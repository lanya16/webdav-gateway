package webdav

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"
	"time"
	"strings"
)

// MockDB 模拟数据库，用于测试
type MockDB struct {
	queries []string
	args    [][]interface{}
	results []interface{}
	err     error
}

type MockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (m *MockResult) LastInsertId() (int64, error) {
	return m.lastInsertID, nil
}

func (m *MockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, nil
}

type MockRows struct {
	columns []string
	values  [][]driver.Value
	closed  bool
}

func (r *MockRows) Columns() []string {
	return r.columns
}

func (r *MockRows) Close() error {
	r.closed = true
	return nil
}

func (r *MockRows) Next(dest []driver.Value) error {
	if r.closed {
		return fmt.Errorf("rows closed")
	}
	// 模拟单行数据
	if len(r.values) > 0 {
		for i, v := range r.values[0] {
			dest[i] = v
		}
		r.values = r.values[1:]
		return nil
	}
	return fmt.Errorf("no more rows")
}

func (m *MockDB) Prepare(query string) (*sql.Stmt, error) {
	return &sql.Stmt{}, m.err
}

func (m *MockDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return &sql.Stmt{}, m.err
}

func (m *MockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	m.queries = append(m.queries, query)
	m.args = append(m.args, args)
	return &sql.Rows{}, m.err
}

func (m *MockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	m.queries = append(m.queries, query)
	m.args = append(m.args, args)
	return &sql.Rows{}, m.err
}

func (m *MockDB) QueryRow(query string, args ...interface{}) *sql.Row {
	m.queries = append(m.queries, query)
	m.args = append(m.args, args)
	return &sql.Row{}
}

func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	m.queries = append(m.queries, query)
	m.args = append(m.args, args)
	return &sql.Row{}
}

func (m *MockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.queries = append(m.queries, query)
	m.args = append(m.args, args)
	return &MockResult{lastInsertID: 1, rowsAffected: 1}, m.err
}

func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	m.queries = append(m.queries, query)
	m.args = append(m.args, args)
	return &MockResult{lastInsertID: 1, rowsAffected: 1}, m.err
}

func (m *MockDB) Begin() (*sql.Tx, error) {
	return &sql.Tx{}, m.err
}

func (m *MockDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return &sql.Tx{}, m.err
}

// TestSQLBuilder 基本功能测试
func TestSQLBuilder(t *testing.T) {
	t.Run("NewSelectBuilder", func(t *testing.T) {
		t.Run("DefaultColumns", func(t *testing.T) {
			builder := NewSelectBuilder("users")
			expectedSQL := "SELECT * FROM users"
			if builder.Build() != expectedSQL {
				t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
			}
		})

		t.Run("SpecificColumns", func(t *testing.T) {
			builder := NewSelectBuilder("users", "id", "name", "email")
			expectedSQL := "SELECT id, name, email FROM users"
			if builder.Build() != expectedSQL {
				t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
			}
		})

		t.Run("EmptyColumns", func(t *testing.T) {
			builder := NewSelectBuilder("users")
			if len(builder.selectCols) != 1 || builder.selectCols[0] != "*" {
				t.Errorf("Expected ['*'], got %v", builder.selectCols)
			}
		})
	})

	t.Run("AddColumn", func(t *testing.T) {
		builder := NewSelectBuilder("users", "id")
		builder.AddColumn("name", "email")
		
		expectedCols := []string{"id", "name", "email"}
		if !reflect.DeepEqual(builder.selectCols, expectedCols) {
			t.Errorf("Expected columns %v, got %v", expectedCols, builder.selectCols)
		}
	})

	t.Run("ChainableMethods", func(t *testing.T) {
		builder := NewSelectBuilder("users", "id", "name").
			Where("age > ?", 18).
			OrderBy("name ASC").
			Limit(10).
			Offset(5)
		
		expectedSQL := "SELECT id, name FROM users WHERE age > ? ORDER BY name ASC LIMIT 10 OFFSET 5"
		expectedArgs := []interface{}{18}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.Args(), expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.Args())
		}
	})
}

