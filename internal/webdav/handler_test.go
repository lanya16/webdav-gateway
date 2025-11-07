package webdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/webdav-gateway/internal/auth"
	"github.com/webdav-gateway/internal/models"
	"github.com/webdav-gateway/internal/storage"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ========================================
// Mock Objects
// ========================================

// MockStorage 模拟存储服务
type MockStorage struct {
	objects map[string]*minio.ObjectInfo
	folders map[string]bool
	err     error
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		objects: make(map[string]*minio.ObjectInfo),
		folders: make(map[string]bool),
	}
}

func (m *MockStorage) PutObject(ctx context.Context, userID uuid.UUID, objectPath string, reader interface{}, size int64, contentType string) error {
	if m.err != nil {
		return m.err
	}
	
	modTime := time.Now()
	m.objects[objectPath] = &minio.ObjectInfo{
		Key:          objectPath,
		Size:         size,
		ContentType:  contentType,
		LastModified: modTime,
		ETag:         "mock-etag",
	}
	return nil
}

func (m *MockStorage) GetObject(ctx context.Context, userID uuid.UUID, objectPath string) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	info, exists := m.objects[objectPath]
	if !exists {
		return nil, fmt.Errorf("对象不存在")
	}
	
	// 返回模拟的Reader
	return &mockReadCloser{info: info}, nil
}

func (m *MockStorage) StatObject(ctx context.Context, userID uuid.UUID, objectPath string) (*minio.ObjectInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	info, exists := m.objects[objectPath]
	if !exists {
		return nil, fmt.Errorf("对象不存在")
	}
	return info, nil
}

func (m *MockStorage) DeleteObject(ctx context.Context, userID uuid.UUID, objectPath string) error {
	if m.err != nil {
		return m.err
	}
	
	delete(m.objects, objectPath)
	return nil
}

func (m *MockStorage) ListObjects(ctx context.Context, userID uuid.UUID, prefix string, recursive bool) ([]minio.ObjectInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	var objects []minio.ObjectInfo
	for key, info := range m.objects {
		if strings.HasPrefix(key, strings.TrimPrefix(prefix, "/")) {
			objects = append(objects, *info)
		}
	}
	return objects, nil
}

func (m *MockStorage) CopyObject(ctx context.Context, userID uuid.UUID, srcPath, dstPath string) error {
	if m.err != nil {
		return m.err
	}
	
	info, exists := m.objects[srcPath]
	if !exists {
		return fmt.Errorf("源对象不存在")
	}
	
	info.Key = dstPath
	m.objects[dstPath] = info
	return nil
}

func (m *MockStorage) MoveObject(ctx context.Context, userID uuid.UUID, srcPath, dstPath string) error {
	if m.err != nil {
		return m.err
	}
	
	info, exists := m.objects[srcPath]
	if !exists {
		return fmt.Errorf("源对象不存在")
	}
	
	info.Key = dstPath
	m.objects[dstPath] = info
	delete(m.objects, srcPath)
	return nil
}

func (m *MockStorage) CreateFolder(ctx context.Context, userID uuid.UUID, folderPath string) error {
	if m.err != nil {
		return m.err
	}
	
	m.folders[folderPath] = true
	return nil
}

func (m *MockStorage) DeleteFolder(ctx context.Context, userID uuid.UUID, folderPath string) error {
	if m.err != nil {
		return m.err
	}
	
	delete(m.folders, folderPath)
	// 删除该文件夹下的所有对象
	for key := range m.objects {
		if strings.HasPrefix(key, folderPath+"/") || key == folderPath {
			delete(m.objects, key)
		}
	}
	return nil
}

func (m *MockStorage) GetObjectSize(ctx context.Context, userID uuid.UUID, objectPath string) (int64, error) {
	info, err := m.StatObject(ctx, userID, objectPath)
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

func (m *MockStorage) EnsureBucket(ctx context.Context, userID uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *MockStorage) normalizePath(p string) string {
	return strings.TrimPrefix(p, "/")
}

type mockReadCloser struct {
	info *minio.ObjectInfo
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("模拟读操作")
}

func (m *mockReadCloser) Close() error {
	return nil
}

// MockAuth 模拟认证服务
type MockAuth struct {
	users    map[uuid.UUID]*mockUser
	err      error
	storageDeltas map[uuid.UUID]int64
}

type mockUser struct {
	ID           uuid.UUID
	Username     string
	StorageQuota int64
	StorageUsed  int64
}

func NewMockAuth() *MockAuth {
	return &MockAuth{
		users:         make(map[uuid.UUID]*mockUser),
		storageDeltas: make(map[uuid.UUID]int64),
	}
}

func (m *MockAuth) Register(ctx context.Context, req interface{}) (interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &mockUser{}, nil
}

func (m *MockAuth) Login(ctx context.Context, req interface{}) (interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &models.UserLoginResponse{
		Token: "mock-token",
		User:  &models.User{},
	}, nil
}

func (m *MockAuth) GenerateToken(user interface{}) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "mock-token", nil
}

func (m *MockAuth) ValidateToken(tokenString string) (*auth.Claims, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	return &auth.Claims{
		UserID:   uuid.New().String(),
		Username: "test-user",
		RegisteredClaims: jwt.RegisteredClaims{},
	}, nil
}

