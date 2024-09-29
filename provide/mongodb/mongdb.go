package mongodb

import (
	"context"

	status_neko "github.com/songzhibin97/status-neko"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	_                   status_neko.Monitor = (*MongoDB)(nil)
	providerMongoDBName                     = "mongodb"
)

type Config struct {
	DSN string `json:"dsn"`
}

type MongoClient interface {
	Connect(ctx context.Context) error
	Ping(ctx context.Context, rp *readpref.ReadPref) error
	Disconnect(ctx context.Context) error
}

type MongoDB struct {
	config Config
	client MongoClient
}

func NewMongoDB(config Config) *MongoDB {
	return &MongoDB{
		config: config,
	}
}

func (m MongoDB) Name() string {
	return providerMongoDBName
}

func (m MongoDB) Check(ctx context.Context) (interface{}, error) {
	if m.client == nil {
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(m.config.DSN))
		if err != nil {
			return nil, err
		}
		m.client = client
	}

	err := m.client.Ping(ctx, readpref.Primary())
	if err != nil {
		m.client = nil
		return nil, err
	}

	return map[string]string{"status": "ok"}, nil
}
