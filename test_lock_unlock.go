package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/webdav-gateway/internal/auth"
	"github.com/webdav-gateway/internal/storage"
	"github.com/webdav-gateway/internal/webdav"
)

// MockStorage 模拟存储服务
type MockStorage struct{}

func (m *MockStorage) StatObject(ctx interface{}, uid uuid.UUID, path string) (interface{}, error) {
	return nil, fmt.Errorf("not found")
}

func (m *MockStorage) ListObjects(ctx interface{}, uid uuid.UUID, path string, recursive bool) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (m *MockStorage) GetObject(ctx interface{}, uid uuid.UUID, path string) (interface{}, error) {
	return nil, fmt.Errorf("not found")
}

func (m *MockStorage) PutObject(ctx interface{}, uid uuid.UUID, path string, body interface{}, size int64, contentType string) error {
	return nil
}

func (m *MockStorage) DeleteObject(ctx interface{}, uid uuid.UUID, path string) error {
	return nil
}

func (m *MockStorage) DeleteFolder(ctx interface{}, uid uuid.UUID, path string) error {
	return nil
}

func (m *MockStorage) CreateFolder(ctx interface{}, uid uuid.UUID, path string) error {
	return nil
}

func (m *MockStorage) MoveObject(ctx interface{}, uid uuid.UUID, srcPath, dstPath string) error {
	return nil
}

func (m *MockStorage) CopyObject(ctx interface{}, uid uuid.UUID, srcPath, dstPath string) error {
	return nil
}

// MockAuth 模拟认证服务
type MockAuth struct{}

func (m *MockAuth) UpdateStorageUsed(ctx interface{}, uid uuid.UUID, delta int64) {}

// SetupTestHandler 设置测试用的处理器
func SetupTestHandler() *webdav.Handler {
	mockStorage := &MockStorage{}
	mockAuth := &MockAuth{}
	return webdav.NewHandler(mockStorage, mockAuth)
}

// TestLockRequest 测试LOCK请求
func TestLockRequest(t *testing.T) {
	handler := SetupTestHandler()
	
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)
	
	// 创建测试请求
	lockXML := `<?xml version="1.0" encoding="utf-8"?>
<lockinfo>
  <lockscope>
    <exclusive/>
  </lockscope>
  <locktype>
    <write/>
  </locktype>
  <owner>
    <href>test@example.com</href>
  </owner>
  <timeout>Second-3600</timeout>
  <depth>0</depth>
</lockinfo>`

	req, _ := http.NewRequest("LOCK", "/test/file.txt", strings.NewReader(lockXML))
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Depth", "0")
	
	// 创建Gin上下文
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("userID", "test-user-123")
	
	// 执行LOCK请求
	handler.HandleLock(c)
	
	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	if !strings.Contains(w.Body.String(), "lockdiscovery") {
		t.Errorf("Expected lockdiscovery in response, got: %s", w.Body.String())
	}
	
	if !strings.Contains(w.Header().Get("Lock-Token"), "opaquelocktoken:") {
		t.Errorf("Expected Lock-Token in header, got: %s", w.Header().Get("Lock-Token"))
	}
	
	fmt.Printf("LOCK test passed: %d\n", w.Code)
	fmt.Printf("Response: %s\n", w.Body.String())
}

// TestUnlockRequest 测试UNLOCK请求
func TestUnlockRequest(t *testing.T) {
	handler := SetupTestHandler()
	
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)
	
	// 先创建一个锁
	lockXML := `<?xml version="1.0" encoding="utf-8"?>
<lockinfo>
  <lockscope>
    <exclusive/>
  </lockscope>
  <locktype>
    <write/>
  </locktype>
  <owner>
    <href>test@example.com</href>
  </owner>
  <timeout>Second-3600</timeout>
  <depth>0</depth>
</lockinfo>`

	lockReq, _ := http.NewRequest("LOCK", "/test/file.txt", strings.NewReader(lockXML))
	lockReq.Header.Set("Content-Type", "application/xml")
	lockReq.Header.Set("Depth", "0")
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = lockReq
	c.Set("userID", "test-user-123")
	
	handler.HandleLock(c)
	lockToken := w.Header().Get("Lock-Token")
	
	// 现在执行UNLOCK请求
	unlockReq, _ := http.NewRequest("UNLOCK", "/test/file.txt", nil)
	unlockReq.Header.Set("Lock-Token", lockToken)
	
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = unlockReq
	c2.Set("userID", "test-user-123")
	
	handler.HandleUnlock(c2)
	
	// 验证响应
	if w2.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w2.Code)
	}
	
	fmt.Printf("UNLOCK test passed: %d\n", w2.Code)
}

// TestLockConflict 测试锁定冲突
func TestLockConflict(t *testing.T) {
	handler := SetupTestHandler()
	
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)
	
	// 第一个用户创建锁
	lockXML := `<?xml version="1.0" encoding="utf-8"?>
<lockinfo>
  <lockscope>
    <exclusive/>
  </lockscope>
  <locktype>
    <write/>
  </locktype>
  <owner>
    <href>user1@example.com</href>
  </owner>
  <timeout>Second-3600</timeout>
  <depth>0</depth>
</lockinfo>`

	req1, _ := http.NewRequest("LOCK", "/test/file.txt", strings.NewReader(lockXML))
	req1.Header.Set("Content-Type", "application/xml")
	req1.Header.Set("Depth", "0")
	
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = req1
	c1.Set("userID", "user1")
	
	handler.HandleLock(c1)
	
	// 第二个用户尝试创建锁
	req2, _ := http.NewRequest("LOCK", "/test/file.txt", strings.NewReader(lockXML))
	req2.Header.Set("Content-Type", "application/xml")
	req2.Header.Set("Depth", "0")
	
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = req2
	c2.Set("userID", "user2")
	
	handler.HandleLock(c2)
	
	// 验证响应
	if w2.Code != http.StatusLocked {
		t.Errorf("Expected status %d, got %d", http.StatusLocked, w2.Code)
	}
	
	if !strings.Contains(w2.Body.String(), "Locked") {
		t.Errorf("Expected Locked error in response, got: %s", w2.Body.String())
	}
	
	fmt.Printf("Lock conflict test passed: %d\n", w2.Code)
	fmt.Printf("Response: %s\n", w2.Body.String())
}

func main() {
	fmt.Println("WebDAV LOCK/UNLOCK 功能测试")
	fmt.Println("================================")
	
	// 运行测试
	t := &testing.T{}
	TestLockRequest(t)
	TestUnlockRequest(t)
	TestLockConflict(t)
	
	fmt.Println("\n所有测试完成!")
}