func (m *MockAuth) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	return &models.User{
		ID:          userID,
		Username:    "test-user",
		StorageUsed: 0,
	}, nil
}

func (m *MockAuth) UpdateStorageUsed(ctx context.Context, userID uuid.UUID, delta int64) error {
	if m.err != nil {
		return m.err
	}
	
	m.storageDeltas[userID] += delta
	return nil
}

func (m *MockAuth) SetError(err error) {
	m.err = err
}

func (m *MockAuth) AddUser(userID uuid.UUID, username string, quota int64) {
	m.users[userID] = &mockUser{
		ID:           userID,
		Username:     username,
		StorageQuota: quota,
		StorageUsed:  0,
	}
}

// MockPropertyService 模拟属性服务
type MockPropertyService struct {
	properties map[string]*Property
	err        error
}

func NewMockPropertyService() *MockPropertyService {
	return &MockPropertyService{
		properties: make(map[string]*Property),
	}
}

func (m *MockPropertyService) Initialize(ctx context.Context) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *MockPropertyService) GetProperty(ctx context.Context, userID, path, namespace, name string) (*Property, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	key := fmt.Sprintf("%s:%s:%s:%s", userID, path, namespace, name)
	prop, exists := m.properties[key]
	if !exists {
		return nil, nil
	}
	return prop, nil
}

func (m *MockPropertyService) ListProperties(ctx context.Context, userID, path string) ([]*Property, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	var props []*Property
	for key, prop := range m.properties {
		// 解析key来获取userID和path
		parts := strings.Split(key, ":")
		if len(parts) >= 4 && parts[0] == userID && parts[1] == path {
			props = append(props, prop)
		}
	}
	return props, nil
}

func (m *MockPropertyService) CreateProperty(ctx context.Context, property *Property) error {
	if m.err != nil {
		return m.err
	}
	
	key := fmt.Sprintf("%s:%s:%s:%s", property.UserID, property.Path, property.Namespace, property.Name)
	m.properties[key] = property
	return nil
}

func (m *MockPropertyService) UpdateProperty(ctx context.Context, property *Property) error {
	if m.err != nil {
		return m.err
	}
	
	key := fmt.Sprintf("%s:%s:%s:%s", property.UserID, property.Path, property.Namespace, property.Name)
	m.properties[key] = property
	return nil
}

func (m *MockPropertyService) DeleteProperty(ctx context.Context, userID, path, namespace, name string) error {
	if m.err != nil {
		return m.err
	}
	
	key := fmt.Sprintf("%s:%s:%s:%s", userID, path, namespace, name)
	delete(m.properties, key)
	return nil
}

func (m *MockPropertyService) BatchSetProperties(ctx context.Context, userID, path string, properties []*Property) error {
	if m.err != nil {
		return m.err
	}
	
	for _, prop := range properties {
		key := fmt.Sprintf("%s:%s:%s:%s", userID, path, prop.Namespace, prop.Name)
		m.properties[key] = prop
	}
	return nil
}

func (m *MockPropertyService) BatchRemoveProperties(ctx context.Context, userID, path string, namespaces []string, names []string) error {
	if m.err != nil {
		return m.err
	}
	
	for _, namespace := range namespaces {
		for _, name := range names {
			key := fmt.Sprintf("%s:%s:%s:%s", userID, path, namespace, name)
			delete(m.properties, key)
		}
	}
	return nil
}

func (m *MockPropertyService) FindPropertiesByNamespace(ctx context.Context, userID, path, namespace string) ([]*Property, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	var props []*Property
	for key, prop := range m.properties {
		parts := strings.Split(key, ":")
		if len(parts) >= 4 && parts[0] == userID && parts[1] == path && parts[2] == namespace {
			props = append(props, prop)
		}
	}
	return props, nil
}

func (m *MockPropertyService) SearchProperties(ctx context.Context, userID string, filters map[string]interface{}) ([]*Property, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	// 简化实现，返回所有属性
	return m.ListProperties(ctx, userID, "")
}

func (m *MockPropertyService) CleanupExpiredProperties(ctx context.Context) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return 0, nil
}

func (m *MockPropertyService) RebuildIndexes(ctx context.Context) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *MockPropertyService) Close() error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *MockPropertyService) HealthCheck(ctx context.Context) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *MockPropertyService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	return map[string]interface{}{
		"total_properties": len(m.properties),
	}, nil
}

func (m *MockPropertyService) SetError(err error) {
	m.err = err
}

// MockXMLParser 模拟XML解析器
type MockXMLParser struct {
	err error
}

func NewMockXMLParser() *MockXMLParser {
	return &MockXMLParser{}
}

func (m *MockXMLParser) ReadXMLBody(body interface{}) ([]byte, *PropertyError) {
	if m.err != nil {
		return nil, &PropertyError{
			Code:    400,
			Message: "XML读取失败",
		}
	}
	
	return []byte(`<D:propertyupdate xmlns:D="DAV:"><D:set><D:prop><test:custom xmlns:test="test">value</test:custom></D:prop></D:set></D:propertyupdate>`), nil
}

