package http

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTP_Check(t *testing.T) {
	tests := []struct {
		name           string
		serverHandler  func(w http.ResponseWriter, r *http.Request)
		config         Config
		expectedStatus int
		expectedBody   string
		expectError    bool
	}{
		{
			name: "Basic Auth GET Request",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				username, password, ok := r.BasicAuth()
				assert.True(t, ok)
				assert.Equal(t, "testuser", username)
				assert.Equal(t, "testpass", password)
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "ok"}`))
			},
			config: Config{
				Method:      GET,
				ContentType: ContentTypeJSON,
				AuthType:    AuthTypeBasic,
				AuthConfig: AuthBasicConfig{
					Username: "testuser",
					Password: "testpass",
				},
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status": "ok"}`,
		},
		// Add more test cases here for different scenarios
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			tt.config.URL = server.URL
			client := resty.New()
			httpClient := NewHTTP(tt.config, SetClient(&option{client: client}))

			result, err := httpClient.Check(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			response, ok := result.(*resty.Response)
			require.True(t, ok)
			assert.Equal(t, tt.expectedStatus, response.StatusCode())
			assert.Equal(t, tt.expectedBody, string(response.Body()))
		})
	}
}

func TestHTTP_CheckWithOAuth2(t *testing.T) {
	// Mock OAuth2 token server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"access_token": "test_access_token",
			"token_type": "Bearer",
			"expires_in": 3600
		}`))
	}))
	defer tokenServer.Close()

	// Mock API server
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test_access_token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "authenticated"}`))
	}))
	defer apiServer.Close()

	config := Config{
		URL:         apiServer.URL,
		Method:      GET,
		ContentType: ContentTypeJSON,
		AuthType:    AuthTypeOAuth2,
		AuthConfig: AuthOAuth2Config{
			AuthenticationMethod: AuthenticationMethodHeader,
			OathTokenURL:         tokenServer.URL,
			ClientID:             "test_client_id",
			ClientSecret:         "test_client_secret",
			OAuthScope:           "test_scope",
		},
	}

	client := resty.New()
	httpClient := NewHTTP(config, SetClient(&option{client: client}))

	result, err := httpClient.Check(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)

	response, ok := result.(*resty.Response)
	require.True(t, ok)
	assert.Equal(t, http.StatusOK, response.StatusCode())
	assert.Equal(t, `{"status": "authenticated"}`, string(response.Body()))
}

func TestLoadCertFromByte(t *testing.T) {
	// Generate test certificates and keys
	cert, key, err := generateTestCert()
	require.NoError(t, err)

	rootCA, _, err := generateTestCert()
	require.NoError(t, err)

	rootCAs, certificates, err := LoadCertFromByte(cert, key, rootCA)
	require.NoError(t, err)
	require.NotNil(t, rootCAs)
	require.Len(t, certificates, 1)

	// Verify the loaded certificate
	parsedCert, err := x509.ParseCertificate(certificates[0].Certificate[0])
	require.NoError(t, err)
	assert.Equal(t, "test.example.com", parsedCert.Subject.CommonName)
}

func TestHTTP_CheckWithNTLM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The Authorization header will be handled by the NTLM negotiator
		// so we won't check for it here
		assert.Equal(t, "testdomain", r.Header.Get("X-NTLM-Domain"))
		assert.Equal(t, "testworkstation", r.Header.Get("X-NTLM-Workstation"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ntlm_configured"}`))
	}))
	defer server.Close()

	config := Config{
		URL:         server.URL,
		Method:      GET,
		ContentType: ContentTypeJSON,
		AuthType:    AuthTypeNTLM,
		AuthConfig: AuthNTLMConfig{
			Username:    "testuser",
			Password:    "testpass",
			Domain:      "testdomain",
			Workstation: "testworkstation",
		},
	}

	client := resty.New()
	httpClient := NewHTTP(config, SetClient(&option{client: client}))

	result, err := httpClient.Check(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)

	response, ok := result.(*resty.Response)
	require.True(t, ok)
	assert.Equal(t, http.StatusOK, response.StatusCode())
	assert.Equal(t, `{"status": "ntlm_configured"}`, string(response.Body()))
}

// Helper function to generate test certificates
func generateTestCert() ([]byte, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: randomInt(),
		Subject: pkix.Name{
			CommonName: "test.example.com",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPEM, keyPEM, nil
}

// Helper function to generate random serial number
func randomInt() *big.Int {
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, max)
	return serialNumber
}
