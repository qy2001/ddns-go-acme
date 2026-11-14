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

	"github.com/jeessy2/ddns-go/v6/config"
	"software.sslmate.com/src/go-pkcs12"
)

const readmeContent = `ddns-go 证书使用说明

文件说明：
  fullchain.pem  证书链（服务器证书 + 中间证书），Nginx/Apache 使用
  privkey.pem    私钥
  cert.pem       服务器证书（不含中间证书）
  cert.pfx       PKCS#12 格式，Windows/IIS 导入使用

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
