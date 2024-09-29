package redis

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
)

func TestRedis_Check(t *testing.T) {
	// 1. 创建一个 Redis mock
	db, mock := redismock.NewClientMock()
	config := Config{
		DSN: "localhost:6379",
	}
	redisMonitor := NewRedis(config)
	redisMonitor.client = db // 使用 mock 客户端

	// 2. 设置预期 PING 响应
	mock.ExpectPing().SetVal("PONG")

	// 3. 调用 Check 方法
	result, err := redisMonitor.Check(context.Background())
	assert.NoError(t, err)

	// 4. 验证结果
	expected := map[string]interface{}{
		"status": "ok",
		"result": "PONG",
	}
	assert.Equal(t, expected, result)

	// 5. 确认所有预期的调用都已完成
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRedis_Check_Error(t *testing.T) {
	// 1. 创建一个 Redis mock
	db, mock := redismock.NewClientMock()
	config := Config{
		DSN: "localhost:6379",
	}
	redisMonitor := NewRedis(config)
	redisMonitor.client = db // 使用 mock 客户端

	// 2. 设置预期 PING 响应并返回错误
	mock.ExpectPing().SetErr(redis.ErrClosed)

	// 3. 调用 Check 方法
	result, err := redisMonitor.Check(context.Background())
	assert.Error(t, err)
	assert.Nil(t, result)

	// 4. 确认所有预期的调用都已完成
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
