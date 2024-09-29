package dns

import (
	"context"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestDNS_Name(t *testing.T) {
	d := DNS{config: Config{Host: "example.com"}}
	assert.Equal(t, providerDNSName, d.Name())
}

func TestDNSCheck(t *testing.T) {
	tests := []struct {
		name         string
		dns          DNS
		expectedHost string
		expectError  bool
	}{
		{
			name: "Valid A Record",
			dns: DNS{
				config: Config{
					Host:         "google.com",
					ParseServer:  "8.8.8.8",     // 使用 Google 公共 DNS
					ResourceType: ResourceTypeA, // 查询 A 记录
				},
			},
			expectedHost: "google.com",
			expectError:  false,
		},
		{
			name: "Invalid Domain",
			dns: DNS{
				config: Config{Host: "nonexistent.domain",
					ParseServer:  "8.8.8.8",     // 使用 Google 公共 DNS
					ResourceType: ResourceTypeA, // 查询 A 记录
				},
			},
			expectedHost: "nonexistent.domain",
			expectError:  true, // 由于域名不存在，应该返回错误
		},
		{
			name: "Timeout Exceeded",
			dns: DNS{
				config: Config{
					Host:         "google.com",
					ParseServer:  "8.8.8.8",     // 使用 Google 公共 DNS
					ResourceType: ResourceTypeA, // 查询 A 记录
				},
			},
			expectedHost: "google.com",
			expectError:  true, // 我们会模拟超时
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// 模拟超时的情况
			if tt.name == "Timeout Exceeded" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 1*time.Millisecond)
				defer cancel()
			}

			result, err := tt.dns.Check(ctx)

			// 如果预期是错误，但没有得到错误
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}

			// 如果没有错误，检查返回结果
			if err == nil {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("result should be a map, got: %T", result)
				}

				// 检查结果中主机名是否正确
				if resultMap["host"] != tt.expectedHost {
					t.Errorf("expected host: %s, got: %s", tt.expectedHost, resultMap["host"])
				}

				// 检查结果是否包含有效的答案（仅在非错误情况）
				answers, ok := resultMap["answers"].([]string)
				if !ok || len(answers) == 0 {
					t.Errorf("expected some answers, got: %v", answers)
				}
			}
		})
	}
}

func TestResourceTypeToInt(t *testing.T) {
	tests := []struct {
		name     string
		rt       ResourceType
		expected uint16
	}{
		{"A", ResourceTypeA, dns.TypeA},
		{"AAAA", ResourceTypeAAAA, dns.TypeAAAA},
		{"CAA", ResourceTypeCAA, dns.TypeCAA},
		{"CNAME", ResourceTypeCNAME, dns.TypeCNAME},
		{"MX", ResourceTypeMX, dns.TypeMX},
		{"NS", ResourceTypeNS, dns.TypeNS},
		{"PTR", ResourceTypePTR, dns.TypePTR},
		{"SOA", ResourceTypeSOA, dns.TypeSOA},
		{"SRV", ResourceTypeSRV, dns.TypeSRV},
		{"TXT", ResourceTypeTXT, dns.TypeTXT},
		{"Invalid", ResourceType("INVALID"), dns.TypeA}, // 默认应该返回 A 记录类型
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resourceTypeToInt(tt.rt)
			assert.Equal(t, tt.expected, result)
		})
	}
}
