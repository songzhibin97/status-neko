package mysql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestMysql_Check(t *testing.T) {
	// 1. 创建一个 sqlmock 对象
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// 2. 创建测试用的 Mysql 实例
	config := Config{
		DSN:      "user:password@tcp(localhost:3306)/dbname",
		QuerySQL: "SELECT COUNT(*) FROM users",
	}
	m := &Mysql{
		config: config,
		db:     db,
	}

	// 3. 设置预期查询
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(42))

	// 4. 调用 Check 方法
	result, err := m.Check(context.Background())
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

func TestMysql_Check_Error(t *testing.T) {
	// 创建一个 sqlmock 对象
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// 创建测试用的 Mysql 实例
	config := Config{
		DSN:      "user:password@tcp(localhost:3306)/dbname",
		QuerySQL: "SELECT COUNT(*) FROM users",
	}
	m := &Mysql{
		config: config,
		db:     db,
	}

	// 设置预期查询并返回错误
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users").
		WillReturnError(sql.ErrNoRows)

	// 调用 Check 方法
	result, err := m.Check(context.Background())
	assert.Error(t, err)
	assert.Nil(t, result)

	// 确认所有预期的调用都已完成
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
