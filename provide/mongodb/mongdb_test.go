package mongodb

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MockMongoClient is a mock implementation of the MongoClient interface
type MockMongoClient struct {
	mock.Mock
}

func (m *MockMongoClient) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMongoClient) Ping(ctx context.Context, rp *readpref.ReadPref) error {
	args := m.Called(ctx, rp)
	return args.Error(0)
}

func (m *MockMongoClient) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestMongoDB_Check(t *testing.T) {
	ctx := context.TODO()
	config := Config{DSN: "mongodb://localhost:27017"}

	// Case 1: Successful check
	mockClient := new(MockMongoClient)
	mockClient.On("Ping", ctx, readpref.Primary()).Return(nil)

	mongoDB := MongoDB{
		config: config,
		client: mockClient,
	}

	status, err := mongoDB.Check(ctx)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"status": "ok"}, status)
	mockClient.AssertExpectations(t)

	// Case 2: Failed to ping
	mockClient = new(MockMongoClient) // Create a new instance of mockClient for a fresh state
	mockClient.On("Ping", ctx, readpref.Primary()).Return(errors.New("ping failed"))

	mongoDB.client = mockClient
	status, err = mongoDB.Check(ctx)
	assert.Error(t, err)
	assert.Nil(t, status)
	mockClient.AssertExpectations(t)
}