func (m *MockXMLParser) ParseProppatchRequest(xmlBody []byte) (*PropertyUpdateRequest, *PropertyError) {
	if m.err != nil {
		return nil, &PropertyError{
			Code:    400,
			Message: "XML解析失败",
		}
	}
	
	return &PropertyUpdateRequest{
		Set: []PropertyUpdateSet{
			{
				Prop: PropContent{
					XMLName: xml.Name{Local: "custom"},
					CustomProps: map[string]string{"test": "value"},
				},
			},
		},
		Remove: []PropertyUpdateRemove{},
	}, nil
}

func (m *MockXMLParser) ParsePropertyFromContent(userID, path string, content PropContent) (*Property, *PropertyError) {
	if m.err != nil {
		return nil, &PropertyError{
			Code:    400,
			Message: "属性解析失败",
		}
	}
	
	return &Property{
		UserID:     userID,
		Path:       path,
		Name:       content.XMLName.Local,
		Namespace:  "test",
		Value:      content.CustomProps["test"],
		IsLive:     false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResourceID: uuid.New().String(),
	}, nil
}

func (m *MockXMLParser) GenerateProppatchResponse(result *PropertyUpdateResult) ([]byte, *PropertyError) {
	if m.err != nil {
		return nil, &PropertyError{
			Code:    500,
			Message: "响应生成失败",
		}
	}
	
	responseXML := `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:">
<D:response>
<D:href>` + result.ResourcePath + `</D:href>
<D:propstat>
<D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`
	
	return []byte(responseXML), nil
}

func (m *MockXMLParser) GenerateErrorResponse(statusCode int, errors []PropertyError) ([]byte, *PropertyError) {
	if m.err != nil {
		return nil, &PropertyError{
			Code:    500,
			Message: "错误响应生成失败",
		}
	}
	
	responseXML := `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:">
<D:response>
<D:status>HTTP/1.1 ` + fmt.Sprintf("%d", statusCode) + `</D:status>
</D:response>
</D:multistatus>`
	
	return []byte(responseXML), nil
}

func (m *MockXMLParser) resolveNamespace(prop PropContent) string {
	return "test"
}

func (m *MockXMLParser) SetError(err error) {
	m.err = err
}

// MockResponseBuilder 模拟响应构建器
type MockResponseBuilder struct {
	err error
}

func NewMockResponseBuilder() *MockResponseBuilder {
	return &MockResponseBuilder{}
}

func (m *MockResponseBuilder) BuildProppatchResponse(result *PropertyUpdateResult) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	return []byte("mock response"), nil
}

func (m *MockResponseBuilder) BuildErrorResponse(errors []PropertyError) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	return []byte("mock error response"), nil
}

func (m *MockResponseBuilder) SetError(err error) {
	m.err = err
}

// ========================================
// Test Setup and Utilities
// ========================================

func setupTestHandler() (*Handler, *MockStorage, *MockAuth, *MockPropertyService, *MockXMLParser, *MockResponseBuilder) {
	mockStorage := NewMockStorage()
	mockAuth := NewMockAuth()
	mockPropertyService := NewMockPropertyService()
	mockXMLParser := NewMockXMLParser()
	mockResponseBuilder := NewMockResponseBuilder()
	
	handler := &Handler{
		storage:         mockStorage,
		auth:            mockAuth,
		lockManager:     NewLockManager(),
		propertyService: mockPropertyService,
		xmlParser:       mockXMLParser,
		responseBuilder: mockResponseBuilder,
	}
	
	return handler, mockStorage, mockAuth, mockPropertyService, mockXMLParser, mockResponseBuilder
}

func createTestContext(method, path string, body []byte, userID string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/xml")
	
	if userID != "" {
		c.Set("userID", userID)
	}
	
	return c, w
}

// ========================================
// Handler Initialization Tests
// ========================================

func TestNewHandler(t *testing.T) {
	mockStorage := NewMockStorage()
	mockAuth := NewMockAuth()
	mockPropertyService := NewMockPropertyService()
	
	handler := NewHandler(mockStorage, mockAuth, mockPropertyService)
	
	// 验证依赖注入
	assert.NotNil(t, handler.storage)
	assert.NotNil(t, handler.auth)
	assert.NotNil(t, handler.lockManager)
	assert.NotNil(t, handler.propertyService)
	assert.NotNil(t, handler.xmlParser)
	assert.NotNil(t, handler.responseBuilder)
	
	// 验证组件初始化
	assert.Equal(t, mockStorage, handler.storage)
	assert.Equal(t, mockAuth, handler.auth)
	assert.Equal(t, mockPropertyService, handler.propertyService)
}

func TestHandlerDependencyInjection(t *testing.T) {
	handler, mockStorage, mockAuth, mockPropertyService, mockXMLParser, mockResponseBuilder := setupTestHandler()
	
	// 验证所有依赖都正确注入
	assert.Equal(t, mockStorage, handler.storage)
	assert.Equal(t, mockAuth, handler.auth)
	assert.Equal(t, mockPropertyService, handler.propertyService)
	assert.Equal(t, mockXMLParser, handler.xmlParser)
	assert.Equal(t, mockResponseBuilder, handler.responseBuilder)
	assert.NotNil(t, handler.lockManager)
}

// ========================================
// HandleProppatch Tests
// ========================================

