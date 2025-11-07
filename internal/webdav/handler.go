package webdav

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/webdav-gateway/internal/auth"
	"github.com/webdav-gateway/internal/storage"
	webdavtypes "github.com/webdav-gateway/internal/types"
)

type Handler struct {
	storage         *storage.Service
	auth            *auth.Service
	lockManager     *LockManager
	propertyService *PropertyService
	xmlParser       *ProppatchXMLParser
	responseBuilder *ProppatchResponseBuilder
}

func NewHandler(storage *storage.Service, auth *auth.Service, propertyService *PropertyService) *Handler {
	return &Handler{
		storage:         storage,
		auth:            auth,
		lockManager:     NewLockManager(),
		propertyService: propertyService,
		xmlParser:       NewProppatchXMLParser(),
		responseBuilder: NewProppatchResponseBuilder(),
	}
}

type PropfindRequest struct {
	XMLName xml.Name `xml:"propfind"`
	Prop    Prop     `xml:"prop"`
}

type Prop struct {
	DisplayName       *struct{} `xml:"displayname"`
	GetContentLength  *struct{} `xml:"getcontentlength"`
	GetContentType    *struct{} `xml:"getcontenttype"`
	GetLastModified   *struct{} `xml:"getlastmodified"`
	CreationDate      *struct{} `xml:"creationdate"`
	ResourceType      *struct{} `xml:"resourcetype"`
	GetETag           *struct{} `xml:"getetag"`
	SupportedLock     *struct{} `xml:"supportedlock"`
	LockDiscovery     *struct{} `xml:"lockdiscovery"`
}

type Multistatus struct {
	XMLName   xml.Name   `xml:"D:multistatus"`
	Xmlns     string     `xml:"xmlns:D,attr"`
	Responses []Response `xml:"D:response"`
}

// Response WebDAV响应结构
type Response struct {
	Href     string                   `xml:"D:href"`
	Propstat []webdavtypes.Propstat   `xml:"D:propstat"`
}

// handler.go中的简化类型别名，兼容现有代码
type Propstat = webdavtypes.Propstat
type ResponseProp = webdavtypes.ResponseProp
type ResourceType = webdavtypes.ResourceType

// 创建响应时的辅助函数
func createSupportedLock() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"lockscope": map[string]interface{}{
				"exclusive": struct{}{},
			},
			"locktype": map[string]interface{}{
				"write": struct{}{},
			},
		},
		map[string]interface{}{
			"lockscope": map[string]interface{}{
				"shared": struct{}{},
			},
			"locktype": map[string]interface{}{
				"write": struct{}{},
			},
		},
	}
}

