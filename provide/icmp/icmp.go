package icmp

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ping/ping"

	status_neko "github.com/songzhibin97/status-neko"
)

var (
	_                status_neko.Monitor = (*ICMP)(nil)
	providerIcmpName                     = "icmp"
)

type ICMP struct {
	config Config
	option *option
}

type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type option struct {
	Timeout time.Duration
}

func SetTimeout(timeout time.Duration) status_neko.Option[*option] {
	return func(o *option) {
		o.Timeout = timeout
	}
}

func NewICMP(config Config, opts ...status_neko.Option[*option]) *ICMP {
	o := &option{
		Timeout: 0,
	}
	for _, opt := range opts {
		opt(o)
	}

	return &ICMP{
		config: config,
		option: o,
	}
}

func (i ICMP) Name() string {
	return providerIcmpName
}

func (i ICMP) Check(ctx context.Context) (interface{}, error) {
	if i.option.Timeout == 0 {
		i.option.Timeout = 5 * time.Second
	}

	pinger, err := ping.NewPinger(i.config.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to create pinger: %w", err)
	}

	// 设置权限模式为解特权
	pinger.SetPrivileged(false)

	pinger.Count = 1
	pinger.Timeout = i.option.Timeout

	err = pinger.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run pinger: %w", err)
	}

	stats := pinger.Statistics()

	if stats.PacketsRecv == 0 {
		return nil, fmt.Errorf("no response from host: %s", i.config.Host)
	}

	return map[string]interface{}{
		"host":     i.config.Host,
		"ip":       stats.IPAddr.String(),
		"latency":  stats.AvgRtt.String(),
		"received": stats.PacketsRecv,
	}, nil
}
