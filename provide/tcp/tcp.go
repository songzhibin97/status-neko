package tcp

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	status_neko "github.com/songzhibin97/status-neko"
)

var (
	_               status_neko.Monitor = (*TCP)(nil)
	providerTcpName                     = "tcp"
)

type TCP struct {
	config Config
}

type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func NewTCP(config Config) *TCP {
	return &TCP{
		config: config,
	}
}

func (t TCP) Name() string {
	return providerTcpName
}

func (t TCP) Check(ctx context.Context) (interface{}, error) {
	address := net.JoinHostPort(t.config.Host, strconv.Itoa(t.config.Port))

	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	defer conn.Close()

	// 如果成功建立连接，返回一个简单的状态信息
	return address, nil
}
