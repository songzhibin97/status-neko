package tcp

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestTCP_Name(t *testing.T) {
	tcp := TCP{Config{Host: "example.com", Port: 80}}
	if name := tcp.Name(); name != providerTcpName {
		t.Errorf("Expected name %s, got %s", providerTcpName, name)
	}
}

func TestTCP_Check(t *testing.T) {
	tests := []struct {
		name    string
		tcp     TCP
		wantErr bool
	}{
		{
			name:    "Valid connection",
			tcp:     TCP{Config{Host: "example.com", Port: 80}},
			wantErr: false,
		},
		{
			name:    "Invalid host",
			tcp:     TCP{Config{Host: "invalid.example.com", Port: 80}},
			wantErr: true,
		},
		{
			name:    "Invalid port",
			tcp:     TCP{Config{Host: "example.com", Port: 99999}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			got, err := tt.tcp.Check(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("TCP.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != net.JoinHostPort(tt.tcp.config.Host, strconv.Itoa(tt.tcp.config.Port)) {
				t.Errorf("TCP.Check() = %v, want %v", got, net.JoinHostPort(tt.tcp.config.Host, strconv.Itoa(tt.tcp.config.Port)))
			}
		})
	}
}

func TestTCP_CheckWithMockServer(t *testing.T) {
	// 启动一个本地 TCP 服务器用于测试
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	// 获取服务器地址
	addr := listener.Addr().(*net.TCPAddr)

	tcp := TCP{
		Config{
			Host: addr.IP.String(),
			Port: addr.Port,
		},
	}

	ctx := context.Background()
	got, err := tcp.Check(ctx)
	if err != nil {
		t.Errorf("TCP.Check() error = %v, wantErr false", err)
		return
	}

	expected := net.JoinHostPort(tcp.config.Host, strconv.Itoa(tcp.config.Port))
	if got != expected {
		t.Errorf("TCP.Check() = %v, want %v", got, expected)
	}
}
