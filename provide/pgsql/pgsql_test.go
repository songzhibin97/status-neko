package pgsql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestPgSql_Check(t *testing.T) {
	// 1. 创建 sqlmock 对象
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// 2. 创建测试用的 PgSql 实例
	config := Config{
		DSN:      "user=username password=yourpassword host=localhost port=5432 dbname=yourdbname sslmode=disable",
		QuerySQL: "SELECT COUNT(*) FROM your_table",
	}
	pgsql := &PgSql{
		config: config,
		db:     db,
	}

	// 3. 设置预期查询
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM your_table").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	// 4. 调用 Check 方法
	result, err := pgsql.Check(context.Background())
	assert.NoError(t, err)

	// 5. 验证结果
	expected := map[string]interface{}{
		"status": "ok",
		"result": 42,
	}
	assert.Equal(t, expected, result)

	// 6. 确认所有预期的调用都已完成
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestPgSql_Check_Error(t *testing.T) {
	// 创建一个 sqlmock 对象
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// 创建测试用的 PgSql 实例
	config := Config{
		DSN:      "user=username password=yourpassword host=localhost port=5432 dbname=yourdbname sslmode=disable",
		QuerySQL: "SELECT COUNT(*) FROM your_table",
	}
	pgsql := &PgSql{
		config: config,
		db:     db,
	}

	// 设置预期查询并返回错误
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM your_table").
		WillReturnError(sql.ErrNoRows)

	// 调用 Check 方法
	result, err := pgsql.Check(context.Background())
	assert.Error(t, err)
	assert.Nil(t, result)

	// 确认所有预期的调用都已完成
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