// TestJoinOperations JOIN操作测试
func TestJoinOperations(t *testing.T) {
	t.Run("SingleJoin", func(t *testing.T) {
		builder := NewSelectBuilder("users u").
			Join("INNER JOIN", "profiles p", "u.id = p.user_id")
		
		expectedSQL := "SELECT * FROM users u INNER JOIN profiles p ON u.id = p.user_id"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("MultipleJoins", func(t *testing.T) {
		builder := NewSelectBuilder("users u").
			Join("LEFT JOIN", "profiles p", "u.id = p.user_id").
			Join("INNER JOIN", "orders o", "u.id = o.user_id")
		
		expectedSQL := "SELECT * FROM users u LEFT JOIN profiles p ON u.id = p.user_id INNER JOIN orders o ON u.id = o.user_id"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("JoinWithWhere", func(t *testing.T) {
		builder := NewSelectBuilder("users u").
			Join("INNER JOIN", "orders o", "u.id = o.user_id").
			Where("u.status = ?", "active").
			And("o.total > ?", 100)
		
		expectedSQL := "SELECT * FROM users u INNER JOIN orders o ON u.id = o.user_id WHERE u.status = ? AND o.total > ?"
		expectedArgs := []interface{}{"active", 100}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.Args(), expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.Args())
		}
	})
}

// TestWhereConditions WHERE条件测试
func TestWhereConditions(t *testing.T) {
	t.Run("SimpleWhere", func(t *testing.T) {
		builder := NewSelectBuilder("users").Where("id = ?", 1)
		expectedSQL := "SELECT * FROM users WHERE id = ?"
		expectedArgs := []interface{}{1}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.Args(), expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.Args())
		}
	})

	t.Run("MultipleWhereConditions", func(t *testing.T) {
		builder := NewSelectBuilder("users").
			Where("age > ?", 18).
			And("status = ?", "active").
			Or("admin = ?", true)
		
		expectedSQL := "SELECT * FROM users WHERE age > ? AND status = ? OR admin = ?"
		expectedArgs := []interface{}{18, "active", true}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.Args(), expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.Args())
		}
	})

	t.Run("ComplexConditions", func(t *testing.T) {
		builder := NewSelectBuilder("users").
			Where("(age > ? AND status = ?)", 18, "active").
			Or("(admin = ? OR vip = ?)", true, true)
		
		expectedSQL := "SELECT * FROM users WHERE (age > ? AND status = ?) OR (admin = ? OR vip = ?)"
		expectedArgs := []interface{}{18, "active", true, true}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.Args(), expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.Args())
		}
	})
}

// TestGroupByAndHaving GROUP BY和HAVING测试
func TestGroupByAndHaving(t *testing.T) {
	t.Run("GroupBy", func(t *testing.T) {
		builder := NewSelectBuilder("users").
			GroupBy("status", "department")
		
		expectedSQL := "SELECT * FROM users GROUP BY status, department"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("Having", func(t *testing.T) {
		builder := NewSelectBuilder("users").
			GroupBy("department").
			Having("COUNT(*) > ?", 10)
		
		expectedSQL := "SELECT * FROM users GROUP BY department HAVING COUNT(*) > ?"
		expectedArgs := []interface{}{10}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.Args(), expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.Args())
		}
	})

	t.Run("GroupByAndHaving", func(t *testing.T) {
		builder := NewSelectBuilder("orders").
			GroupBy("user_id").
			Having("SUM(amount) > ?", 1000).
			OrderBy("SUM(amount) DESC")
		
		expectedSQL := "SELECT * FROM orders GROUP BY user_id HAVING SUM(amount) > ? ORDER BY SUM(amount) DESC"
		expectedArgs := []interface{}{1000}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.Args(), expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.Args())
		}
	})
}

