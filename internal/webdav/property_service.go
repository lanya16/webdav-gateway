package webdav

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/webdav-gateway/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

// DatabaseProperty 数据库属性类型（带ID字段）
type DatabaseProperty struct {
	ID          int    `json:"id"`
	UserID      string `json:"user_id"`
	ResourceID  string `json:"resource_id"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Value       string `json:"value"`
	IsLive      bool   `json:"is_live"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// PropertyToDatabaseProperty 将Property转换为DatabaseProperty
func PropertyToDatabaseProperty(prop Property) *DatabaseProperty {
	return &DatabaseProperty{
		ID:         0, // 将由数据库自动生成
		UserID:     prop.UserID,
		ResourceID: prop.ResourceID,
		Path:       prop.Path,
		Name:       prop.Name,
		Namespace:  prop.Namespace,
		Value:      prop.Value,
		IsLive:     prop.IsLive,
		CreatedAt:  0, // 将由数据库自动生成
		UpdatedAt:  0, // 将由数据库自动生成
	}
}

// DatabasePropertyToProperty 将DatabaseProperty转换为Property
func DatabasePropertyToProperty(dbProp *DatabaseProperty) Property {
	return Property{
		Name:        dbProp.Name,
		Namespace:   dbProp.Namespace,
		Value:       dbProp.Value,
		IsLive:      dbProp.IsLive,
		UserID:      dbProp.UserID,
		ResourceID:  dbProp.ResourceID,
		Path:        dbProp.Path,
		CreatedAt:   time.Unix(dbProp.CreatedAt, 0),
		UpdatedAt:   time.Unix(dbProp.UpdatedAt, 0),
	}
}

// DatabasePropertyToPropertySlice 将DatabaseProperty切片转换为Property切片
func DatabasePropertyToPropertySlice(dbProps []*DatabaseProperty) []*Property {
	props := make([]*Property, len(dbProps))
	for i, dbProp := range dbProps {
		prop := DatabasePropertyToProperty(dbProp)
		props[i] = &prop
	}
	return props
}

// ========================================
// 重构后的属性存储服务
// ========================================

// PropertyService 属性存储服务（重构版）
type PropertyService struct {
	db      *sql.DB
	dbPath  string
	mu      sync.RWMutex
	initialised bool
}

// NewPropertyService 创建属性存储服务
func NewPropertyService(dbPath string) (*PropertyService, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	service := &PropertyService{
		db:     db,
		dbPath: dbPath,
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return service, nil
}

// Initialize 初始化数据库表
func (s *PropertyService) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialised {
		return nil
	}

	// 创建属性表
	if err := s.createPropertiesTable(ctx); err != nil {
		return fmt.Errorf("创建属性表失败: %v", err)
	}

	// 创建索引
	if err := s.createIndexes(ctx); err != nil {
		return fmt.Errorf("创建索引失败: %v", err)
	}

	s.initialised = true
	return nil
}

// createPropertiesTable 创建属性表
func (s *PropertyService) createPropertiesTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS properties (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			path TEXT NOT NULL,
			name TEXT NOT NULL,
			namespace TEXT NOT NULL,
			value TEXT,
			is_live INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			UNIQUE(user_id, path, namespace, name)
		);
	`
	
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// createIndexes 创建索引
func (s *PropertyService) createIndexes(ctx context.Context) error {
	indexes := []struct {
		name string
		sql  string
	}{
		{"idx_properties_user_path", "CREATE INDEX IF NOT EXISTS idx_properties_user_path ON properties(user_id, path)"},
		{"idx_properties_namespace", "CREATE INDEX IF NOT EXISTS idx_properties_namespace ON properties(namespace)"},
		{"idx_properties_name", "CREATE INDEX IF NOT EXISTS idx_properties_name ON properties(name)"},
		{"idx_properties_user_path_namespace", "CREATE INDEX IF NOT EXISTS idx_properties_user_path_namespace ON properties(user_id, path, namespace)"},
		{"idx_properties_user_path_name", "CREATE INDEX IF NOT EXISTS idx_properties_user_path_name ON properties(user_id, path, name)"},
		{"idx_properties_created_at", "CREATE INDEX IF NOT EXISTS idx_properties_created_at ON properties(created_at)"},
		{"idx_properties_is_live", "CREATE INDEX IF NOT EXISTS idx_properties_is_live ON properties(is_live)"},
	}

	for _, index := range indexes {
		if _, err := s.db.ExecContext(ctx, index.sql); err != nil {
			return fmt.Errorf("创建索引 %s 失败: %v", index.name, err)
		}
	}

	return nil
}

// ========================================
// 基本CRUD操作（使用SQLBuilder）
// ========================================

// GetProperty 获取单个属性
func (s *PropertyService) GetProperty(ctx context.Context, userID, path, namespace, name string) (*DatabaseProperty, error) {
	builder := NewSelectBuilder("properties").
		Where("user_id = ? AND path = ? AND namespace = ? AND name = ?", userID, path, namespace, name)

	row := builder.ExecuteQueryRow(ctx, s.db)
	
	property, err := s.scanProperty(row)
	if err == sql.ErrNoRows {
		return nil, nil // 属性不存在
	}
	return property, err
}

// ListProperties 列出路径下的所有属性
func (s *PropertyService) ListProperties(ctx context.Context, userID, path string) ([]*Property, error) {
	dbProps, err := s.listProperties(ctx, userID, path)
	if err != nil {
		return nil, err
	}
	return DatabasePropertyToPropertySlice(dbProps), nil
}

// listProperties 内部方法，返回DatabaseProperty
func (s *PropertyService) listProperties(ctx context.Context, userID, path string) ([]*DatabaseProperty, error) {
	builder := NewSelectBuilder("properties", "id", "user_id", "resource_id", "path", "name", "namespace", "value", "is_live", "created_at", "updated_at").
		Where("user_id = ? AND path = ?", userID, path).
		OrderBy("namespace", "name")

	rows, err := builder.ExecuteQuery(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("查询属性列表失败: %v", err)
	}
	defer rows.Close()

	return s.scanProperties(rows)
}

// CreateProperty 创建新属性
func (s *PropertyService) CreateProperty(ctx context.Context, property *DatabaseProperty) error {
	now := time.Now()
	property.CreatedAt = now.Unix()
	property.UpdatedAt = now.Unix()

	builder := NewInsertBuilder("properties").
		Columns("user_id", "resource_id", "path", "name", "namespace", "value", "is_live", "created_at", "updated_at").
		Values(property.UserID, property.ResourceID, property.Path, property.Name, property.Namespace, property.Value, property.IsLive, now.Unix(), now.Unix()).
		OnConflict("user_id", "path", "namespace", "name")

	result, err := builder.Execute(ctx, s.db)
	if err != nil {
		return fmt.Errorf("创建属性失败: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取新属性ID失败: %v", err)
	}

	property.ID = int(id)
	return nil
}

// UpdateProperty 更新属性
func (s *PropertyService) UpdateProperty(ctx context.Context, property *DatabaseProperty) error {
	now := time.Now()
	property.UpdatedAt = now.Unix()

	builder := NewUpdateBuilder("properties").
		Set("value = ?", property.Value).
		Set("is_live = ?", property.IsLive).
		Set("updated_at = ?", now.Unix()).
		Where("user_id = ? AND path = ? AND namespace = ? AND name = ?", property.UserID, property.Path, property.Namespace, property.Name)

	result, err := builder.Execute(ctx, s.db)
	if err != nil {
		return fmt.Errorf("更新属性失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("属性不存在")
	}

	return nil
}

// DeleteProperty 删除属性
func (s *PropertyService) DeleteProperty(ctx context.Context, userID, path, namespace, name string) error {
	builder := NewDeleteBuilder("properties").
		Where("user_id = ? AND path = ? AND namespace = ? AND name = ?", userID, path, namespace, name)

	result, err := builder.Execute(ctx, s.db)
	if err != nil {
		return fmt.Errorf("删除属性失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("属性不存在")
	}

	return nil
}

// ========================================
// 批量操作
// ========================================

// BatchSetProperties 批量设置属性
func (s *PropertyService) BatchSetProperties(ctx context.Context, userID, path string, properties []*Property) error {
	dbProps := make([]*DatabaseProperty, len(properties))
	for i, prop := range properties {
		dbProps[i] = PropertyToDatabaseProperty(*prop)
	}
	return s.batchSetProperties(ctx, userID, path, dbProps)
}

// batchSetProperties 内部方法
func (s *PropertyService) batchSetProperties(ctx context.Context, userID, path string, properties []*DatabaseProperty) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	for _, property := range properties {
		// 检查属性是否已存在
		existing, err := s.getPropertyTx(tx, userID, path, property.Namespace, property.Name)
		if err != nil {
			return fmt.Errorf("检查属性存在性失败: %v", err)
		}

		if existing != nil {
			// 更新现有属性
			if err := s.updatePropertyTx(tx, property); err != nil {
				return fmt.Errorf("更新属性失败: %v", err)
			}
		} else {
			// 创建新属性
			if err := s.createPropertyTx(tx, property); err != nil {
				return fmt.Errorf("创建属性失败: %v", err)
			}
		}
	}

	return tx.Commit()
}

// BatchRemoveProperties 批量删除属性
func (s *PropertyService) BatchRemoveProperties(ctx context.Context, userID, path string, namespaces []string, names []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	for _, namespace := range namespaces {
		for _, name := range names {
			if err := s.deletePropertyTx(tx, userID, path, namespace, name); err != nil {
				return fmt.Errorf("删除属性失败: %v", err)
			}
		}
	}

	return tx.Commit()
}

// ========================================
// 事务辅助方法（保持简洁）
// ========================================

// getPropertyTx 事务中获取属性
func (s *PropertyService) getPropertyTx(tx *sql.Tx, userID, path, namespace, name string) (*DatabaseProperty, error) {
	builder := NewSelectBuilder("properties").
		Where("user_id = ? AND path = ? AND namespace = ? AND name = ?", userID, path, namespace, name)

	row := tx.QueryRow(builder.Build(), builder.Args()...)
	
	property, err := s.scanProperty(row)
	if err == sql.ErrNoRows {
		return nil, nil // 属性不存在
	}
	return property, err
}

// createPropertyTx 事务中创建属性
func (s *PropertyService) createPropertyTx(tx *sql.Tx, property *DatabaseProperty) error {
	now := time.Now()
	property.CreatedAt = now.Unix()
	property.UpdatedAt = now.Unix()

	builder := NewInsertBuilder("properties").
		Columns("user_id", "resource_id", "path", "name", "namespace", "value", "is_live", "created_at", "updated_at").
		Values(property.UserID, property.ResourceID, property.Path, property.Name, property.Namespace, property.Value, property.IsLive, now.Unix(), now.Unix()).
		OnConflict("user_id", "path", "namespace", "name")

	result, err := tx.Exec(builder.Build(), builder.Args()...)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	property.ID = int(id)
	return nil
}

// updatePropertyTx 事务中更新属性
func (s *PropertyService) updatePropertyTx(tx *sql.Tx, property *DatabaseProperty) error {
	now := time.Now()
	property.UpdatedAt = now.Unix()

	builder := NewUpdateBuilder("properties").
		Set("value = ?", property.Value).
		Set("is_live = ?", property.IsLive).
		Set("updated_at = ?", now.Unix()).
		Where("user_id = ? AND path = ? AND namespace = ? AND name = ?", property.UserID, property.Path, property.Namespace, property.Name)

	_, err := tx.Exec(builder.Build(), builder.Args()...)
	return err
}

// deletePropertyTx 事务中删除属性
func (s *PropertyService) deletePropertyTx(tx *sql.Tx, userID, path, namespace, name string) error {
	builder := NewDeleteBuilder("properties").
		Where("user_id = ? AND path = ? AND namespace = ? AND name = ?", userID, path, namespace, name)

	_, err := tx.Exec(builder.Build(), builder.Args()...)
	return err
}

// ========================================
// 通用扫描方法
// ========================================

// scanProperty 扫描单个属性
func (s *PropertyService) scanProperty(row *sql.Row) (*DatabaseProperty, error) {
	property := &DatabaseProperty{}
	var createdAt, updatedAt int64
	
	err := row.Scan(
		&property.ID,
		&property.UserID,
		&property.ResourceID,
		&property.Path,
		&property.Name,
		&property.Namespace,
		&property.Value,
		&property.IsLive,
		&createdAt,
		&updatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	property.CreatedAt = createdAt
	property.UpdatedAt = updatedAt
	
	return property, nil
}

// scanProperties 扫描多个属性
func (s *PropertyService) scanProperties(rows *sql.Rows) ([]*DatabaseProperty, error) {
	var properties []*DatabaseProperty
	
	for rows.Next() {
		property := &DatabaseProperty{}
		var createdAt, updatedAt int64
		
		err := rows.Scan(
			&property.ID,
			&property.UserID,
			&property.ResourceID,
			&property.Path,
			&property.Name,
			&property.Namespace,
			&property.Value,
			&property.IsLive,
			&createdAt,
			&updatedAt,
		)
		
		if err != nil {
			return nil, fmt.Errorf("扫描属性记录失败: %v", err)
		}
		
		property.CreatedAt = createdAt
		property.UpdatedAt = updatedAt
		properties = append(properties, property)
	}
	
	return properties, nil
}

// Close 关闭数据库连接
func (s *PropertyService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	return s.db.Close()
}

// HealthCheck 健康检查
func (s *PropertyService) HealthCheck(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "SELECT 1")
	return err
}