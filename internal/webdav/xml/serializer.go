package xml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/webdav-gateway/internal/types"
)

// Serializer XML序列化器
type Serializer struct {
	encoderOptions encoderOptions
}

// encoderOptions 编码选项
type encoderOptions struct {
	Indent string
	Prefix string
}

// NewSerializer 创建新的XML序列化器
func NewSerializer() *Serializer {
	return &Serializer{
		encoderOptions: encoderOptions{
			Indent: "  ",
			Prefix: "  ",
		},
	}
}

// WithIndent 设置缩进
func (s *Serializer) WithIndent(prefix string, indent string) *Serializer {
	s.encoderOptions.Prefix = prefix
	s.encoderOptions.Indent = indent
	return s
}

// EncodeProppatchRequest 编码PROPPATCH请求
func (s *Serializer) EncodeProppatchRequest(request *types.PropertyUpdateRequest) ([]byte, error) {
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent(s.encoderOptions.Prefix, s.encoderOptions.Indent)
	
	if err := encoder.Encode(request); err != nil {
		return nil, fmt.Errorf("编码PROPPATCH请求失败: %v", err)
	}
	
	return buf.Bytes(), nil
}

// DecodeProppatchRequest 解码PROPPATCH请求
func (s *Serializer) DecodeProppatchRequest(data []byte) (*types.PropertyUpdateRequest, error) {
	var request types.PropertyUpdateRequest
	if err := xml.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("解码PROPPATCH请求失败: %v", err)
	}
	
	return &request, nil
}

// EncodeProppatchResponse 编码PROPPATCH响应
func (s *Serializer) EncodeProppatchResponse(response *types.ProppatchResponse) ([]byte, error) {
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent(s.encoderOptions.Prefix, s.encoderOptions.Indent)
	
	if err := encoder.Encode(response); err != nil {
		return nil, fmt.Errorf("编码PROPPATCH响应失败: %v", err)
	}
	
	return buf.Bytes(), nil
}

// DecodeProppatchResponse 解码PROPPATCH响应
func (s *Serializer) DecodeProppatchResponse(data []byte) (*types.ProppatchResponse, error) {
	var response types.ProppatchResponse
	if err := xml.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("解码PROPPATCH响应失败: %v", err)
	}
	
	return &response, nil
}

// EncodeMultiStatusResponse 编码Multi-Status响应
func (s *Serializer) EncodeMultiStatusResponse(responses []types.ProppatchResponse) ([]byte, error) {
	multistatus := struct {
		XMLName   xml.Name             `xml:"D:multistatus"`
		Xmlns     string               `xml:"xmlns:D,attr"`
		Responses []types.ProppatchResponse `xml:"D:response"`
	}{
		Xmlns:     "DAV:",
		Responses: responses,
	}
	
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	encoder.Indent(s.encoderOptions.Prefix, s.encoderOptions.Indent)
	
	if err := encoder.Encode(multistatus); err != nil {
		return nil, fmt.Errorf("编码Multi-Status响应失败: %v", err)
	}
	
	return buf.Bytes(), nil
}

// Validator XML验证器
type Validator struct {
	maxContentLength int
}

// NewValidator 创建新的XML验证器
func NewValidator() *Validator {
	return &Validator{
		maxContentLength: 1024 * 1024, // 1MB
	}
}

// WithMaxContentLength 设置最大内容长度
func (v *Validator) WithMaxContentLength(length int) *Validator {
	v.maxContentLength = length
	return v
}