// TestOrderByAndLimit ORDER BY和LIMIT测试
func TestOrderByAndLimit(t *testing.T) {
	t.Run("OrderBy", func(t *testing.T) {
		builder := NewSelectBuilder("users").
			OrderBy("name ASC", "created_at DESC")
		
		expectedSQL := "SELECT * FROM users ORDER BY name ASC, created_at DESC"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("LimitOnly", func(t *testing.T) {
		builder := NewSelectBuilder("users").Limit(10)
		expectedSQL := "SELECT * FROM users LIMIT 10"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("OffsetOnly", func(t *testing.T) {
		builder := NewSelectBuilder("users").Offset(20)
		expectedSQL := "SELECT * FROM users OFFSET 20"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("LimitAndOffset", func(t *testing.T) {
		builder := NewSelectBuilder("users").
			OrderBy("id DESC").
			Limit(10).
			Offset(30)
		
		expectedSQL := "SELECT * FROM users ORDER BY id DESC LIMIT 10 OFFSET 30"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("ZeroLimit", func(t *testing.T) {
		builder := NewSelectBuilder("users").Limit(0)
		expectedSQL := "SELECT * FROM users"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("ZeroOffset", func(t *testing.T) {
		builder := NewSelectBuilder("users").Offset(0)
		expectedSQL := "SELECT * FROM users"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})
}

// TestInsertBuilder INSERT构建器测试
func TestInsertBuilder(t *testing.T) {
	t.Run("BasicInsert", func(t *testing.T) {
		builder := NewInsertBuilder("users").
			Columns("name", "email").
			Values("John", "john@example.com")
		
		expectedSQL := "INSERT INTO users (name, email) VALUES (?, ?)"
		expectedArgs := []interface{}{"John", "john@example.com"}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.Args(), expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.Args())
		}
	})

	t.Run("MultipleRows", func(t *testing.T) {
		builder := NewInsertBuilder("users").
			Columns("name", "email").
			Values("John", "john@example.com").
			Values("Jane", "jane@example.com").
			Values("Bob", "bob@example.com")
		
		expectedSQL := "INSERT INTO users (name, email) VALUES (?, ?), (?, ?), (?, ?)"
		expectedArgs := []interface{}{"John", "john@example.com", "Jane", "jane@example.com", "Bob", "bob@example.com"}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.Args(), expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.Args())
		}
	})

	t.Run("NoColumns", func(t *testing.T) {
		builder := NewInsertBuilder("users").
			Values("John", "john@example.com")
		
		expectedSQL := "INSERT INTO users VALUES (?, ?)"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("OnConflict", func(t *testing.T) {
		builder := NewInsertBuilder("users").
			Columns("id", "name", "email").
			Values(1, "John", "john@example.com").
			OnConflict("id")
		
		expectedSQL := "INSERT INTO users (id, name, email) VALUES (?, ?, ?) ON CONFLICT (id) DO NOTHING"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("ChainableMethods", func(t *testing.T) {
		builder := NewInsertBuilder("users").
			Columns("name", "email").
			Values("John", "john@example.com").
			OnConflict("email")
		
		if builder == nil {
			t.Error("Builder should not be nil")
		}
		if len(builder.cols) != 2 {
			t.Errorf("Expected 2 columns, got %d", len(builder.cols))
		}
		if len(builder.values) != 1 {
			t.Errorf("Expected 1 value row, got %d", len(builder.values))
		}
	})
}

// TestUpdateBuilder UPDATE构建器测试
func TestUpdateBuilder(t *testing.T) {
	t.Run("BasicUpdate", func(t *testing.T) {
		builder := NewUpdateBuilder("users").
			Set("name", "John Doe").
			Set("status", "active").
			Where("id = ?", 1)
		
		expectedSQL := "UPDATE users SET name=?, status=? WHERE id = ?"
		expectedArgs := []interface{}{"John Doe", "active", 1}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.args, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.args)
		}
	})

	t.Run("UpdateWithOrderBy", func(t *testing.T) {
		builder := NewUpdateBuilder("users").
			Set("status", "inactive").
			Where("last_login < ?", time.Now().AddDate(0, -1, 0)).
			OrderBy("last_login ASC").
			Limit(100)
		
		expectedSQL := "UPDATE users SET status=? WHERE last_login < ? ORDER BY last_login ASC LIMIT 100"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("UpdateWithoutWhere", func(t *testing.T) {
		builder := NewUpdateBuilder("users").
			Set("updated_at", time.Now())
		
		expectedSQL := "UPDATE users SET updated_at=?"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("MultipleSets", func(t *testing.T) {
		builder := NewUpdateBuilder("products").
			Set("price", 99.99).
			Set("stock", 0).
			Set("status", "discontinued")
		
		expectedSQL := "UPDATE products SET price=?, stock=?, status=?"
		expectedArgs := []interface{}{99.99, 0, "discontinued"}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.args, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.args)
		}
	})
}

// TestDeleteBuilder DELETE构建器测试
func TestDeleteBuilder(t *testing.T) {
	t.Run("BasicDelete", func(t *testing.T) {
		builder := NewDeleteBuilder("users").
			Where("status = ?", "inactive")
		
		expectedSQL := "DELETE FROM users WHERE status = ?"
		expectedArgs := []interface{}{"inactive"}
		
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
		
		if !reflect.DeepEqual(builder.args, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, builder.args)
		}
	})

	t.Run("DeleteWithOrderBy", func(t *testing.T) {
		builder := NewDeleteBuilder("logs").
			Where("created_at < ?", time.Now().AddDate(0, -1, 0)).
			OrderBy("created_at ASC").
			Limit(1000)
		
		expectedSQL := "DELETE FROM logs WHERE created_at < ? ORDER BY created_at ASC LIMIT 1000"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("DeleteWithoutWhere", func(t *testing.T) {
		builder := NewDeleteBuilder("temp_data")
		
		expectedSQL := "DELETE FROM temp_data"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})

	t.Run("ComplexDelete", func(t *testing.T) {
		builder := NewDeleteBuilder("sessions").
			Where("expires_at < ?", time.Now()).
			And("user_id NOT IN (?)", sql.NullInt64{Int64: 0, Valid: false})
		
		expectedSQL := "DELETE FROM sessions WHERE expires_at < ? AND user_id NOT IN (?)"
		if builder.Build() != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, builder.Build())
		}
	})
}

// TestExecuteMethods 执行方法测试
func TestExecuteMethods(t *testing.T) {
	mockDB := &MockDB{}
	ctx := context.Background()

	t.Run("ExecuteQuery", func(t *testing.T) {
		builder := NewSelectBuilder("users").Where("id = ?", 1)
		_, err := builder.ExecuteQuery(ctx, mockDB)
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if len(mockDB.queries) != 1 {
			t.Errorf("Expected 1 query, got %d", len(mockDB.queries))
		}
		
		expectedQuery := "SELECT * FROM users WHERE id = ?"
		if mockDB.queries[0] != expectedQuery {
			t.Errorf("Expected query %s, got %s", expectedQuery, mockDB.queries[0])
		}
	})

	t.Run("ExecuteQueryRow", func(t *testing.T) {
		builder := NewSelectBuilder("users").Where("id = ?", 1)
		row := builder.ExecuteQueryRow(ctx, mockDB)
		
		if row == nil {
			t.Error("Expected row, got nil")
		}
		
		if len(mockDB.queries) != 1 {
			t.Errorf("Expected 1 query, got %d", len(mockDB.queries))
		}
	})

	t.Run("ExecuteExec", func(t *testing.T) {
		builder := NewSelectBuilder("users").Where("id = ?", 1)
		result, err := builder.ExecuteExec(ctx, mockDB)
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if result == nil {
			t.Error("Expected result, got nil")
		}
		
		if len(mockDB.queries) != 1 {
			t.Errorf("Expected 1 query, got %d", len(mockDB.queries))
		}
	})

	t.Run("InsertExecute", func(t *testing.T) {
		builder := NewInsertBuilder("users").
			Columns("name", "email").
			Values("John", "john@example.com")
		
		result, err := builder.Execute(ctx, mockDB)
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if result == nil {
			t.Error("Expected result, got nil")
		}
		
		if len(mockDB.queries) != 1 {
			t.Errorf("Expected 1 query, got %d", len(mockDB.queries))
		}
		
		expectedQuery := "INSERT INTO users (name, email) VALUES (?, ?)"
		if mockDB.queries[0] != expectedQuery {
			t.Errorf("Expected query %s, got %s", expectedQuery, mockDB.queries[0])
		}
	})

	t.Run("UpdateExecute", func(t *testing.T) {
		builder := NewUpdateBuilder("users").
			Set("name", "John Doe").
			Where("id = ?", 1)
		
		result, err := builder.Execute(ctx, mockDB)
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if result == nil {
			t.Error("Expected result, got nil")
		}
		
		if len(mockDB.queries) != 1 {
			t.Errorf("Expected 1 query, got %d", len(mockDB.queries))
		}
		
		expectedQuery := "UPDATE users SET name=? WHERE id = ?"
		if mockDB.queries[0] != expectedQuery {
			t.Errorf("Expected query %s, got %s", expectedQuery, mockDB.queries[0])
		}
	})

	t.Run("DeleteExecute", func(t *testing.T) {
		builder := NewDeleteBuilder("users").
			Where("status = ?", "inactive")
		
		result, err := builder.Execute(ctx, mockDB)
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if result == nil {
			t.Error("Expected result, got nil")
		}
		
		if len(mockDB.queries) != 1 {
			t.Errorf("Expected 1 query, got %d", len(mockDB.queries))
		}
		
		expectedQuery := "DELETE FROM users WHERE status = ?"
		if mockDB.queries[0] != expectedQuery {
			t.Errorf("Expected query %s, got %s", expectedQuery, mockDB.queries[0])
		}
	})
}

// TestSQLUtil SQL工具类测试
func TestSQLUtil(t *testing.T) {
	util := SQLUtil{}

	t.Run("EscapeIdentifier", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{"Normal", "users", `"users"`},
			{"WithSpaces", "my table", `"my table"`},
			{"WithQuotes", `table"name`, `"table""name"`},
			{"Asterisk", "*", "*"},
			{"Empty", "", `""`},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := util.EscapeIdentifier(tt.input)
				if result != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result)
				}
			})
		}
	})

	t.Run("QuoteString", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{"Normal", "hello", "'hello'"},
			{"WithSingleQuote", "hello'world", "'hello''world'"},
			{"Empty", "", "''"},
			{"MultipleQuotes", "a'b'c'd", "'a''b''c''d'"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := util.QuoteString(tt.input)
				if result != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result)
				}
			})
		}
	})

	t.Run("IsValidIdentifier", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected bool
		}{
			{"ValidLowercase", "users", true},
			{"ValidUppercase", "USERS", true},
			{"ValidMixed", "UsersTable", true},
			{"ValidWithUnderscore", "user_profiles", true},
			{"ValidWithNumbers", "table123", true},
			{"ValidWithDollar", "user$id", true},
			{"Empty", "", false},
			{"WithSpaces", "user table", false},
			{"WithSpecialChars", "user-table", false},
			{"WithDot", "user.table", false},
			{"OnlyNumbers", "123", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := util.IsValidIdentifier(tt.input)
				if result != tt.expected {
					t.Errorf("Expected %t, got %t", tt.expected, result)
				}
			})
		}
	})
}

