package cert

import (
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/dnspod"
	"github.com/go-acme/lego/v4/providers/dns/tencentcloud"
	"github.com/jeessy2/ddns-go/v6/config"
)

// newDNSProvider 将 ddns-go 的 DNS 配置转换为 lego 的 DNS-01 Provider
func newDNSProvider(dnsConf config.DnsConfig) (challenge.Provider, error) {
	switch dnsConf.DNS.Name {
	case "cloudflare":
		cfg := cloudflare.NewDefaultConfig()
		cfg.AuthToken = dnsConf.DNS.Secret
		cfg.PropagationTimeout = 5 * time.Minute
		return cloudflare.NewDNSProviderConfig(cfg)
	case "dnspod":
		cfg := dnspod.NewDefaultConfig()
		// DNSPod 的 API Key 格式为 "ID,Token"
		cfg.LoginToken = dnsConf.DNS.ID + "," + dnsConf.DNS.Secret
		cfg.PropagationTimeout = 5 * time.Minute
		return dnspod.NewDNSProviderConfig(cfg)
	case "tencentcloud":
		cfg := tencentcloud.NewDefaultConfig()
		cfg.SecretID = dnsConf.DNS.ID
		cfg.SecretKey = dnsConf.DNS.Secret
		cfg.PropagationTimeout = 5 * time.Minute
		return tencentcloud.NewDNSProviderConfig(cfg)
	default:
		return nil, fmt.Errorf("DNS 服务商 %s 暂不支持证书 DNS-01 验证", dnsConf.DNS.Name)
	}
}
