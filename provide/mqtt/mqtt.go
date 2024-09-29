package mqtt

import (
	"context"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	status_neko "github.com/songzhibin97/status-neko"
)

var (
	_                status_neko.Monitor = (*MQTT)(nil)
	providerMqttName                     = "mqtt"
)

type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Topic    string `json:"topic"`
}

type MQTT struct {
	config Config
}

func NewMQTT(config Config) *MQTT {
	return &MQTT{
		config: config,
	}
}

func (m MQTT) Name() string {
	return providerMqttName
}

func (m MQTT) Check(ctx context.Context) (interface{}, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", m.config.Host, m.config.Port))
	opts.SetUsername(m.config.Username)
	opts.SetPassword(m.config.Password)
	opts.SetClientID(fmt.Sprintf("status-neko-mqtt-%d", time.Now().UnixNano()))

	client := mqtt.NewClient(opts)

	token := client.Connect()
	if token.WaitTimeout(5*time.Second) && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %v", token.Error())
	}

	if !client.IsConnected() {
		return nil, fmt.Errorf("failed to connect to MQTT broker")
	}

	defer client.Disconnect(250)

	status := map[string]interface{}{
		"connected": true,
		"broker":    fmt.Sprintf("%s:%d", m.config.Host, m.config.Port),
		"topic":     m.config.Topic,
	}

	return status, nil
}
