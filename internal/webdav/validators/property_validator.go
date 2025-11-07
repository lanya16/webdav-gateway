package validators

import (
	"strings"
	"time"

	"github.com/webdav-gateway/internal/types"
)

// PropertyValidator 属性验证器接口
type PropertyValidator interface {
	Validate(property types.Property) error
	Normalize(property types.Property) (types.Property, error)
}

// DefaultPropertyValidator 默认属性验证器
type DefaultPropertyValidator struct{}

// Validate 验证属性
func (v *DefaultPropertyValidator) Validate(property types.Property) error {
	if strings.TrimSpace(property.Name) == "" {
		return &types.PropertyError{
			Code:    409,
			Message: "属性名不能为空",
		}
	}
	
	if strings.TrimSpace(property.Namespace) == "" {
		return &types.PropertyError{
			Code:    409,
			Message: "命名空间不能为空",
		}
	}
	
	if len(property.Value) > 10240 { // 10KB 限制
		return &types.PropertyError{
			Code:    413,
			Message: "属性值过大",
		}
	}
	
	return nil
}

// Normalize 规范化属性
func (v *DefaultPropertyValidator) Normalize(property types.Property) (types.Property, error) {
	normalized := property
	
	// 清理空白字符
	normalized.Value = strings.TrimSpace(property.Value)
	
	// 设置默认命名空间
	if normalized.Namespace == "" {
		normalized.Namespace = types.NamespaceCustom
	}
	
	// 设置活属性标志
	normalized.IsLive = isKnownLiveProperty(normalized.Name, normalized.Namespace)
	
	return normalized, nil
}

// isKnownLiveProperty 检查是否为已知活属性
func isKnownLiveProperty(name, namespace string) bool {
	if namespace != types.NamespaceDAV {
		return false
	}
	return types.KnownLiveProperties[name]
}

// ValidationRule 验证规则接口
type ValidationRule interface {
	Validate(prop types.PropContent) error
	GetRuleName() string
}

// StringLengthRule 字符串长度验证规则
type StringLengthRule struct {
	MaxLength int
	RuleName  string
}

func (r *StringLengthRule) Validate(prop types.PropContent) error {
	if len(prop.Value) > r.MaxLength {
		return &types.PropertyError{
			Code:    413,
			Message: "属性值超过最大长度限制",
		}
	}
	return nil
}

func (r *StringLengthRule) GetRuleName() string {
	return r.RuleName
}

// RequiredFieldRule 必填字段验证规则
type RequiredFieldRule struct {
	FieldName string
	RuleName  string
}

func (r *RequiredFieldRule) Validate(prop types.PropContent) error {
	if strings.TrimSpace(prop.XMLName.Local) == "" {
		return &types.PropertyError{
			Code:    400,
			Message: "缺少必需字段",
		}
	}
	return nil
}

func (r *RequiredFieldRule) GetRuleName() string {
	return r.RuleName
}

// CompositeValidator 复合验证器
type CompositeValidator struct {
	rules []ValidationRule
}

func NewCompositeValidator(rules ...ValidationRule) *CompositeValidator {
	return &CompositeValidator{rules: rules}
}

func (cv *CompositeValidator) AddRule(rule ValidationRule) {
	cv.rules = append(cv.rules, rule)
}

func (cv *CompositeValidator) Validate(prop types.PropContent) error {
	for _, rule := range cv.rules {
		if err := rule.Validate(prop); err != nil {
			return err
		}
	}
	return nil
}

// PropertyValidationContext 验证上下文
type PropertyValidationContext struct {
	ResourcePath string
	UserID       string
	Time         time.Time
}

type ValidationContextProvider interface {
	GetContext() PropertyValidationContext
}

// PropertyOperationValidator 属性操作验证器
type PropertyOperationValidator struct {
	rules    []ValidationRule
	provider ValidationContextProvider
}

func NewPropertyOperationValidator(provider ValidationContextProvider, rules ...ValidationRule) *PropertyOperationValidator {
	return &PropertyOperationValidator{
		rules:    rules,
		provider: provider,
	}
}

func (v *PropertyOperationValidator) ValidateOperation(operation string, prop types.PropContent) error {
	for _, rule := range v.rules {
		if err := rule.Validate(prop); err != nil {
			return err
		}
	}
	
	// 业务逻辑验证
	switch operation {
	case "set":
		return v.validateSetOperation(prop)
	case "remove":
		return v.validateRemoveOperation(prop)
	default:
		return &types.PropertyError{
			Code:    400,
			Message: "不支持的操作类型",
		}
	}
}

func (v *PropertyOperationValidator) validateSetOperation(prop types.PropContent) error {
	// 防止设置活属性（这里可以根据业务需求调整）
	if isKnownLiveProperty(prop.XMLName.Local, prop.XMLName.Space) {
		return &types.PropertyError{
			Code:    403,
			Message: "不能直接设置活属性",
		}
	}
	return nil
}

func (v *PropertyOperationValidator) validateRemoveOperation(prop types.PropContent) error {
	// 防止移除核心DAV属性
	if prop.XMLName.Space == "DAV" {
		return &types.PropertyError{
			Code:    403,
			Message: "不能移除核心DAV属性",
		}
	}
	return nil
}

// NewDefaultValidator 创建默认验证器实例
func NewDefaultValidator() PropertyValidator {
	return &DefaultPropertyValidator{}
}