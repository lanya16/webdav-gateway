package webdav

import (
	"encoding/xml"
	"time"

	webdavtypes "github.com/webdav-gateway/internal/types"
)

// ========================================
// PROPPATCH 请求结构体
// ========================================

// PropertyUpdateRequest PROPPATCH请求主体结构
type PropertyUpdateRequest = webdavtypes.PropertyUpdateRequest

// SetOperation 属性设置操作
type SetOperation struct {
	XMLName xml.Name    `xml:"set"`
	Prop    PropContent `xml:"prop"`
}

// RemoveOperation 属性移除操作
type RemoveOperation struct {
	XMLName xml.Name    `xml:"remove"`
	Prop    PropContent `xml:"prop"`
}

// PropContent 属性内容（支持任意属性）
type PropContent = webdavtypes.PropContent

// ========================================
// PROPPATCH 响应结构体
// ========================================

// ProppatchResponse PROPPATCH响应主体
type ProppatchResponse struct {
	XMLName   xml.Name   `xml:"D:response"`
	Xmlns     string     `xml:"xmlns:D,attr"`
	Href      string     `xml:"D:href"`
	Propstats []PropstatResponse `xml:"D:propstat"`
}

// PropstatResponse 属性状态响应（使用统一类型）
type PropstatResponse = webdavtypes.Propstat

// PropContentResponse 属性内容响应（使用统一类型）
type PropContentResponse = webdavtypes.PropContentResponse

// ========================================
// 属性存储结构体
// ========================================

// Property 属性数据结构（使用统一类型）


// PropertyOperation 属性操作记录
type PropertyOperation = webdavtypes.PropertyOperation

// PropertyUpdateResult 属性更新结果
type PropertyUpdateResult = webdavtypes.PropertyUpdateResult

// Property 属性类型
type Property = webdavtypes.Property

// ========================================
// 错误响应结构体
// ========================================

// PropertyError 属性操作错误
type PropertyError = webdavtypes.PropertyError

// ErrorCondition 错误条件结构（使用统一类型）
// 注意：这里使用PropertyErrorCondition，与xml_handlers.go中的LockErrorCondition不同
type ErrorCondition = webdavtypes.PropertyErrorCondition

// PropNotFound 属性未找到错误（使用统一类型）
type PropNotFound = webdavtypes.PropNotFound

// ========================================
// 常量定义
// ========================================

// PropertyNamespace 属性命名空间常量（使用统一的类型定义）
const (
	NamespaceDAV      = webdavtypes.NamespaceDAV
	NamespaceCustom   = webdavtypes.NamespaceCustom
	NamespaceUser     = webdavtypes.NamespaceUser
	NamespaceMetadata = "http://webdav-gateway.org/metadata"
)

// KnownLiveProperties 已知活属性列表
var KnownLiveProperties = map[string]bool{
	"displayname":       true,
	"getcontentlength":  true,
	"getcontenttype":    true,
	"getlastmodified":   true,
	"creationdate":      true,
	"resourcetype":      true,
	"getetag":           true,
	"supportedlock":     true,
	"lockdiscovery":     true,
}

// ========================================
// XML处理实现
// ========================================

// UnmarshalXML 和 MarshalXML 方法已在 webdavtypes.PropContent 中定义

// ========================================
// Error 接口实现
// ========================================

// Error 方法已在 webdavtypes.PropertyError 中定义

// IsPropertyError 检查是否为属性错误
func IsPropertyError(err error) bool {
	_, ok := err.(*webdavtypes.PropertyError)
	return ok
}