// TestTableDriven 综合测试表驱动测试
func TestTableDrivenBuilders(t *testing.T) {
	tests := []struct {
		name     string
		builder  interface {
			Build() string
		}
		expectedSQL string
	}{
		{
			name: "SimpleSelect",
			builder: NewSelectBuilder("users"),
			expectedSQL: "SELECT * FROM users",
		},
		{
			name: "SelectWithWhere",
			builder: NewSelectBuilder("users").Where("active = ?", true),
			expectedSQL: "SELECT * FROM users WHERE active = ?",
		},
		{
			name: "SelectWithJoin",
			builder: NewSelectBuilder("users u").Join("LEFT JOIN", "profiles p", "u.id = p.user_id"),
			expectedSQL: "SELECT * FROM users u LEFT JOIN profiles p ON u.id = p.user_id",
		},
		{
			name: "SelectComplex",
			builder: NewSelectBuilder("users", "id", "name").
				Where("age > ?", 18).
				And("status = ?", "active").
				GroupBy("department").
				Having("COUNT(*) > ?", 5).
				OrderBy("name ASC").
				Limit(10),
			expectedSQL: "SELECT id, name FROM users WHERE age > ? AND status = ? GROUP BY department HAVING COUNT(*) > ? ORDER BY name ASC LIMIT 10",
		},
		{
			name: "SimpleInsert",
			builder: NewInsertBuilder("users").Columns("name").Values("John"),
			expectedSQL: "INSERT INTO users (name) VALUES (?)",
		},
		{
			name: "InsertWithConflict",
			builder: NewInsertBuilder("users").Columns("id", "name").Values(1, "John").OnConflict("id"),
			expectedSQL: "INSERT INTO users (id, name) VALUES (?, ?) ON CONFLICT (id) DO NOTHING",
		},
		{
			name: "SimpleUpdate",
			builder: NewUpdateBuilder("users").Set("name", "John").Where("id = ?", 1),
			expectedSQL: "UPDATE users SET name=? WHERE id = ?",
		},
		{
			name: "UpdateWithLimit",
			builder: NewUpdateBuilder("users").
				Set("status", "inactive").
				Where("last_login < ?", time.Now()).
				OrderBy("last_login ASC").
				Limit(100),
			expectedSQL: "UPDATE users SET status=? WHERE last_login < ? ORDER BY last_login ASC LIMIT 100",
		},
		{
			name: "SimpleDelete",
			builder: NewDeleteBuilder("users").Where("status = ?", "deleted"),
			expectedSQL: "DELETE FROM users WHERE status = ?",
		},
		{
			name: "DeleteWithLimit",
			builder: NewDeleteBuilder("logs").
				Where("created_at < ?", time.Now().AddDate(0, -1, 0)).
				OrderBy("created_at ASC").
				Limit(1000),
			expectedSQL: "DELETE FROM logs WHERE created_at < ? ORDER BY created_at ASC LIMIT 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.builder.Build()
			if result != tt.expectedSQL {
				t.Errorf("Expected %s, got %s", tt.expectedSQL, result)
			}
		})
	}
}

