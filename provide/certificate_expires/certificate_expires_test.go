package certificate_expires

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"testing"
	"time"
)

type MockHTTPClient struct {
	GetFunc func(url string) (*http.Response, error)
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	return m.GetFunc(url)
}

func TestCertificateExpires(t *testing.T) {
	// 创建一个模拟的证书
	cert := &tls.Certificate{
		Leaf: &x509.Certificate{
			NotAfter: time.Now().Add(24 * time.Hour),
		},
	}

	// 创建一个模拟的 HTTP 客户端
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return &http.Response{
				TLS: &tls.ConnectionState{
					PeerCertificates: []*x509.Certificate{cert.Leaf},
				},
				Body: http.NoBody,
			}, nil
		},
	}

	// 创建一个新的 CertificateExpires 实例
	ce := NewCertificateExpires("https://example.com", SetClient(mockClient))

	// 测试 Name 方法
	if ce.Name() != providerCertificateExpiresName {
		t.Errorf("期望名称为 %s，实际得到 %s", providerCertificateExpiresName, ce.Name())
	}

	// 测试 Check 方法
	ctx := context.Background()
	result, err := ce.Check(ctx)
	if err != nil {
		t.Fatalf("Check 方法返回错误：%v", err)
	}

	expiry, ok := result.(time.Time)
	if !ok {
		t.Fatalf("期望结果类型为 time.Time，实际得到 %T", result)
	}

	// 验证返回的过期时间是否正确
	expectedExpiry := cert.Leaf.NotAfter
	if !expiry.Equal(expectedExpiry) {
		t.Errorf("期望过期时间为 %v，实际得到 %v", expectedExpiry, expiry)
	}

	// 测试 getDomainFromURL 函数
	testCases := []struct {
		url      string
		expected string
	}{
		{"https://example.com", "example.com"},
		{"https://example.com:443", "example.com"},
		{"http://subdomain.example.com:8080", "subdomain.example.com"},
	}

	for _, tc := range testCases {
		domain, err := getDomainFromURL(tc.url)
		if err != nil {
			t.Errorf("getDomainFromURL(%s) 返回错误：%v", tc.url, err)
		}
		if domain != tc.expected {
			t.Errorf("getDomainFromURL(%s) = %s，期望值为 %s", tc.url, domain, tc.expected)
		}
	}

	// 测试空客户端的情况
	ceNilClient := NewCertificateExpires("https://example.com", SetClient(nil))
	_, err = ceNilClient.Check(ctx)
	if err == nil || err.Error() != "HTTP client is not initialized" {
		t.Errorf("对于空客户端，期望错误 'HTTP client is not initialized'，实际得到 %v", err)
	}

	// 测试 nil response 的情况
	nilResponseClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return nil, nil
		},
	}
	ceNilResponse := NewCertificateExpires("https://example.com", SetClient(nilResponseClient))
	_, err = ceNilResponse.Check(ctx)
	if err == nil || err.Error() != "received nil response" {
		t.Errorf("对于 nil response，期望错误 'received nil response'，实际得到 %v", err)
	}

	// 测试 Get 方法返回错误的情况
	errorClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}
	ceError := NewCertificateExpires("https://example.com", SetClient(errorClient))
	_, err = ceError.Check(ctx)
	if err == nil || err.Error() != "network error" {
		t.Errorf("对于网络错误，期望错误 'network error'，实际得到 %v", err)
	}
}