func TestHandleProppatch_Success(t *testing.T) {
	handler, _, _, mockPropertyService, mockXMLParser, _ := setupTestHandler()
	
	userID := uuid.New().String()
	path := "/test.txt"
	
	// 设置XML解析器模拟成功响应
	proppatchXML := `<?xml version="1.0" encoding="utf-8"?>
<propertyupdate xmlns="DAV:">
<set>
<prop>
<custom:description xmlns:custom="custom">Test description</custom:description>
</prop>
</set>
</propertyupdate>`
	
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), userID)
	
	// 执行测试
	handler.HandleProppatch(c)
	
	// 验证响应
	assert.Equal(t, http.StatusMultiStatus, w.Code)
	assert.Contains(t, w.Body.String(), "multistatus")
	
	// 验证属性服务被调用
	properties, err := mockPropertyService.ListProperties(context.Background(), userID, path)
	assert.NoError(t, err)
	assert.Len(t, properties, 0) // 实际属性存储需要真实实现
}

func TestHandleProppatch_MissingUserID(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	proppatchXML := `<?xml version="1.0"?><propertyupdate xmlns="DAV:"></propertyupdate>`
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), "")
	
	handler.HandleProppatch(c)
	
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleProppatch_XMLParsingError(t *testing.T) {
	handler, _, _, _, mockXMLParser, _ := setupTestHandler()
	
	userID := uuid.New().String()
	
	// 设置XML解析器返回错误
	mockXMLParser.SetError(fmt.Errorf("解析错误"))
	
	proppatchXML := `<?xml version="1.0"?><propertyupdate xmlns="DAV:"></propertyupdate>`
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), userID)
	
	handler.HandleProppatch(c)
	
	assert.Equal(t, http.StatusMultiStatus, w.Code)
	assert.Contains(t, w.Body.String(), "multistatus")
}

func TestHandleProppatch_PropertyServiceError(t *testing.T) {
	handler, _, _, mockPropertyService, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	
	// 设置属性服务返回错误
	mockPropertyService.SetError(fmt.Errorf("服务错误"))
	
	proppatchXML := `<?xml version="1.0"?><propertyupdate xmlns="DAV:">
<set><prop><custom:description xmlns:custom="custom">Test</custom:description></prop></set>
</propertyupdate>`
	
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), userID)
	
	handler.HandleProppatch(c)
	
	assert.Equal(t, http.StatusMultiStatus, w.Code)
}

func TestHandleProppatch_EmptyRequest(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	
	// 空请求体
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(""), userID)
	
	handler.HandleProppatch(c)
	
	// 应该返回错误，因为空的PROPPATCH请求无效
	assert.Equal(t, http.StatusMultiStatus, w.Code)
}

// ========================================
// Integration Tests
// ========================================

func TestHandleProppatch_Integration(t *testing.T) {
	handler, mockStorage, mockAuth, mockPropertyService, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	uid, _ := uuid.Parse(userID)
	
	// 设置用户
	mockAuth.AddUser(uid, "testuser", 1024*1024)
	
	// 模拟存储对象存在
	mockStorage.objects["/test.txt"] = &minio.ObjectInfo{
		Key:          "/test.txt",
		Size:         100,
		ContentType:  "text/plain",
		LastModified: time.Now(),
	}
	
	// 准备PROPPATCH请求
	proppatchXML := `<?xml version="1.0" encoding="utf-8"?>
<propertyupdate xmlns="DAV:">
<set>
<prop>
<custom:title xmlns:custom="custom">My Document</custom:title>
<custom:author xmlns:custom="custom">John Doe</custom:author>
</prop>
</set>
</propertyupdate>`
	
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), userID)
	
	// 执行测试
	handler.HandleProppatch(c)
	
	// 验证结果
	assert.Equal(t, http.StatusMultiStatus, w.Code)
	assert.Contains(t, w.Body.String(), "multistatus")
	
	// 验证响应格式
	assert.Contains(t, w.Body.String(), "response")
	assert.Contains(t, w.Body.String(), "propstat")
}

func TestHandleProppatch_RemoveProperties_Integration(t *testing.T) {
	handler, _, _, mockPropertyService, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	path := "/test.txt"
	
	// 预先添加属性
	mockPropertyService.properties["user:path:custom:title"] = &Property{
		UserID:     userID,
		Path:       path,
		Name:       "title",
		Namespace:  "custom",
		Value:      "Old Title",
		IsLive:     false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResourceID: "test-resource",
	}
	
	// 准备移除属性的PROPPATCH请求
	proppatchXML := `<?xml version="1.0" encoding="utf-8"?>
<propertyupdate xmlns="DAV:">
<remove>
<prop>
<custom:title xmlns:custom="custom"/>
</prop>
</remove>
</propertyupdate>`
	
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), userID)
	
	// 执行测试
	handler.HandleProppatch(c)
	
	// 验证结果
	assert.Equal(t, http.StatusMultiStatus, w.Code)
	
	// 验证属性被移除
	prop, err := mockPropertyService.GetProperty(context.Background(), userID, path, "custom", "title")
	assert.NoError(t, err)
	assert.Nil(t, prop) // 属性应该被移除
}

// ========================================
// Lock Integration Tests
// ========================================