// TestEdgeCases 边界条件和错误处理测试
func TestEdgeCases(t *testing.T) {
	t.Run("EmptyTableName", func(t *testing.T) {
		builder := NewSelectBuilder("")
		sql := builder.Build()
		if !strings.Contains(sql, "SELECT * FROM") {
			t.Errorf("Should contain SELECT FROM, got %s", sql)
		}
	})

	t.Run("SpecialCharactersInConditions", func(t *testing.T) {
		builder := NewSelectBuilder("users").
			Where("name LIKE ?", "%O'Brien%").
			And("email = ?", "test@example.com")
		
		sql := builder.Build()
		expectedSQL := "SELECT * FROM users WHERE name LIKE ? AND email = ?"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
		
		args := builder.Args()
		if len(args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(args))
		}
	})

	t.Run("NilArguments", func(t *testing.T) {
		builder := NewSelectBuilder("users").
			Where("status = ?", nil)
		
		sql := builder.Build()
		args := builder.Args()
		
		if len(args) != 1 {
			t.Errorf("Expected 1 arg, got %d", len(args))
		}
		if args[0] != nil {
			t.Errorf("Expected nil, got %v", args[0])
		}
	})

	t.Run("ComplexNestedQuery", func(t *testing.T) {
		builder := NewSelectBuilder("users u").
			Join("LEFT JOIN", "orders o", "u.id = o.user_id").
			Where("u.id IN (SELECT user_id FROM vip_users WHERE level > ?)", 5).
			And("(o.total > ? OR o.total IS NULL)", 1000).
			GroupBy("u.id", "u.name").
			Having("COUNT(o.id) > ?", 1).
			OrderBy("u.name ASC", "COUNT(o.id) DESC").
			Limit(20).
			Offset(40)
		
		sql := builder.Build()
		expectedSQL := "SELECT * FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.id IN (SELECT user_id FROM vip_users WHERE level > ?) AND (o.total > ? OR o.total IS NULL) GROUP BY u.id, u.name HAVING COUNT(o.id) > ? ORDER BY u.name ASC, COUNT(o.id) DESC LIMIT 20 OFFSET 40"
		
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
		
		args := builder.Args()
		if len(args) != 3 {
			t.Errorf("Expected 3 args, got %d", len(args))
		}
	})

	t.Run("MultipleAddColumnCalls", func(t *testing.T) {
		builder := NewSelectBuilder("users")
		builder.AddColumn("id")
		builder.AddColumn("name", "email")
		builder.AddColumn("status", "created_at", "updated_at")
		
		expectedCols := []string{"*", "id", "name", "email", "status", "created_at", "updated_at"}
		if !reflect.DeepEqual(builder.selectCols, expectedCols) {
			t.Errorf("Expected columns %v, got %v", expectedCols, builder.selectCols)
		}
	})

	t.Run("InsertWithoutColumns", func(t *testing.T) {
		builder := NewInsertBuilder("users")
		builder.Values("John", "john@example.com", true)
		
		sql := builder.Build()
		expectedSQL := "INSERT INTO users VALUES (?, ?, ?)"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
	})

	t.Run("UpdateWithoutSets", func(t *testing.T) {
		builder := NewUpdateBuilder("users")
		builder.Where("id = ?", 1)
		
		sql := builder.Build()
		expectedSQL := "UPDATE users WHERE id = ?"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
	})

	t.Run("DeleteWithoutConditions", func(t *testing.T) {
		builder := NewDeleteBuilder("temp_data")
		
		sql := builder.Build()
		expectedSQL := "DELETE FROM temp_data"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
	})

	t.Run("NegativeLimitAndOffset", func(t *testing.T) {
		builder := NewSelectBuilder("users").
			Limit(-1).
			Offset(-5)
		
		sql := builder.Build()
		expectedSQL := "SELECT * FROM users LIMIT -1 OFFSET -5"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
	})
}