// ValidateProppatchFormat 验证PROPPATCH XML格式
func (v *Validator) ValidateProppatchFormat(xmlBytes []byte) error {
	// 基本XML语法检查
	if !bytes.HasPrefix(xmlBytes, []byte("<?xml")) {
		return fmt.Errorf("XML声明缺失")
	}
	
	// 检查是否以"<propertyupdate>"开头
	trimmed := strings.TrimSpace(string(xmlBytes))
	if !strings.HasPrefix(trimmed, "<propertyupdate") {
		return fmt.Errorf("必须以<propertyupdate>元素开始")
	}
	
	// 检查是否以"</propertyupdate>"结尾
	if !strings.HasSuffix(trimmed, "</propertyupdate>") {
		return fmt.Errorf("必须以</propertyupdate>元素结尾")
	}
	
	// 检查内容长度
	if len(xmlBytes) > v.maxContentLength {
		return fmt.Errorf("请求体过大")
	}
	
	// 尝试解析XML结构
	var temp interface{}
	if err := xml.Unmarshal(xmlBytes, &temp); err != nil {
		return fmt.Errorf("XML语法错误: %v", err)
	}
	
	return nil
}

// ExtractPropertyNames 提取属性名列表
func (v *Validator) ExtractPropertyNames(xmlBytes []byte) ([]string, error) {
	// 使用正则表达式提取属性名
	re := regexp.MustCompile(`<[^/>]*[^\w:]([a-zA-Z_][\w:-]*)[^>]*>`)
	matches := re.FindAllStringSubmatch(string(xmlBytes), -1)
	
	properties := make([]string, 0)
	for _, match := range matches {
		if len(match) > 1 {
			propName := match[1]
			// 排除元素名
			if propName != "propertyupdate" && propName != "set" && propName != "remove" && propName != "prop" {
				properties = append(properties, propName)
			}
		}
	}
	
	return properties, nil
}

// ParseNamespaceDeclarations 解析命名空间声明
func (v *Validator) ParseNamespaceDeclarations(xmlBytes []byte) map[string]string {
	namespaces := make(map[string]string)
	
	// 使用正则表达式提取命名空间声明
	re := regexp.MustCompile(`xmlns:([^=]+)=["']([^"']+)["']`)
	matches := re.FindAllStringSubmatch(string(xmlBytes), -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			prefix := match[1]
			url := match[2]
			namespaces[prefix] = url
		}
	}
	
	return namespaces
}

// NamespaceResolver 命名空间解析器
type NamespaceResolver struct {
	mappings map[string]string
}

// NewNamespaceResolver 创建新的命名空间解析器
func NewNamespaceResolver() *NamespaceResolver {
	return &NamespaceResolver{
		mappings: make(map[string]string),
	}
}

// AddMapping 添加命名空间映射
func (nr *NamespaceResolver) AddMapping(prefix, url string) {
	nr.mappings[prefix] = url
}

// Resolve 解析命名空间URL
func (nr *NamespaceResolver) Resolve(prefix string, defaults map[string]string) string {
	if url, exists := nr.mappings[prefix]; exists {
		return url
	}
	
	// 默认映射
	if url, exists := defaults[prefix]; exists {
		return url
	}
	
	// 根据常见命名空间前缀推断
	commonMappings := map[string]string{
		"D":     "DAV:",
		"DAV":   "DAV:",
		"":      "http://webdav-gateway.org/properties",
		"custom": "http://webdav-gateway.org/properties",
		"user":  "http://webdav-gateway.org/user",
		"meta":  "http://webdav-gateway.org/metadata",
	}
	
	return commonMappings[prefix]
}

// Builder XML构建器
type Builder struct {
	indentLevel int
	indentStr   string
}

// NewBuilder 创建新的XML构建器
func NewBuilder() *Builder {
	return &Builder{
		indentLevel: 0,
		indentStr:   "  ",
	}
}

// WithIndentString 设置缩进字符串
func (b *Builder) WithIndentString(indent string) *Builder {
	b.indentStr = indent
	return b
}

// StartElement 开始元素
func (b *Builder) StartElement(name string) string {
	b.indentLevel++
	return strings.Repeat(b.indentStr, b.indentLevel-1) + "<" + name + ">"
}