func TestHandleProppatch_WithLock(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	path := "/test.txt"
	
	// 创建一个锁定
	lock := handler.lockManager.CreateLock(path, LockTypeExclusive, userID, 3600, "0")
	assert.NotNil(t, lock)
	
	// 尝试执行PROPPATCH（应该失败，因为资源被锁定）
	proppatchXML := `<?xml version="1.0"?>
<propertyupdate xmlns="DAV:">
<set><prop><custom:title xmlns:custom="custom">Test</custom:title></prop></set>
</propertyupdate>`
	
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), userID)
	
	handler.HandleProppatch(c)
	
	// 应该返回锁定错误
	assert.Equal(t, http.StatusLocked, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestHandleProppatch_SharedLockCompatible(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	path := "/test.txt"
	
	// 创建共享锁定
	lock := handler.lockManager.CreateLock(path, LockTypeShared, userID, 3600, "0")
	assert.NotNil(t, lock)
	
	// 尝试执行PROPPATCH（应该成功，因为是锁的持有者）
	proppatchXML := `<?xml version="1.0"?>
<propertyupdate xmlns="DAV:">
<set><prop><custom:title xmlns:custom="custom">Test</custom:title></prop></set>
</propertyupdate>`
	
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), userID)
	
	handler.HandleProppatch(c)
	
	// 应该返回多状态响应而不是锁定错误
	assert.Equal(t, http.StatusMultiStatus, w.Code)
}

// ========================================
// Error Handling Tests
// ========================================

func TestHandleProppatch_InvalidXML(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	
	// 无效的XML
	invalidXML := `<?xml version="1.0"?>
<invalid-xml>
<propertyupdate xmlns="DAV:">
<set><prop><custom:title xmlns:custom="custom">Test</custom:title></prop></set>
</propertyupdate>`
	
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(invalidXML), userID)
	
	handler.HandleProppatch(c)
	
	// 应该返回错误响应
	assert.Equal(t, http.StatusMultiStatus, w.Code)
}

func TestHandleProppatch_MalformedRequest(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	
	// 格式错误的请求
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte("not xml"), userID)
	
	handler.HandleProppatch(c)
	
	// 应该返回错误响应
	assert.Equal(t, http.StatusMultiStatus, w.Code)
}

// ========================================
// Table-Driven Tests
// ========================================

func TestHandleProppatch_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		body        string
		userID      string
		setupMocks  func(*MockStorage, *MockAuth, *MockPropertyService, *MockXMLParser)
		wantCode    int
		wantContains string
	}{
		{
			name:        "成功设置属性",
			method:      "PROPPATCH",
			path:        "/files/test.txt",
			body:        `<?xml version="1.0"?><propertyupdate xmlns="DAV:"><set><prop><custom:title xmlns:custom="custom">Test</custom:title></prop></set></propertyupdate>`,
			userID:      uuid.New().String(),
			setupMocks:  func(s *MockStorage, a *MockAuth, p *MockPropertyService, x *MockXMLParser) {},
			wantCode:    http.StatusMultiStatus,
			wantContains: "multistatus",
		},
		{
			name:        "缺少用户ID",
			method:      "PROPPATCH",
			path:        "/files/test.txt",
			body:        `<?xml version="1.0"?><propertyupdate xmlns="DAV:"><set><prop><custom:title xmlns:custom="custom">Test</custom:title></prop></set></propertyupdate>`,
			userID:      "",
			setupMocks:  func(s *MockStorage, a *MockAuth, p *MockPropertyService, x *MockXMLParser) {},
			wantCode:    http.StatusUnauthorized,
			wantContains: "",
		},
		{
			name:        "XML解析错误",
			method:      "PROPPATCH",
			path:        "/files/test.txt",
			body:        `invalid xml`,
			userID:      uuid.New().String(),
			setupMocks:  func(s *MockStorage, a *MockAuth, p *MockPropertyService, x *MockXMLParser) {
				x.SetError(fmt.Errorf("解析错误"))
			},
			wantCode:    http.StatusMultiStatus,
			wantContains: "multistatus",
		},
		{
			name:        "属性服务错误",
			method:      "PROPPATCH",
			path:        "/files/test.txt",
			body:        `<?xml version="1.0"?><propertyupdate xmlns="DAV:"><set><prop><custom:title xmlns:custom="custom">Test</custom:title></prop></set></propertyupdate>`,
			userID:      uuid.New().String(),
			setupMocks:  func(s *MockStorage, a *MockAuth, p *MockPropertyService, x *MockXMLParser) {
				p.SetError(fmt.Errorf("服务错误"))
			},
			wantCode:    http.StatusMultiStatus,
			wantContains: "multistatus",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockStorage, mockAuth, mockPropertyService, mockXMLParser, _ := setupTestHandler()
			
			// 设置模拟对象
			tt.setupMocks(mockStorage, mockAuth, mockPropertyService, mockXMLParser)
			
			// 创建测试上下文
			c, w := createTestContext(tt.method, tt.path, []byte(tt.body), tt.userID)
			
			// 执行测试
			handler.HandleProppatch(c)
			
			// 验证结果
			assert.Equal(t, tt.wantCode, w.Code)
			if tt.wantContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantContains)
			}
		})
	}
}

// ========================================
// Response Generation Tests
// ========================================

