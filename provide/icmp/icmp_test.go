package icmp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestICMP_Name(t *testing.T) {
	i := ICMP{config: Config{Host: "example.com", Port: 0}}
	assert.Equal(t, providerIcmpName, i.Name())
}

func TestICMP_Check(t *testing.T) {
	tests := []struct {
		name    string
		icmp    *ICMP
		wantErr bool
	}{
		{
			name:    "Valid host (Google DNS)",
			icmp:    NewICMP(Config{"8.8.8.8", 0}, SetTimeout(2*time.Second)),
			wantErr: false,
		},
		{
			name:    "Invalid host",
			icmp:    NewICMP(Config{"256.256.256.256", 0}, SetTimeout(2*time.Second)),
			wantErr: true,
		},
		{
			name:    "Unreachable host",
			icmp:    NewICMP(Config{"10.255.255.255", 0}, SetTimeout(2*time.Second)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := tt.icmp.Check(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				result, ok := got.(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, tt.icmp.config.Host, result["host"])
				assert.IsType(t, "", result["latency"])
				assert.IsType(t, 0, result["received"])
			}
		})
	}
}
