package web

import (
	"net/http"
	"os"

	"github.com/jeessy2/ddns-go/v6/config"
)

type configForJS struct {
	Version           string
	Username          string
	Lang              string
	NotAllowWanAccess bool
	config.Webhook
	DnsConf                []dnsConf4JS
	CertConf               []config.CertConfig
	Ipv4                   []config.NetInterface
	Ipv6                   []config.NetInterface
	AllInterfaces          []config.NetInterface
	CertSupportedProviders []string
}

// ConfigAPI 返回完整配置（ID/Secret 已脱敏）
func ConfigAPI(w http.ResponseWriter, r *http.Request) {
	conf, err := config.GetConfigCached()
	if err != nil {
		// 与旧页面保持一致：无配置时默认禁止公网访问
		conf.NotAllowWanAccess = true
	}

	ipv4, ipv6, _ := config.GetNetInterface()

	allIfaceNames := map[string]bool{}
	for _, iface := range ipv4 {
		allIfaceNames[iface.Name] = true
	}
	for _, iface := range ipv6 {
		allIfaceNames[iface.Name] = true
	}
	allInterfaces := make([]config.NetInterface, 0, len(allIfaceNames))
	for name := range allIfaceNames {
		allInterfaces = append(allInterfaces, config.NetInterface{Name: name})
	}

	writeJSON(w, http.StatusOK, configForJS{
		Version:                os.Getenv(VersionEnv),
		Username:               conf.Username,
		Lang:                   conf.Lang,
		NotAllowWanAccess:      conf.NotAllowWanAccess,
		Webhook:                conf.Webhook,
		DnsConf:                getDnsConf4JS(conf.DnsConf),
		CertConf:               conf.CertConf,
		Ipv4:                   ipv4,
		Ipv6:                   ipv6,
		AllInterfaces:          allInterfaces,
		CertSupportedProviders: []string{"cloudflare", "dnspod", "tencentcloud"},
	})
}
