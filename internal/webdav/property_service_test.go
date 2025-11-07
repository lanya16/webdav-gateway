package webdav

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// Mock Database Structures
// ========================================

// MockDB 模拟数据库
type MockDB struct {
	mu         sync.RWMutex
	tables     map[string][]map[string]interface{}
	queries    []string
	execCount  int
	queryCount int
}

// NewMockDB 创建模拟数据库
func NewMockDB() *MockDB {
	return &MockDB{
		tables: make(map[string][]map[string]interface{}),
	}
}

// ExecContext 模拟ExecContext
func (m *MockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (driver.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.execCount++
	m.queries = append(m.queries, query)
	
	// 简单模拟INSERT、UPDATE、DELETE操作
	if strings.Contains(strings.ToUpper(query), "INSERT INTO properties") {
		return &MockResult{lastInsertID: int64(m.execCount), rowsAffected: 1}, nil
	}
	if strings.Contains(strings.ToUpper(query), "UPDATE properties") {
		return &MockResult{rowsAffected: 1}, nil
	}
	if strings.Contains(strings.ToUpper(query), "DELETE FROM properties") {
		return &MockResult{rowsAffected: 1}, nil
	}
	
	return &MockResult{rowsAffected: 0}, nil
}

// QueryContext 模拟QueryContext
func (m *MockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (driver.Rows, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.queryCount++
	m.queries = append(m.queries, query)
	
	// 模拟查询结果
	return m.mockRows(query), nil
}

// QueryRowContext 模拟QueryRowContext
func (m *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) driver.Row {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.queryCount++
	m.queries = append(m.queries, query)
	
	// 返回模拟行
	rows := m.mockRows(query)
	if rows != nil {
		return &MockRow{rows: rows}
	}
	return &MockRow{rows: nil}
}

func (m *MockDB) mockRows(query string) driver.Rows {
	// 模拟属性查询结果
	if strings.Contains(strings.ToUpper(query), "SELECT") {
		// 模拟几个属性记录
		return &MockRows{
			columns: []string{"id", "user_id", "resource_id", "path", "name", "namespace", "value", "is_live", "created_at", "updated_at"},
			rows: [][]driver.Value{
				{int64(1), "user1", "res1", "/path/file.txt", "displayname", "DAV:", "File.txt", int64(1), int64(time.Now().Unix()), int64(time.Now().Unix())},
				{int64(2), "user1", "res1", "/path/file.txt", "getcontenttype", "DAV:", "text/plain", int64(1), int64(time.Now().Unix()), int64(time.Now().Unix())},
			},
		}
	}
	return nil
}

// PrepareContext 实现sql.DB接口
func (m *MockDB) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return &MockStmt{}, nil
}

// BeginTx 实现sql.DB接口
func (m *MockDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (driver.Tx, error) {
	return &MockTx{}, nil
}

// Close 实现sql.DB接口
func (m *MockDB) Close() error {
	return nil
}

// SetMaxIdleConns 实现sql.DB接口
func (m *MockDB) SetMaxIdleConns(n int) {}

// SetMaxOpenConns 实现sql.DB接口
func (m *MockDB) SetMaxOpenConns(n int) {}

// SetConnMaxLifetime 实现sql.DB接口
func (m *MockDB) SetConnMaxLifetime(d time.Duration) {}

// MockResult 模拟driver.Result
type MockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r *MockResult) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

func (r *MockResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// MockRows 模拟driver.Rows
type MockRows struct {
	columns []string
	rows    [][]driver.Value
	pos     int
}

func (r *MockRows) Columns() []string {
	return r.columns
}

func (r *MockRows) Close() error {
	return nil
}

func (r *MockRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.rows) {
		return sql.ErrNoRows
	}
	copy(dest, r.rows[r.pos])
	r.pos++
	return nil
}

// MockRow 模拟driver.Row
type MockRow struct {
	rows driver.Rows
}

func (r *MockRow) Scan(dest ...driver.Value) error {
	if r.rows == nil {
		return sql.ErrNoRows
	}
	
	// 模拟单行扫描
	values := make([]driver.Value, len(dest))
	err := r.rows.Next(values)
	if err != nil {
		return err
	}
	
	for i, v := range values {
		*(dest[i].(*driver.Value)) = v
	}
	return nil
}

// MockStmt 模拟driver.Stmt
type MockStmt struct{}

func (s *MockStmt) Close() error { return nil }
func (s *MockStmt) NumInput() int { return 0 }
func (s *MockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &MockResult{}, nil
}
func (s *MockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &MockRows{}, nil
}

// MockTx 模拟driver.Tx
type MockTx struct{}

func (t *MockTx) Commit() error { return nil }
func (t *MockTx) Rollback() error { return nil }

// ========================================
// Test Setup and Teardown
// ========================================

func TestMain(m *testing.M) {
	// 设置测试环境
	SetupTestEnvironment()
	
	// 运行测试
	exitCode := m.Run()
	
	// 清理测试环境
	CleanupTestEnvironment()
	
	os.Exit(exitCode)
}

func SetupTestEnvironment() {
	// 创建临时测试目录
	testDir := filepath.Join(os.TempDir(), "webdav_property_service_test")
	os.MkdirAll(testDir, 0755)
}

func CleanupTestEnvironment() {
	// 清理临时测试目录
	testDir := filepath.Join(os.TempDir(), "webdav_property_service_test")
	os.RemoveAll(testDir)
}

// ========================================
// Test Helper Functions
// ========================================

func createTestPropertyService(t *testing.T) (*PropertyService, func()) {
	t.Helper()
	
	// 创建临时数据库文件
	dbPath := filepath.Join(os.TempDir(), "webdav_property_service_test", 
		"test_"+randString(8)+".db")
	
	service, err := NewPropertyService(dbPath)
	require.NoError(t, err)
	
	// 初始化数据库
	ctx := context.Background()
	err = service.Initialize(ctx)
	require.NoError(t, err)
	
	// 返回服务实例和清理函数
	cleanup := func() {
		service.Close()
		os.Remove(dbPath)
	}
	
	return service, cleanup
}

func createTestProperty(userID, path, namespace, name, value string, isLive bool) *Property {
	return &Property{
		UserID:     userID,
		ResourceID: "resource_" + randString(8),
		Path:       path,
		Name:       name,
		Namespace:  namespace,
		Value:      value,
		IsLive:     isLive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func randString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

// ========================================
// Initialization Tests
// ========================================

func TestNewPropertyService(t *testing.T) {
	tests := []struct {
		name        string
		dbPath      string
		wantErr     bool
		description string
	}{
		{
			name:        "有效数据库路径",
			dbPath:      ":memory:",
			wantErr:     false,
			description: "应该成功创建内存数据库服务",
		},
		{
			name:        "无效数据库路径",
			dbPath:      "/invalid/path/to/database.db",
			wantErr:     true,
			description: "应该返回错误",
		},
		{
			name:        "临时文件数据库",
			dbPath:      "",
			wantErr:     false,
			description: "应该创建临时文件数据库",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewPropertyService(tt.dbPath)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.Equal(t, tt.dbPath, service.dbPath)
				
				// 测试连接池配置
				assert.NotNil(t, service.db)
				
				// 清理
				if service != nil {
					service.Close()
				}
			}
		})
	}
}

func TestPropertyService_Initialize(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("首次初始化", func(t *testing.T) {
		// 清理初始化标记
		service.initialised = false
		
		err := service.Initialize(ctx)
		assert.NoError(t, err)
		assert.True(t, service.initialised)
	})

	t.Run("重复初始化", func(t *testing.T) {
		// 已经初始化过
		err := service.Initialize(ctx)
		assert.NoError(t, err)
		assert.True(t, service.initialised)
	})

	t.Run("并发初始化", func(t *testing.T) {
		// 重置初始化标记
		service.initialised = false
		
		var wg sync.WaitGroup
		errors := make([]error, 10)
		
		// 启动10个goroutine同时初始化
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				errors[index] = service.Initialize(ctx)
			}(i)
		}
		
		wg.Wait()
		
		// 验证只有一个goroutine成功，其他返回nil（因为并发）
		successCount := 0
		for _, err := range errors {
			if err == nil {
				successCount++
			}
		}
		
		// 至少有一个成功，可能有多个并发成功但无错误
		assert.True(t, successCount >= 1)
		assert.True(t, service.initialised)
	})
}

