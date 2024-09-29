package kafka_producer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/songzhibin97/status-neko/provide/kafka_producer"
	"github.com/stretchr/testify/assert"

	"github.com/IBM/sarama"
)

// MockSyncProducer is a mock implementation of sarama.SyncProducer
type MockSyncProducer struct {
	returnError bool
}

// Implement the SendMessage method
func (m *MockSyncProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	if m.returnError {
		return 0, 0, errors.New("mock error")
	}
	return 0, 0, nil
}

// Implement the SendMessages method (empty implementation for mock)
func (m *MockSyncProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	return nil
}

// Implement the Close method (empty implementation for mock)
func (m *MockSyncProducer) Close() error {
	return nil
}

// Implement the TxnStatus method
func (m *MockSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return sarama.ProducerTxnFlagInError // or any other default value
}

// Implement the IsTransactional method
func (m *MockSyncProducer) IsTransactional() bool {
	return false
}

// Implement the BeginTxn method
func (m *MockSyncProducer) BeginTxn() error {
	return nil
}

// Implement the CommitTxn method
func (m *MockSyncProducer) CommitTxn() error {
	return nil
}

// Implement the AbortTxn method
func (m *MockSyncProducer) AbortTxn() error {
	return nil
}

// Implement the AddOffsetsToTxn method with correct signature
func (m *MockSyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	return nil
}

// Implement the AddMessageToTxn method with correct signature
func (m *MockSyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	return nil
}

func TestKafkaProducer_CheckSuccess(t *testing.T) {
	// 创建一个 mock broker
	mockBroker := sarama.NewMockBroker(t, 1)

	defer mockBroker.Close()

	// 配置 mock broker 返回的响应
	mockBroker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(mockBroker.Addr(), mockBroker.BrokerID()).
			SetLeader("test-topic", 0, mockBroker.BrokerID()),
		"ProduceRequest": sarama.NewMockProduceResponse(t).
			SetError("test-topic", 0, sarama.ErrNoError),
	})

	// 设置 Kafka 配置
	config := kafka_producer.Config{
		Brokers:         []string{mockBroker.Addr()},
		Topic:           "test-topic",
		ProducerMessage: "Test Message",
	}

	// 创建 Kafka producer
	producer := kafka_producer.NewKafkaProducer(config)

	// 执行检查
	result, err := producer.Check(context.Background())

	// 确保没有错误
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// 确保返回的结果包含期望的字段
	status := result.(map[string]interface{})
	assert.Equal(t, config.Brokers, status["brokers"])
	assert.Equal(t, config.Topic, status["topic"])
	assert.Equal(t, config.ProducerMessage, status["producer_message"])
}

func TestKafkaProducer_CheckFail(t *testing.T) {
	// Setup configuration
	config := kafka_producer.Config{
		Brokers:         []string{"localhost:9092"},
		Topic:           "test_topic",
		ProducerMessage: "test_message",
	}

	// Create KafkaProducer instance with mock producer
	kProducer := kafka_producer.NewKafkaProducer(config)
	kProducer.SyncProducer = func([]string, *sarama.Config) (sarama.SyncProducer, error) {
		return &MockSyncProducer{returnError: true}, nil
	}

	// Call Check method and expect failure
	ctx := context.Background()
	status, err := kProducer.Check(ctx)

	// Assert error and nil status
	assert.Error(t, err)
	assert.Nil(t, status)
}
