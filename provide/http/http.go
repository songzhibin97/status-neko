package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/go-ntlmssp"

	"golang.org/x/oauth2/clientcredentials"

	"golang.org/x/oauth2"

	"github.com/go-resty/resty/v2"
	status_neko "github.com/songzhibin97/status-neko"
)

type (
	// Method is the HTTP method to be used in the request.
	Method string

	// ContentType is the content type of the request.
	ContentType string

	// AuthType is the type of authentication to be used in the request.
	AuthType string

	// AuthenticationMethod is the interface that wraps the basic methods for authentication.
	AuthenticationMethod string

	// ProxyType is the type of proxy to be used in the request.
	ProxyType string
)

var (
	_ status_neko.Monitor = (*HTTP)(nil)

	providerHttpName = "http"

	GET     Method = resty.MethodGet
	POST    Method = resty.MethodPost
	PUT     Method = resty.MethodPut
	PATCH   Method = resty.MethodPatch
	DELETE  Method = resty.MethodDelete
	HEAD    Method = resty.MethodHead
	OPTIONS Method = resty.MethodOptions

	ContentTypeJSON ContentType = "application/json"
	ContentTypeXML  ContentType = "application/xml"

	AuthTypeNone   AuthType = ""
	AuthTypeBasic  AuthType = "Basic"
	AuthTypeOAuth2 AuthType = "Oauth2"
	AuthTypeNTLM   AuthType = "NTLM"
	AuthTypeMTLS   AuthType = "mTls"

	AuthenticationMethodHeader AuthenticationMethod = "client_secret_basic"
	AuthenticationMethodParam  AuthenticationMethod = "client_secret_post"

	ProxyTypeNone       ProxyType = ""
	ProxyTypeHTTP       ProxyType = "HTTP"
	ProxyTypeHTTPS      ProxyType = "HTTPS"
	ProxyTypeSocks      ProxyType = "SOCKS"
	ProxyTypeSocksV5    ProxyType = "SOCKS5"
	ProxyTypeSocksV5DNS ProxyType = "SOCKS5DNS"
	ProxyTypeSocksV4    ProxyType = "SOCKS4"
)

type AuthBasicConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthOAuth2Config struct {
	AuthenticationMethod AuthenticationMethod `json:"authentication_method"`
	OathTokenURL         string               `json:"oauth_token_url"`
	ClientID             string               `json:"client_id"`
	ClientSecret         string               `json:"client_secret"`
	OAuthScope           string               `json:"oauth_scope"`
	// 这两个是后续请求时需要的 token 和过期时间
	// 作为cache存储
	*TokenSet `json:"-"`
}

type AuthNTLMConfig struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Domain      string `json:"domain"`
	Workstation string `json:"workstation"`
}

type TokenSet struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Expiry      time.Time `json:"expires_in"`
	Scope       string    `json:"scope,omitempty"`
}

type AuthMTLSConfig struct {
	Cert string `json:"cert"` // 证书
	Key  string `json:"key"`  // 私钥
	CA   string `json:"ca"`   // CA
}

type ProxyAuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type HTTP struct {
	option *option

	config Config
}

type Config struct {
	URL                    string            `json:"url"`
	Method                 Method            `json:"method"`
	ContentType            ContentType       `json:"content_type"`
	Body                   string            `json:"body"`
	Headers                map[string]string `json:"headers"`
	AuthType               AuthType          `json:"auth_type"`
	AuthConfig             interface{}       `json:"auth_config"`
	ProxyType              ProxyType         `json:"proxy_type"`
	ProxyAddress           string            `json:"proxy_address"`
	ProxyAuthEnabled       bool              `json:"proxy_auth_enabled"`
	ProxyAuthConfig        ProxyAuthConfig   `json:"proxy_auth_config"`
	SkipCertificateExpires bool              `json:"skip_certificate_expires"`
}

type option struct {
	client *resty.Client
}

func SetClient(c *option) status_neko.Option[*option] {
	return func(o *option) {
		o.client = c.client
	}
}

func NewHTTP(c Config, opts ...status_neko.Option[*option]) *HTTP {
	o := &option{}
	for _, opt := range opts {
		opt(o)
	}

	return &HTTP{
		option: o,
		config: c,
	}
}

func (h HTTP) Name() string {
	return providerHttpName
}

