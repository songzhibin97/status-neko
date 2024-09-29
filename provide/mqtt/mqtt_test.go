package mqtt

import (
	"context"
	"fmt"
	"net"
	"testing"
)

func TestMQTT_Check(t *testing.T) {
	// 启动一个模拟的 MQTT broker
	broker := startMockBroker(t)
	defer broker.Close()

	// 获取模拟broker的地址
	addr := broker.Addr().(*net.TCPAddr)

	testCases := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "Successful connection",
			config: Config{
				Host:     addr.IP.String(),
				Port:     addr.Port,
				Username: "testuser",
				Password: "testpass",
				Topic:    "test/topic",
			},
			expectError: false,
		},
		{
			name: "Failed connection - wrong port",
			config: Config{
				Host:     addr.IP.String(),
				Port:     12345, // Wrong port
				Username: "testuser",
				Password: "testpass",
				Topic:    "test/topic",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mqtt := NewMQTT(tc.config)
			result, err := mqtt.Check(context.Background())

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					t.Logf("Connection successful")
				}

				if result == nil {
					t.Errorf("Expected non-nil result, got nil")
				} else {
					status, ok := result.(map[string]interface{})
					if !ok {
						t.Errorf("Expected result to be map[string]interface{}, got %T", result)
					} else {
						t.Logf("Result: %+v", status)

						connected, ok := status["connected"].(bool)
						if !ok {
							t.Errorf("Expected 'connected' to be bool, got %T", status["connected"])
						} else if !connected {
							t.Errorf("Expected 'connected' to be true, got false")
						}

						expectedBroker := fmt.Sprintf("%s:%d", tc.config.Host, tc.config.Port)
						if status["broker"] != expectedBroker {
							t.Errorf("Expected broker to be %s, got %s", expectedBroker, status["broker"])
						}

						if status["topic"] != tc.config.Topic {
							t.Errorf("Expected topic to be %s, got %s", tc.config.Topic, status["topic"])
						}
					}
				}
			}
		})
	}
}

func startMockBroker(t *testing.T) net.Listener {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock broker: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleMockConnection(conn)
		}
	}()

	return listener
}

func handleMockConnection(conn net.Conn) {
	defer conn.Close()

	// 模拟 MQTT CONNECT 包的响应
	connectAck := []byte{0x20, 0x02, 0x00, 0x00} // CONNACK packet
	_, err := conn.Write(connectAck)
	if err != nil {
		fmt.Printf("Error writing CONNACK: %v\n", err)
		return
	}

	// 继续读取连接数据，但不做任何处理
	buffer := make([]byte, 1024)
	for {
		_, err := conn.Read(buffer)
		if err != nil {
			return
		}
	}
}
