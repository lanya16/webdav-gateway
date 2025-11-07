package utils

import (
	"fmt"
	"strings"
	"time"
)

// StringUtil 字符串工具类
type StringUtil struct{}

var String StringUtil

// TrimWhitespace 移除字符串首尾空白字符
func (s StringUtil) TrimWhitespace(str string) string {
	if len(str) == 0 {
		return str
	}
	
	// 移除首尾空白字符
	start := 0
	end := len(str)
	
	for start < end && (str[start] == ' ' || str[start] == '\t' || str[start] == '\n' || str[start] == '\r') {
		start++
	}
	
	for end > start && (str[end-1] == ' ' || str[end-1] == '\t' || str[end-1] == '\n' || str[end-1] == '\r') {
		end--
	}
	
	return str[start:end]
}

// EscapeForXML XML转义处理
func (s StringUtil) EscapeForXML(str string) string {
	cleaned := strings.ReplaceAll(str, "<", "&lt;")
	cleaned = strings.ReplaceAll(cleaned, ">", "&gt;")
	cleaned = strings.ReplaceAll(cleaned, "&", "&amp;")
	cleaned = strings.ReplaceAll(cleaned, "\"", "&quot;")
	cleaned = strings.ReplaceAll(cleaned, "'", "&apos;")
	
	return s.TrimWhitespace(cleaned)
}

// PropertyUtil 属性工具类
type PropertyUtil struct{}

var Property PropertyUtil

// GenerateKey 生成属性唯一键
func (p PropertyUtil) GenerateKey(namespace, name string) string {
	return namespace + ":" + name
}

// ParseKey 解析属性键
func (p PropertyUtil) ParseKey(key string) (namespace, name string) {
	if idx := p.indexOfLastColon(key); idx > 0 {
		return key[:idx], key[idx+1:]
	}
	return "custom", key // 默认命名空间
}

// indexOfLastColon 找到最后一个冒号的位置
func (p PropertyUtil) indexOfLastColon(s string) int {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

// CreateSuccessResponse 创建成功响应
func (p PropertyUtil) CreateSuccessResponse(path string, properties []interface{}) []interface{} {
	if properties == nil {
		return []interface{}{}
	}
	return properties
}

// TimeUtil 时间工具类
type TimeUtil struct{}

var Time TimeUtil

// UnixToTime Unix时间戳转换为时间
func (t TimeUtil) UnixToTime(unix int64) time.Time {
	return time.Unix(unix, 0)
}

// TimeToUnix 时间转换为Unix时间戳
func (t TimeUtil) TimeToUnix(tm time.Time) int64 {
	return tm.Unix()
}

// NowUnix 获取当前Unix时间戳
func (t TimeUtil) NowUnix() int64 {
	return time.Now().Unix()
}

// FormatDuration 格式化持续时间
func (t TimeUtil) FormatDuration(duration time.Duration) string {
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
}

// Slugify 转换为URL友好格式
func (s StringUtil) Slugify(str string) string {
	// 移除或替换特殊字符
	replacements := map[string]string{
		" ": "-",
		"_": "-",
		".": "-",
		"/": "-",
		"\\": "-",
	}
	
	result := str
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	
	// 转为小写
	result = strings.ToLower(result)
	
	// 移除其他特殊字符
	var allowed string
	for _, r := range result {
		if (r >= 'a' && r <= 'z') || 
		   (r >= '0' && r <= '9') || 
		   r == '-' {
			allowed += string(r)
		}
	}
	
	// 清理多余连字符
	result = strings.ReplaceAll(allowed, "--", "-")
	result = strings.Trim(result, "-")
	
	return result
}

// SplitNameSpace 拆分命名空间和属性名
func (p PropertyUtil) SplitNameSpace(fullName string) (namespace, name string) {
	parts := strings.SplitN(fullName, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "custom", fullName
}

// MergeSlices 合并切片
func MergeSlices[T any](slices ...[]T) []T {
	if len(slices) == 0 {
		return nil
	}
	
	result := make([]T, 0, len(slices[0]))
	for _, slice := range slices {
		result = append(result, slice...)
	}
	
	return result
}

// RemoveDuplicates 移除重复元素
func RemoveDuplicates[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := make([]T, 0, len(slice))
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// MapSlice 映射切片元素
func MapSlice[T any, R any](slice []T, fn func(T) R) []R {
	if slice == nil {
		return nil
	}
	
	result := make([]R, len(slice))
	for i, item := range slice {
		result[i] = fn(item)
	}
	
	return result
}

// FilterSlice 过滤切片元素
func FilterSlice[T any](slice []T, fn func(T) bool) []T {
	if slice == nil {
		return nil
	}
	
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if fn(item) {
			result = append(result, item)
		}
	}
	
	return result
}

// ReduceSlice 归约切片元素
func ReduceSlice[T any, R any](slice []T, initial R, fn func(R, T) R) R {
	result := initial
	for _, item := range slice {
		result = fn(result, item)
	}
	return result
}

// IsEmpty 检查切片是否为空
func IsEmpty[T any](slice []T) bool {
	return len(slice) == 0
}

// IsNotEmpty 检查切片是否不为空
func IsNotEmpty[T any](slice []T) bool {
	return len(slice) > 0
}

// Contains 检查切片是否包含元素
func Contains[T comparable](slice []T, item T) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// MapContains 检查映射是否包含键
func MapContains[K comparable, V any](m map[K]V, key K) bool {
	_, exists := m[key]
	return exists
}

// SafeMapGet 安全获取映射值
func SafeMapGet[K comparable, V any](m map[K]V, key K) (V, bool) {
	val, exists := m[key]
	return val, exists
}

// WithDefault 返回默认值如果值是零值
func WithDefault[T comparable](value, defaultValue T) T {
	var zero T
	if value == zero {
		return defaultValue
	}
	return value
}

// SafeSliceAccess 安全访问切片元素
func SafeSliceAccess[T any](slice []T, index int) (T, bool) {
	var zero T
	if index < 0 || index >= len(slice) {
		return zero, false
	}
	return slice[index], true
}