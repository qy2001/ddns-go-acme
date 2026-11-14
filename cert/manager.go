package cert

import (
	"crypto"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/jeessy2/ddns-go/v6/config"
	"github.com/jeessy2/ddns-go/v6/util"
)

const (
	// DefaultRenewBefore 默认提前续期天数
	DefaultRenewBefore = config.DefaultCertRenewBefore
	// AutoCheckInterval 自动检查证书的间隔
	AutoCheckInterval  = 6 * time.Hour
	metaFileName       = "meta.json"
	certFileName       = "cert.pem"
	fullChainFileName  = "fullchain.pem"
	privateKeyFileName = "privkey.pem"
)

// Meta 证书元信息
type Meta struct {
	Domains     []string  `json:"domains"`
	CA          string    `json:"ca"`
	NotBefore   time.Time `json:"notBefore"`
	NotAfter    time.Time `json:"notAfter"`
	LastIssue   time.Time `json:"lastIssue"`
	LastAttempt time.Time `json:"lastAttempt"`
	LastError   string    `json:"lastError"`
}

// Status 证书状态（用于前端展示）
type Status struct {
	Name      string    `json:"Name"`
	Domains   []string  `json:"Domains"`
	Enable    bool      `json:"Enable"`
	CA        string    `json:"CA"`
	Issued    bool      `json:"Issued"`
	NotBefore time.Time `json:"NotBefore"`
	NotAfter  time.Time `json:"NotAfter"`
	DaysLeft  int       `json:"DaysLeft"`
	Issuing   bool      `json:"Issuing"`
	LastError string    `json:"LastError"`
	LastIssue time.Time `json:"LastIssue"`
	CertDir   string    `json:"CertDir"`
}

var (
	// 同一时间只允许一个 ACME 申请任务，避免触发 CA 限流
	issueLock sync.Mutex
	stateMu   sync.Mutex
	issuing   = map[string]bool{}
)

// Name 返回配置名称，未填写时使用域名拼接
func Name(c config.CertConfig) string {
	if strings.TrimSpace(c.Name) != "" {
		return strings.TrimSpace(c.Name)
	}
	return strings.Join(c.Domains, ",")
}

// SafeName 将名称转换为可用于目录/文件名的字符串
func SafeName(name string) string {
	original := strings.TrimSpace(name)
	if original == "" {
		return "cert"
	}
	cleaned := safeNameReg.ReplaceAllString(original, "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		cleaned = "cert"
	}
	// 名称包含中文等字符时追加哈希，避免不同配置落到同一目录
	if cleaned != original {
		sum := sha1.Sum([]byte(original))
		cleaned = cleaned + "-" + hex.EncodeToString(sum[:4])
	}
	return cleaned
}

var safeNameReg = regexp.MustCompile(`[^0-9A-Za-z._-]+`)

// BaseDir 证书默认保存根目录（与配置文件同级的隐藏目录）
func BaseDir() string {
	return filepath.Join(filepath.Dir(util.GetConfigFilePath()), ".ddns_go_certs")
}

// Dir 返回指定证书配置的保存目录
func Dir(c config.CertConfig) string {
	if strings.TrimSpace(c.CertDir) == "" {
		return filepath.Join(BaseDir(), SafeName(Name(c)))
	}
	dir := strings.TrimSpace(c.CertDir)
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(BaseDir(), dir)
	}
	return dir
}

// SupportedProvider 是否支持 DNS-01 验证的服务商
func SupportedProvider(name string) bool {
	switch name {
	case "cloudflare", "dnspod", "tencentcloud":
		return true
	default:
		return false
	}
}

// List 返回所有证书配置的状态
func List() []Status {
	conf, err := config.GetConfigCached()
	if err != nil {
		return []Status{}
	}
	result := make([]Status, 0, len(conf.CertConf))
	for _, c := range conf.CertConf {
		result = append(result, StatusOf(c))
	}
	return result
}

// StatusOf 返回单个证书配置的状态
func StatusOf(c config.CertConfig) Status {
	meta, _ := readMeta(Dir(c))
	status := Status{
		Name:      Name(c),
		Domains:   c.Domains,
		Enable:    c.Enable,
		CA:        c.CA,
		CertDir:   Dir(c),
		LastError: meta.LastError,
		LastIssue: meta.LastIssue,
		Issuing:   isIssuing(Name(c)),
	}
	leaf, err := readLeaf(c)
	if err == nil {
		status.Issued = true
		status.NotBefore = leaf.NotBefore
		status.NotAfter = leaf.NotAfter
		status.DaysLeft = int(time.Until(leaf.NotAfter).Hours() / 24)
	}
	return status
}

