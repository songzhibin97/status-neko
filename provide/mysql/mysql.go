package mysql

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	status_neko "github.com/songzhibin97/status-neko"
)

var (
	_                 status_neko.Monitor = (*Mysql)(nil)
	providerMysqlName                     = "mysql"
)

type Config struct {
	DSN      string `json:"dsn"`
	QuerySQL string `json:"query_sql"`
}

type Mysql struct {
	config Config
	db     *sql.DB
}

func NewMysql(config Config) *Mysql {
	return &Mysql{
		config: config,
	}
}

func (m Mysql) Name() string {
	return providerMysqlName
}

func (m Mysql) Check(ctx context.Context) (interface{}, error) {
	if m.db == nil {
		db, err := sql.Open("mysql", m.config.DSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
		m.db = db
	}

	if err := m.db.PingContext(ctx); err != nil {
		m.db = nil
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
