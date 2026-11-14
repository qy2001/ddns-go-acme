package web

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jeessy2/ddns-go/v6/cert"
	"github.com/jeessy2/ddns-go/v6/config"
	"github.com/jeessy2/ddns-go/v6/util"
)

// CertList 返回所有证书状态
func CertList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"result": "ok", "data": cert.List()})
}

// CertIssue 立即申请/续期指定证书
func CertIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"Name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"result": util.LogStr("数据解析失败, 请刷新页面重试")})
		return
	}

	conf, _ := config.GetConfigCached()
	for _, c := range conf.CertConf {
		if cert.Name(c) != req.Name {
			continue
		}
		certConf := c
		go func() {
			if err := cert.Issue(certConf, conf); err != nil {
				util.Log("证书申请失败! 域名: %s, 异常信息: %s", strings.Join(certConf.Domains, ","), err)
			}
		}()
		writeJSON(w, http.StatusOK, map[string]string{"result": "ok"})
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"result": "未找到该证书配置"})
}

// CertContent 返回证书内容，用于前端生成终端部署命令
// 默认返回 fullchain/privkey/cert 的 PEM 文本；format=pfx 时返回 base64 编码的 PFX
func CertContent(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")

	conf, _ := config.GetConfigCached()
	for i := range conf.CertConf {
		if cert.Name(conf.CertConf[i]) != name {
			continue
		}
		if query.Get("format") == "pfx" {
			pfx, _, err := cert.ExportPFX(conf.CertConf[i], query.Get("password"))
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"result": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"result": "ok", "data": map[string]string{
				"pfx": base64.StdEncoding.EncodeToString(pfx),
			}})
			return
		}
		contents, err := cert.PEMContents(conf.CertConf[i])
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"result": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": "ok", "data": contents})
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"result": "未找到该证书配置"})
}

// CertDownload 下载证书文件
func CertDownload(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	format := query.Get("format")
	password := query.Get("password")

	conf, _ := config.GetConfigCached()
	var target *config.CertConfig
	for i := range conf.CertConf {
		if cert.Name(conf.CertConf[i]) == name {
			target = &conf.CertConf[i]
			break
		}
	}
	if target == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"result": "未找到该证书配置"})
		return
	}

	var data []byte
	var filename string
	var err error
	switch format {
	case "pem":
		data, filename, err = cert.ExportPEM(*target)
	case "pfx":
		data, filename, err = cert.ExportPFX(*target, password)
	case "zip":
		data, filename, err = cert.ExportAll(*target, password)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"result": "不支持的导出格式"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"result": err.Error()})
		return
	}

	contentType := "application/zip"
	if format == "pfx" {
		contentType = "application/x-pkcs12"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(data))
}