// TestSQLDialectCompatibility SQL方言兼容性测试
func TestSQLDialectCompatibility(t *testing.T) {
	t.Run("PostgreSQLCompatible", func(t *testing.T) {
		// PostgreSQL兼容的查询构建
		builder := NewSelectBuilder("users").
			Where("created_at >= ?", time.Now().AddDate(-1, 0, 0)).
			OrderBy("created_at DESC").
			Limit(100)
		
		sql := builder.Build()
		// PostgreSQL使用OFFSET而不是SKIP
		expectedSQL := "SELECT * FROM users WHERE created_at >= ? ORDER BY created_at DESC LIMIT 100"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
	})

	t.Run("MySQLCompatible", func(t *testing.T) {
		// MySQL兼容的LIMIT语法
		builder := NewSelectBuilder("users").
			Limit(10).
			Offset(20)
		
		sql := builder.Build()
		// MySQL使用标准的LIMIT OFFSET语法
		expectedSQL := "SELECT * FROM users LIMIT 10 OFFSET 20"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
	})

	t.Run("SQLServerCompatible", func(t *testing.T) {
		// SQL Server在UPDATE中支持ORDER BY和LIMIT
		builder := NewUpdateBuilder("users").
			Set("status", "inactive").
			Where("last_login < ?", time.Now().AddDate(0, -6, 0)).
			OrderBy("last_login ASC").
			Limit(500)
		
		sql := builder.Build()
		expectedSQL := "UPDATE users SET status=? WHERE last_login < ? ORDER BY last_login ASC LIMIT 500"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
	})

	t.Run("StandardSQLCompliance", func(t *testing.T) {
		// 测试标准SQL兼容性
		builder := NewInsertBuilder("users").
			Columns("id", "name", "email", "created_at").
			Values(1, "John Doe", "john@example.com", time.Now()).
			OnConflict("id")
		
		sql := builder.Build()
		// ON CONFLICT是PostgreSQL语法，标准SQL使用MERGE或INSERT...ON DUPLICATE KEY UPDATE
		expectedSQL := "INSERT INTO users (id, name, email, created_at) VALUES (?, ?, ?, ?) ON CONFLICT (id) DO NOTHING"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
	})

	t.Run("ParameterBindingStyle", func(t *testing.T) {
		// 测试参数绑定风格兼容性
		builder := NewSelectBuilder("users").
			Where("age BETWEEN ? AND ?", 18, 65).
			And("salary > ?", 50000).
			Or("title = ?", "Manager")
		
		sql := builder.Build()
		expectedSQL := "SELECT * FROM users WHERE age BETWEEN ? AND ? AND salary > ? OR title = ?"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
		
		args := builder.Args()
		expectedArgs := []interface{}{18, 65, 50000, "Manager"}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, args)
		}
	})
}