func (h HTTP) Check(ctx context.Context) (interface{}, error) {
	// 创建请求对象

	client := h.option.client

	if h.config.SkipCertificateExpires {
		client = client.SetTLSClientConfig(&tls.Config{
			InsecureSkipVerify: h.config.SkipCertificateExpires,
		})
	}

	switch h.config.AuthType {
	case AuthTypeNTLM:
		ntlmTransport := &ntlmssp.Negotiator{
			RoundTripper: &http.Transport{},
		}
		client.SetTransport(ntlmTransport)

	case AuthTypeMTLS:
		if mTlsConfig, ok := h.config.AuthConfig.(AuthMTLSConfig); ok {
			if mTlsConfig.Cert != "" {
				rootCAs, certificates, err := LoadCertFromByte([]byte(mTlsConfig.Cert), []byte(mTlsConfig.Key), []byte(mTlsConfig.CA))
				if err != nil {
					return nil, err
				}
				client = client.SetTLSClientConfig(&tls.Config{
					RootCAs:            rootCAs,
					Certificates:       certificates,
					InsecureSkipVerify: h.config.SkipCertificateExpires,
				})
			}
		}
	}

	// 处理代理设置
	if h.config.ProxyType != ProxyTypeNone {
		client.SetProxy(h.config.ProxyAddress)

	}

	req := client.R().SetContext(ctx)

	// 设置请求方法、URL、Content-Type
	req.Method = string(h.config.Method)
	req.URL = h.config.URL
	req = req.SetHeader("Content-Type", string(h.config.ContentType))

	// 设置请求头
	for key, value := range h.config.Headers {
		req = req.SetHeader(key, value)
	}

	// 设置请求体
	if h.config.Body != "" {
		req = req.SetBody(h.config.Body)
	}

	switch h.config.AuthType {
	case AuthTypeBasic:
		if basicConfig, ok := h.config.AuthConfig.(AuthBasicConfig); ok {
			req = req.SetBasicAuth(basicConfig.Username, basicConfig.Password)
		}

	case AuthTypeOAuth2:
		if oauthConfig, ok := h.config.AuthConfig.(AuthOAuth2Config); ok {
			if oauthConfig.TokenSet == nil || oauthConfig.TokenSet.Expiry.Before(time.Now()) {
				tokenSet, err := getOidcTokenClient(ctx, oauthConfig)
				if err != nil {
					return nil, err
				}
				oauthConfig.TokenSet = tokenSet
			}
			req = req.SetAuthScheme(oauthConfig.TokenSet.TokenType)
			req = req.SetAuthToken(oauthConfig.TokenSet.AccessToken)
		}
	case AuthTypeNTLM:
		if ntlmConfig, ok := h.config.AuthConfig.(AuthNTLMConfig); ok {
			req = req.SetBasicAuth(ntlmConfig.Username, ntlmConfig.Password)
			req = req.SetHeader("Authorization", "NTLM")
			if ntlmConfig.Domain != "" {
				req.SetHeader("X-NTLM-Domain", ntlmConfig.Domain)
			}
			if ntlmConfig.Workstation != "" {
				req.SetHeader("X-NTLM-Workstation", ntlmConfig.Workstation)
			}
		}
	}

	if h.config.ProxyAddress != "" && h.config.ProxyAuthEnabled {
		authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(h.config.ProxyAuthConfig.Username+":"+h.config.ProxyAuthConfig.Password))
		req = req.SetHeader("Proxy-Authorization", authHeader)
	}

	// 发送请求并获取响应
	resp, err := req.Send()
	if err != nil {
		return nil, err
	}

	// 返回响应内容
	return resp, nil
}

func getOidcTokenClient(ctx context.Context, authOAuth2Config AuthOAuth2Config) (*TokenSet, error) {
	// Determine the AuthStyle based on the authMethod parameter
	var authStyle oauth2.AuthStyle
	switch authOAuth2Config.AuthenticationMethod {
	case AuthenticationMethodParam:
		authStyle = oauth2.AuthStyleInParams
	case AuthenticationMethodHeader:
		fallthrough
	default:
		authStyle = oauth2.AuthStyleInHeader
	}

	// Create an OAuth2 config for client credentials flow
	config := clientcredentials.Config{
		ClientID:     authOAuth2Config.ClientID,
		ClientSecret: authOAuth2Config.ClientSecret,
		TokenURL:     authOAuth2Config.OathTokenURL,
		Scopes:       []string{authOAuth2Config.OAuthScope},
		AuthStyle:    authStyle, // Use the determined authStyle
	}

	// Create a context with a custom HTTP client
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	// Retrieve the token
	token, err := config.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve token: %v", err)
	}

	// Create a TokenSet from the OAuth2 token response
	tokenSet := &TokenSet{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Expiry:      token.Expiry,
		Scope:       authOAuth2Config.OAuthScope,
	}

	return tokenSet, nil
}

func LoadCertFromByte(clientCrt []byte, childKey []byte, rootCaChain []byte) (*x509.CertPool, []tls.Certificate, error) {
	cert, err := tls.X509KeyPair(clientCrt, childKey)
	if err != nil {
		return nil, nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(rootCaChain)
	return caCertPool, []tls.Certificate{cert}, nil
}
