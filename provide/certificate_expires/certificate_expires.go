package certificate_expires

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	status_neko "github.com/songzhibin97/status-neko"
)

var (
	_                              status_neko.Monitor = (*CertificateExpires)(nil)
	providerCertificateExpiresName                     = "certificate_expires"
)

type CertificateExpires struct {
	url    string
	option *option
}

type option struct {
	client HTTPClient
}

func SetClient(client HTTPClient) status_neko.Option[*option] {
	return func(o *option) {
		o.client = client
	}
}

type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

func NewCertificateExpires(url string, opts ...status_neko.Option[*option]) *CertificateExpires {
	o := &option{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}

	for _, opt := range opts {
		opt(o)
	}

	return &CertificateExpires{
		url:    url,
		option: o,
	}
}

func (c CertificateExpires) Name() string {
	return providerCertificateExpiresName
}

func (c CertificateExpires) Check(ctx context.Context) (interface{}, error) {
	domain, err := getDomainFromURL(c.url)
	if err != nil {
		return nil, err
	}

	return c.getCertExpiry(domain)
}

func (c CertificateExpires) getCertExpiry(domain string) (time.Time, error) {
	if c.option.client == nil {
		return time.Time{}, errors.New("HTTP client is not initialized")
	}

	resp, err := c.option.client.Get("https://" + domain)
	if err != nil {
		return time.Time{}, err
	}
	if resp == nil {
		return time.Time{}, errors.New("received nil response")
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return time.Time{}, errors.New("未能获取证书信息")
	}

	return resp.TLS.PeerCertificates[0].NotAfter, nil
}

func getDomainFromURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	host := parsedURL.Host

	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	return host, nil
}
