package cert

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeessy2/ddns-go/v6/config"
	"github.com/pavlo-v-chernykh/keystore-go/v4"
	"software.sslmate.com/src/go-pkcs12"
)

const readmeContent = `ddns-go 证书使用说明

文件说明：
  fullchain.pem  证书链（服务器证书 + 中间证书），Nginx/Apache 使用
  privkey.pem    私钥
  cert.pem       服务器证书（不含中间证书）
  chain.pem      中间证书链（不含服务器证书），Apache SSLCertificateChainFile 使用
  cert.der       DER 二进制编码的服务器证书
  cert.pfx       PKCS#12 格式，Windows/IIS 导入使用
  cert.jks       Java KeyStore，Tomcat 等 Java 应用使用（无密码时默认 changeit）

Nginx:
  ssl_certificate     /path/to/fullchain.pem;
  ssl_certificate_key /path/to/privkey.pem;

Apache:
  SSLCertificateFile    /path/to/fullchain.pem
  SSLCertificateKeyFile /path/to/privkey.pem

IIS / Windows:
  导入 cert.pfx 到“个人”证书存储，并勾选“允许导出私钥”。
`

// ExportPEM 导出 PEM 文件压缩包
func ExportPEM(c config.CertConfig) ([]byte, string, error) {
	files, err := readPEMFiles(Dir(c))
	if err != nil {
		return nil, "", err
	}
	buf, err := zipFiles(files)
	if err != nil {
		return nil, "", err
	}
	return buf, SafeName(Name(c)) + "-pem.zip", nil
}

// ExportPFX 导出 PKCS#12 文件
func ExportPFX(c config.CertConfig, password string) ([]byte, string, error) {
	pfx, err := buildPFX(c, password)
	if err != nil {
		return nil, "", err
	}
	return pfx, SafeName(Name(c)) + ".pfx", nil
}

// ExportAll 导出全部格式并附带使用说明
func ExportAll(c config.CertConfig, password string) ([]byte, string, error) {
	files, err := readPEMFiles(Dir(c))
	if err != nil {
		return nil, "", err
	}
	pfx, err := buildPFX(c, password)
	if err != nil {
		return nil, "", err
	}
	files = append(files, zipFile{Name: "cert.pfx", Data: pfx})
	contents, _ := PEMContents(c)
	files = append(files, zipFile{Name: "chain.pem", Data: []byte(contents["chain"])})
	if jks, jerr := buildJKS(c, password); jerr == nil {
		files = append(files, zipFile{Name: "cert.jks", Data: jks})
	}
	if der, derr := leafDER(c); derr == nil {
		files = append(files, zipFile{Name: "cert.der", Data: der})
	}
	files = append(files, zipFile{Name: "README.txt", Data: []byte(readmeContent)})

	buf, err := zipFiles(files)
	if err != nil {
		return nil, "", err
	}
	return buf, SafeName(Name(c)) + ".zip", nil
}

// PEMContents 返回 fullchain / privkey / cert / chain 的 PEM 文本内容，用于生成终端部署命令
// chain 为中间证书链（fullchain 去掉服务器证书），Apache 等需要单独配置
func PEMContents(c config.CertConfig) (map[string]string, error) {
	dir := Dir(c)
	contents := make(map[string]string, 4)
	for key, name := range map[string]string{
		"fullchain": fullChainFileName,
		"privkey":   privateKeyFileName,
		"cert":      certFileName,
	} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("读取 %s 失败: %w", name, err)
		}
		contents[key] = string(data)
	}
	contents["chain"] = chainFromFullChain(contents["fullchain"])
	return contents, nil
}

// chainFromFullChain 从完整证书链中去掉第一个（服务器）证书，返回中间证书链
func chainFromFullChain(fullChain string) string {
	var chain strings.Builder
	rest := []byte(fullChain)
	index := 0
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		index++
		if index > 1 {
			chain.Write(pem.EncodeToMemory(block))
		}
	}
	return chain.String()
}

func readPEMFiles(dir string) ([]zipFile, error) {
	names := []string{fullChainFileName, privateKeyFileName, certFileName}
	files := make([]zipFile, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("读取 %s 失败: %w", name, err)
		}
		files = append(files, zipFile{Name: name, Data: data})
	}
	return files, nil
}