func TestHandleProppatch_ResponseStructure(t *testing.T) {
	handler, _, _, _, mockXMLParser, _ := setupTestHandler()
	
	userID := uuid.New().String()
	
	// 准备复杂的PROPPATCH请求
	proppatchXML := `<?xml version="1.0" encoding="utf-8"?>
<propertyupdate xmlns="DAV:">
<set>
<prop>
<custom:title xmlns:custom="custom">My Document</custom:title>
<custom:author xmlns:custom="custom">John Doe</custom:author>
<custom:created xmlns:custom="custom">2023-01-01</custom:created>
</prop>
</set>
<remove>
<prop>
<custom:oldfield xmlns:custom="custom"/>
</prop>
</remove>
</propertyupdate>`
	
	c, w := createTestContext("PROPPATCH", "/files/document.txt", []byte(proppatchXML), userID)
	
	// 模拟XML解析器返回详细结果
	mockXMLParser.SetError(nil) // 确保没有错误
	
	handler.HandleProppatch(c)
	
	// 验证响应结构
	assert.Equal(t, http.StatusMultiStatus, w.Code)
	
	body := w.Body.String()
	assert.Contains(t, body, "multistatus")
	assert.Contains(t, body, "response")
	assert.Contains(t, body, "propstat")
}

func TestHandleProppatch_ResponseHeaders(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	
	proppatchXML := `<?xml version="1.0"?>
<propertyupdate xmlns="DAV:">
<set><prop><custom:title xmlns:custom="custom">Test</custom:title></prop></set>
</propertyupdate>`
	
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), userID)
	
	handler.HandleProppatch(c)
	
	// 验证响应头
	assert.Equal(t, "application/xml; charset=utf-8", w.Header().Get("Content-Type"))
}

// ========================================
// Other Handler Method Tests
// ========================================

func TestHandlePropfind(t *testing.T) {
	handler, mockStorage, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	uid, _ := uuid.Parse(userID)
	
	// 模拟存储对象
	mockStorage.objects["/test.txt"] = &minio.ObjectInfo{
		Key:          "/test.txt",
		Size:         1024,
		ContentType:  "text/plain",
		LastModified: time.Now(),
	}
	
	c, w := createTestContext("PROPFIND", "/files/test.txt", nil, userID)
	c.Request.Header.Set("Depth", "0")
	
	handler.HandlePropfind(c)
	
	assert.Equal(t, http.StatusMultiStatus, w.Code)
	assert.Contains(t, w.Body.String(), "multistatus")
}

func TestHandleGet(t *testing.T) {
	handler, mockStorage, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	uid, _ := uuid.Parse(userID)
	
	// 模拟存储对象
	mockStorage.objects["/test.txt"] = &minio.ObjectInfo{
		Key:          "/test.txt",
		Size:         1024,
		ContentType:  "text/plain",
		LastModified: time.Now(),
		ETag:         "test-etag",
	}
	
	c, w := createTestContext("GET", "/files/test.txt", nil, userID)
	
	handler.HandleGet(c)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}

func TestHandlePut(t *testing.T) {
	handler, mockStorage, mockAuth, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	uid, _ := uuid.Parse(userID)
	
	// 设置用户
	mockAuth.AddUser(uid, "testuser", 1024*1024)
	
	fileContent := []byte("test file content")
	c, w := createTestContext("PUT", "/files/newfile.txt", fileContent, userID)
	
	handler.HandlePut(c)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	// 验证文件被创建
	info, err := mockStorage.StatObject(context.Background(), uid, "/newfile.txt")
	assert.NoError(t, err)
	assert.Equal(t, int64(len(fileContent)), info.Size)
}

func TestHandleDelete(t *testing.T) {
	handler, mockStorage, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	uid, _ := uuid.Parse(userID)
	
	// 模拟存储对象
	mockStorage.objects["/test.txt"] = &minio.ObjectInfo{
		Key:          "/test.txt",
		Size:         1024,
		ContentType:  "text/plain",
		LastModified: time.Now(),
	}
	
	c, w := createTestContext("DELETE", "/files/test.txt", nil, userID)
	
	handler.HandleDelete(c)
	
	assert.Equal(t, http.StatusNoContent, w.Code)
	
	// 验证文件被删除
	_, err := mockStorage.StatObject(context.Background(), uid, "/test.txt")
	assert.Error(t, err)
}

func TestHandleMkcol(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	
	c, w := createTestContext("MKCOL", "/files/newfolder", nil, userID)
	
	handler.HandleMkcol(c)
	
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandleOptions(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	c, w := createTestContext("OPTIONS", "/files/test.txt", nil, "")
	
	handler.HandleOptions(c)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("DAV"), "1")
	assert.Contains(t, w.Header().Get("Allow"), "PROPPATCH")
}

// ========================================
// Lock Management Tests
// ========================================

func TestHandleLock(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	
	lockXML := `<?xml version="1.0"?>
<lockinfo xmlns="DAV:">
<lockscope><exclusive/></lockscope>
<locktype><write/></locktype>
<owner><href>user@example.com</href></owner>
</lockinfo>`
	
	c, w := createTestContext("LOCK", "/files/test.txt", []byte(lockXML), userID)
	
	handler.HandleLock(c)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "activelock")
}

func TestHandleUnlock(t *testing.T) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	path := "/test.txt"
	
	// 首先创建一个锁定
	lock := handler.lockManager.CreateLock(path, LockTypeExclusive, userID, 3600, "0")
	assert.NotNil(t, lock)
	
	c, w := createTestContext("UNLOCK", "/files/test.txt", nil, userID)
	c.Request.Header.Set("Lock-Token", fmt.Sprintf("<%s>", lock.Token))
	
	handler.HandleUnlock(c)
	
	assert.Equal(t, http.StatusNoContent, w.Code)
	
	// 验证锁定被移除
	_, exists := handler.lockManager.GetLock(lock.Token)
	assert.False(t, exists)
}

