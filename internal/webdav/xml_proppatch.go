package webdav

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/webdav-gateway/internal/webdav/validators"
	"github.com/webdav-gateway/internal/webdav/xml"
)

// ========================================
// PROPPATCH XML解析器 (简化版)
// ========================================

// ProppatchXMLParser PROPPATCH专用XML解析器
type ProppatchXMLParser struct {
	xmlParser    *xml.XMLParser
	validator    *validators.PropertyOperationValidator
	serializer   *xml.Serializer
}

// NewProppatchXMLParser 创建新的PROPPATCH XML解析器
func NewProppatchXMLParser() *ProppatchXMLParser {
	return &ProppatchXMLParser{
		xmlParser:  xml.NewParser(),
		validator:  validators.NewPropertyOperationValidator(nil),
		serializer: xml.NewSerializer(),
	}
}

// ParseRequest 解析PROPPATCH请求
func (p *ProppatchXMLParser) ParseRequest(ctx context.Context, body []byte) (*PropertyUpdateRequest, *PropertyError) {
	// 解析XML
	request, err := p.serializer.DecodeProppatchRequest(body)
	if err != nil {
		return nil, &PropertyError{
			Code:        400,
			Message:     "无效的XML格式",
			Description: err.Error(),
		}
	}

	// 验证请求结构
	if len(request.SetOperations) == 0 && len(request.RemoveOperations) == 0 {
		return nil, &PropertyError{
			Code:    400,
			Message: "PROPPATCH请求必须包含set或remove操作",
		}
	}

	// 验证操作内容
	if err := p.validateOperations(request); err != nil {
		return nil, err
	}

	return request, nil
}

