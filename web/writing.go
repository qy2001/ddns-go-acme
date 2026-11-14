package web

import (
	"encoding/json"
	"strings"

	"github.com/jeessy2/ddns-go/v6/config"
)

const VersionEnv = "DDNS_GO_VERSION"

// js中的dns配置
type dnsConf4JS struct {
	Name             string
	DnsName          string
	DnsID            string
	DnsSecret        string
	DnsExtParam      string
	TTL              string
	Ipv4Enable       bool
	Ipv4GetType      string
	Ipv4Url          string
	Ipv4NetInterface string
	Ipv4Cmd          string
	Ipv4Domains      string
	Ipv6Enable       bool
	Ipv6GetType      string
	Ipv6Url          string
	Ipv6NetInterface string
	Ipv6Cmd          string
	Ipv6Reg          string
	Ipv6Domains      string
	HttpInterface    string
}

func getDnsConfStr(dnsConf []config.DnsConfig) string {
	byt, _ := json.Marshal(getDnsConf4JS(dnsConf))
	return string(byt)
}

func getDnsConf4JS(dnsConf []config.DnsConfig) []dnsConf4JS {
	dnsConfArray := []dnsConf4JS{}
	for _, conf := range dnsConf {
		// 已存在配置文件，隐藏真实的ID、Secret
		idHide, secretHide := getHideIDSecret(&conf)
		dnsConfArray = append(dnsConfArray, dnsConf4JS{
			Name:             conf.Name,
			DnsName:          conf.DNS.Name,
			DnsID:            idHide,
			DnsSecret:        secretHide,
			DnsExtParam:      conf.DNS.ExtParam,
			TTL:              conf.TTL,
			Ipv4Enable:       conf.Ipv4.Enable,
			Ipv4GetType:      conf.Ipv4.GetType,
			Ipv4Url:          conf.Ipv4.URL,
			Ipv4NetInterface: conf.Ipv4.NetInterface,
			Ipv4Cmd:          conf.Ipv4.Cmd,
			Ipv4Domains:      strings.Join(conf.Ipv4.Domains, "\r\n"),
			Ipv6Enable:       conf.Ipv6.Enable,
			Ipv6GetType:      conf.Ipv6.GetType,
			Ipv6Url:          conf.Ipv6.URL,
			Ipv6NetInterface: conf.Ipv6.NetInterface,
			Ipv6Cmd:          conf.Ipv6.Cmd,
			Ipv6Reg:          conf.Ipv6.Ipv6Reg,
			Ipv6Domains:      strings.Join(conf.Ipv6.Domains, "\r\n"),
			HttpInterface:    conf.HttpInterface,
		})
	}
	return dnsConfArray
}

// 显示的数量
const displayCount int = 3

// hideIDSecret 隐藏真实的ID、Secret
func getHideIDSecret(conf *config.DnsConfig) (idHide string, secretHide string) {
	if len(conf.DNS.ID) > displayCount && conf.DNS.Name != "callback" {
		idHide = conf.DNS.ID[:displayCount] + strings.Repeat("*", len(conf.DNS.ID)-displayCount)
	} else {
		idHide = conf.DNS.ID
	}
	if len(conf.DNS.Secret) > displayCount && conf.DNS.Name != "callback" {
		secretHide = conf.DNS.Secret[:displayCount] + strings.Repeat("*", len(conf.DNS.Secret)-displayCount)
	} else {
		secretHide = conf.DNS.Secret
	}
	return
}