// EndElement 结束元素
func (b *Builder) EndElement(name string) string {
	level := b.indentLevel
	b.indentLevel--
	return strings.Repeat(b.indentStr, level-1) + "</" + name + ">"
}

// Element 完整元素
func (b *Builder) Element(name, content string) string {
	indent := strings.Repeat(b.indentStr, b.indentLevel)
	return indent + "<" + name + ">" + content + "</" + name + ">"
}

// SelfClosingElement 自闭合元素
func (b *Builder) SelfClosingElement(name string, attrs ...string) string {
	indent := strings.Repeat(b.indentStr, b.indentLevel)
	if len(attrs) > 0 {
		return indent + "<" + name + " " + strings.Join(attrs, " ") + "/>"
	}
	return indent + "<" + name + "/>"
}

// RawContent 原始内容
func (b *Builder) RawContent(content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	indent := strings.Repeat(b.indentStr, b.indentLevel)
	return indent + content
}

// CreatePropContent 创建PropContent的XML表示
func (b *Builder) CreatePropContent(prop types.PropContent) string {
	var buffer strings.Builder
	
	// 开始prop元素
	buffer.WriteString(b.StartElement("D:prop"))
	
	// 如果有值，添加值
	if prop.Value != "" {
		value := strings.TrimSpace(prop.Value)
		if value != "" {
			buffer.WriteString(b.Element(prop.XMLName.Local, value))
		}
	} else {
		// 只有元素名
		indent := strings.Repeat(b.indentStr, b.indentLevel)
		buffer.WriteString(indent + "<" + prop.XMLName.Local + "/>")
	}
	
	// 结束prop元素
	buffer.WriteString(b.EndElement("D:prop"))
	
	return buffer.String()
}

// XMLParser XML解析器
type XMLParser struct {
	validator *Validator
	resolver  *NamespaceResolver
}

// NewParser 创建新的XML解析器
func NewParser() *XMLParser {
	return &XMLParser{
		validator: NewValidator(),
		resolver:  NewNamespaceResolver(),
	}
}

// SetValidator 设置验证器
func (p *XMLParser) SetValidator(validator *Validator) *XMLParser {
	p.validator = validator
	return p
}

// ValidateProppatchFormat 验证PROPPATCH XML格式
func (p *XMLParser) ValidateProppatchFormat(xmlBytes []byte) error {
	// 简单的XML格式检查
	if len(xmlBytes) == 0 {
		return fmt.Errorf("XML内容不能为空")
	}
	if !bytes.HasPrefix(xmlBytes, []byte("<?xml")) && !bytes.HasPrefix(xmlBytes, []byte("<")) {
		return fmt.Errorf("无效的XML格式")
	}
	return nil
}

// ValidateOperation 验证操作
func (p *XMLParser) ValidateOperation(operation string, prop interface{}) *types.PropertyError {
	// 简单的验证实现
	return nil // 没有错误
}

// SetResolver 设置命名空间解析器
func (p *XMLParser) SetResolver(resolver *NamespaceResolver) *XMLParser {
	p.resolver = resolver
	return p
}

// ReadBody 读取并验证请求体
func (p *XMLParser) ReadBody(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读取请求体失败: %v", err)
	}
	
	if len(body) == 0 {
		return nil, fmt.Errorf("请求体不能为空")
	}
	
	// 验证XML格式
	if err := p.validator.ValidateProppatchFormat(body); err != nil {
		return nil, fmt.Errorf("XML格式错误: %v", err)
	}
	
	return body, nil
}