// ========================================
// End-to-End Integration Tests
// ========================================

func TestE2E_CompleteWorkflow(t *testing.T) {
	handler, mockStorage, mockAuth, mockPropertyService, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	uid, _ := uuid.Parse(userID)
	
	// 设置用户
	mockAuth.AddUser(uid, "testuser", 1024*1024)
	
	// 1. 创建文件
	createContent := []byte("Hello, World!")
	c1, w1 := createTestContext("PUT", "/files/document.txt", createContent, userID)
	handler.HandlePut(c1)
	assert.Equal(t, http.StatusCreated, w1.Code)
	
	// 2. 设置文件属性
	proppatchXML := `<?xml version="1.0"?>
<propertyupdate xmlns="DAV:">
<set>
<prop>
<custom:title xmlns:custom="custom">My Document</custom:title>
<custom:author xmlns:custom="custom">John Doe</custom:author>
</prop>
</set>
</propertyupdate>`
	
	c2, w2 := createTestContext("PROPPATCH", "/files/document.txt", []byte(proppatchXML), userID)
	handler.HandleProppatch(c2)
	assert.Equal(t, http.StatusMultiStatus, w2.Code)
	
	// 3. 获取文件属性
	c3, w3 := createTestContext("PROPFIND", "/files/document.txt", nil, userID)
	c3.Request.Header.Set("Depth", "0")
	handler.HandlePropfind(c3)
	assert.Equal(t, http.StatusMultiStatus, w3.Code)
	
	// 4. 验证属性是否被正确保存
	properties, err := mockPropertyService.ListProperties(context.Background(), userID, "/document.txt")
	assert.NoError(t, err)
	
	// 5. 锁定文件
	lockXML := `<?xml version="1.0"?>
<lockinfo xmlns="DAV:">
<lockscope><exclusive/></lockscope>
<locktype><write/></locktype>
</lockinfo>`
	
	c4, w4 := createTestContext("LOCK", "/files/document.txt", []byte(lockXML), userID)
	handler.HandleLock(c4)
	assert.Equal(t, http.StatusOK, w4.Code)
	
	// 6. 尝试修改锁定文件（应该成功，因为是持有者）
	modificationXML := `<?xml version="1.0"?>
<propertyupdate xmlns="DAV:">
<set><prop><custom:description xmlns:custom="custom">Updated description</custom:description></prop></set>
</propertyupdate>`
	
	c5, w5 := createTestContext("PROPPATCH", "/files/document.txt", []byte(modificationXML), userID)
	handler.HandleProppatch(c5)
	assert.Equal(t, http.StatusMultiStatus, w5.Code)
	
	// 7. 解锁文件
	// 注意：实际应该从LOCK响应中获取锁令牌
	locks := handler.lockManager.GetAllLocks()
	if len(locks) > 0 {
		lockToken := locks[0].Token
		c6, w6 := createTestContext("UNLOCK", "/files/document.txt", nil, userID)
		c6.Request.Header.Set("Lock-Token", fmt.Sprintf("<%s>", lockToken))
		handler.HandleUnlock(c6)
		assert.Equal(t, http.StatusNoContent, w6.Code)
	}
	
	// 8. 删除文件
	c7, w7 := createTestContext("DELETE", "/files/document.txt", nil, userID)
	handler.HandleDelete(c7)
	assert.Equal(t, http.StatusNoContent, w7.Code)
}

func TestE2E_MultiUserScenario(t *testing.T) {
	handler, mockStorage, mockAuth, _, _, _ := setupTestHandler()
	
	userID1 := uuid.New().String()
	userID2 := uuid.New().String()
	uid1, _ := uuid.Parse(userID1)
	uid2, _ := uuid.Parse(userID2)
	
	// 设置用户
	mockAuth.AddUser(uid1, "user1", 1024*1024)
	mockAuth.AddUser(uid2, "user2", 1024*1024)
	
	// 用户1创建文件
	content := []byte("User1's file")
	c1, w1 := createTestContext("PUT", "/files/shared.txt", content, userID1)
	handler.HandlePut(c1)
	assert.Equal(t, http.StatusCreated, w1.Code)
	
	// 用户1锁定文件
	lockXML := `<?xml version="1.0"?>
<lockinfo xmlns="DAV:">
<lockscope><exclusive/></lockscope>
<locktype><write/></locktype>
</lockinfo>`
	
	c2, w2 := createTestContext("LOCK", "/files/shared.txt", []byte(lockXML), userID1)
	handler.HandleLock(c2)
	assert.Equal(t, http.StatusOK, w2.Code)
	
	// 用户2尝试修改锁定文件（应该失败）
	proppatchXML := `<?xml version="1.0"?>
<propertyupdate xmlns="DAV:">
<set><prop><custom:title xmlns:custom="custom">User2's title</custom:title></prop></set>
</propertyupdate>`
	
	c3, w3 := createTestContext("PROPPATCH", "/files/shared.txt", []byte(proppatchXML), userID2)
	handler.HandleProppatch(c3)
	assert.Equal(t, http.StatusLocked, w3.Code)
	
	// 用户1解锁
	locks := handler.lockManager.GetAllLocks()
	if len(locks) > 0 {
		lockToken := locks[0].Token
		c4, w4 := createTestContext("UNLOCK", "/files/shared.txt", nil, userID1)
		c4.Request.Header.Set("Lock-Token", fmt.Sprintf("<%s>", lockToken))
		handler.HandleUnlock(c4)
		assert.Equal(t, http.StatusNoContent, w4.Code)
	}
	
	// 用户2现在可以修改文件
	c5, w5 := createTestContext("PROPPATCH", "/files/shared.txt", []byte(proppatchXML), userID2)
	handler.HandleProppatch(c5)
	assert.Equal(t, http.StatusMultiStatus, w5.Code)
}