func (h *Handler) HandlePropfind(c *gin.Context) {
	userID := c.GetString("userID")
	uid, _ := uuid.Parse(userID)
	
	requestPath := c.Param("path")
	if requestPath == "" {
		requestPath = "/"
	}

	depth := c.GetHeader("Depth")
	if depth == "" {
		depth = "infinity"
	}

	var responses []Response
	userIDString := uid.String()

	if depth == "0" {
		// Only the resource itself
		info, err := h.storage.StatObject(c.Request.Context(), uid, requestPath)
		if err != nil {
			// It might be a folder or root
			responses = append(responses, h.createFolderResponse(requestPath, time.Now(), userIDString))
		} else {
			responses = append(responses, h.createFileResponse(requestPath, info.Size, info.LastModified, info.ContentType, userIDString))
		}
	} else {
		// List directory contents
		objects, err := h.storage.ListObjects(c.Request.Context(), uid, requestPath, depth == "infinity")
		if err != nil {
			// Return root folder
			responses = append(responses, h.createFolderResponse(requestPath, time.Now(), userIDString))
		} else {
			// Add parent folder
			responses = append(responses, h.createFolderResponse(requestPath, time.Now(), userIDString))
			
			// Add files and folders
			for _, obj := range objects {
				objPath := "/" + obj.Key
				if strings.HasSuffix(obj.Key, "/") {
					responses = append(responses, h.createFolderResponse(objPath, obj.LastModified, userIDString))
				} else {
					responses = append(responses, h.createFileResponse(objPath, obj.Size, obj.LastModified, obj.ContentType, userIDString))
				}
			}
		}
	}

	multistatus := Multistatus{
		Xmlns:     "DAV:",
		Responses: responses,
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Status(http.StatusMultiStatus)
	
	c.Writer.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(c.Writer)
	encoder.Indent("", "  ")
	encoder.Encode(multistatus)
}

func (h *Handler) HandleGet(c *gin.Context) {
	userID := c.GetString("userID")
	uid, _ := uuid.Parse(userID)
	
	requestPath := c.Param("path")

	// 检查共享锁定（允许读取）
	if _, lock := h.CheckSharedLock(c, requestPath); lock != nil {
		// 允许SHARED锁定的读取操作
		userID := c.GetString("userID")
		if lock.Type == LockTypeExclusive && lock.Owner != userID {
			return // CheckSharedLock已经发送了423错误
		}
	}

	obj, err := h.storage.GetObject(c.Request.Context(), uid, requestPath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", stat.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))
	c.Header("Last-Modified", stat.LastModified.Format(http.TimeFormat))
	c.Header("ETag", fmt.Sprintf(`"%s"`, stat.ETag))

	c.Status(http.StatusOK)
	io.Copy(c.Writer, obj)
}

func (h *Handler) HandleHead(c *gin.Context) {
	userID := c.GetString("userID")
	uid, _ := uuid.Parse(userID)
	
	requestPath := c.Param("path")

	// 检查共享锁定（允许读取）
	if _, lock := h.CheckSharedLock(c, requestPath); lock != nil {
		// 允许SHARED锁定的读取操作
		userID := c.GetString("userID")
		if lock.Type == LockTypeExclusive && lock.Owner != userID {
			return // CheckSharedLock已经发送了423错误
		}
	}

	info, err := h.storage.StatObject(c.Request.Context(), uid, requestPath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Content-Type", info.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	c.Header("Last-Modified", info.LastModified.Format(http.TimeFormat))
	c.Header("ETag", fmt.Sprintf(`"%s"`, info.ETag))
	c.Status(http.StatusOK)
}

func (h *Handler) HandlePut(c *gin.Context) {
	userID := c.GetString("userID")
	uid, _ := uuid.Parse(userID)
	
	requestPath := c.Param("path")

	// 检查EXCLUSIVE锁定
	if locked, _ := h.CheckExclusiveLock(c, requestPath); locked {
		return // CheckExclusiveLock已经发送了423错误
	}

	// 检查父目录锁定
	if locked, _ := h.CheckParentLocks(c, requestPath); locked {
		return // CheckParentLocks已经发送了423错误
	}

	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	err := h.storage.PutObject(c.Request.Context(), uid, requestPath, c.Request.Body, c.Request.ContentLength, contentType)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Update user storage
	h.auth.UpdateStorageUsed(c.Request.Context(), uid, c.Request.ContentLength)

	c.Status(http.StatusCreated)
}

func (h *Handler) HandleDelete(c *gin.Context) {
	userID := c.GetString("userID")
	uid, _ := uuid.Parse(userID)
	
	requestPath := c.Param("path")

	// 检查任何类型的锁定
	if locked, _ := h.CheckAnyLock(c, requestPath); locked {
		return // CheckAnyLock已经发送了423错误
	}

	// 检查父目录锁定
	if locked, _ := h.CheckParentLocks(c, requestPath); locked {
		return // CheckParentLocks已经发送了423错误
	}

	// Get size before deletion
	info, err := h.storage.StatObject(c.Request.Context(), uid, requestPath)
	if err == nil {
		// It's a file
		if err := h.storage.DeleteObject(c.Request.Context(), uid, requestPath); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		// Update storage
		h.auth.UpdateStorageUsed(c.Request.Context(), uid, -info.Size)
	} else {
		// Try as folder
		if err := h.storage.DeleteFolder(c.Request.Context(), uid, requestPath); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) HandleMkcol(c *gin.Context) {
	userID := c.GetString("userID")
	uid, _ := uuid.Parse(userID)
	
	requestPath := c.Param("path")

	// 检查父目录锁定
	if locked, _ := h.CheckParentLocks(c, requestPath); locked {
		return // CheckParentLocks已经发送了423错误
	}

	err := h.storage.CreateFolder(c.Request.Context(), uid, requestPath)
	if err != nil {
		c.Status(http.StatusConflict)
		return
	}

	c.Status(http.StatusCreated)
}

func (h *Handler) HandleMove(c *gin.Context) {
	userID := c.GetString("userID")
	uid, _ := uuid.Parse(userID)
	
	srcPath := c.Param("path")
	dstPath := c.GetHeader("Destination")
	
	if dstPath == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	// Extract path from destination URL
	dstPath = strings.TrimPrefix(dstPath, "http://")
	dstPath = strings.TrimPrefix(dstPath, "https://")
	if idx := strings.Index(dstPath, "/"); idx >= 0 {
		dstPath = dstPath[idx:]
	}

	// 检查源资源锁定
	if locked, _ := h.CheckAnyLock(c, srcPath); locked {
		return // CheckAnyLock已经发送了423错误
	}

	// 检查目标资源锁定
	if locked, _ := h.CheckExclusiveLock(c, dstPath); locked {
		return // CheckExclusiveLock已经发送了423错误
	}

	overwrite := c.GetHeader("Overwrite")
	if overwrite != "T" {
		// Check if destination exists
		_, err := h.storage.StatObject(c.Request.Context(), uid, dstPath)
		if err == nil {
			c.Status(http.StatusPreconditionFailed)
			return
		}
	}

	err := h.storage.MoveObject(c.Request.Context(), uid, srcPath, dstPath)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusCreated)
}

func (h *Handler) HandleCopy(c *gin.Context) {
	userID := c.GetString("userID")
	uid, _ := uuid.Parse(userID)
	
	srcPath := c.Param("path")
	dstPath := c.GetHeader("Destination")
	
	if dstPath == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	dstPath = strings.TrimPrefix(dstPath, "http://")
	dstPath = strings.TrimPrefix(dstPath, "https://")
	if idx := strings.Index(dstPath, "/"); idx >= 0 {
		dstPath = dstPath[idx:]
	}

	// 检查源资源锁定（允许SHARED锁定的读取）
	if locked, lock := h.CheckSharedLock(c, srcPath); locked && lock != nil {
		if lock.Type == LockTypeExclusive && lock.Owner != userID {
			return // CheckSharedLock已经发送了423错误
		}
	}

	// 检查目标资源锁定
	if locked, _ := h.CheckExclusiveLock(c, dstPath); locked {
		return // CheckExclusiveLock已经发送了423错误
	}

	err := h.storage.CopyObject(c.Request.Context(), uid, srcPath, dstPath)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusCreated)
}

func (h *Handler) HandleOptions(c *gin.Context) {
	c.Header("DAV", "1, 2")
	c.Header("MS-Author-Via", "DAV")
	c.Header("Allow", "OPTIONS, GET, HEAD, POST, PUT, DELETE, PROPFIND, PROPPATCH, MKCOL, COPY, MOVE, LOCK, UNLOCK")
	c.Status(http.StatusOK)
}

func (h *Handler) createFileResponse(href string, size int64, modTime time.Time, contentType string, userID string) Response {
	// 获取自定义属性
	customProperties, _ := h.GetCustomPropertiesForUser(userID, href)
	
	return Response{
		Href: href,
		Propstat: []webdavtypes.Propstat{{
			Prop: webdavtypes.ResponseProp{
				DisplayName:       path.Base(href),
				GetContentLength:  size,
				GetContentType:    contentType,
				GetLastModified:   modTime.Format(http.TimeFormat),
				CreationDate:      modTime.Format(time.RFC3339),
				ResourceType:      &webdavtypes.ResourceType{},
				GetETag:           fmt.Sprintf(`"%d-%d"`, modTime.Unix(), size),
				SupportedLock:     createSupportedLock(),
				LockDiscovery:     nil, // 临时设为nil避免类型错误
				CustomProperties:  customProperties,
			},
			Status: "HTTP/1.1 200 OK",
		}},
	}
}

func (h *Handler) createFolderResponse(href string, modTime time.Time, userID string) Response {
	if !strings.HasSuffix(href, "/") {
		href += "/"
	}
	
	// 获取自定义属性
	customProperties, _ := h.GetCustomPropertiesForUser(userID, href)
	
	return Response{
		Href: href,
		Propstat: []webdavtypes.Propstat{{
			Prop: webdavtypes.ResponseProp{
				DisplayName:       path.Base(strings.TrimSuffix(href, "/")),
				GetLastModified:   modTime.Format(http.TimeFormat),
				CreationDate:      modTime.Format(time.RFC3339),
				ResourceType: &webdavtypes.ResourceType{
					Collection: &struct{}{},
				},
				SupportedLock:     createSupportedLock(),
				LockDiscovery:     nil, // 临时设为nil避免类型错误
				CustomProperties:  customProperties,
			},
			Status: "HTTP/1.1 200 OK",
		}},
	}
}
// LockedError 423 Locked错误响应
type LockedError struct {
	XMLName    xml.Name `xml:"D:error"`
	XMLNS      string   `xml:"xmlns:D,attr"`
	LockToken  string   `xml:"D:locktoken"`
	Owner      string   `xml:"D:owner"`
	Message    string   `xml:"D:message"`
}

// SendLockedError 发送423 Locked错误响应
func (h *Handler) SendLockedError(c *gin.Context, lockToken, owner, message string) {
	errorResponse := LockedError{
		XMLNS:     "DAV:",
		LockToken: lockToken,
		Owner:     owner,
		Message:   message,
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Retry-After", "60")
	c.Status(http.StatusLocked)

	// 添加锁定令牌引用
	if lockToken != "" {
		c.Header("Lock-Token", fmt.Sprintf("<%s>", lockToken))
	}

	encoder := xml.NewEncoder(c.Writer)
	encoder.Indent("", "  ")
	encoder.Encode(errorResponse)
}

// CheckExclusiveLock 检查EXCLUSIVE锁定
func (h *Handler) CheckExclusiveLock(c *gin.Context, path string) (bool, *Lock) {
	userID := c.GetString("userID")
	
	locked, lock, err := h.lockManager.CheckExclusiveLock(path, userID)
	if err != nil {
		// 确保lock不为nil，避免空指针panic
		if lock == nil {
			lock = &Lock{
				Token: "unknown",
				Owner: "unknown",
				Type:  LockTypeExclusive,
			}
		}
		h.SendLockedError(c, lock.Token, lock.Owner, err.Error())
		return true, lock
	}
	
	return locked, lock
}

// CheckSharedLock 检查共享锁定
func (h *Handler) CheckSharedLock(c *gin.Context, path string) (bool, *Lock) {
	userID := c.GetString("userID")
	
	locked, lock, err := h.lockManager.CheckLock(path, userID)
	if err != nil {
		// 确保lock不为nil
		if lock == nil {
			lock = &Lock{
				Token: "unknown",
				Owner: "unknown",
				Type:  LockTypeExclusive,
			}
		}
		// 如果是EXCLUSIVE锁定且不是持有者，返回423
		if lock.Type == LockTypeExclusive && lock.Owner != userID {
			h.SendLockedError(c, lock.Token, lock.Owner, err.Error())
			return true, lock
		}
	}
	
	return locked, lock
}

// CheckParentLocks 检查父目录锁定
func (h *Handler) CheckParentLocks(c *gin.Context, path string) (bool, *Lock) {
	userID := c.GetString("userID")
	
	locked, lock, err := h.lockManager.CheckParentLocks(path, userID)
	if err != nil {
		// 确保lock不为nil
		if lock == nil {
			lock = &Lock{
				Token: "unknown",
				Owner: "unknown",
				Type:  LockTypeExclusive,
			}
		}
		h.SendLockedError(c, lock.Token, lock.Owner, err.Error())
		return true, lock
	}
	
	return locked, lock
}

// CheckAnyLock 检查任何类型的锁定
func (h *Handler) CheckAnyLock(c *gin.Context, path string) (bool, *Lock) {
	userID := c.GetString("userID")
	
	locks := h.lockManager.GetLocksForPath(path)
	for _, lock := range locks {
		// 检查锁定是否过期
		if time.Now().Sub(lock.CreatedAt).Seconds() > float64(lock.Timeout) {
			h.lockManager.RemoveLock(lock.Token)
			continue
		}
		
		// 如果有EXCLUSIVE锁定且不是持有者，返回423
		if lock.Type == LockTypeExclusive && lock.Owner != userID {
			h.SendLockedError(c, lock.Token, lock.Owner, "Resource is locked exclusively")
			return true, lock
		}
		
		// 如果有SHARED锁定，返回但不阻止操作
		if lock.Type == LockTypeShared {
			return true, lock
		}
	}
	
	return false, nil
}// HandleLock 处理LOCK请求
func (h *Handler) HandleLock(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.Status(http.StatusUnauthorized)
		return
	}

	requestPath := c.Param("path")
	if requestPath == "" {
		requestPath = "/"
	}

	// 获取完整的请求URL（用于lockroot）
	requestURL := h.buildRequestURL(c, requestPath)

	// 检查是否为刷新操作（通过If头检测）
	ifHeader := c.GetHeader("If")
	if ifHeader != "" {
		// 这是锁定刷新请求
		h.handleLockRefresh(c, requestPath, ifHeader, requestURL)
		return
	}

	// 解析LOCK请求体
	var lockInfo *webdavtypes.LockInfoRequest
	var err error

	if c.Request.Body != nil && c.Request.ContentLength > 0 {
		body, readErr := io.ReadAll(c.Request.Body)
		if readErr != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		lockInfo, err = ParseLockInfoFromBytes(body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
	} else {
		// 无请求体，使用默认值
		lockInfo = &webdavtypes.LockInfoRequest{
			LockScope: webdavtypes.LockScopeInfo{
				Exclusive: &struct{}{},
			},
			LockType: webdavtypes.LockTypeInfo{
				Write: &struct{}{},
			},
		}
	}

	// 确定锁定类型
	var lockType LockType
	if lockInfo.LockScope.Exclusive != nil {
		lockType = LockTypeExclusive
	} else if lockInfo.LockScope.Shared != nil {
		lockType = LockTypeShared
	} else {
		// 默认使用exclusive锁
		lockType = LockTypeExclusive
	}

	// 解析超时时间
	timeoutHeader := c.GetHeader("Timeout")
	timeout := ParseTimeout(timeoutHeader)

	// 解析深度
	depthHeader := c.GetHeader("Depth")
	depth := ParseDepth(depthHeader)

	// 提取所有者信息
	owner := userID
	if lockInfo.Owner != nil {
		if lockInfo.Owner.Href != "" {
			owner = lockInfo.Owner.Href
		} else {
			owner = userID // 使用默认值
		}
	}

	// 检查锁定冲突
	if conflict, existingLock, err := h.lockManager.CheckLockConflict(requestPath, lockType, userID, depth); conflict {
		// 返回423 Locked错误
		h.sendConflictError(c, existingLock, err)
		return
	}

	// 创建锁定
	lock := h.lockManager.CreateLock(requestPath, lockType, owner, timeout, depth)

	// 生成响应
	h.sendLockResponse(c, lock, requestURL)
}

// handleLockRefresh 处理锁定刷新请求
func (h *Handler) handleLockRefresh(c *gin.Context, requestPath string, ifHeader string, requestURL string) {
	userID := c.GetString("userID")

	// 解析If头获取锁令牌
	parsed, err := ParseIfHeader(ifHeader)
	if err != nil || parsed == nil || len(parsed.Lists) == 0 {
		c.Status(http.StatusBadRequest)
		return
	}

	// 获取第一个令牌
	var token string
	for _, list := range parsed.Lists {
		for _, condition := range list.Conditions {
			if condition.Token != "" {
				token = condition.Token
				break
			}
		}
		if token != "" {
			break
		}
	}

	if token == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	// 验证锁定是否存在
	lock, exists := h.lockManager.GetLock(token)
	if !exists {
		c.Status(http.StatusPreconditionFailed)
		return
	}

	// 验证路径匹配
	if lock.Path != requestPath {
		c.Status(http.StatusConflict)
		return
	}

	// 验证所有者
	if lock.Owner != userID {
		c.Status(http.StatusForbidden)
		return
	}

	// 解析新的超时时间
	timeoutHeader := c.GetHeader("Timeout")
	timeout := ParseTimeout(timeoutHeader)

	// 刷新锁定
	refreshedLock, err := h.lockManager.RefreshLock(token, timeout)
	if err != nil {
		c.Status(http.StatusConflict)
		return
	}

	// 返回刷新后的锁定信息
	h.sendLockResponse(c, refreshedLock, requestURL)
}

// HandleUnlock 处理UNLOCK请求
func (h *Handler) HandleUnlock(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.Status(http.StatusUnauthorized)
		return
	}

	requestPath := c.Param("path")
	if requestPath == "" {
		requestPath = "/"
	}

	// 获取Lock-Token头
	lockToken := c.GetHeader("Lock-Token")
	if lockToken == "" {
		// 返回400 Bad Request
		c.Status(http.StatusBadRequest)
		return
	}

	// 清理token格式（移除< >）
	lockToken = strings.TrimPrefix(lockToken, "<")
	lockToken = strings.TrimSuffix(lockToken, ">")

	// 验证锁定是否存在
	lock, exists := h.lockManager.GetLock(lockToken)
	if !exists {
		// 锁不存在或已过期，返回409 Conflict
		// 根据RFC 4918，也可以返回423
		c.Status(http.StatusConflict)
		return
	}

	// 验证路径匹配
	if lock.Path != requestPath {
		// 锁令牌范围不匹配Request-URI
		// 返回409错误并包含lock-token-matches-request-uri前置条件
		errorCondition := CreateLockTokenMismatchError()
		h.sendXMLError(c, http.StatusConflict, errorCondition)
		return
	}

	// 验证所有者（只有锁的持有者可以释放）
	if lock.Owner != userID {
		// 返回403 Forbidden
		c.Status(http.StatusForbidden)
		return
	}

	// 移除锁定
	if !h.lockManager.RemoveLock(lockToken) {
		// 移除失败（理论上不应该发生）
		c.Status(http.StatusConflict)
		return
	}

	// 成功返回204 No Content
	c.Status(http.StatusNoContent)
}

// sendLockResponse 发送LOCK响应
func (h *Handler) sendLockResponse(c *gin.Context, lock *Lock, requestURL string) {
	// 创建活动锁定信息
	activeLock := CreateActiveLockResponse(lock, requestURL)

	// 创建响应
	response := PropResponse{
		Namespace:     "DAV:",
		LockDiscovery: []ActiveLock{activeLock},
	}

	// 设置响应头
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Lock-Token", fmt.Sprintf("<%s>", lock.Token))
	c.Status(http.StatusOK)

	// 发送XML响应
	c.Writer.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(c.Writer)
	encoder.Indent("", "  ")
	encoder.Encode(response)
}

// sendConflictError 发送锁定冲突错误
func (h *Handler) sendConflictError(c *gin.Context, lock *Lock, err error) {
	var errorCondition webdavtypes.LockErrorCondition

	if lock != nil {
		// 创建no-conflicting-lock错误
		errorCondition = CreateNoConflictingLockError([]string{lock.Path})
	} else {
		// 创建通用冲突错误
		errorCondition = CreateNoConflictingLockError([]string{})
	}

	_ = errorCondition // 使用变量避免编译器警告

	h.sendXMLError(c, http.StatusLocked, errorCondition)
}

// sendXMLError 发送XML格式的错误响应
func (h *Handler) sendXMLError(c *gin.Context, statusCode int, errorCondition webdavtypes.LockErrorCondition) {
	errorResponse := ErrorResponse{
		Namespace: "DAV:",
		Condition: errorCondition,
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Status(statusCode)

	c.Writer.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(c.Writer)
	encoder.Indent("", "  ")
	encoder.Encode(errorResponse)
}

// buildRequestURL 构建完整的请求URL
func (h *Handler) buildRequestURL(c *gin.Context, path string) string {
	// 获取请求的scheme
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}

	// 检查X-Forwarded-Proto头
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	// 构建完整URL
	host := c.Request.Host
	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}

// CheckAndEnforceLocks 检查并强制执行锁定规则
func (h *Handler) CheckAndEnforceLocks(c *gin.Context, path string) bool {
	userID := c.GetString("userID")
	if userID == "" {
		return false
	}

	// 检查路径锁定
	if locked, existingLock, err := h.lockManager.CheckLock(path, userID); err != nil {
		c.Status(http.StatusLocked)
		h.SendLockedError(c, existingLock.Token, existingLock.Owner, err.Error())
		return true
	} else if locked {
		c.Status(http.StatusLocked)
		h.SendLockedError(c, existingLock.Token, existingLock.Owner, "Resource is locked")
		return true
	}

	// 检查父目录锁定
	if locked, existingLock, err := h.lockManager.CheckParentLocks(path, userID); err != nil {
		c.Status(http.StatusLocked)
		h.SendLockedError(c, existingLock.Token, existingLock.Owner, err.Error())
		return true
	} else if locked {
		c.Status(http.StatusLocked)
		h.SendLockedError(c, existingLock.Token, existingLock.Owner, "Parent resource is locked")
		return true
	}

	return false
}

// ========================================
// PROPPATCH 方法实现
// ========================================

// HandleProppatch 处理PROPPATCH请求
func (h *Handler) HandleProppatch(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.Status(http.StatusUnauthorized)
		return
	}
	uid, _ := uuid.Parse(userID)
	
	requestPath := c.Param("path")
	if requestPath == "" {
		requestPath = "/"
	}

	// 检查资源锁定状态
	// 使用优化的锁定检查
	if locked, lock, err := h.OptimizedProppatchLockCheck(c, requestPath, userID); err != nil {
		// 处理锁定错误
		h.SendProppatchLockedError(c, requestPath, lock.Token, lock.Owner, err.Error())
		return
	} else if locked {
		// 资源被锁定
		return
	}

	// 验证锁定所有权（如果有If头）
	if err := h.ValidateProppatchLockOwnership(c, requestPath, userID); err != nil {
		c.Status(http.StatusPreconditionFailed)
		h.sendProppatchErrorResponse(c, requestPath, []webdavtypes.PropertyError{{
			Code:        412,
			Message:     "锁定验证失败: " + err.Error(),
		}})
		return
	}

	// 初始化属性存储服务
	if err := h.propertyService.Initialize(c.Request.Context()); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// 读取和解析XML请求体
	xmlBody, propError := h.xmlParser.ReadXMLBody(c.Request.Body)
	if propError != nil {
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.Status(propError.Code)
		h.sendProppatchErrorResponse(c, requestPath, []webdavtypes.PropertyError{*propError})
		return
	}

	// 解析PROPPATCH请求
	propRequest, propError := h.xmlParser.ParseProppatchRequest(xmlBody)
	if propError != nil {
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.Status(propError.Code)
		h.sendProppatchErrorResponse(c, requestPath, []webdavtypes.PropertyError{*propError})
		return
	}

	// 处理属性操作
	result, propErrors := h.processProppatchOperations(c, uid, requestPath, propRequest)
	
	// 生成响应
	if len(propErrors) > 0 {
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.Status(http.StatusMultiStatus)
		h.sendProppatchErrorResponse(c, requestPath, propErrors)
	} else {
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.Status(http.StatusMultiStatus)
		h.sendProppatchSuccessResponse(c, result)
	}
}

// processProppatchOperations 处理PROPPATCH操作
func (h *Handler) processProppatchOperations(ctx context.Context, uid uuid.UUID, requestPath string, propRequest *PropertyUpdateRequest) (*PropertyUpdateResult, []webdavtypes.PropertyError) {
	result := &PropertyUpdateResult{
		ResourcePath: requestPath,
		Propstats:    make([]Propstat, 0),
		Operations:   make([]PropertyOperation, 0),
	}
	
	var propErrors []webdavtypes.PropertyError
	var propertiesToSet []*Property
	userID := uid.String()
	
	// 预检查所有操作的锁定兼容性
	preCheckErrors := h.performPreOperationLockCheck(requestPath, propRequest.SetOperations, propRequest.RemoveOperations, userID)
	if len(preCheckErrors) > 0 {
		return result, preCheckErrors
	}
	
	// 处理set操作
	for _, setOp := range propRequest.SetOperations {
		prop, propError := h.processSetOperation(setOp.PropContent[0], ctx, uid.String(), requestPath)
		if propError != nil {
			propErrors = append(propErrors, *propError)
		} else if prop != nil {
			propertiesToSet = append(propertiesToSet, prop)
			
			// 添加到操作记录
			operation := webdavtypes.PropertyOperation{
				Property:  *prop,
				Operation: "set",
				Success:   true,
				Timestamp: time.Now(),
			}
			result.Operations = append(result.Operations, operation)
		}
	}
	
	// 处理remove操作
	for _, removeOp := range propRequest.RemoveOperations {
		propError := h.processRemoveOperation(removeOp.PropContent[0], ctx, uid.String(), requestPath)
		if propError != nil {
			propErrors = append(propErrors, *propError)
		} else {
			// 添加到操作记录
			operation := webdavtypes.PropertyOperation{
				Operation: "remove",
				Property:  webdavtypes.Property{
					Name: removeOp.PropContent[0].XMLName.Local,
				},
				Success:   true,
				Timestamp: time.Now(),
			}
			result.Operations = append(result.Operations, operation)
		}
	}
	
	// 如果有要设置的属性，批量创建/更新
	if len(propertiesToSet) > 0 && len(propErrors) == 0 {
		if err := h.propertyService.BatchSetProperties(ctx, uid.String(), requestPath, propertiesToSet); err != nil {
			propErrors = append(propErrors, webdavtypes.PropertyError{
				Code:    500,
				Message: "批量设置属性失败",
			})
		} else {
			result.SuccessCount = len(propertiesToSet)
			// 为成功操作的属性创建propstat响应
			for _, prop := range propertiesToSet {
				propContent := webdavtypes.PropContentResponse{
					DisplayName: prop.Name,
					CustomProps: map[string]string{
						fmt.Sprintf("%s:%s", prop.Namespace, prop.Name): prop.Value,
					},
				}
				result.Propstats = append(result.Propstats, webdavtypes.Propstat{
					Prop:   webdavtypes.ResponseProp{
						DisplayName: propContent.DisplayName,
						CustomProperties: propContent.CustomProps,
					},
					Status: "HTTP/1.1 200 OK",
				})
			}
		}
	}
	
	result.ErrorCount = len(propErrors)
	return result, propErrors
}

// performPreOperationLockCheck 执行操作前的锁定检查
func (h *Handler) performPreOperationLockCheck(requestPath string, setOps []webdavtypes.SetOperation, removeOps []webdavtypes.RemoveOperation, userID string) []webdavtypes.PropertyError {
	var errors []webdavtypes.PropertyError
	
	// 检查set操作
	for i, setOp := range setOps {
		_ = setOp  // 标记为已使用
		if lock, err := h.CheckProppatchLockCompatibility(requestPath, "set", userID); err != nil {
			errors = append(errors, webdavtypes.PropertyError{
				Code:        423,
				Message:     fmt.Sprintf("Set操作 %d 被锁定", i+1),
				Property:    setOp.PropContent[0].XMLName.Local,
			})
			if lock != nil {
				// 添加锁定令牌信息
				errors[len(errors)-1].PropertyObj = webdavtypes.Property{
					Name:  fmt.Sprintf("锁定令牌: %s, 持有者: %s", lock.Token, lock.Owner),
					Value: err.Error(),
				}
			}
		}
	}
	
	// 检查remove操作
	for i, removeOp := range removeOps {
		_ = removeOp  // 标记为已使用
		if lock, err := h.CheckProppatchLockCompatibility(requestPath, "remove", userID); err != nil {
			errors = append(errors, webdavtypes.PropertyError{
				Code:        423,
				Message:     fmt.Sprintf("Remove操作 %d 被锁定", i+1),
				Property:    removeOp.PropContent[0].XMLName.Local,
			})
			if lock != nil {
				// 添加锁定令牌信息
				errors[len(errors)-1].PropertyObj = webdavtypes.Property{
					Name:  fmt.Sprintf("锁定令牌: %s, 持有者: %s", lock.Token, lock.Owner),
					Value: err.Error(),
				}
			}
		}
	}
	
	return errors
}

// processSetOperation 处理单个set操作
func (h *Handler) processSetOperation(prop webdavtypes.PropContent, ctx context.Context, userID, path string) (*webdavtypes.Property, *webdavtypes.PropertyError) {
	// 解析属性
	property, propError := h.xmlParser.ParsePropertyFromContent(userID, path, prop)
	if propError != nil {
		return nil, propError
	}
	
	// 检查权限（这里可以实现更复杂的权限检查）
	if !h.canModifyProperty(userID, property) {
		return nil, &webdavtypes.PropertyError{
			Code:    403,
			Message: "没有权限修改此属性",
		}
	}
	
	return property, nil
}

// processRemoveOperation 处理单个remove操作
func (h *Handler) processRemoveOperation(prop webdavtypes.PropContent, ctx context.Context, userID, path string) *webdavtypes.PropertyError {
	namespace := h.xmlParser.resolveNamespace(prop)
	propertyName := prop.XMLName.Local
	
	// 检查权限
	if !h.canRemoveProperty(userID, namespace, propertyName) {
		return &webdavtypes.PropertyError{
			Code:    403,
			Message: "没有权限移除此属性",
		}
	}
	
	// 检查属性是否存在
	existingProp, err := h.propertyService.GetProperty(ctx, userID, path, namespace, propertyName)
	if err != nil {
		return &webdavtypes.PropertyError{
			Code:    500,
			Message: "检查属性存在性失败",
		}
	}
	
	if existingProp == nil {
		return &webdavtypes.PropertyError{
			Code:    404,
			Message: "属性不存在",
		}
	}
	
	// 检查是否为活属性（活属性通常不能被删除）
	if existingProp.IsLive {
		return &webdavtypes.PropertyError{
			Code:    403,
			Message: "活属性不能被删除",
		}
	}
	
	// 删除属性
	if err := h.propertyService.DeleteProperty(ctx, userID, path, namespace, propertyName); err != nil {
		return &webdavtypes.PropertyError{
			Code:    500,
			Message: "删除属性失败",
		}
	}
	
	return nil
}

// canModifyProperty 检查是否可以修改属性
func (h *Handler) canModifyProperty(userID string, property *webdavtypes.Property) bool {
	// 基本权限检查：用户可以修改自己的属性
	// 这里可以实现更复杂的权限逻辑
	return true
}

// canRemoveProperty 检查是否可以删除属性
func (h *Handler) canRemoveProperty(userID, namespace, propertyName string) bool {
	// 基本权限检查：用户可以删除自己的属性
	// 特殊命名空间的属性可能有特殊规则
	if namespace == NamespaceDAV {
		// DAV命名空间的活属性通常不能被删除
		if propertyName == "displayname" || propertyName == "resourcetype" {
			return false
		}
	}
	return true
}

// sendProppatchSuccessResponse 发送成功的PROPPATCH响应
func (h *Handler) sendProppatchSuccessResponse(c *gin.Context, result *PropertyUpdateResult) {
	responseXML, propError := h.xmlParser.GenerateProppatchResponse(result)
	if propError != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	
	c.Writer.Write(responseXML)
}

// sendProppatchErrorResponse 发送错误的PROPPATCH响应
func (h *Handler) sendProppatchErrorResponse(c *gin.Context, path string, errors []webdavtypes.PropertyError) {
	responseXML, propError := h.xmlParser.GenerateErrorResponse(http.StatusMultiStatus, errors)
	if propError != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	
	// 在响应中设置正确的路径
	responseStr := string(responseXML)
	// 简单的字符串替换，实际应该用XML处理
	responseStr = strings.Replace(responseStr, `href=""`, `href="`+path+`"`, 1)
	
	c.Writer.Write([]byte(responseStr))
}

// ========================================
// 扩展PROPFIND支持自定义属性
// ========================================

// HandleProppatchList 获取资源的自定义属性（用于PROPFIND）
func (h *Handler) HandleProppatchList(ctx context.Context, userID string, path string) ([]*Property, error) {
	if err := h.propertyService.Initialize(ctx); err != nil {
		return nil, err
	}
	
	return h.propertyService.ListProperties(ctx, userID, path)
}

// EnhancedProppatchResponse 增强的PROPFIND响应（包含自定义属性）
type EnhancedProppatchResponse struct {
	ResponseProp `xml:",inline"`
	CustomProperties map[string]string `xml:"-"`
}

// AddCustomProperty 添加自定义属性到响应中
func (e *EnhancedProppatchResponse) AddCustomProperty(namespace, name, value string) {
	key := fmt.Sprintf("%s:%s", namespace, name)
	if e.CustomProperties == nil {
		e.CustomProperties = make(map[string]string)
	}
	e.CustomProperties[key] = value
}

// getCustomProperties 获取路径的自定义属性（已废弃，请使用GetCustomPropertiesForUser）
func (h *Handler) getCustomProperties(path string) map[string]string {
	return nil // 返回nil避免在旧代码中出现问题
}

// GetCustomPropertiesForUser 获取指定用户的自定义属性
func (h *Handler) GetCustomPropertiesForUser(userID, path string) (map[string]string, error) {
	ctx := context.Background()
	
	// 初始化属性服务
	if err := h.propertyService.Initialize(ctx); err != nil {
		return nil, err
	}
	
	// 查询属性
	properties, err := h.propertyService.ListProperties(ctx, userID, path)
	if err != nil {
		return nil, err
	}
	
	// 转换为map
	customProps := make(map[string]string)
	for _, prop := range properties {
		key := fmt.Sprintf("%s:%s", prop.Namespace, prop.Name)
		customProps[key] = prop.Value
	}
	
	return customProps, nil
}

// ========================================
// PROPPATCH 锁定检查增强
// ========================================

// CheckProppatchLocks 检查PROPPATCH操作所需的锁定
func (h *Handler) CheckProppatchLocks(c *gin.Context, requestPath string, propRequest *PropertyUpdateRequest) (bool, *Lock, error) {
	userID := c.GetString("userID")
	
	// 1. 检查直接资源锁定
	if locked, existingLock, err := h.lockManager.CheckExclusiveLock(requestPath, userID); err != nil {
		return true, existingLock, err
	} else if locked {
		return true, existingLock, fmt.Errorf("资源被锁定")
	}
	
	// 2. 检查父目录锁定
	if locked, existingLock, err := h.lockManager.CheckParentLocks(requestPath, userID); err != nil {
		return true, existingLock, err
	} else if locked {
		return true, existingLock, fmt.Errorf("父资源被锁定")
	}
	
	// 3. 检查深度锁定的影响
	// 检查深度锁定的资源
	if h.hasDepthLockOnDescendants(requestPath, userID) {
		return true, nil, fmt.Errorf("子资源存在深度锁定")
	}
	
	return false, nil, nil
}

// hasDepthLockOnDescendants 检查路径的子资源是否有深度锁定
func (h *Handler) hasDepthLockOnDescendants(path string, userID string) bool {
	// 获取路径下的所有锁定
	allLocks := h.lockManager.GetAllLocks()
	
	for _, lock := range allLocks {
		// 检查是否为深度锁定
		if lock.Depth == -1 {
			// 检查锁定路径是否为当前路径的子路径
			if strings.HasPrefix(lock.Path, path) && lock.Path != path {
				// 检查是否属于同一用户或用户有访问权限
				if lock.Owner != userID {
					return true
				}
			}
		}
	}
	
	return false
}

// ValidateProppatchLockOwnership 验证PROPPATCH操作的锁定所有权
func (h *Handler) ValidateProppatchLockOwnership(c *gin.Context, requestPath string, userID string) error {
	// 检查If头中的锁定令牌（如果存在）
	ifHeader := c.GetHeader("If")
	if ifHeader != "" {
		if tokens, err := ParseIfHeader(ifHeader); err == nil && len(tokens.Lists) > 0 {
			for _, list := range tokens.Lists {
				for _, condition := range list.Conditions {
					if condition.Token != "" {
						// 验证锁定令牌
						lock, exists := h.lockManager.GetLock(condition.Token)
						if !exists {
							return fmt.Errorf("锁定令牌无效: %s", condition.Token)
						}
						
						// 检查锁定令牌是否适用于当前路径
						if lock.Path != requestPath {
							return fmt.Errorf("锁定令牌不匹配路径")
						}
						
						// 检查锁定是否属于当前用户
						if lock.Owner != userID {
							return fmt.Errorf("锁定不属于当前用户")
						}
						
						// 检查锁定是否过期
						if time.Now().After(lock.ExpiresAt) {
							return fmt.Errorf("锁定已过期")
						}
					}
				}
			}
		}
	}
	
	return nil
}

// SendProppatchLockedError 发送PROPPATCH特定的锁定错误
func (h *Handler) SendProppatchLockedError(c *gin.Context, path string, lockToken, owner, message string) {
	// 创建PROPPATCH特定的锁定错误响应
	_ = CreateNoConflictingLockError([]string{path}) // 标记为已使用
	
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Lock-Token", fmt.Sprintf("<%s>", lockToken))
	c.Status(http.StatusLocked)
	
	// 生成详细的锁定错误信息
	errorXML := fmt.Sprintf(`<?xml version="1.1" encoding="utf-8"?>
<D:error xmlns:D="DAV:">
<D:no-conflicting-lock>
<D:locktoken>%s</D:locktoken>
<D:owner>%s</D:owner>
<D:message>%s</D:message>
</D:no-conflicting-lock>
</D:error>`, lockToken, owner, message)
	
	c.Writer.Write([]byte(errorXML))
}

// OptimizedProppatchLockCheck 优化的PROPPATCH锁定检查
func (h *Handler) OptimizedProppatchLockCheck(c *gin.Context, requestPath string, userID string) (bool, *Lock, error) {
	// 缓存锁定检查结果以提高性能
	// 这里可以实现简单的缓存机制
	
	// 1. 快速检查直接锁定
	if lock := h.lockManager.GetLockForPathAndUser(requestPath, userID); lock != nil {
		if lock.Type == LockTypeExclusive {
			return true, lock, fmt.Errorf("资源被独占锁定")
		}
	}
	
	// 2. 快速检查父目录锁定
	parentPath := getParentPath(requestPath)
	for parentPath != "" && parentPath != "/" {
		if lock := h.lockManager.GetLockForPathAndUser(parentPath, userID); lock != nil {
			if lock.Type == LockTypeExclusive && lock.Depth == -1 {
				return true, lock, fmt.Errorf("父资源被深度锁定")
			}
		}
		parentPath = getParentPath(parentPath)
	}
	
	return false, nil, nil
}

// getParentPath 获取路径的父目录
func getParentPath(path string) string {
	if path == "" || path == "/" {
		return ""
	}
	
	// 移除尾部的斜杠
	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	
	// 查找最后一个斜杠
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash <= 0 {
		return "/"
	}
	
	return path[:lastSlash]
}

// CheckProppatchLockCompatibility 检查PROPPATCH操作与现有锁的兼容性
func (h *Handler) CheckProppatchLockCompatibility(requestPath, operationType string, userID string) (*Lock, error) {
	// 获取路径上的所有锁定
	locks := h.lockManager.GetLocksForPath(requestPath)
	
	for _, lock := range locks {
		// 检查锁定是否过期
		if time.Now().After(lock.ExpiresAt) {
			continue // 跳过过期的锁定
		}
		
		// 根据操作类型检查兼容性
		switch operationType {
		case "set":
			// 设置属性需要独占访问
			if lock.Type == LockTypeExclusive && lock.Owner != userID {
				return lock, fmt.Errorf("资源被其他用户独占锁定")
			}
		case "remove":
			// 移除属性需要独占访问
			if lock.Type == LockTypeExclusive && lock.Owner != userID {
				return lock, fmt.Errorf("资源被其他用户独占锁定")
			}
		default:
			return lock, fmt.Errorf("未知的操作类型: %s", operationType)
		}
	}
	
	return nil, nil
}

// HandleProppatchEnhancedPROPFIND 增强的PROPFIND处理（包含自定义属性）
func (h *Handler) HandleProppatchEnhancedPROPFIND(c *gin.Context) {
	userID := c.GetString("userID")
	uid, _ := uuid.Parse(userID)
	
	requestPath := c.Param("path")
	if requestPath == "" {
		requestPath = "/"
	}

	depth := c.GetHeader("Depth")
	if depth == "" {
		depth = "infinity"
	}

	var responses []Response
	userIDString := uid.String()

	if depth == "0" {
		// Only the resource itself
		info, err := h.storage.StatObject(c.Request.Context(), uid, requestPath)
		if err != nil {
			// It might be a folder or root
			responses = append(responses, h.createFolderResponse(requestPath, time.Now(), userIDString))
		} else {
			responses = append(responses, h.createFileResponse(requestPath, info.Size, info.LastModified, info.ContentType, userIDString))
		}
	} else {
		// List directory contents
		objects, err := h.storage.ListObjects(c.Request.Context(), uid, requestPath, depth == "infinity")
		if err != nil {
			// Return root folder
			responses = append(responses, h.createFolderResponse(requestPath, time.Now(), userIDString))
		} else {
			// Add parent folder
			responses = append(responses, h.createFolderResponse(requestPath, time.Now(), userIDString))
			
			// Add files and folders
			for _, obj := range objects {
				objPath := "/" + obj.Key
				if strings.HasSuffix(obj.Key, "/") {
					responses = append(responses, h.createFolderResponse(objPath, obj.LastModified, userIDString))
				} else {
					responses = append(responses, h.createFileResponse(objPath, obj.Size, obj.LastModified, obj.ContentType, userIDString))
				}
			}
		}
	}

	multistatus := Multistatus{
		Xmlns:     "DAV:",
		Responses: responses,
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Status(http.StatusMultiStatus)
	
	c.Writer.Write([]byte(xml.Header))
	encoder := xml.NewEncoder(c.Writer)
	encoder.Indent("", "  ")
	encoder.Encode(multistatus)
}

// GetCustomPropertiesForPath 获取指定路径的自定义属性列表
func (h *Handler) GetCustomPropertiesForPath(userID, path string) ([]*Property, error) {
	ctx := context.Background()
	
	// 初始化属性服务
	if err := h.propertyService.Initialize(ctx); err != nil {
		return nil, err
	}
	
	// 查询属性
	return h.propertyService.ListProperties(ctx, userID, path)
}

// UpdatePROPFINDResponseWithCustomProperties 更新PROPFIND响应以包含自定义属性
func (h *Handler) UpdatePROPFINDResponseWithCustomProperties(c *gin.Context, responses []Response) {
	userID := c.GetString("userID")
	
	// 为每个响应添加自定义属性
	for i := range responses {
		customProps, err := h.GetCustomPropertiesForUser(userID, responses[i].Href)
		if err == nil {
			responses[i].Propstat[0].Prop.CustomProperties = customProps
		}
	}
}