func buildPFX(c config.CertConfig, password string) ([]byte, error) {
	dir := Dir(c)
	fullChain, err := os.ReadFile(filepath.Join(dir, fullChainFileName))
	if err != nil {
		return nil, fmt.Errorf("读取证书失败: %w", err)
	}
	privateKey, err := os.ReadFile(filepath.Join(dir, privateKeyFileName))
	if err != nil {
		return nil, fmt.Errorf("读取私钥失败: %w", err)
	}

	pair, err := tls.X509KeyPair(fullChain, privateKey)
	if err != nil {
		return nil, fmt.Errorf("证书与私钥不匹配: %w", err)
	}
	if len(pair.Certificate) == 0 {
		return nil, errors.New("证书内容为空")
	}

	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, err
	}
	caCerts := make([]*x509.Certificate, 0, len(pair.Certificate)-1)
	for _, der := range pair.Certificate[1:] {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, err
		}
		caCerts = append(caCerts, cert)
	}

	pfx, err := pkcs12.Modern.Encode(pair.PrivateKey, leaf, caCerts, password)
	if err != nil {
		return nil, fmt.Errorf("生成 PFX 失败: %w", err)
	}
	return pfx, nil
}

func zipFiles(files []zipFile) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f.Name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(f.Data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type zipFile struct {
	Name string
	Data []byte
}

// ExportSingle 导出单个文件：fullchain / privkey / cert / chain（PEM）或 der（二进制服务器证书）
func ExportSingle(c config.CertConfig, kind string) ([]byte, string, error) {
	base := SafeName(Name(c))
	if kind == "der" {
		der, err := leafDER(c)
		return der, base + ".der", err
	}
	if kind == "chain" {
		fullChain, err := os.ReadFile(filepath.Join(Dir(c), fullChainFileName))
		if err != nil {
			return nil, "", fmt.Errorf("读取证书失败: %w", err)
		}
		return []byte(chainFromFullChain(string(fullChain))), base + "-chain.pem", nil
	}
	fileName, ok := map[string]string{
		"fullchain": fullChainFileName,
		"privkey":   privateKeyFileName,
		"cert":      certFileName,
	}[kind]
	if !ok {
		return nil, "", errors.New("不支持的文件类型")
	}
	data, err := os.ReadFile(filepath.Join(Dir(c), fileName))
	if err != nil {
		return nil, "", fmt.Errorf("读取 %s 失败: %w", fileName, err)
	}
	return data, base + "-" + kind + ".pem", nil
}

// ExportJKS 导出 Java KeyStore（Tomcat 等 Java 应用）
func ExportJKS(c config.CertConfig, password string) ([]byte, string, error) {
	jks, err := buildJKS(c, password)
	if err != nil {
		return nil, "", err
	}
	return jks, SafeName(Name(c)) + ".jks", nil
}

// leafDER 返回 DER 二进制编码的服务器证书
func leafDER(c config.CertConfig) ([]byte, error) {
	fullChain, err := os.ReadFile(filepath.Join(Dir(c), fullChainFileName))
	if err != nil {
		return nil, fmt.Errorf("读取证书失败: %w", err)
	}
	block, _ := pem.Decode(fullChain)
	if block == nil {
		return nil, errors.New("证书内容为空")
	}
	return block.Bytes, nil
}

// buildJKS 生成 JKS，密码为空时使用 Java 默认密码 changeit
func buildJKS(c config.CertConfig, password string) ([]byte, error) {
	if password == "" {
		password = "changeit"
	}
	dir := Dir(c)
	fullChain, err := os.ReadFile(filepath.Join(dir, fullChainFileName))
	if err != nil {
		return nil, fmt.Errorf("读取证书失败: %w", err)
	}
	privateKey, err := os.ReadFile(filepath.Join(dir, privateKeyFileName))
	if err != nil {
		return nil, fmt.Errorf("读取私钥失败: %w", err)
	}
	pair, err := tls.X509KeyPair(fullChain, privateKey)
	if err != nil {
		return nil, fmt.Errorf("证书与私钥不匹配: %w", err)
	}
	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(pair.PrivateKey)
	if err != nil {
		return nil, err
	}
	chain := make([]keystore.Certificate, 0, len(pair.Certificate))
	for _, der := range pair.Certificate {
		chain = append(chain, keystore.Certificate{Type: "X509", Content: der})
	}
	ks := keystore.New()
	entry := keystore.PrivateKeyEntry{
		CreationTime:     time.Now(),
		PrivateKey:       pkcs8Key,
		CertificateChain: chain,
	}
	if err := ks.SetPrivateKeyEntry(Name(c), entry, []byte(password)); err != nil {
		return nil, fmt.Errorf("生成 JKS 失败: %w", err)
	}
	var buf bytes.Buffer
	if err := ks.Store(&buf, []byte(password)); err != nil {
		return nil, fmt.Errorf("生成 JKS 失败: %w", err)
	}
	return buf.Bytes(), nil
}
