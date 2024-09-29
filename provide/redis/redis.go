package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"

	status_neko "github.com/songzhibin97/status-neko"
)

var (
	_                 status_neko.Monitor = (*Redis)(nil)
	providerRedisName                     = "redis"
)

type Config struct {
	DSN string `json:"dsn"`
}

type Redis struct {
	config Config
	client *redis.Client
}

func NewRedis(config Config) *Redis {
	return &Redis{
		config: config,
	}
}

func (r *Redis) Name() string {
	return providerRedisName
}

func (r *Redis) Check(ctx context.Context) (interface{}, error) {
	if r.client == nil {
		client := redis.NewClient(&redis.Options{
			Addr: r.config.DSN,
		})
		r.client = client
	}
	// 发送 PING 命令以检查连接
	result, err := r.client.Ping(ctx).Result()
	if err != nil {
		r.client = nil
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return map[string]interface{}{
		"status": "ok",
		"result": result,
	}, nil
}
