package pgsql

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	status_neko "github.com/songzhibin97/status-neko"
)

var (
	_                 status_neko.Monitor = (*PgSql)(nil)
	providerPgsqlName                     = "pgsql"
)

type Config struct {
	DSN      string `json:"dsn"`
	QuerySQL string `json:"query_sql"`
}

type PgSql struct {
	config Config
	db     *sql.DB
}

func NewPgsql(config Config) *PgSql {
	return &PgSql{
		config: config,
	}
}

func (p *PgSql) Name() string {
	return providerPgsqlName
}

func (p *PgSql) Check(ctx context.Context) (interface{}, error) {
	if p.db == nil {
		db, err := sql.Open("postgres", p.config.DSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
		p.db = db
	}

	if err := p.db.PingContext(ctx); err != nil {
		p.db = nil
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	var result int
	err := p.db.QueryRowContext(ctx, p.config.QuerySQL).Scan(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return map[string]interface{}{
		"status": "ok",
		"result": result,
	}, nil
}
