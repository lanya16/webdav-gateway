package webdav

import (
	"encoding/xml"
	"fmt"
	"io"

	webdavtypes "github.com/webdav-gateway/internal/types"
)

// WebDAV XML 请求结构

// LockInfoRequest LOCK请求体结构（使用统一类型）
type LockInfoRequest = webdavtypes.LockInfoRequest

// OwnerInfo 锁定所有者信息
type OwnerInfo struct {
	XMLName xml.Name `xml:"owner"`
	Href    string   `xml:"href,omitempty"`
	Value   string   `xml:",chardata"`
}

// WebDAV XML 响应结构

// PropResponse LOCK响应体结构（DAV:prop）
type PropResponse struct {
	XMLName       xml.Name     `xml:"D:prop"`
	Namespace     string       `xml:"xmlns:D,attr"`
	LockDiscovery []ActiveLock `xml:"D:lockdiscovery>D:activelock"`
}

// ErrorResponse WebDAV错误响应
type ErrorResponse struct {
	XMLName   xml.Name `xml:"D:error"`
	Namespace string   `xml:"xmlns:D,attr"`
	Condition ErrorConditionDetail
}

// ErrorConditionDetail 错误条件（使用统一类型）
type ErrorConditionDetail = webdavtypes.LockErrorCondition

// NoConflictingLock 冲突锁错误（使用统一类型）
type NoConflictingLock = webdavtypes.NoConflictingLock

// LockTokenSubmitted 未提交锁令牌错误（使用统一类型）
type LockTokenSubmitted = webdavtypes.LockTokenSubmitted

// LockTokenMismatch 锁令牌不匹配错误（使用统一类型）
type LockTokenMismatch = webdavtypes.LockTokenMismatch

// XML 解析辅助函数

// ParseLockInfo 解析LOCK请求体
func ParseLockInfo(body io.Reader) (*LockInfoRequest, error) {
	var lockInfo LockInfoRequest

	decoder := xml.NewDecoder(body)
	if err := decoder.Decode(&lockInfo); err != nil {
		return nil, fmt.Errorf("failed to parse lock info: %w", err)
	}

	return &lockInfo, nil
}

// ParseLockInfoFromBytes 从字节数组解析LOCK请求体
func ParseLockInfoFromBytes(data []byte) (*LockInfoRequest, error) {
	var lockInfo LockInfoRequest

	if err := xml.Unmarshal(data, &lockInfo); err != nil {
		return nil, fmt.Errorf("failed to parse lock info: %w", err)
	}

	return &lockInfo, nil
}

// XML 生成辅助函数

// MarshalPropResponse 序列化LOCK响应
func MarshalPropResponse(activeLocks []ActiveLock) ([]byte, error) {
	response := PropResponse{
		Namespace:     "DAV:",
		LockDiscovery: activeLocks,
	}

	output, err := xml.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prop response: %w", err)
	}

	return append([]byte(xml.Header), output...), nil
}

// MarshalErrorResponse 序列化错误响应
func MarshalErrorResponse(condition ErrorConditionDetail) ([]byte, error) {
	response := ErrorResponse{
		Namespace: "DAV:",
		Condition: condition,
	}

	output, err := xml.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal error response: %w", err)
	}

	return append([]byte(xml.Header), output...), nil
}

// CreateNoConflictingLockError 创建冲突锁错误
func CreateNoConflictingLockError(conflictingPaths []string) webdavtypes.LockErrorCondition {
	return webdavtypes.LockErrorCondition{
		NoConflictingLock: &webdavtypes.NoConflictingLock{
			Href: conflictingPaths,
		},
	}
}

// CreateLockTokenSubmittedError 创建未提交令牌错误
func CreateLockTokenSubmittedError(lockedPaths []string) webdavtypes.LockErrorCondition {
	return webdavtypes.LockErrorCondition{
		LockTokenSubmitted: &webdavtypes.LockTokenSubmitted{
			Href: lockedPaths,
		},
	}
}

// CreateLockTokenMismatchError 创建令牌不匹配错误
func CreateLockTokenMismatchError() webdavtypes.LockErrorCondition {
	return webdavtypes.LockErrorCondition{
		LockTokenMismatch: &webdavtypes.LockTokenMismatch{},
	}
}

// If 头解析

// IfHeader If头部结构
type IfHeader struct {
	Lists []IfList
}

// IfList If列表（Tagged或No-tag）
type IfList struct {
	ResourceTag string      // 资源标签（Tagged list）
	Conditions  []Condition // 条件列表
}

