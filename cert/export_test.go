package cert

import (
	"archive/zip"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jeessy2/ddns-go/v6/config"
	"software.sslmate.com/src/go-pkcs12"
)

func TestExportAndStatus(t *testing.T) {
	dir := t.TempDir()
	c := config.CertConfig{Name: "test-cert", Domains: []string{"example.com"}, CertDir: dir}
	writeTestCert(t, dir)

	status := StatusOf(c)
	if !status.Issued {
		t.Fatalf("expected issued status, got %+v", status)
	}
	if status.DaysLeft < 80 {
		t.Fatalf("unexpected days left: %d", status.DaysLeft)
	}

	pfxData, filename, err := ExportPFX(c, "secret")
	if err != nil {
		t.Fatalf("export pfx: %v", err)
	}
	if filename != "test-cert.pfx" {
		t.Fatalf("unexpected filename: %s", filename)
	}
	key, leaf, _, err := pkcs12.DecodeChain(pfxData, "secret")
	if err != nil {
		t.Fatalf("decode pfx: %v", err)
	}
	if key == nil || leaf.Subject.CommonName != "example.com" {
		t.Fatalf("unexpected pfx content: %+v", leaf)
	}

	pemZip, _, err := ExportPEM(c)
	if err != nil {
		t.Fatalf("export pem: %v", err)
	}
	names := zipNames(t, pemZip)
	for _, want := range []string{certFileName, fullChainFileName, privateKeyFileName} {
		if !names[want] {
			t.Fatalf("pem zip missing %s, got %v", want, names)
		}
	}

	allZip, _, err := ExportAll(c, "")
	if err != nil {
		t.Fatalf("export all: %v", err)
	}
	allNames := zipNames(t, allZip)
	for _, want := range []string{"cert.pfx", "README.txt", certFileName} {
		if !allNames[want] {
			t.Fatalf("all zip missing %s, got %v", want, allNames)
		}
	}
}

func TestSafeNameDistinctForChinese(t *testing.T) {
	a := SafeName("我的证书")
	b := SafeName("备用证书")
	if a == b || !strings.HasPrefix(a, "cert-") {
		t.Fatalf("unexpected safe names: %s %s", a, b)
	}
	if SafeName("nas-cert") != "nas-cert" {
		t.Fatalf("ascii name should be unchanged")
	}
}

func writeTestCert(t *testing.T, dir string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "example.com"},
		DNSNames:              []string{"example.com"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(90 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		certFileName:       certPEM,
		fullChainFileName:  certPEM,
		privateKeyFileName: keyPEM,
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func zipNames(t *testing.T, data []byte) map[string]bool {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	return names
}
