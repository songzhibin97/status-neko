package kafka_producer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	status_neko "github.com/songzhibin97/status-neko"
)

type (
	SASLAuthType string
)

var (
	_ status_neko.Monitor = (*KafkaProducer)(nil)

	providerKafkaProducerName = "kafka_producer"

	SASLAuthTypeNone   SASLAuthType = "none"
	SASLAuthTypePlain  SASLAuthType = "plain"
	SASLAuthTypeSha256 SASLAuthType = "sha256"
	SASLAuthTypeSha512 SASLAuthType = "sha512"
)

type Config struct {
	Brokers         []string `json:"brokers"`
	Topic           string   `json:"topic"`
	ProducerMessage string   `json:"producer_message"`
	CreateTopic     bool     `json:"create_topic"` // producer 自动创建 topic
}

type AuthMTLSConfig struct {
	Cert string `json:"cert"` // 证书
	Key  string `json:"key"`  // 私钥
	CA   string `json:"ca"`   // CA
}

type option struct {
	SASLAuthType           SASLAuthType
	Username               string
	Password               string
	SSL                    bool `json:"ssl"`
	SkipCertificateExpires bool `json:"skip_certificate_expires"`
	AuthMTLSConfig         AuthMTLSConfig
	Check                  func(ctx context.Context) (interface{}, error)
}

type KafkaProducer struct {
	config       Config
	option       *option
	SyncProducer func([]string, *sarama.Config) (sarama.SyncProducer, error)
}

func SetSASLAuthType(authType SASLAuthType) status_neko.Option[*option] {
	return func(o *option) {
		o.SASLAuthType = authType
	}
}

func SetUsernameAndPassword(username, password string) status_neko.Option[*option] {
	return func(o *option) {
		o.Username = username
		o.Password = password
	}
}

func SetSSL(ssl bool, skipCertificateExpires bool, config AuthMTLSConfig) status_neko.Option[*option] {
	return func(o *option) {
		o.SSL = ssl
		o.SkipCertificateExpires = skipCertificateExpires
		o.AuthMTLSConfig = config

	}
}

func NewKafkaProducer(config Config, opts ...status_neko.Option[*option]) *KafkaProducer {
	o := &option{}
	for _, opt := range opts {
		opt(o)
	}

	return &KafkaProducer{
		config:       config,
		option:       o,
		SyncProducer: sarama.NewSyncProducer,
	}
}

func (k KafkaProducer) Name() string {
	return providerKafkaProducerName
}

func (k KafkaProducer) Check(ctx context.Context) (interface{}, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Net.DialTimeout = 5 * time.Second
	config.Net.ReadTimeout = 5 * time.Second
	config.Net.WriteTimeout = 5 * time.Second

	// 配置 SSL
	if k.option.SSL {
		config.Net.TLS.Enable = true
		var rootCAs *x509.CertPool
		var certificates []tls.Certificate
		var err error
		if !k.option.SkipCertificateExpires {
			rootCAs, certificates, err = LoadCertFromByte([]byte(k.option.AuthMTLSConfig.Cert), []byte(k.option.AuthMTLSConfig.Key), []byte(k.option.AuthMTLSConfig.CA))
			if err != nil {
				return nil, err
			}
		}

		config.Net.TLS.Config = &tls.Config{
			RootCAs:            rootCAs,
			Certificates:       certificates,
			InsecureSkipVerify: k.option.SkipCertificateExpires,
		}
	}

	// 配置 SASL
	switch k.option.SASLAuthType {
	case SASLAuthTypePlain:
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		config.Net.SASL.User = k.option.Username
		config.Net.SASL.Password = k.option.Password
	case SASLAuthTypeSha256:
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		config.Net.SASL.User = k.option.Username
		config.Net.SASL.Password = k.option.Password
	case SASLAuthTypeSha512:
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		config.Net.SASL.User = k.option.Username
		config.Net.SASL.Password = k.option.Password
	}

	producer, err := sarama.NewSyncProducer(k.config.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %v", err)
	}
	defer producer.Close()

	message := &sarama.ProducerMessage{
		Topic: k.config.Topic,
		Value: sarama.StringEncoder(k.config.ProducerMessage),
	}

	partition, offset, err := producer.SendMessage(message)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %v", err)
	}

	status := map[string]interface{}{
		"connected":        true,
		"brokers":          k.config.Brokers,
		"topic":            k.config.Topic,
		"last_partition":   partition,
		"last_offset":      offset,
		"ssl_enabled":      k.option.SSL,
		"sasl_auth_type":   k.option.SASLAuthType,
		"producer_message": k.config.ProducerMessage,
	}

	return status, nil
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
