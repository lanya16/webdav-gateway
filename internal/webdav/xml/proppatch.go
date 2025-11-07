package xml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/webdav-gateway/internal/types"
)

// ReadXMLBody 从HTTP请求体中读取XML数据，返回XML解码器
func ReadXMLBody(r io.Reader) (*xml.Decoder, error) {
	// 读取所有数据
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读取请求体失败: %v", err)
	}

	// 检查请求体是否为空
	if len(body) == 0 {
		return nil, fmt.Errorf("请求体不能为空")
	}

	// 基本XML格式验证
	if !bytes.HasPrefix(body, []byte("<?xml")) && !bytes.HasPrefix(body, []byte("<propertyupdate")) {
		return nil, fmt.Errorf("无效的XML格式: 必须以<?xml或<propertyupdate开始")
	}

	// 创建XML解码器
	decoder := xml.NewDecoder(bytes.NewReader(body))
	
	// 尝试解析XML结构以验证语法
	var temp interface{}
	if err := decoder.Decode(&temp); err != nil {
		return nil, fmt.Errorf("XML语法错误: %v", err)
	}

	// 重新创建解码器供实际使用
	decoder = xml.NewDecoder(bytes.NewReader(body))
	return decoder, nil
}

// ParseProppatchRequest 解析PROPPATCH请求XML，返回PropertyUpdateRequest列表
func ParseProppatchRequest(decoder *xml.Decoder) ([]types.PropertyUpdateRequest, error) {
	// 解析整个XML文档
	var request types.PropertyUpdateRequest
	if err := decoder.Decode(&request); err != nil {
		return nil, fmt.Errorf("解析PROPPATCH请求失败: %v", err)
	}

	// 验证请求结构
	if len(request.SetOperations) == 0 && len(request.RemoveOperations) == 0 {
		return nil, fmt.Errorf("PROPPATCH请求必须包含set或remove操作")
	}

	// 验证每个操作的有效性
	if err := validatePropertyOperations(request); err != nil {
		return nil, fmt.Errorf("验证属性操作失败: %v", err)
	}

	return []types.PropertyUpdateRequest{request}, nil
}

// ResolveNamespace 解析命名空间前缀，返回完整的命名空间字符串
func ResolveNamespace(prefix, uri string) string {
	// 如果URI已经提供了完整的命名空间，直接返回
	if uri != "" {
		return uri
	}

	// 如果前缀为空，返回默认命名空间
	if prefix == "" {
		return types.NamespaceCustom
	}

	// 根据常见前缀返回标准命名空间
	namespaceMap := map[string]string{
		"D":      types.NamespaceDAV,
		"DAV":    types.NamespaceDAV,
		"d":      types.NamespaceDAV,
		"dav":    types.NamespaceDAV,
		"custom": "http://webdav-gateway.org/properties",
		"user":   types.NamespaceUser,
		"meta":   "http://webdav-gateway.org/metadata",
		"sys":    "http://webdav-gateway.org/system",
	}

	// 查找映射的命名空间
	if ns, exists := namespaceMap[prefix]; exists {
		return ns
	}

	// 如果没有找到映射，返回自定义命名空间
	return fmt.Sprintf("http://webdav-gateway.org/%s", prefix)
}

// GetPropertyKey 生成属性的唯一键，用于映射和查找
func GetPropertyKey(namespace, name string) string {
	// 构建唯一键: namespace:name
	if namespace == "" || namespace == types.NamespaceCustom {
		return name
	}
	return namespace + ":" + name
}

// GetPropertyKeyFromProperty 从Property结构生成唯一键
func GetPropertyKeyFromProperty(property types.Property) string {
	// 构建唯一键: namespace:name
	if property.Namespace == "" || property.Namespace == types.NamespaceCustom {
		return property.Name
	}
	return property.Namespace + ":" + property.Name
}

// validatePropertyOperations 验证属性操作的辅助函数
func validatePropertyOperations(request types.PropertyUpdateRequest) error {
	// 验证set操作
	for i, setOp := range request.SetOperations {
		if err := validatePropertyOperation("set", setOp.PropContent, fmt.Sprintf("set[%d]", i)); err != nil {
			return err
		}
	}

	// 验证remove操作
	for i, removeOp := range request.RemoveOperations {
		if err := validatePropertyOperation("remove", removeOp.PropContent, fmt.Sprintf("remove[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

// validatePropertyOperation 验证单个属性操作的辅助函数
func validatePropertyOperation(operation string, propContent []types.PropContent, context string) error {
	for _, prop := range propContent {
		// 检查属性名是否为空
		if prop.XMLName.Local == "" {
			return fmt.Errorf("%s: 属性名不能为空", context)
		}

		// 检查set操作是否有值
		if operation == "set" && prop.Value == "" {
			// 空值是允许的，表示设置为空字符串
			continue
		}

		// 检查属性值长度
		if len(prop.Value) > 1024*1024 { // 1MB限制
			return fmt.Errorf("%s: 属性值过大", context)
		}
	}

	return nil
}