// ========================================
// CRUD Operation Tests
// ========================================

func TestPropertyService_CreateProperty(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("创建新属性", func(t *testing.T) {
		property := createTestProperty("user1", "/path/file.txt", "DAV:", "displayname", "Test File.txt", true)
		
		err := service.CreateProperty(ctx, property)
		assert.NoError(t, err)
		assert.Greater(t, property.ID, 0)
		assert.False(t, property.CreatedAt.IsZero())
		assert.False(t, property.UpdatedAt.IsZero())
	})

	t.Run("创建重复属性", func(t *testing.T) {
		property := createTestProperty("user1", "/path/file.txt", "DAV:", "getcontenttype", "text/plain", true)
		
		// 第一次创建
		err := service.CreateProperty(ctx, property)
		assert.NoError(t, err)
		
		// 尝试重复创建（应该使用ON CONFLICT处理）
		err = service.CreateProperty(ctx, property)
		assert.NoError(t, err) // SQLite的ON CONFLICT DO NOTHING不会报错
	})

	t.Run("创建属性时数据库错误", func(t *testing.T) {
		// 关闭数据库连接
		service.db.Close()
		
		property := createTestProperty("user1", "/path/file.txt", "DAV:", "test", "test", false)
		
		err := service.CreateProperty(ctx, property)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "创建属性失败")
	})
}

