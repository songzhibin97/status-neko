package mss

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/denisenkom/go-mssqldb"
	status_neko "github.com/songzhibin97/status-neko"
)

var (
	_               status_neko.Monitor = (*MSS)(nil)
	providerMSSName                     = "mss"
)

type Config struct {
	DSN      string `json:"dsn"`
	QuerySQL string `json:"query_sql"`
}

type MSS struct {
	config Config
	db     *sql.DB
}

func NewMSS(config Config) *MSS {
	return &MSS{
		config: config,
	}
}

func (m MSS) Name() string {
	return providerMSSName
}

func (m MSS) Check(ctx context.Context) (interface{}, error) {
	if m.db == nil {
		db, err := sql.Open("sqlserver", m.config.DSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
		m.db = db
	}

	if err := m.db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	var result int
	err := m.db.QueryRowContext(ctx, m.config.QuerySQL).Scan(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return map[string]interface{}{
		"status": "ok",
		"result": result,
	}, nil
}