// RunTimer 定时检查并续期证书
func RunTimer(interval time.Duration) {
	if interval <= 0 {
		interval = AutoCheckInterval
	}
	for {
		RunOnce()
		time.Sleep(interval)
	}
}

// RunOnce 检查一次所有证书配置，必要时异步申请/续期
func RunOnce() {
	conf, err := config.GetConfigCached()
	if err != nil {
		return
	}
	for _, c := range conf.CertConf {
		if !c.Enable {
			continue
		}
		if !NeedRenew(c) {
			continue
		}
		certConf := c
		go func() {
			if err := issue(certConf, conf, false); err != nil {
				util.Log("证书申请失败! 域名: %s, 异常信息: %s", strings.Join(certConf.Domains, ","), err)
			}
		}()
	}
}

// Issue 立即申请或续期证书（忽略是否到期）
func Issue(c config.CertConfig, conf config.Config) error {
	return issue(c, conf, true)
}

func issue(c config.CertConfig, conf config.Config, force bool) error {
	if err := validate(c, conf); err != nil {
		recordFailure(c, err)
		return err
	}
	name := Name(c)
	if isIssuing(name) {
		return errors.New("该证书正在申请中，请稍后再试")
	}
	if !force && !NeedRenew(c) {
		return nil
	}

	issueLock.Lock()
	defer issueLock.Unlock()
	// 等待锁期间可能已被其它任务完成
	if !force && !NeedRenew(c) {
		return nil
	}

	setIssuing(name, true)
	defer setIssuing(name, false)

	if err := issueNow(c, conf); err != nil {
		recordFailure(c, err)
		return err
	}
	return nil
}

func issueNow(c config.CertConfig, conf config.Config) error {
	accountKey, err := accountPrivateKey(c.CA, c.Email)
	if err != nil {
		return err
	}

	user := &acmeUser{email: c.Email, key: accountKey}
	legoConfig := lego.NewConfig(user)
	legoConfig.CADirURL = caURL(c.CA)
	legoConfig.UserAgent = "ddns-go"
	legoConfig.Certificate.KeyType = keyType(c.KeyType)

	client, err := lego.NewClient(legoConfig)
	if err != nil {
		return err
	}

	provider, err := newDNSProvider(conf.DnsConf[c.DNSProvider])
	if err != nil {
		return err
	}
	// 使用公共 DNS 检查 TXT 记录传播，避免本机 DNS 异常导致验证失败
	err = client.Challenge.SetDNS01Provider(provider,
		dns01.AddRecursiveNameservers([]string{"1.1.1.1:53", "8.8.8.8:53"}))
	if err != nil {
		return err
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return fmt.Errorf("注册 ACME 账号失败: %w", err)
	}
	user.reg = reg

	util.Log("开始申请证书, 域名: %s", strings.Join(c.Domains, ","))
	res, err := client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: c.Domains,
		Bundle:  true,
	})
	if err != nil {
		return fmt.Errorf("申请证书失败: %w", err)
	}

	leaf, err := saveCertificate(c, res)
	if err != nil {
		return err
	}

	meta := Meta{
		Domains:     c.Domains,
		CA:          c.CA,
		NotBefore:   leaf.NotBefore,
		NotAfter:    leaf.NotAfter,
		LastIssue:   time.Now(),
		LastAttempt: time.Now(),
	}
	if err := writeMeta(Dir(c), meta); err != nil {
		return err
	}
	util.Log("证书申请成功! 域名: %s, 有效期至: %s", strings.Join(c.Domains, ","), leaf.NotAfter.Format("2006-01-02 15:04:05"))
	return nil
}

// NeedRenew 证书是否需要申请/续期
func NeedRenew(c config.CertConfig) bool {
	leaf, err := readLeaf(c)
	if err != nil || leaf == nil {
		return true
	}

	renewBefore := c.RenewBefore
	if renewBefore <= 0 {
		renewBefore = DefaultRenewBefore
	}
	return time.Until(leaf.NotAfter) <= time.Duration(renewBefore)*24*time.Hour
}

func recordFailure(c config.CertConfig, err error) {
	meta, _ := readMeta(Dir(c))
	meta.LastAttempt = time.Now()
	meta.LastError = err.Error()
	if len(meta.Domains) == 0 {
		meta.Domains = c.Domains
	}
	if meta.CA == "" {
		meta.CA = c.CA
	}
	_ = writeMeta(Dir(c), meta)
}

