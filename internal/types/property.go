package types

import (
	"encoding/xml"
	"time"
)

// ========================================
// Property Types - 共享的属性类型定义
// ========================================

// PropertyError 属性错误
type PropertyError struct {
	Code        int       `json:"code"`
	Message     string    `json:"message"`
	Description string    `json:"description,omitempty"`
	Property    string    `json:"property,omitempty"`
	PropertyObj Property  `json:"property_obj,omitempty"`
}

func (e *PropertyError) Error() string {
	return e.Message
}

// Property 属性定义
type Property struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
	IsLive    bool   `json:"is_live"`
	
	// 可选的额外字段，用于XML处理
	UserID     string `json:"user_id,omitempty"`
	ResourceID string `json:"resource_id,omitempty"`
	Path       string `json:"path,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

// PropContent 属性内容
type PropContent struct {
	XMLName struct {
		Space string
		Local string
	}
	Value string
}

// ========================================
// PROPPATCH Types - PROPPATCH相关类型
// ========================================

// PropertyUpdateRequest PROPPATCH请求
type PropertyUpdateRequest struct {
	SetOperations    []SetOperation    `xml:"D:set>prop"`
	RemoveOperations []RemoveOperation `xml:"D:remove>prop"`
}

// SetOperation 设置操作
type SetOperation struct {
	PropContent []PropContent `xml:"*"`
}

// RemoveOperation 移除操作
type RemoveOperation struct {
	PropContent []PropContent `xml:"*"`
}

// ProppatchResponse PROPPATCH响应
type ProppatchResponse struct {
	Status        string                        `xml:"D:status"`
	PropertyNames []string                      `xml:"D:prop>propstat>D:prop>D:*>D:*"`
}

// PropertyUpdateResult 属性更新结果
type PropertyUpdateResult struct {
	Success   bool                 `json:"success"`
	Property  Property             `json:"property"`
	Operation string               `json:"operation"`
	Error     *PropertyError       `json:"error,omitempty"`
	ResourcePath string             `json:"resource_path,omitempty"`
	Propstats []Propstat           `json:"propstats,omitempty"`
	Operations []PropertyOperation `json:"operations,omitempty"`
	SuccessCount int               `json:"success_count,omitempty"`
	ErrorCount int                 `json:"error_count,omitempty"`
}

// Propstat 属性状态
type Propstat struct {
	Prop ResponseProp `json:"prop"`
	Status string     `json:"status"`
}

// PropContentResponse 属性内容响应
type PropContentResponse struct {
	DisplayName string            `json:"displayname"`
	CustomProps map[string]string `json:"custom_props"`
}

// PropertyOperation 属性操作
type PropertyOperation struct {
	Property Property `json:"property"`
	Operation string  `json:"operation"`
	Success  bool     `json:"success"`
	PropertyName string `json:"property_name,omitempty"`
	Namespace string  `json:"namespace,omitempty"`
	Value     *string `json:"value,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// ========================================
// WebDAV Response Types - 统一响应类型
// ========================================

// WebDAVPropstat 通用WebDAV属性状态结构
type WebDAVPropstat struct {
	Prop   interface{} `xml:"D:prop"`
	Status string      `xml:"D:status"`
}

// ResponseProp WebDAV响应属性（兼容handler.go的ResponseProp）
type ResponseProp struct {
	DisplayName       string        `xml:"D:displayname,omitempty"`
	GetContentLength  int64         `xml:"D:getcontentlength,omitempty"`
	GetContentType    string        `xml:"D:getlastmodified,omitempty"`
	GetLastModified   string        `xml:"D:getlastmodified,omitempty"`
	CreationDate      string        `xml:"D:creationdate,omitempty"`
	ResourceType      *ResourceType `xml:"D:resourcetype,omitempty"`
	GetETag           string        `xml:"D:getetag,omitempty"`
	SupportedLock     []interface{} `xml:"D:supportedlock>DAV:lockentry,omitempty"`
	LockDiscovery     []ActiveLock  `xml:"D:lockdiscovery,omitempty"`
	// 自定义属性支持
	CustomProperties  map[string]string `xml:"-"`
}

// ResourceType 资源类型
type ResourceType struct {
	Collection *struct{} `xml:"D:collection,omitempty"`
}

// LockScopeInfo 锁作用域信息（XML格式）
type LockScopeInfo struct {
	Exclusive *struct{} `xml:"D:exclusive,omitempty"`
	Shared    *struct{} `xml:"D:shared,omitempty"`
}

// LockTypeInfo 锁类型信息（XML格式）
type LockTypeInfo struct {
	Write *struct{} `xml:"D:write,omitempty"`
}

// ActiveLock 活跃锁
type ActiveLock struct {
	XMLName   xml.Name      `xml:"D:activelock"`
	LockScope LockScopeInfo `xml:"D:lockscope"`
	LockType  LockTypeInfo  `xml:"D:locktype"`
	Depth     string        `xml:"D:depth"`
	Owner     string        `xml:"D:owner,omitempty"`
	Timeout   string        `xml:"D:timeout"`
	LockToken string        `xml:"D:locktoken,omitempty"`
}