// ParsePropertyFromContent 从PropContent解析Property结构
func (p *XMLParser) ParsePropertyFromContent(userID, resourcePath string, prop types.PropContent) (*types.Property, error) {
	// 解析命名空间
	namespace := p.resolver.Resolve(prop.XMLName.Space, nil)
	if namespace == "" {
		namespace = types.NamespaceCustom
	}
	
	property := types.Property{
		UserID:     userID,
		ResourceID: uuid.New().String(),
		Path:       resourcePath,
		Name:       prop.XMLName.Local,
		Namespace:  namespace,
		Value:      strings.TrimSpace(prop.Value),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	// 验证属性
	if err := p.validator.ValidateProppatchFormat([]byte(prop.Value)); err != nil {
		return nil, fmt.Errorf("属性值验证失败: %v", err)
	}
	
	return &property, nil
}

// GenerateCustomPropertyXML 生成自定义属性的XML元素
func (s *Serializer) GenerateCustomPropertyXML(namespace, name, value string) (string, error) {
	// 解析命名空间和属性名
	var nsPrefix, propName string
	if idx := strings.Index(name, ":"); idx > 0 {
		nsPrefix = name[:idx]
		propName = name[idx+1:]
	} else {
		nsPrefix = "custom"
		propName = name
	}
	
	// 创建XML元素
	escapedValue := escapeXML(value)
	xmlStr := fmt.Sprintf("<%s:%s>%s</%s:%s>", nsPrefix, propName, escapedValue, nsPrefix, propName)
	
	return xmlStr, nil
}

// ConvertToPropertyUpdateResult 转换为属性更新结果
func ConvertToPropertyUpdateResult(path string, props []types.Property, errors []types.PropertyError) *types.PropertyUpdateResult {
	result := &types.PropertyUpdateResult{
		ResourcePath: path,
		Propstats:    make([]types.Propstat, 0),
		Operations:   make([]types.PropertyOperation, 0),
	}
	
	// 添加成功操作
	for _, prop := range props {
		propstat := types.Propstat{
			Prop: types.ResponseProp{
				DisplayName: prop.Name,
				CustomProperties: map[string]string{
					prop.Namespace + ":" + prop.Name: prop.Value,
				},
			},
			Status: "HTTP/1.1 200 OK",
		}
		result.Propstats = append(result.Propstats, propstat)
		result.Operations = append(result.Operations, types.PropertyOperation{
			Operation:   "set",
			Property:    prop,
			PropertyName: prop.Name,
			Namespace:   prop.Namespace,
			Value:       &prop.Value,
			Timestamp:  time.Now(),
		})
	}
	
	// 添加错误
	for _, err := range errors {
		propstat := types.Propstat{
			Prop: types.ResponseProp{
				DisplayName: err.Property,
			},
			Status: getHTTPStatus(err.Code),
		}
		result.Propstats = append(result.Propstats, propstat)
	}
	
	result.SuccessCount = len(props)
	result.ErrorCount = len(errors)
	
	return result
}

// getHTTPStatus 根据错误码获取HTTP状态
func getHTTPStatus(statusCode int) string {
	switch statusCode {
	case 200:
		return "HTTP/1.1 200 OK"
	case 403:
		return "HTTP/1.1 403 Forbidden"
	case 404:
		return "HTTP/1.1 404 Not Found"
	case 409:
		return "HTTP/1.1 409 Conflict"
	case 412:
		return "HTTP/1.1 412 Precondition Failed"
	case 423:
		return "HTTP/1.1 423 Locked"
	case 424:
		return "HTTP/1.1 424 Failed Dependency"
	case 500:
		return "HTTP/1.1 500 Internal Server Error"
	case 507:
		return "HTTP/1.1 507 Insufficient Storage"
	default:
		return fmt.Sprintf("HTTP/1.1 %d", statusCode)
	}
}

// escapeXML 转义XML特殊字符
func escapeXML(s string) string {
	// 使用标准库的编码/转义功能
	result := make([]byte, 0, len(s)*2)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '<':
			result = append(result, []byte("&lt;")...)
		case '>':
			result = append(result, []byte("&gt;")...)
		case '&':
			result = append(result, []byte("&amp;")...)
		case '"':
			result = append(result, []byte("&quot;")...)
		case '\'':
			result = append(result, []byte("&apos;")...)
		default:
			result = append(result, c)
		}
	}
	return string(result)
}