// validateOperations 验证操作内容
func (p *ProppatchXMLParser) validateOperations(req *PropertyUpdateRequest) *PropertyError {
	// 验证set操作
	for i, setOp := range req.SetOperations {
		if err := p.validateOperation("set", setOp.PropContent[0], fmt.Sprintf("set[%d]", i)); err != nil {
			return err
		}
	}

	// 验证remove操作
	for i, removeOp := range req.RemoveOperations {
		if err := p.validateOperation("remove", removeOp.PropContent[0], fmt.Sprintf("remove[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

// validateOperation 验证单个操作
func (p *ProppatchXMLParser) validateOperation(operation string, prop PropContent, context string) *PropertyError {
	// 基本验证
	if prop.XMLName.Local == "" {
		return &PropertyError{
			Code:    400,
			Message: fmt.Sprintf("%s: 属性名不能为空", context),
		}
	}

	// 业务逻辑验证
	return p.xmlParser.ValidateOperation(operation, prop)
}

// ParsePropertyFromContent 从PropContent解析Property结构
func (p *ProppatchXMLParser) ParsePropertyFromContent(userID, resourcePath string, prop PropContent) (*Property, *PropertyError) {
	property, err := p.xmlParser.ParsePropertyFromContent(userID, resourcePath, prop)
	if err != nil {
		return nil, &PropertyError{
			Code:        400,
			Message:     "属性解析失败",
			Description: err.Error(),
		}
	}

	// 应用验证器
	if err := p.xmlParser.ValidateOperation("set", prop); err != nil {
		return nil, &PropertyError{
			Code:        400,
			Message:     "属性验证失败",
			Description: err.Error(),
		}
	} else {
		property.IsLive = (normalized != nil)
	}

	return property, nil
}

// ReadXMLBody 从HTTP请求体中读取XML数据
func (p *ProppatchXMLParser) ReadXMLBody(r io.Reader) ([]byte, *PropertyError) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, &PropertyError{
			Code:        400,
			Message:     "读取请求体失败",
			Description: err.Error(),
		}
	}

	if len(body) == 0 {
		return nil, &PropertyError{
			Code:    400,
			Message: "请求体不能为空",
		}
	}

	// 验证XML格式
	if err := p.xmlParser.ValidateProppatchFormat(body); err != nil {
		return nil, &PropertyError{
			Code:        400,
			Message:     "XML格式错误",
			Description: err.Error(),
		}
	}

	return body, nil
}

// ParseProppatchRequest 解析PROPPATCH请求XML
func (p *ProppatchXMLParser) ParseProppatchRequest(xmlBody []byte) (*PropertyUpdateRequest, *PropertyError) {
	// 解析XML
	request, err := p.serializer.DecodeProppatchRequest(xmlBody)
	if err != nil {
		return nil, &PropertyError{
			Code:        400,
			Message:     "无效的XML格式",
			Description: err.Error(),
		}
	}

	// 验证请求结构
	if len(request.SetOperations) == 0 && len(request.RemoveOperations) == 0 {
		return nil, &PropertyError{
			Code:    400,
			Message: "PROPPATCH请求必须包含set或remove操作",
		}
	}

	// 转换格式以匹配PropertyUpdateRequest
	propertyRequest := &PropertyUpdateRequest{
		SetOperations:    make([]SetOperation, len(request.SetOperations)),
		RemoveOperations: make([]RemoveOperation, len(request.RemoveOperations)),
	}

	// 转换set操作
	for i, setOp := range request.SetOperations {
		propertyRequest.SetOperations[i] = SetOperation{
			PropContent: setOp.PropContent[0]Content,
		}
	}

	// 转换remove操作
	for i, removeOp := range request.RemoveOperations {
		propertyRequest.RemoveOperations[i] = RemoveOperation{
			PropContent: removeOp.PropContent[0]Content,
		}
	}

	// 验证操作内容
	if err := p.validateOperations(propertyRequest); err != nil {
		return nil, err
	}

	return propertyRequest, nil
}

// resolveNamespace 解析命名空间前缀
func (p *ProppatchXMLParser) resolveNamespace(prop PropContent) string {
	// 使用内置的ResolveNamespace函数
	return ResolveNamespace(prop.XMLName.Space, prop.XMLName.Space)
}

// GenerateProppatchResponse 生成PROPPATCH响应XML
func (p *ProppatchXMLParser) GenerateProppatchResponse(result *PropertyUpdateResult) ([]byte, *PropertyError) {
	// 构建响应
	responses := make([]ProppatchResponse, 0)
	
	for _, propstat := range result.Propstats {
		response := ProppatchResponse{
			Status:        propstat.Status,
			PropertyNames: []string{propstat.Prop.DisplayName},
		}
		responses = append(responses, response)
	}

	// 编码为XML
	responseXML, err := p.serializer.EncodeMultiStatusResponse(responses)
	if err != nil {
		return nil, &PropertyError{
			Code:        500,
			Message:     "生成响应失败",
			Description: err.Error(),
		}
	}

	return responseXML, nil
}

// GenerateErrorResponse 生成错误响应XML
func (p *ProppatchXMLParser) GenerateErrorResponse(statusCode int, errors []PropertyError) ([]byte, *PropertyError) {
	// 构建错误响应
	responses := make([]ProppatchResponse, 0)
	
	for _, err := range errors {
		response := ProppatchResponse{
			Status:        getHTTPStatus(err.Code),
			PropertyNames: []string{err.Property},
		}
		responses = append(responses, response)
	}

	// 编码为XML
	responseXML, err := p.serializer.EncodeMultiStatusResponse(responses)
	if err != nil {
		return nil, &PropertyError{
			Code:        500,
			Message:     "生成错误响应失败",
			Description: err.Error(),
		}
	}

	return responseXML, nil
}

// ========================================
// PROPPATCH 响应生成器
// ========================================

// ProppatchResponseBuilder PROPPATCH响应构建器
type ProppatchResponseBuilder struct {
	serializer *xml.Serializer
}

// NewProppatchResponseBuilder 创建新的响应构建器
func NewProppatchResponseBuilder() *ProppatchResponseBuilder {
	return &ProppatchResponseBuilder{
		serializer: xml.NewSerializer(),
	}
}

// BuildSuccessResponse 构建成功响应
func (b *ProppatchResponseBuilder) BuildSuccessResponse(path string, properties []Property) (*ProppatchResponse, error) {
	propstats := make([]Propstat, 0, len(properties))
	
	for _, prop := range properties {
		propContent := PropContentResponse{
			DisplayName: prop.Name,
			CustomProps: map[string]string{
				prop.Namespace + ":" + prop.Name: prop.Value,
			},
		}
		
		propstats = append(propstats, Propstat{
			Prop:   propContent,
			Status: "HTTP/1.1 200 OK",
		})
	}
	
	return &ProppatchResponse{
		Xmlns:    "DAV:",
		Href:     path,
		Propstats: propstats,
	}, nil
}

// BuildErrorResponse 构建错误响应
func (b *ProppatchResponseBuilder) BuildErrorResponse(path string, errors []PropertyError) (*ProppatchResponse, error) {
	propstats := make([]Propstat, 0, len(errors))
	
	for _, err := range errors {
		status := getHTTPStatus(err.Code)
		
		propContent := PropContentResponse{
			DisplayName: err.Property,
		}
		
		propstats = append(propstats, Propstat{
			Prop:   propContent,
			Status: status,
		})
	}
	
	return &ProppatchResponse{
		Xmlns:    "DAV:",
		Href:     path,
		Propstats: propstats,
	}, nil
}

// BuildMultiStatusResponse 构建Multi-Status响应
func (b *ProppatchResponseBuilder) BuildMultiStatusResponse(results []*PropertyUpdateResult) ([]byte, error) {
	responses := make([]ProppatchResponse, 0, len(results))
	
	for _, result := range results {
		response := ProppatchResponse{
			Xmlns:     "DAV:",
			Href:      result.ResourcePath,
			Propstats: result.Propstats,
		}
		responses = append(responses, response)
	}
	
	return b.serializer.EncodeMultiStatusResponse(responses)
}

// BuildPropertyUpdateResult 构建属性更新结果
func (b *ProppatchResponseBuilder) BuildPropertyUpdateResult(path string, props []Property, errors []PropertyError) *PropertyUpdateResult {
	result := &PropertyUpdateResult{
		ResourcePath: path,
		Propstats:    make([]Propstat, 0),
		Operations:   make([]PropertyOperation, 0),
	}
	
	// 添加成功操作
	for _, prop := range props {
		propstat := Propstat{
			Prop: PropContentResponse{
				DisplayName: prop.Name,
				CustomProps: map[string]string{
					prop.Namespace + ":" + prop.Name: prop.Value,
				},
			},
			Status: "HTTP/1.1 200 OK",
		}
		result.Propstats = append(result.Propstats, propstat)
		result.Operations = append(result.Operations, PropertyOperation{
			Operation:  "set",
			Property:   prop.Name,
			Namespace:  prop.Namespace,
			Value:      &prop.Value,
			Timestamp:  time.Now(),
		})
	}
	
	// 添加错误
	for _, err := range errors {
		propstat := Propstat{
			Prop: PropContentResponse{
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

// ========================================
// 工具函数 (保留必要的)
// ========================================

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

// ExtractCustomPropertiesFromResponse 从Response中提取自定义属性
func ExtractCustomPropertiesFromResponse(response Response) map[string]string {
	customProps := make(map[string]string)
	
	// 遍历propstat中的prop元素
	for _, propstat := range response.Propstat {
		// 解析自定义属性
		for key, value := range propstat.Prop.CustomProperties {
			customProps[key] = value
		}
	}
	
	return customProps
}

// GenerateCustomPropertyXML 生成自定义属性的XML元素
func GenerateCustomPropertyXML(namespace, name, value string) (string, error) {
	return xml.NewSerializer().GenerateCustomPropertyXML(namespace, name, value)
}

// IsLiveProperty 检查属性是否为活属性
func IsLiveProperty(name, namespace string) bool {
	if namespace != NamespaceDAV {
		return false
	}
	return KnownLiveProperties[name]
}