// TestTransactionLikeOperations 事务相关测试
func TestTransactionLikeOperations(t *testing.T) {
	mockDB := &MockDB{}
	ctx := context.Background()

	t.Run("MultipleOperationsInSequence", func(t *testing.T) {
		// 模拟事务操作序列
		operations := []struct {
			name string
			build func() interface {
				Build() string
				Execute(context.Context, *sql.DB) (sql.Result, error)
			}
		}{
			{
				name: "insert_user",
				build: func() interface {
					Build() string
					Execute(context.Context, *sql.DB) (sql.Result, error)
				} {
					return NewInsertBuilder("users").
						Columns("name", "email").
						Values("John", "john@example.com")
				},
			},
			{
				name: "insert_profile",
				build: func() interface {
					Build() string
					Execute(context.Context, *sql.DB) (sql.Result, error)
				} {
					return NewInsertBuilder("profiles").
						Columns("user_id", "bio").
						Values(1, "Software Developer")
				},
			},
			{
				name: "update_user_status",
				build: func() interface {
					Build() string
					Execute(context.Context, *sql.DB) (sql.Result, error)
				} {
					return NewUpdateBuilder("users").
						Set("status", "active").
						Where("id = ?", 1)
				},
			},
		}

		for _, op := range operations {
			t.Run(op.name, func(t *testing.T) {
				builder := op.build()
				result, err := builder.Execute(ctx, mockDB)
				
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				
				if result == nil {
					t.Error("Expected result, got nil")
				}
			})
		}

		// 验证所有操作都被执行
		if len(mockDB.queries) != 3 {
			t.Errorf("Expected 3 queries, got %d", len(mockDB.queries))
		}

		expectedQueries := []string{
			"INSERT INTO users (name, email) VALUES (?, ?)",
			"INSERT INTO profiles (user_id, bio) VALUES (?, ?)",
			"UPDATE users SET status=? WHERE id = ?",
		}

		for i, expected := range expectedQueries {
			if i < len(mockDB.queries) && mockDB.queries[i] != expected {
				t.Errorf("Query %d: expected %s, got %s", i, expected, mockDB.queries[i])
			}
		}
	})

	t.Run("ComplexTransactionSequence", func(t *testing.T) {
		// 重置mock数据库
		mockDB = &MockDB{}

		// 创建订单事务序列
		step1 := NewInsertBuilder("orders").
			Columns("user_id", "total", "status").
			Values(123, 299.99, "pending").
			Execute(ctx, mockDB)
		
		step2 := NewInsertBuilder("order_items").
			Columns("order_id", "product_id", "quantity", "price").
			Values(1, 456, 2, 149.99).
			Execute(ctx, mockDB)
		
		step3 := NewUpdateBuilder("products").
			Set("stock", 98).
			Where("id = ?", 456).
			Execute(ctx, mockDB)
		
		step4 := NewUpdateBuilder("users").
			Set("last_order_at", time.Now()).
			Where("id = ?", 123).
			Execute(ctx, mockDB)

		// 验证所有步骤都成功执行
		errors := []error{}
		if step1[1] != nil {
			errors = append(errors, step1[1])
		}
		if step2[1] != nil {
			errors = append(errors, step2[1])
		}
		if step3[1] != nil {
			errors = append(errors, step3[1])
		}
		if step4[1] != nil {
			errors = append(errors, step4[1])
		}

		if len(errors) > 0 {
			t.Errorf("Expected no errors, got %v", errors)
		}

		if len(mockDB.queries) != 4 {
			t.Errorf("Expected 4 queries, got %d", len(mockDB.queries))
		}

		// 验证SQL语句顺序
		expectedOrder := []string{
			"INSERT INTO orders (user_id, total, status) VALUES (?, ?, ?)",
			"INSERT INTO order_items (order_id, product_id, quantity, price) VALUES (?, ?, ?, ?)",
			"UPDATE products SET stock=? WHERE id = ?",
			"UPDATE users SET last_order_at=? WHERE id = ?",
		}

		for i, expected := range expectedOrder {
			if mockDB.queries[i] != expected {
				t.Errorf("Query %d: expected %s, got %s", i, expected, mockDB.queries[i])
			}
		}
	})
}