// Condition If条件
type Condition struct {
	Token   string // 锁令牌
	ETag    string // ETag值
	Not     bool   // 是否为NOT条件
}

// ParseIfHeader 解析If头部
// 简化实现，支持基本的令牌匹配
func ParseIfHeader(ifHeader string) (*IfHeader, error) {
	if ifHeader == "" {
		return nil, nil
	}

	// 简单解析：提取所有 <token> 格式的令牌
	var lists []IfList
	var currentList IfList

	// 移除外层括号并分割
	ifHeader = trimSpaces(ifHeader)

	// 查找所有 <...> 格式的令牌
	tokens := extractTokens(ifHeader)
	for _, token := range tokens {
		currentList.Conditions = append(currentList.Conditions, Condition{
			Token: token,
			Not:   false,
		})
	}

	if len(currentList.Conditions) > 0 {
		lists = append(lists, currentList)
	}

	return &IfHeader{Lists: lists}, nil
}

// extractTokens 提取If头中的所有令牌
func extractTokens(header string) []string {
	var tokens []string
	inToken := false
	currentToken := ""

	for _, ch := range header {
		if ch == '<' {
			inToken = true
			currentToken = ""
		} else if ch == '>' && inToken {
			inToken = false
			if currentToken != "" {
				tokens = append(tokens, currentToken)
			}
		} else if inToken {
			currentToken += string(ch)
		}
	}

	return tokens
}

// trimSpaces 移除字符串首尾空格
func trimSpaces(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}

	return s[start:end]
}

// ValidateIfHeader 验证If头中的令牌
func ValidateIfHeader(ifHeader string, validTokens []string) (bool, string) {
	if ifHeader == "" {
		return len(validTokens) == 0, ""
	}

	parsed, err := ParseIfHeader(ifHeader)
	if err != nil {
		return false, ""
	}

	// 检查是否有任何令牌匹配
	for _, list := range parsed.Lists {
		for _, condition := range list.Conditions {
			for _, validToken := range validTokens {
				if condition.Token == validToken {
					return true, validToken
				}
			}
		}
	}

	return false, ""
}

// Timeout 头解析

// ParseTimeout 解析Timeout头部
func ParseTimeout(timeoutHeader string) int64 {
	if timeoutHeader == "" {
		return 3600 // 默认1小时
	}

	// 支持格式：Second-3600, Infinite
	if timeoutHeader == "Infinite" || timeoutHeader == "infinite" {
		return 86400 * 365 // 1年
	}

	// 解析 Second-XXX 格式
	var seconds int64
	_, err := fmt.Sscanf(timeoutHeader, "Second-%d", &seconds)
	if err != nil {
		return 3600 // 默认1小时
	}

	if seconds <= 0 {
		return 3600
	}

	return seconds
}

// Depth 头解析

// ParseDepth 解析Depth头部
func ParseDepth(depthHeader string) int {
	if depthHeader == "" {
		return 0 // 默认深度0
	}

	if depthHeader == "infinity" || depthHeader == "Infinity" {
		return -1 // 使用-1表示infinity
	}

	var depth int
	_, err := fmt.Sscanf(depthHeader, "%d", &depth)
	if err != nil {
		return 0
	}

	return depth
}

// FormatDepth 格式化深度值
func FormatDepth(depth int) string {
	if depth == -1 {
		return "infinity"
	}
	return fmt.Sprintf("%d", depth)
}

// CreateActiveLockResponse 创建ActiveLock响应
func CreateActiveLockResponse(lock *Lock, requestURL string) ActiveLock {
	activeLock := ActiveLock{
		LockScope: LockScopeInfo{},
		LockType: LockTypeInfo{
			Write: &struct{}{},
		},
		Depth:   FormatDepth(lock.Depth),
		Owner:   lock.Owner,
		Timeout: fmt.Sprintf("Second-%d", lock.Timeout),
		LockToken: LockToken{
			Href: lock.Token,
		},
		LockRoot: LockRoot{
			Href: requestURL,
		},
	}

	// 设置锁范围
	if lock.Type == LockTypeExclusive {
		activeLock.LockScope.Exclusive = &struct{}{}
	} else {
		activeLock.LockScope.Shared = &struct{}{}
	}

	return activeLock
}

// CreateMultiStatusResponse 创建207 Multi-Status响应
func CreateMultiStatusResponse(responses []Response) ([]byte, error) {
	multistatus := Multistatus{
		Xmlns:     "DAV:",
		Responses: responses,
	}

	output, err := xml.MarshalIndent(multistatus, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal multistatus: %w", err)
	}

	return append([]byte(xml.Header), output...), nil
}