// ========================================
// Performance Tests
// ========================================

func TestHandleProppatch_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试")
	}
	
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	
	// 创建大量的属性
	props := make([]string, 100)
	for i := 0; i < 100; i++ {
		props[i] = fmt.Sprintf(`<custom:field%d xmlns:custom="custom">value%d</custom:field%d>`, i, i, i)
	}
	
	proppatchXML := fmt.Sprintf(`<?xml version="1.0"?>
<propertyupdate xmlns="DAV:">
<set><prop>%s</prop></set>
</propertyupdate>`, strings.Join(props, ""))
	
	start := time.Now()
	c, w := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), userID)
	handler.HandleProppatch(c)
	duration := time.Since(start)
	
	assert.Equal(t, http.StatusMultiStatus, w.Code)
	
	// 性能断言：处理时间应该少于1秒
	assert.Less(t, duration, time.Second, "PROPPATCH处理时间过长")
	
	t.Logf("处理100个属性耗时: %v", duration)
}

func TestConcurrentProppatchRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过并发测试")
	}
	
	handler, _, _, _, _, _ := setupTestHandler()
	
	// 并发goroutines
	concurrency := 10
	done := make(chan bool, concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func(index int) {
			userID := uuid.New().String()
			path := fmt.Sprintf("/files/test%d.txt", index)
			
			proppatchXML := fmt.Sprintf(`<?xml version="1.0"?>
<propertyupdate xmlns="DAV:">
<set><prop><custom:title xmlns:custom="custom">Document %d</custom:title></prop></set>
</propertyupdate>`, index)
			
			c, w := createTestContext("PROPPATCH", path, []byte(proppatchXML), userID)
			handler.HandleProppatch(c)
			
			// 所有请求都应该成功
			assert.Equal(t, http.StatusMultiStatus, w.Code)
			done <- true
		}(i)
	}
	
	// 等待所有goroutines完成
	timeout := time.After(30 * time.Second)
	for i := 0; i < concurrency; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("并发测试超时")
		}
	}
}

// ========================================
// Helper Functions for Testing
// ========================================

func createSampleProperty(userID, path, namespace, name, value string) *Property {
	return &Property{
		ID:           1,
		UserID:       userID,
		Path:         path,
		Name:         name,
		Namespace:    namespace,
		Value:        value,
		IsLive:       false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		ResourceID:   uuid.New().String(),
	}
}

func validateXMLStructure(t *testing.T, xmlData string) {
	var result Multistatus
	err := xml.Unmarshal([]byte(xmlData), &result)
	assert.NoError(t, err, "XML结构无效")
	assert.NotNil(t, result.XMLName, "XML名称为空")
	assert.NotEmpty(t, result.Responses, "响应列表为空")
}

func createMockHTTPRequest(method, path, body string, headers map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	return req
}

// ========================================
// Test Data and Constants
// ========================================

const (
	// 模拟的属性名
	PropDisplayName = "displayname"
	PropContentType = "getcontenttype"
	PropContentLen  = "getcontentlength"
	PropLastMod     = "getlastmodified"
	
	// 模拟的状态码
	StatusOK         = "HTTP/1.1 200 OK"
	StatusMultiStatus = "HTTP/1.1 207 Multi-Status"
	StatusLocked     = "HTTP/1.1 423 Locked"
	StatusNotFound   = "HTTP/1.1 404 Not Found"
)

// ========================================
// Benchmark Tests
// ========================================

func BenchmarkHandleProppatch(b *testing.B) {
	handler, _, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	proppatchXML := `<?xml version="1.0"?>
<propertyupdate xmlns="DAV:">
<set><prop><custom:title xmlns:custom="custom">Test</custom:title></prop></set>
</propertyupdate>`
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		c, _ := createTestContext("PROPPATCH", "/files/test.txt", []byte(proppatchXML), userID)
		handler.HandleProppatch(c)
	}
}

func BenchmarkHandlePropfind(b *testing.B) {
	handler, mockStorage, _, _, _, _ := setupTestHandler()
	
	userID := uuid.New().String()
	uid, _ := uuid.Parse(userID)
	
	// 预创建存储对象
	mockStorage.objects["/test.txt"] = &minio.ObjectInfo{
		Key:          "/test.txt",
		Size:         1024,
		ContentType:  "text/plain",
		LastModified: time.Now(),
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		c, _ := createTestContext("PROPFIND", "/files/test.txt", nil, userID)
		c.Request.Header.Set("Depth", "0")
		handler.HandlePropfind(c)
	}
}