// LockInfoRequest LOCK请求体结构
type LockInfoRequest struct {
	XMLName   xml.Name      `xml:"lockinfo"`
	Namespace string        `xml:"xmlns,attr,omitempty"`
	LockScope LockScopeInfo `xml:"lockscope"`
	LockType  LockTypeInfo  `xml:"locktype"`
	Owner     *OwnerInfo    `xml:"owner,omitempty"`
}

// OwnerInfo 锁定所有者信息
type OwnerInfo struct {
	XMLName xml.Name `xml:"owner"`
	Href    string   `xml:",chardata"`
}

// LockInfo 锁信息
type LockInfo struct {
	LockType   *LockTypeInfo  `xml:"D:lockscope>DAV:locktype>D:*"`
	LockScope  *LockScopeInfo `xml:"D:lockscope>DAV:lockscope>D:*"`
	LockOwner  string         `xml:"D:owner>D:owner>"`
	LockToken  *LockToken     `xml:"D:locktoken>D:href"`
	Timeout    string         `xml:"D:timeout"`
}

// LockToken 锁令牌
type LockToken struct {
	XMLName xml.Name `xml:"D:locktoken"`
	Href    string   `xml:"D:href"`
}

// Legacy LockType/LockScope for backwards compatibility
// LockType 锁类型（保留向后兼容）
type LockType struct {
	Write *struct{} `xml:"D:write,omitempty"`
}

// LockScope 锁作用域（保留向后兼容）
type LockScope struct {
	Exclusive *struct{} `xml:"D:exclusive,omitempty"`
	Shared    *struct{} `xml:"D:shared,omitempty"`
}

// DAVResponseProp DAV响应属性结构（简化版）
type DAVResponseProp struct {
	DisplayName  string `xml:"D:displayname,omitempty"`
}

// CustomResponseProp 自定义响应属性结构
type CustomResponseProp struct {
	CustomProps map[string]string `xml:"-"`
}



// ========================================
// Error Condition Types - 错误条件类型
// ========================================

// PropertyErrorCondition 属性错误条件（用于PROPPATCH错误）
type PropertyErrorCondition struct {
	XMLName       xml.Name         `xml:"D:error"`
	Xmlns         string           `xml:"xmlns:D,attr"`
	PropNotFound  *PropNotFound    `xml:"D:prop-not-found,omitempty"`
}

// PropNotFound 属性未找到错误
type PropNotFound struct {
	XMLName  xml.Name `xml:"D:prop-not-found"`
	PropName string   `xml:",chardata"`
}

// LockErrorCondition 锁定错误条件（用于LOCK/UNLOCK错误）
type LockErrorCondition struct {
	XMLName             xml.Name              `xml:"D:error"`
	Xmlns               string                `xml:"xmlns:D,attr"`
	NoConflictingLock   *NoConflictingLock    `xml:"D:no-conflicting-lock,omitempty"`
	LockTokenSubmitted  *LockTokenSubmitted   `xml:"D:lock-token-submitted,omitempty"`
	LockTokenMismatch   *LockTokenMismatch    `xml:"D:lock-token-matches-request-uri,omitempty"`
}

// NoConflictingLock 冲突锁错误
type NoConflictingLock struct {
	XMLName xml.Name  `xml:"D:no-conflicting-lock"`
	Href    []string  `xml:"D:href,omitempty"`
}

// LockTokenSubmitted 未提交锁令牌错误
type LockTokenSubmitted struct {
	XMLName xml.Name  `xml:"D:lock-token-submitted"`
	Href    []string  `xml:"D:href,omitempty"`
}

// LockTokenMismatch 锁令牌不匹配错误
type LockTokenMismatch struct {
	XMLName xml.Name `xml:"D:lock-token-matches-request-uri"`
}

// ErrorConditionDetail 错误条件详情（向后兼容别名）
type ErrorConditionDetail = LockErrorCondition

// ========================================
// Namespace Constants - 命名空间常量
// ========================================

const (
	// NamespaceDAV DAV命名空间
	NamespaceDAV = "DAV"
	
	// NamespaceCustom 自定义命名空间
	NamespaceCustom = "CUSTOM"
	
	// NamespaceUser 用户命名空间
	NamespaceUser = "USER"
)

// ========================================
// Known Live Properties - 已知的活属性
// ========================================

// KnownLiveProperties 已知的活属性映射
var KnownLiveProperties = map[string]bool{
	"creationdate":        true,
	"getcontentlanguage":  true,
	"getcontentlength":    true,
	"getcontenttype":      true,
	"getetag":             true,
	"getlastmodified":     true,
	"lockdiscovery":       true,
	"resourcetype":        true,
	"source":              true,
	"supportedlock":       true,
	"displayname":         true,
}