func validate(c config.CertConfig, conf config.Config) error {
	if len(c.Domains) == 0 {
		return errors.New("请填写证书域名")
	}
	for _, d := range c.Domains {
		d = strings.TrimSpace(d)
		if d == "" || !strings.Contains(d, ".") {
			return fmt.Errorf("域名 %s 不正确", d)
		}
	}
	if strings.TrimSpace(c.Email) == "" {
		return errors.New("请填写 ACME 账号邮箱")
	}
	if c.DNSProvider < 0 || c.DNSProvider >= len(conf.DnsConf) {
		return errors.New("请选择用于 DNS-01 验证的 DNS 配置")
	}
	if !SupportedProvider(conf.DnsConf[c.DNSProvider].DNS.Name) {
		return fmt.Errorf("DNS 服务商 %s 暂不支持证书申请，请使用 Cloudflare、DNSPod 或腾讯云", conf.DnsConf[c.DNSProvider].DNS.Name)
	}
	return nil
}

func readLeaf(c config.CertConfig) (*x509.Certificate, error) {
	certPEM, err := os.ReadFile(filepath.Join(Dir(c), certFileName))
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("证书内容无效")
	}
	return x509.ParseCertificate(block.Bytes)
}

func readMeta(dir string) (Meta, error) {
	var meta Meta
	byt, err := os.ReadFile(filepath.Join(dir, metaFileName))
	if err != nil {
		return meta, err
	}
	err = json.Unmarshal(byt, &meta)
	return meta, err
}

func writeMeta(dir string, meta Meta) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	byt, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, metaFileName+".tmp")
	if err := os.WriteFile(tmp, byt, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, metaFileName))
}

func saveCertificate(c config.CertConfig, res *certificate.Resource) (*x509.Certificate, error) {
	certs, err := certcrypto.ParsePEMBundle(res.Certificate)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}
	if len(certs) == 0 {
		return nil, errors.New("证书内容为空")
	}
	if len(certs) == 1 && len(res.IssuerCertificate) > 0 {
		issuer, err := certcrypto.ParsePEMBundle(res.IssuerCertificate)
		if err == nil {
			certs = append(certs, issuer...)
		}
	}

	dir := Dir(c)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	leafPEM := encodeCert(certs[0])
	if err := os.WriteFile(filepath.Join(dir, certFileName), leafPEM, 0600); err != nil {
		return nil, err
	}

	fullChain := make([]byte, 0, 4096)
	for _, cert := range certs {
		fullChain = append(fullChain, encodeCert(cert)...)
	}
	if err := os.WriteFile(filepath.Join(dir, fullChainFileName), fullChain, 0600); err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(dir, privateKeyFileName), res.PrivateKey, 0600); err != nil {
		return nil, err
	}
	return certs[0], nil
}

func encodeCert(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}

func accountPrivateKey(ca, email string) (crypto.PrivateKey, error) {
	path := filepath.Join(BaseDir(), "accounts", SafeName(ca+"_"+email)+".key")
	if data, err := os.ReadFile(path); err == nil {
		return certcrypto.ParsePEMPrivateKey(data)
	}
	key, err := certcrypto.GeneratePrivateKey(certcrypto.EC256)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, certcrypto.PEMEncode(key), 0600); err != nil {
		return nil, err
	}
	return key, nil
}

func caURL(ca string) string {
	switch ca {
	case "letsencrypt-staging":
		return lego.LEDirectoryStaging
	default:
		return lego.LEDirectoryProduction
	}
}

func keyType(kt string) certcrypto.KeyType {
	switch kt {
	case "ec384":
		return certcrypto.EC384
	case "rsa2048":
		return certcrypto.RSA2048
	case "rsa4096":
		return certcrypto.RSA4096
	default:
		return certcrypto.EC256
	}
}

func isIssuing(name string) bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	return issuing[name]
}

func setIssuing(name string, value bool) {
	stateMu.Lock()
	defer stateMu.Unlock()
	issuing[name] = value
}

// acmeUser 实现 lego 的 registration.User 接口
type acmeUser struct {
	email string
	key   crypto.PrivateKey
	reg   *registration.Resource
}

func (u *acmeUser) GetEmail() string                        { return u.email }
func (u *acmeUser) GetRegistration() *registration.Resource { return u.reg }
func (u *acmeUser) GetPrivateKey() crypto.PrivateKey        { return u.key }
