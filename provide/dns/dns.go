package dns

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/miekg/dns"
	status_neko "github.com/songzhibin97/status-neko"
)

type ResourceType string

var (
	_               status_neko.Monitor = (*DNS)(nil)
	providerDNSName                     = "dns"

	ResourceTypeA     ResourceType = "A"
	ResourceTypeAAAA  ResourceType = "AAAA"
	ResourceTypeCAA   ResourceType = "CAA"
	ResourceTypeCNAME ResourceType = "CNAME"
	ResourceTypeMX    ResourceType = "MX"
	ResourceTypeNS    ResourceType = "NS"
	ResourceTypePTR   ResourceType = "PTR"
	ResourceTypeSOA   ResourceType = "SOA"
	ResourceTypeSRV   ResourceType = "SRV"
	ResourceTypeTXT   ResourceType = "TXT"
)

type Config struct {
	Host         string       `json:"host"`
	Port         int          `json:"port"`
	ParseServer  string       `json:"parse_server"`  // 解析服务器
	ResourceType ResourceType `json:"resource_type"` // 资源类型
}

type DNS struct {
	config Config
}

func NewDNS(config Config) *DNS {
	return &DNS{
		config: config,
	}
}

func (d DNS) Name() string {
	return providerDNSName
}

func (d DNS) Check(ctx context.Context) (interface{}, error) {
	if d.config.Port == 0 {
		d.config.Port = 53 // 使用默认 DNS 端口
	}

	if d.config.ParseServer == "" {
		d.config.ParseServer = "8.8.8.8" // 如果未指定，使用 Google 的公共 DNS 服务器
	}

	c := new(dns.Client)
	c.Timeout = 5 * time.Second // 设置超时

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(d.config.Host), resourceTypeToInt(d.config.ResourceType))
	m.RecursionDesired = true

	serverAddr := net.JoinHostPort(d.config.ParseServer, strconv.Itoa(d.config.Port))

	start := time.Now()
	r, rtt, err := c.ExchangeContext(ctx, m, serverAddr)
	if err != nil {
		return nil, fmt.Errorf("DNS query failed: %w", err)
	}

	// 如果返回的应答为空，则认为查询失败
	if len(r.Answer) == 0 {
		return nil, fmt.Errorf("no DNS answers for host: %s", d.config.Host)
	}

	result := map[string]interface{}{
		"host":            d.config.Host,
		"parse_server":    d.config.ParseServer,
		"resource_type":   string(d.config.ResourceType),
		"latency":         rtt.String(),
		"resolution_time": time.Since(start).String(),
		"answers":         []string{},
	}

	for _, ans := range r.Answer {
		result["answers"] = append(result["answers"].([]string), ans.String())
	}

	return result, nil
}

func resourceTypeToInt(rt ResourceType) uint16 {
	switch rt {
	case ResourceTypeA:
		return dns.TypeA
	case ResourceTypeAAAA:
		return dns.TypeAAAA
	case ResourceTypeCAA:
		return dns.TypeCAA
	case ResourceTypeCNAME:
		return dns.TypeCNAME
	case ResourceTypeMX:
		return dns.TypeMX
	case ResourceTypeNS:
		return dns.TypeNS
	case ResourceTypePTR:
		return dns.TypePTR
	case ResourceTypeSOA:
		return dns.TypeSOA
	case ResourceTypeSRV:
		return dns.TypeSRV
	case ResourceTypeTXT:
		return dns.TypeTXT
	default:
		return dns.TypeA // 默认使用 A 记录
	}
}