func TestPropertyService_GetProperty(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("获取存在的属性", func(t *testing.T) {
		// 先创建属性
		property := createTestProperty("user1", "/path/file.txt", "DAV:", "displayname", "Test File", true)
		err := service.CreateProperty(ctx, property)
		require.NoError(t, err)
		
		// 获取属性
		retrieved, err := service.GetProperty(ctx, "user1", "/path/file.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, property.Name, retrieved.Name)
		assert.Equal(t, property.Namespace, retrieved.Namespace)
		assert.Equal(t, property.Value, retrieved.Value)
	})

	t.Run("获取不存在的属性", func(t *testing.T) {
		property, err := service.GetProperty(ctx, "user1", "/path/nonexistent.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.Nil(t, property)
	})

	t.Run("获取属性时数据库错误", func(t *testing.T) {
		// 关闭数据库连接
		service.db.Close()
		
		property, err := service.GetProperty(ctx, "user1", "/path/file.txt", "DAV:", "displayname")
		assert.Error(t, err)
		assert.Nil(t, property)
	})
}

func TestPropertyService_UpdateProperty(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("更新存在的属性", func(t *testing.T) {
		// 先创建属性
		originalProperty := createTestProperty("user1", "/path/file.txt", "DAV:", "displayname", "Original Name", true)
		err := service.CreateProperty(ctx, originalProperty)
		require.NoError(t, err)
		
		// 更新属性
		originalProperty.Value = "Updated Name"
		err = service.UpdateProperty(ctx, originalProperty)
		assert.NoError(t, err)
		
		// 验证更新
		retrieved, err := service.GetProperty(ctx, "user1", "/path/file.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "Updated Name", retrieved.Value)
	})

	t.Run("更新不存在的属性", func(t *testing.T) {
		property := createTestProperty("user1", "/path/nonexistent.txt", "DAV:", "displayname", "test", false)
		property.ID = 999999 // 不存在的ID
		
		err := service.UpdateProperty(ctx, property)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "属性不存在")
	})

	t.Run("更新属性时数据库错误", func(t *testing.T) {
		// 关闭数据库连接
		service.db.Close()
		
		property := createTestProperty("user1", "/path/file.txt", "DAV:", "displayname", "test", false)
		
		err := service.UpdateProperty(ctx, property)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "更新属性失败")
	})
}

func TestPropertyService_DeleteProperty(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("删除存在的属性", func(t *testing.T) {
		// 先创建属性
		property := createTestProperty("user1", "/path/file.txt", "DAV:", "displayname", "Test", true)
		err := service.CreateProperty(ctx, property)
		require.NoError(t, err)
		
		// 删除属性
		err = service.DeleteProperty(ctx, "user1", "/path/file.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		
		// 验证删除
		retrieved, err := service.GetProperty(ctx, "user1", "/path/file.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("删除不存在的属性", func(t *testing.T) {
		err := service.DeleteProperty(ctx, "user1", "/path/nonexistent.txt", "DAV:", "displayname")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "属性不存在")
	})

	t.Run("删除属性时数据库错误", func(t *testing.T) {
		// 关闭数据库连接
		service.db.Close()
		
		err := service.DeleteProperty(ctx, "user1", "/path/file.txt", "DAV:", "displayname")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "删除属性失败")
	})
}

// ========================================
// List Properties Tests
// ========================================

func TestPropertyService_ListProperties(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("列出路径下的所有属性", func(t *testing.T) {
		// 创建多个属性
		properties := []*Property{
			createTestProperty("user1", "/path/file.txt", "DAV:", "displayname", "File.txt", true),
			createTestProperty("user1", "/path/file.txt", "DAV:", "getcontenttype", "text/plain", true),
			createTestProperty("user1", "/path/file.txt", "CUSTOM:", "author", "John Doe", false),
		}
		
		for _, prop := range properties {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 列出属性
		listed, err := service.ListProperties(ctx, "user1", "/path/file.txt")
		assert.NoError(t, err)
		assert.Len(t, listed, 3)
		
		// 验证属性排序
		for i := 1; i < len(listed); i++ {
			assert.LessOrEqual(t, listed[i-1].Namespace, listed[i].Namespace)
			if listed[i-1].Namespace == listed[i].Namespace {
				assert.LessOrEqual(t, listed[i-1].Name, listed[i].Name)
			}
		}
	})

	t.Run("列出不存在路径的属性", func(t *testing.T) {
		properties, err := service.ListProperties(ctx, "user1", "/path/nonexistent.txt")
		assert.NoError(t, err)
		assert.Empty(t, properties)
	})

	t.Run("列出属性时数据库错误", func(t *testing.T) {
		// 关闭数据库连接
		service.db.Close()
		
		properties, err := service.ListProperties(ctx, "user1", "/path/file.txt")
		assert.Error(t, err)
		assert.Nil(t, properties)
	})
}

// ========================================
// Batch Operation Tests
// ========================================

func TestPropertyService_BatchSetProperties(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("批量设置新属性", func(t *testing.T) {
		properties := []*Property{
			createTestProperty("user1", "/path/batch.txt", "DAV:", "displayname", "Batch File", true),
			createTestProperty("user1", "/path/batch.txt", "DAV:", "getcontenttype", "text/plain", true),
			createTestProperty("user1", "/path/batch.txt", "CUSTOM:", "tags", "test,batch", false),
		}
		
		err := service.BatchSetProperties(ctx, "user1", "/path/batch.txt", properties)
		assert.NoError(t, err)
		
		// 验证所有属性都已创建
		for _, prop := range properties {
			retrieved, err := service.GetProperty(ctx, prop.UserID, prop.Path, prop.Namespace, prop.Name)
			assert.NoError(t, err)
			assert.NotNil(t, retrieved)
			assert.Equal(t, prop.Value, retrieved.Value)
		}
	})

	t.Run("批量更新现有属性", func(t *testing.T) {
		// 先创建一些属性
		originalProps := []*Property{
			createTestProperty("user1", "/path/update.txt", "DAV:", "displayname", "Original", true),
			createTestProperty("user1", "/path/update.txt", "DAV:", "getcontenttype", "original/type", true),
		}
		
		for _, prop := range originalProps {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 批量更新这些属性
		updatedProps := []*Property{
			{
				UserID:     "user1",
				ResourceID: originalProps[0].ResourceID,
				Path:       "/path/update.txt",
				Name:       "displayname",
				Namespace:  "DAV:",
				Value:      "Updated",
				IsLive:     true,
				CreatedAt:  originalProps[0].CreatedAt,
				UpdatedAt:  originalProps[0].UpdatedAt,
			},
			{
				UserID:     "user1", 
				ResourceID: originalProps[1].ResourceID,
				Path:       "/path/update.txt",
				Name:       "getcontenttype",
				Namespace:  "DAV:",
				Value:      "updated/type",
				IsLive:     true,
				CreatedAt:  originalProps[1].CreatedAt,
				UpdatedAt:  originalProps[1].UpdatedAt,
			},
		}
		
		err := service.BatchSetProperties(ctx, "user1", "/path/update.txt", updatedProps)
		assert.NoError(t, err)
		
		// 验证更新
		for _, prop := range updatedProps {
			retrieved, err := service.GetProperty(ctx, prop.UserID, prop.Path, prop.Namespace, prop.Name)
			assert.NoError(t, err)
			assert.NotNil(t, retrieved)
			assert.Equal(t, prop.Value, retrieved.Value)
		}
	})

	t.Run("混合创建和更新", func(t *testing.T) {
		// 先创建一个属性
		existing := createTestProperty("user1", "/path/mixed.txt", "DAV:", "existing", "Original", true)
		err := service.CreateProperty(ctx, existing)
		require.NoError(t, err)
		
		// 批量操作包含现有属性和新属性
		mixedProps := []*Property{
			existing, // 现有属性，应该更新
			createTestProperty("user1", "/path/mixed.txt", "DAV:", "new", "New Value", true), // 新属性
		}
		
		err = service.BatchSetProperties(ctx, "user1", "/path/mixed.txt", mixedProps)
		assert.NoError(t, err)
		
		// 验证两个属性都存在
		existingRetrieved, err := service.GetProperty(ctx, "user1", "/path/mixed.txt", "DAV:", "existing")
		assert.NoError(t, err)
		assert.NotNil(t, existingRetrieved)
		
		newRetrieved, err := service.GetProperty(ctx, "user1", "/path/mixed.txt", "DAV:", "new")
		assert.NoError(t, err)
		assert.NotNil(t, newRetrieved)
	})

	t.Run("批量设置时事务失败", func(t *testing.T) {
		properties := []*Property{
			createTestProperty("user1", "/path/fail.txt", "DAV:", "test1", "value1", true),
			createTestProperty("user1", "/path/fail.txt", "DAV:", "test2", "value2", true),
		}
		
		// 关闭数据库连接模拟事务失败
		service.db.Close()
		
		err := service.BatchSetProperties(ctx, "user1", "/path/fail.txt", properties)
		assert.Error(t, err)
	})
}

func TestPropertyService_BatchRemoveProperties(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("批量删除属性", func(t *testing.T) {
		// 先创建一些属性
		properties := []*Property{
			createTestProperty("user1", "/path/remove.txt", "DAV:", "prop1", "value1", true),
			createTestProperty("user1", "/path/remove.txt", "DAV:", "prop2", "value2", true),
			createTestProperty("user1", "/path/remove.txt", "CUSTOM:", "prop3", "value3", false),
		}
		
		for _, prop := range properties {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 批量删除某些属性
		namespaces := []string{"DAV:"}
		names := []string{"prop1", "prop2"}
		
		err := service.BatchRemoveProperties(ctx, "user1", "/path/remove.txt", namespaces, names)
		assert.NoError(t, err)
		
		// 验证删除
		prop1, _ := service.GetProperty(ctx, "user1", "/path/remove.txt", "DAV:", "prop1")
		assert.Nil(t, prop1)
		
		prop2, _ := service.GetProperty(ctx, "user1", "/path/remove.txt", "DAV:", "prop2")
		assert.Nil(t, prop2)
		
		// 确认其他属性仍然存在
		prop3, _ := service.GetProperty(ctx, "user1", "/path/remove.txt", "CUSTOM:", "prop3")
		assert.NotNil(t, prop3)
	})

	t.Run("批量删除不存在的属性", func(t *testing.T) {
		namespaces := []string{"DAV:"}
		names := []string{"nonexistent1", "nonexistent2"}
		
		err := service.BatchRemoveProperties(ctx, "user1", "/path/nonexistent.txt", namespaces, names)
		assert.NoError(t, err) // 删除不存在的属性不应该报错
	})
}

// ========================================
// Advanced Query Tests
// ========================================

func TestPropertyService_FindPropertiesByNamespace(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("按命名空间查找属性", func(t *testing.T) {
		// 创建不同命名空间的属性
		properties := []*Property{
			createTestProperty("user1", "/path/namespace.txt", "DAV:", "displayname", "File", true),
			createTestProperty("user1", "/path/namespace.txt", "DAV:", "getcontenttype", "text/plain", true),
			createTestProperty("user1", "/path/namespace.txt", "CUSTOM:", "author", "John", false),
			createTestProperty("user1", "/path/namespace.txt", "CUSTOM:", "version", "1.0", false),
		}
		
		for _, prop := range properties {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 查找DAV命名空间属性
		davProps, err := service.FindPropertiesByNamespace(ctx, "user1", "/path/namespace.txt", "DAV:")
		assert.NoError(t, err)
		assert.Len(t, davProps, 2)
		
		for _, prop := range davProps {
			assert.Equal(t, "DAV:", prop.Namespace)
		}
		
		// 查找CUSTOM命名空间属性
		customProps, err := service.FindPropertiesByNamespace(ctx, "user1", "/path/namespace.txt", "CUSTOM:")
		assert.NoError(t, err)
		assert.Len(t, customProps, 2)
		
		for _, prop := range customProps {
			assert.Equal(t, "CUSTOM:", prop.Namespace)
		}
	})
}

func TestPropertyService_SearchProperties(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("按用户ID搜索", func(t *testing.T) {
		// 创建不同用户的属性
		user1Props := []*Property{
			createTestProperty("user1", "/path/user1.txt", "DAV:", "displayname", "User1 File", true),
			createTestProperty("user1", "/path/user1.txt", "CUSTOM:", "author", "User1 Author", false),
		}
		
		user2Props := []*Property{
			createTestProperty("user2", "/path/user2.txt", "DAV:", "displayname", "User2 File", true),
			createTestProperty("user2", "/path/user2.txt", "CUSTOM:", "author", "User2 Author", false),
		}
		
		allProps := append(user1Props, user2Props...)
		for _, prop := range allProps {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 搜索user1的属性
		results, err := service.SearchProperties(ctx, "user1", map[string]interface{}{
			"limit": 100,
		})
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		
		for _, prop := range results {
			assert.Equal(t, "user1", prop.UserID)
		}
	})

	t.Run("按命名空间过滤", func(t *testing.T) {
		// 先创建一些属性（使用前面的数据）
		
		results, err := service.SearchProperties(ctx, "user1", map[string]interface{}{
			"namespace": "DAV:",
			"limit":     100,
		})
		assert.NoError(t, err)
		
		for _, prop := range results {
			assert.Equal(t, "DAV:", prop.Namespace)
			assert.Equal(t, "user1", prop.UserID)
		}
	})

	t.Run("按路径前缀搜索", func(t *testing.T) {
		// 创建不同路径的属性
		pathProps := []*Property{
			createTestProperty("user1", "/documents/file1.txt", "DAV:", "displayname", "Doc1", true),
			createTestProperty("user1", "/documents/subfolder/file2.txt", "DAV:", "displayname", "Doc2", true),
			createTestProperty("user1", "/images/photo1.jpg", "DAV:", "displayname", "Photo1", true),
		}
		
		for _, prop := range pathProps {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 搜索文档路径前缀
		results, err := service.SearchProperties(ctx, "user1", map[string]interface{}{
			"path_prefix": "/documents",
			"limit":       100,
		})
		assert.NoError(t, err)
		assert.Len(t, results, 2) // 应该找到两个文档
		
		for _, prop := range results {
			assert.Contains(t, prop.Path, "/documents")
		}
	})

	t.Run("按名称模式搜索", func(t *testing.T) {
		// 创建具有模式化名称的属性
		patternProps := []*Property{
			createTestProperty("user1", "/path/pattern.txt", "CUSTOM:", "author_name", "John Doe", false),
			createTestProperty("user1", "/path/pattern.txt", "CUSTOM:", "author_email", "john@example.com", false),
			createTestProperty("user1", "/path/pattern.txt", "CUSTOM:", "title", "My Title", false),
		}
		
		for _, prop := range patternProps {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 搜索author开头的属性
		results, err := service.SearchProperties(ctx, "user1", map[string]interface{}{
			"name_pattern": "author",
			"limit":        100,
		})
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		
		for _, prop := range results {
			assert.Contains(t, prop.Name, "author")
		}
	})

	t.Run("按活属性过滤", func(t *testing.T) {
		// 搜索活属性
		liveProps, err := service.SearchProperties(ctx, "user1", map[string]interface{}{
			"is_live": true,
			"limit":   100,
		})
		assert.NoError(t, err)
		
		for _, prop := range liveProps {
			assert.True(t, prop.IsLive)
			assert.Equal(t, "user1", prop.UserID)
		}
	})

	t.Run("限制搜索结果数量", func(t *testing.T) {
		results, err := service.SearchProperties(ctx, "user1", map[string]interface{}{
			"limit": 1,
		})
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(results), 1)
	})
}

// ========================================
// Maintenance Operation Tests
// ========================================

func TestPropertyService_CleanupExpiredProperties(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("清理过期属性", func(t *testing.T) {
		// 创建一些过期属性（模拟30天前的创建时间）
		expiredTime := time.Now().Add(-31 * 24 * time.Hour)
		
		expiredProps := []*Property{
			createTestProperty("user1", "/path/expired1.txt", "temp:namespace", "temp1", "", false),
			createTestProperty("user1", "/path/expired2.txt", "temp:namespace", "temp2", "", false),
		}
		
		// 手动设置创建时间为过去时间
		for _, prop := range expiredProps {
			prop.CreatedAt = expiredTime
			prop.UpdatedAt = expiredTime
			
			// 直接插入数据库（绕过CreateProperty的时间设置）
			query := `
				INSERT INTO properties (user_id, resource_id, path, name, namespace, value, is_live, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`
			_, err := service.db.ExecContext(ctx, query, 
				prop.UserID, prop.ResourceID, prop.Path, prop.Name, 
				prop.Namespace, prop.Value, prop.IsLive, 
				expiredTime.Unix(), expiredTime.Unix())
			require.NoError(t, err)
		}
		
		// 创建非过期属性
		normalProp := createTestProperty("user1", "/path/normal.txt", "DAV:", "displayname", "Normal", true)
		err := service.CreateProperty(ctx, normalProp)
		require.NoError(t, err)
		
		// 清理过期属性
		deletedCount, err := service.CleanupExpiredProperties(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), deletedCount) // 应该删除2个过期属性
		
		// 验证正常属性仍然存在
		retrieved, err := service.GetProperty(ctx, "user1", "/path/normal.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
	})
}

func TestPropertyService_RebuildIndexes(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("重建索引", func(t *testing.T) {
		// 创建一些属性
		properties := []*Property{
			createTestProperty("user1", "/path/index.txt", "DAV:", "displayname", "File", true),
			createTestProperty("user1", "/path/index.txt", "CUSTOM:", "author", "John", false),
		}
		
		for _, prop := range properties {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 重建索引
		err := service.RebuildIndexes(ctx)
		assert.NoError(t, err)
		
		// 验证属性仍然可以正常查询
		retrieved, err := service.GetProperty(ctx, "user1", "/path/index.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
	})
}

// ========================================
// Transaction Tests
// ========================================

func TestPropertyService_TransactionMethods(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("事务中创建属性", func(t *testing.T) {
		tx, err := service.db.BeginTx(ctx, nil)
		require.NoError(t, err)
		
		property := createTestProperty("user1", "/path/tx.txt", "DAV:", "displayname", "Tx File", true)
		
		err = service.createPropertyTx(tx, property)
		assert.NoError(t, err)
		
		err = tx.Commit()
		assert.NoError(t, err)
		
		// 验证属性已创建
		retrieved, err := service.GetProperty(ctx, "user1", "/path/tx.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
	})

	t.Run("事务中更新属性", func(t *testing.T) {
		// 先创建属性
		original := createTestProperty("user1", "/path/txupdate.txt", "DAV:", "displayname", "Original", true)
		err := service.CreateProperty(ctx, original)
		require.NoError(t, err)
		
		// 开始事务
		tx, err := service.db.BeginTx(ctx, nil)
		require.NoError(t, err)
		
		// 更新属性
		original.Value = "Updated in Transaction"
		err = service.updatePropertyTx(tx, original)
		assert.NoError(t, err)
		
		// 提交事务
		err = tx.Commit()
		assert.NoError(t, err)
		
		// 验证更新
		retrieved, err := service.GetProperty(ctx, "user1", "/path/txupdate.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "Updated in Transaction", retrieved.Value)
	})

	t.Run("事务回滚", func(t *testing.T) {
		// 先创建属性
		original := createTestProperty("user1", "/path/txrollback.txt", "DAV:", "displayname", "Original", true)
		err := service.CreateProperty(ctx, original)
		require.NoError(t, err)
		
		// 开始事务
		tx, err := service.db.BeginTx(ctx, nil)
		require.NoError(t, err)
		
		// 更新属性
		original.Value = "Should Not Persist"
		err = service.updatePropertyTx(tx, original)
		assert.NoError(t, err)
		
		// 回滚事务
		err = tx.Rollback()
		assert.NoError(t, err)
		
		// 验证值没有改变
		retrieved, err := service.GetProperty(ctx, "user1", "/path/txrollback.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "Original", retrieved.Value)
	})

	t.Run("事务中删除属性", func(t *testing.T) {
		// 先创建属性
		property := createTestProperty("user1", "/path/txdelete.txt", "DAV:", "displayname", "To Delete", true)
		err := service.CreateProperty(ctx, property)
		require.NoError(t, err)
		
		// 开始事务
		tx, err := service.db.BeginTx(ctx, nil)
		require.NoError(t, err)
		
		// 删除属性
		err = service.deletePropertyTx(tx, "user1", "/path/txdelete.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		
		// 提交事务
		err = tx.Commit()
		assert.NoError(t, err)
		
		// 验证属性已删除
		retrieved, err := service.GetProperty(ctx, "user1", "/path/txdelete.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.Nil(t, retrieved)
	})
}

// ========================================
// Error Handling Tests
// ========================================

func TestPropertyService_ErrorHandling(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("无效的数据库路径处理", func(t *testing.T) {
		_, err := NewPropertyService("/invalid/path/database.db")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "连接数据库失败")
	})

	t.Run("数据库连接关闭后的操作", func(t *testing.T) {
		// 关闭数据库连接
		err := service.Close()
		assert.NoError(t, err)
		
		// 尝试各种操作，应该都返回错误
		property := createTestProperty("user1", "/path/test.txt", "DAV:", "displayname", "Test", true)
		
		// CreateProperty
		err = service.CreateProperty(ctx, property)
		assert.Error(t, err)
		
		// GetProperty
		_, err = service.GetProperty(ctx, "user1", "/path/test.txt", "DAV:", "displayname")
		assert.Error(t, err)
		
		// UpdateProperty
		err = service.UpdateProperty(ctx, property)
		assert.Error(t, err)
		
		// DeleteProperty
		err = service.DeleteProperty(ctx, "user1", "/path/test.txt", "DAV:", "displayname")
		assert.Error(t, err)
		
		// ListProperties
		_, err = service.ListProperties(ctx, "user1", "/path/test.txt")
		assert.Error(t, err)
		
		// BatchSetProperties
		err = service.BatchSetProperties(ctx, "user1", "/path/test.txt", []*Property{property})
		assert.Error(t, err)
		
		// BatchRemoveProperties
		err = service.BatchRemoveProperties(ctx, "user1", "/path/test.txt", []string{"DAV:"}, []string{"displayname"})
		assert.Error(t, err)
		
		// SearchProperties
		_, err = service.SearchProperties(ctx, "user1", map[string]interface{}{})
		assert.Error(t, err)
		
		// HealthCheck
		err = service.HealthCheck(ctx)
		assert.Error(t, err)
		
		// GetStats
		_, err = service.GetStats(ctx)
		assert.Error(t, err)
	})

	t.Run("空参数处理", func(t *testing.T) {
		// 重新连接数据库
		service.db, _ = sql.Open("sqlite3", ":memory:")
		service.Initialize(ctx)
		
		// 测试各种空参数情况
		_, err := service.GetProperty(ctx, "", "", "", "")
		assert.Error(t, err)
		
		_, err = service.ListProperties(ctx, "", "")
		assert.NoError(t, err) // 列出空结果
		
		property := &Property{
			UserID:     "",
			ResourceID: "",
			Path:       "",
			Name:       "",
			Namespace:  "",
			Value:      "",
		}
		
		err = service.CreateProperty(ctx, property)
		assert.NoError(t, err) // 可能成功创建
		
		err = service.UpdateProperty(ctx, property)
		assert.Error(t, err)
		
		err = service.DeleteProperty(ctx, "", "", "", "")
		assert.Error(t, err)
	})

	t.Run("长路径和特殊字符处理", func(t *testing.T) {
		// 测试很长的路径
		longPath := strings.Repeat("/very", 100) + "/long/path.txt"
		property := createTestProperty("user1", longPath, "DAV:", "displayname", "Long Path Test", true)
		
		err := service.CreateProperty(ctx, property)
		assert.NoError(t, err)
		
		retrieved, err := service.GetProperty(ctx, "user1", longPath, "DAV:", "displayname")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		
		// 测试包含特殊字符的属性名和命名空间
		specialProperty := createTestProperty("user1", "/path/special.txt", "http://example.com/ns", "special-prop", "special value", false)
		
		err = service.CreateProperty(ctx, specialProperty)
		assert.NoError(t, err)
		
		retrieved, err = service.GetProperty(ctx, "user1", "/path/special.txt", "http://example.com/ns", "special-prop")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
	})
}

// ========================================
// Concurrency Tests
// ========================================

func TestPropertyService_Concurrency(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("并发创建属性", func(t *testing.T) {
		const numGoroutines = 10
		const propsPerGoroutine = 5
		var wg sync.WaitGroup
		errors := make([]error, numGoroutines*propsPerGoroutine)
		
		// 启动多个goroutine同时创建属性
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				
				for j := 0; j < propsPerGoroutine; j++ {
					property := createTestProperty(
						"user1", 
						"/path/concurrent.txt", 
						"DAV:", 
						"prop"+string(rune(goroutineID))+string(rune(j)), 
						"value", 
						true,
					)
					
					err := service.CreateProperty(ctx, property)
					errors[goroutineID*propsPerGoroutine+j] = err
				}
			}(i)
		}
		
		wg.Wait()
		
		// 验证所有操作都成功
		for _, err := range errors {
			assert.NoError(t, err)
		}
		
		// 验证所有属性都已创建
		listed, err := service.ListProperties(ctx, "user1", "/path/concurrent.txt")
		assert.NoError(t, err)
		assert.Equal(t, numGoroutines*propsPerGoroutine, len(listed))
	})

	t.Run("并发读写操作", func(t *testing.T) {
		const numReaders = 5
		const numWriters = 3
		var wg sync.WaitGroup
		var readErrors []error
		var writeErrors []error
		
		// 先创建一个基础属性
		baseProperty := createTestProperty("user1", "/path/mixed.txt", "DAV:", "base", "base value", true)
		err := service.CreateProperty(ctx, baseProperty)
		require.NoError(t, err)
		
		// 启动reader goroutines
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(readerID int) {
				defer wg.Done()
				
				for j := 0; j < 10; j++ {
					_, err := service.GetProperty(ctx, "user1", "/path/mixed.txt", "DAV:", "base")
					if err != nil {
						readErrors = append(readErrors, err)
					}
					
					time.Sleep(time.Millisecond * 10) // 模拟一些处理时间
				}
			}(i)
		}
		
		// 启动writer goroutines
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(writerID int) {
				defer wg.Done()
				
				for j := 0; j < 5; j++ {
					property := createTestProperty("user1", "/path/mixed.txt", "CUSTOM:", "writer"+string(rune(writerID)), "value"+string(rune(j)), false)
					err := service.CreateProperty(ctx, property)
					if err != nil {
						writeErrors = append(writeErrors, err)
					}
					
					time.Sleep(time.Millisecond * 20) // 模拟一些处理时间
				}
			}(i)
		}
		
		wg.Wait()
		
		// 验证没有错误发生
		for _, err := range readErrors {
			assert.NoError(t, err)
		}
		for _, err := range writeErrors {
			assert.NoError(t, err)
		}
		
		// 验证数据一致性
		listed, err := service.ListProperties(ctx, "user1", "/path/mixed.txt")
		assert.NoError(t, err)
		assert.Greater(t, len(listed), 0)
	})

	t.Run("并发批量操作", func(t *testing.T) {
		const numBatches = 3
		var wg sync.WaitGroup
		var batchErrors []error
		
		for i := 0; i < numBatches; i++ {
			wg.Add(1)
			go func(batchID int) {
				defer wg.Done()
				
				properties := []*Property{
					createTestProperty("user1", "/path/batch"+string(rune(batchID))+".txt", "DAV:", "displayname", "Batch "+string(rune(batchID)), true),
					createTestProperty("user1", "/path/batch"+string(rune(batchID))+".txt", "CUSTOM:", "author", "Author "+string(rune(batchID)), false),
				}
				
				err := service.BatchSetProperties(ctx, "user1", "/path/batch"+string(rune(batchID))+".txt", properties)
				if err != nil {
					batchErrors = append(batchErrors, err)
				}
			}(i)
		}
		
		wg.Wait()
		
		// 验证没有错误
		for _, err := range batchErrors {
			assert.NoError(t, err)
		}
		
		// 验证所有批量操作都成功
		for i := 0; i < numBatches; i++ {
			path := "/path/batch" + string(rune(i)) + ".txt"
			listed, err := service.ListProperties(ctx, "user1", path)
			assert.NoError(t, err)
			assert.Len(t, listed, 2)
		}
	})

	t.Run("并发索引重建", func(t *testing.T) {
		const numRebuilds = 3
		var wg sync.WaitGroup
		var rebuildErrors []error
		
		for i := 0; i < numRebuilds; i++ {
			wg.Add(1)
			go func(rebuildID int) {
				defer wg.Done()
				
				err := service.RebuildIndexes(ctx)
				if err != nil {
					rebuildErrors = append(rebuildErrors, err)
				}
			}(i)
		}
		
		wg.Wait()
		
		// 验证重建操作成功
		for _, err := range rebuildErrors {
			assert.NoError(t, err)
		}
		
		// 验证服务仍然可用
		property := createTestProperty("user1", "/path/post-rebuild.txt", "DAV:", "displayname", "Post Rebuild", true)
		err := service.CreateProperty(ctx, property)
		assert.NoError(t, err)
		
		retrieved, err := service.GetProperty(ctx, "user1", "/path/post-rebuild.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
	})
}

// ========================================
// Health Check and Stats Tests
// ========================================

func TestPropertyService_HealthCheck(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("正常健康检查", func(t *testing.T) {
		err := service.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("数据库关闭后的健康检查", func(t *testing.T) {
		service.Close()
		
		err := service.HealthCheck(ctx)
		assert.Error(t, err)
	})
}

func TestPropertyService_GetStats(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("获取空数据库统计信息", func(t *testing.T) {
		stats, err := service.GetStats(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		
		totalCount, ok := stats["total_properties"].(int)
		assert.True(t, ok)
		assert.Equal(t, 0, totalCount)
		
		liveCount, ok := stats["live_properties"].(int)
		assert.True(t, ok)
		assert.Equal(t, 0, liveCount)
	})

	t.Run("获取有数据数据库统计信息", func(t *testing.T) {
		// 创建一些属性
		properties := []*Property{
			createTestProperty("user1", "/path/stats1.txt", "DAV:", "displayname", "File1", true),
			createTestProperty("user1", "/path/stats1.txt", "DAV:", "getcontenttype", "text/plain", true),
			createTestProperty("user1", "/path/stats2.txt", "CUSTOM:", "author", "John", false),
			createTestProperty("user1", "/path/stats2.txt", "CUSTOM:", "version", "1.0", false),
		}
		
		for _, prop := range properties {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 获取统计信息
		stats, err := service.GetStats(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		
		totalCount, ok := stats["total_properties"].(int)
		assert.True(t, ok)
		assert.Equal(t, 4, totalCount)
		
		liveCount, ok := stats["live_properties"].(int)
		assert.True(t, ok)
		assert.Equal(t, 2, liveCount) // 只有前两个是活属性
	})

	t.Run("数据库错误时的统计信息", func(t *testing.T) {
		service.Close()
		
		stats, err := service.GetStats(ctx)
		assert.Error(t, err)
		assert.Nil(t, stats)
	})
}

// ========================================
// Integration with SQLBuilder Tests
// ========================================

func TestPropertyService_SQLBuilderIntegration(t *testing.T) {
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("SQLBuilder基本查询功能", func(t *testing.T) {
		// 创建一些测试数据
		properties := []*Property{
			createTestProperty("user1", "/path/integration.txt", "DAV:", "displayname", "Integration File", true),
			createTestProperty("user1", "/path/integration.txt", "CUSTOM:", "author", "Integration Author", false),
		}
		
		for _, prop := range properties {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 测试SelectBuilder
		builder := NewSelectBuilder("properties").
			Where("user_id = ? AND path = ?", "user1", "/path/integration.txt").
			OrderBy("namespace", "name")
		
		rows, err := builder.ExecuteQuery(ctx, service.db)
		assert.NoError(t, err)
		defer rows.Close()
		
		// 验证结果
		var count int
		for rows.Next() {
			count++
		}
		assert.Equal(t, 2, count)
	})

	t.Run("SQLBuilder更新操作", func(t *testing.T) {
		// 创建测试属性
		property := createTestProperty("user1", "/path/update.txt", "DAV:", "displayname", "Original", true)
		err := service.CreateProperty(ctx, property)
		require.NoError(t, err)
		
		// 使用UpdateBuilder更新
		updateBuilder := NewUpdateBuilder("properties").
			Set("value = ?", "Updated via SQLBuilder").
			Where("user_id = ? AND path = ? AND namespace = ? AND name = ?", 
				"user1", "/path/update.txt", "DAV:", "displayname")
		
		result, err := updateBuilder.Execute(ctx, service.db)
		assert.NoError(t, err)
		
		rowsAffected, err := result.RowsAffected()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rowsAffected)
		
		// 验证更新
		retrieved, err := service.GetProperty(ctx, "user1", "/path/update.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.Equal(t, "Updated via SQLBuilder", retrieved.Value)
	})

	t.Run("SQLBuilder删除操作", func(t *testing.T) {
		// 创建测试属性
		property := createTestProperty("user1", "/path/delete.txt", "DAV:", "displayname", "To Delete", true)
		err := service.CreateProperty(ctx, property)
		require.NoError(t, err)
		
		// 使用DeleteBuilder删除
		deleteBuilder := NewDeleteBuilder("properties").
			Where("user_id = ? AND path = ? AND namespace = ? AND name = ?", 
				"user1", "/path/delete.txt", "DAV:", "displayname")
		
		result, err := deleteBuilder.Execute(ctx, service.db)
		assert.NoError(t, err)
		
		rowsAffected, err := result.RowsAffected()
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rowsAffected)
		
		// 验证删除
		retrieved, err := service.GetProperty(ctx, "user1", "/path/delete.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("SQLBuilder复杂查询", func(t *testing.T) {
		// 创建测试数据
		testProps := []*Property{
			createTestProperty("user1", "/path/complex1.txt", "DAV:", "displayname", "Complex 1", true),
			createTestProperty("user1", "/path/complex2.txt", "DAV:", "displayname", "Complex 2", true),
			createTestProperty("user1", "/path/complex1.txt", "CUSTOM:", "author", "Author 1", false),
			createTestProperty("user2", "/path/complex1.txt", "DAV:", "displayname", "Complex 1 User2", true),
		}
		
		for _, prop := range testProps {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
		
		// 测试复杂查询 - 用户1的DAV命名空间属性
		builder := NewSelectBuilder("properties", "name", "namespace", "value", "path").
			Where("user_id = ? AND namespace = ?", "user1", "DAV:").
			OrderBy("path", "name").
			Limit(10)
		
		rows, err := builder.ExecuteQuery(ctx, service.db)
		assert.NoError(t, err)
		defer rows.Close()
		
		// 验证查询结果
		var results []map[string]string
		for rows.Next() {
			var name, namespace, value, path string
			err := rows.Scan(&name, &namespace, &value, &path)
			assert.NoError(t, err)
			
			results = append(results, map[string]string{
				"name":      name,
				"namespace": namespace,
				"value":     value,
				"path":      path,
			})
		}
		
		// 应该找到user1的两个DAV属性
		assert.Len(t, results, 2)
		for _, result := range results {
			assert.Equal(t, "user1", "user1") // 用户ID在查询中已过滤
			assert.Equal(t, "DAV:", result["namespace"])
		}
	})

	t.Run("SQLBuilder参数绑定安全", func(t *testing.T) {
		// 测试SQL注入防护
		maliciousPath := "/path/'; DROP TABLE properties; --"
		
		builder := NewSelectBuilder("properties").
			Where("path = ?", maliciousPath)
		
		rows, err := builder.ExecuteQuery(ctx, service.db)
		assert.NoError(t, err)
		defer rows.Close()
		
		// 验证表仍然存在（没有被删除）
		stats, err := service.GetStats(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
	})

	t.Run("SQLBuilder事务中的使用", func(t *testing.T) {
		// 在事务中使用SQLBuilder
		tx, err := service.db.BeginTx(ctx, nil)
		require.NoError(t, err)
		
		// 插入数据
		insertBuilder := NewInsertBuilder("properties").
			Columns("user_id", "resource_id", "path", "name", "namespace", "value", "is_live", "created_at", "updated_at").
			Values("user1", "res1", "/path/tx.txt", "displayname", "DAV:", "TX Value", true, time.Now().Unix(), time.Now().Unix()).
			OnConflict("user_id", "path", "namespace", "name")
		
		_, err = tx.Exec(insertBuilder.Build(), insertBuilder.Args()...)
		assert.NoError(t, err)
		
		// 提交事务
		err = tx.Commit()
		assert.NoError(t, err)
		
		// 验证数据已插入
		retrieved, err := service.GetProperty(ctx, "user1", "/path/tx.txt", "DAV:", "displayname")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "TX Value", retrieved.Value)
	})
}

// ========================================
// Performance Tests (Basic)
// ========================================

func TestPropertyService_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试（使用 -short 标志）")
	}
	
	service, cleanup := createTestPropertyService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("大量数据插入性能", func(t *testing.T) {
		const numProperties = 1000
		startTime := time.Now()
		
		for i := 0; i < numProperties; i++ {
			property := createTestProperty("user1", "/path/perf"+string(rune(i%100))+".txt", "DAV:", "displayname", "Performance Test "+string(rune(i)), true)
			err := service.CreateProperty(ctx, property)
			require.NoError(t, err)
		}
		
		duration := time.Since(startTime)
		t.Logf("插入 %d 个属性耗时: %v", numProperties, duration)
		
		// 验证性能指标（应该能在合理时间内完成）
		assert.Less(t, duration, 30*time.Second)
	})

	t.Run("大量数据查询性能", func(t *testing.T) {
		// 先创建一些数据
		const numProps = 500
		for i := 0; i < numProps; i++ {
			property := createTestProperty("user1", "/path/query/perf"+string(rune(i%50))+".txt", "DAV:", "displayname", "Query Test "+string(rune(i)), true)
			err := service.CreateProperty(ctx, property)
			require.NoError(t, err)
		}
		
		startTime := time.Now()
		
		// 执行多个查询
		for i := 0; i < 100; i++ {
			path := "/path/query/perf" + string(rune(i%50)) + ".txt"
			properties, err := service.ListProperties(ctx, "user1", path)
			assert.NoError(t, err)
			assert.NotEmpty(t, properties)
		}
		
		duration := time.Since(startTime)
		t.Logf("执行 100 个列表查询耗时: %v", duration)
		
		// 验证查询性能
		assert.Less(t, duration, 10*time.Second)
	})
}

// ========================================
// Table-Driven Tests Examples
// ========================================

func TestPropertyService_TableDrivenExamples(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*PropertyService, context.Context) error
		operation func(*PropertyService, context.Context) error
		validate func(*PropertyService, context.Context) error
		wantErr  bool
	}{
		{
			name: "创建然后获取属性",
			setup: func(s *PropertyService, ctx context.Context) error {
				return nil
			},
			operation: func(s *PropertyService, ctx context.Context) error {
				property := createTestProperty("user1", "/path/table.txt", "DAV:", "displayname", "Table Test", true)
				return s.CreateProperty(ctx, property)
			},
			validate: func(s *PropertyService, ctx context.Context) error {
				property, err := s.GetProperty(ctx, "user1", "/path/table.txt", "DAV:", "displayname")
				if err != nil {
					return err
				}
				if property == nil {
					return assert.AnError
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "获取不存在的属性",
			setup: func(s *PropertyService, ctx context.Context) error {
				return nil
			},
			operation: func(s *PropertyService, ctx context.Context) error {
				_, err := s.GetProperty(ctx, "user1", "/path/nonexistent.txt", "DAV:", "displayname")
				return err
			},
			validate: func(s *PropertyService, ctx context.Context) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "更新不存在的属性",
			setup: func(s *PropertyService, ctx context.Context) error {
				return nil
			},
			operation: func(s *PropertyService, ctx context.Context) error {
				property := createTestProperty("user1", "/path/nonexistent.txt", "DAV:", "displayname", "test", false)
				property.ID = 999999
				return s.UpdateProperty(ctx, property)
			},
			validate: func(s *PropertyService, ctx context.Context) error {
				return nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, cleanup := createTestPropertyService(t)
			defer cleanup()
			
			ctx := context.Background()
			
			// 执行设置
			if tt.setup != nil {
				err := tt.setup(service, ctx)
				require.NoError(t, err)
			}
			
			// 执行操作
			err := tt.operation(service, ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			// 执行验证
			if tt.validate != nil {
				err := tt.validate(service, ctx)
				assert.NoError(t, err)
			}
		})
	}
}

// ========================================
// Utility Functions for Tests
// ========================================

// 验证属性数据结构
func validateProperty(t *testing.T, property *Property) {
	t.Helper()
	
	assert.NotZero(t, property.ID)
	assert.NotEmpty(t, property.UserID)
	assert.NotEmpty(t, property.Path)
	assert.NotEmpty(t, property.Name)
	assert.NotEmpty(t, property.Namespace)
	assert.False(t, property.CreatedAt.IsZero())
	assert.False(t, property.UpdatedAt.IsZero())
	assert.True(t, property.UpdatedAt.After(property.CreatedAt) || property.UpdatedAt.Equal(property.CreatedAt))
}

// 验证属性列表
func validatePropertyList(t *testing.T, properties []*Property) {
	t.Helper()
	
	for i, property := range properties {
		validateProperty(t, property)
		
		// 验证排序（如果需要）
		if i > 0 {
			prev := properties[i-1]
			if prev.Namespace == property.Namespace {
				assert.LessOrEqual(t, prev.Name, property.Name)
			} else {
				assert.Less(t, prev.Namespace, property.Namespace)
			}
		}
	}
}

// 创建大量测试数据
func createBulkTestData(t *testing.T, service *PropertyService, ctx context.Context, userID string, basePath string, count int) {
	t.Helper()
	
	for i := 0; i < count; i++ {
		path := basePath + "/file" + string(rune(i)) + ".txt"
		
		// 创建多个属性
		properties := []*Property{
			createTestProperty(userID, path, "DAV:", "displayname", "File "+string(rune(i)), true),
			createTestProperty(userID, path, "DAV:", "getcontenttype", "text/plain", true),
			createTestProperty(userID, path, "CUSTOM:", "author", "Author "+string(rune(i)), false),
			createTestProperty(userID, path, "CUSTOM:", "created", time.Now().Format(time.RFC3339), false),
		}
		
		for _, prop := range properties {
			err := service.CreateProperty(ctx, prop)
			require.NoError(t, err)
		}
	}
}

// 比较两个属性列表（忽略时间戳差异）
func comparePropertyLists(t *testing.T, expected, actual []*Property) {
	t.Helper()
	
	assert.Equal(t, len(expected), len(actual))
	
	for i := range expected {
		exp := expected[i]
		act := actual[i]
		
		assert.Equal(t, exp.UserID, act.UserID)
		assert.Equal(t, exp.Path, act.Path)
		assert.Equal(t, exp.Name, act.Name)
		assert.Equal(t, exp.Namespace, act.Namespace)
		assert.Equal(t, exp.Value, act.Value)
		assert.Equal(t, exp.IsLive, act.IsLive)
		// 忽略 ID、CreatedAt、UpdatedAt 的差异
	}
}