// TestPerformancePatterns 性能模式测试
func TestPerformancePatterns(t *testing.T) {
	t.Run("ReuseBuilder", func(t *testing.T) {
		// 测试构建器重用模式
		baseBuilder := NewSelectBuilder("users").
			Where("status = ?", "active").
			OrderBy("created_at DESC")
		
		// 执行第一次查询
		query1 := baseBuilder.Limit(10).Build()
		expectedQuery1 := "SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT 10"
		if query1 != expectedQuery1 {
			t.Errorf("Expected %s, got %s", expectedQuery1, query1)
		}
		
		// 执行第二次查询（不同limit）
		query2 := baseBuilder.Limit(20).Build()
		expectedQuery2 := "SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT 20"
		if query2 != expectedQuery2 {
			t.Errorf("Expected %s, got %s", expectedQuery2, query2)
		}
		
		// 验证参数正确
		args1 := baseBuilder.Limit(10).Args()
		args2 := baseBuilder.Limit(20).Args()
		expectedArgs := []interface{}{"active"}
		
		if !reflect.DeepEqual(args1, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, args1)
		}
		
		if !reflect.DeepEqual(args2, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, args2)
		}
	})

	t.Run("BatchInsertPattern", func(t *testing.T) {
		// 测试批量插入模式
		builder := NewInsertBuilder("users").Columns("name", "email", "age")
		
		// 模拟批量数据
		users := []struct{ name, email string; age int }{
			{"John", "john@example.com", 25},
			{"Jane", "jane@example.com", 30},
			{"Bob", "bob@example.com", 35},
		}
		
		for _, user := range users {
			builder.Values(user.name, user.email, user.age)
		}
		
		sql := builder.Build()
		expectedSQL := "INSERT INTO users (name, email, age) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
		
		args := builder.Args()
		expectedArgs := []interface{}{
			"John", "john@example.com", 25,
			"Jane", "jane@example.com", 30,
			"Bob", "bob@example.com", 35,
		}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, args)
		}
	})

	t.Run("ConditionalQueryBuilding", func(t *testing.T) {
		// 测试条件查询构建
		filters := struct {
			status  *string
			minAge  *int
			maxAge  *int
			search  *string
			limit   int
			offset  int
		}{
			status:  stringPtr("active"),
			minAge:  intPtr(18),
			search:  stringPtr("john"),
			limit:   10,
			offset:  0,
		}
		
		builder := NewSelectBuilder("users")
		
		// 动态添加条件
		if filters.status != nil {
			builder = builder.Where("status = ?", *filters.status)
		}
		if filters.minAge != nil {
			builder = builder.And("age >= ?", *filters.minAge)
		}
		if filters.maxAge != nil {
			builder = builder.And("age <= ?", *filters.maxAge)
		}
		if filters.search != nil {
			builder = builder.And("(name LIKE ? OR email LIKE ?)", "%"+*filters.search+"%", "%"+*filters.search+"%")
		}
		
		builder = builder.OrderBy("created_at DESC").Limit(filters.limit).Offset(filters.offset)
		
		sql := builder.Build()
		expectedSQL := "SELECT * FROM users WHERE status = ? AND age >= ? AND (name LIKE ? OR email LIKE ?) ORDER BY created_at DESC LIMIT 10"
		if sql != expectedSQL {
			t.Errorf("Expected %s, got %s", expectedSQL, sql)
		}
		
		args := builder.Args()
		expectedArgs := []interface{}{"active", 18, "%john%", "%john%"}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Errorf("Expected args %v, got %v", expectedArgs, args)
		}
	})
}

// 辅助函数
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

// 基准测试
func BenchmarkSQLBuilder(b *testing.B) {
	b.Run("SimpleSelect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := NewSelectBuilder("users").
				Where("status = ?", "active").
				OrderBy("created_at DESC").
				Limit(10)
			_ = builder.Build()
		}
	})

	b.Run("ComplexQuery", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := NewSelectBuilder("users u").
				Join("LEFT JOIN", "profiles p", "u.id = p.user_id").
				Join("INNER JOIN", "orders o", "u.id = o.user_id").
				Where("u.status = ?", "active").
				And("o.total > ?", 100).
				GroupBy("u.id", "u.name").
				Having("COUNT(o.id) > ?", 5).
				OrderBy("u.name ASC", "COUNT(o.id) DESC").
				Limit(20).
				Offset(40)
			_ = builder.Build()
		}
	})

	b.Run("BatchInsert", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := NewInsertBuilder("users").
				Columns("name", "email", "age")
			
			for j := 0; j < 100; j++ {
				builder.Values(fmt.Sprintf("User%d", j), fmt.Sprintf("user%d@example.com", j), 20+j%50)
			}
			_ = builder.Build()
		}
	})
}

// 测试并发安全性（虽然SQLBuilder不应该是并发安全的，但测试一下当前实现）
func TestConcurrencySafety(t *testing.T) {
	t.Run("ConcurrentBuilds", func(t *testing.T) {
		// 这个测试主要是验证当前实现不会出现竞态条件
		// 实际生产中，SQLBuilder应该是每次使用新的实例
		
		done := make(chan bool, 10)
		
		for i := 0; i < 10; i++ {
			go func(index int) {
				// 每个goroutine创建自己的builder实例
				builder := NewSelectBuilder("users").
					Where("id = ?", index).
					OrderBy("created_at DESC").
					Limit(10)
				
				sql := builder.Build()
				args := builder.Args()
				
				if !strings.Contains(sql, "SELECT * FROM users") {
					t.Errorf("Invalid SQL: %s", sql)
				}
				
				if len(args) != 1 || args[0] != index {
					t.Errorf("Invalid args: %v", args)
				}
				
				done <- true
			}(i)
		}
		
		// 等待所有goroutine完成
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}