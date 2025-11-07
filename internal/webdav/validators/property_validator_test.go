package validators

import (
	"strings"
	"testing"
	"time"

	"github.com/webdav-gateway/internal/types"
)

// ========================================
// DefaultPropertyValidator 测试
// ========================================

func TestDefaultPropertyValidator_Validate(t *testing.T) {
	tests := []struct {
		name          string
		property      types.Property
		expectedError string
	}{
		{
			name: "有效属性",
			property: types.Property{
				Name:      "custom-property",
				Namespace: types.NamespaceCustom,
				Value:     "test value",
			},
			expectedError: "",
		},
		{
			name: "空属性名",
			property: types.Property{
				Name:      "",
				Namespace: types.NamespaceCustom,
				Value:     "test value",
			},
			expectedError: "属性名不能为空",
		},
		{
			name: "仅包含空格的属性名",
			property: types.Property{
				Name:      "   ",
				Namespace: types.NamespaceCustom,
				Value:     "test value",
			},
			expectedError: "属性名不能为空",
		},
		{
			name: "空命名空间",
			property: types.Property{
				Name:      "custom-property",
				Namespace: "",
				Value:     "test value",
			},
			expectedError: "命名空间不能为空",
		},
		{
			name: "仅包含空格的命名空间",
			property: types.Property{
				Name:      "custom-property",
				Namespace: "   ",
				Value:     "test value",
			},
			expectedError: "命名空间不能为空",
		},
		{
			name: "属性值超过10KB限制",
			property: types.Property{
				Name:      "custom-property",
				Namespace: types.NamespaceCustom,
				Value:     strings.Repeat("a", 10241), // 10KB + 1 byte
			},
			expectedError: "属性值过大",
		},
		{
			name: "属性值恰好10KB",
			property: types.Property{
				Name:      "custom-property",
				Namespace: types.NamespaceCustom,
				Value:     strings.Repeat("a", 10240), // 10KB exact
			},
			expectedError: "",
		},
		{
			name: "DAV命名空间的有效属性",
			property: types.Property{
				Name:      "displayname",
				Namespace: types.NamespaceDAV,
				Value:     "My Resource",
			},
			expectedError: "",
		},
		{
			name: "用户命名空间的有效属性",
			property: types.Property{
				Name:      "author",
				Namespace: types.NamespaceUser,
				Value:     "John Doe",
			},
			expectedError: "",
		},
	}

	validator := &DefaultPropertyValidator{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.property)
			
			if tt.expectedError == "" {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error: %s, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %s, got: %v", tt.expectedError, err)
				}
			}
		})
	}
}

func TestDefaultPropertyValidator_Normalize(t *testing.T) {
	tests := []struct {
		name               string
		inputProperty      types.Property
		expectedProperty   types.Property
		expectedError      string
	}{
		{
			name: "带空格的属性值应该被清理",
			inputProperty: types.Property{
				Name:      "custom-property",
				Namespace: types.NamespaceCustom,
				Value:     "  test value  ",
			},
			expectedProperty: types.Property{
				Name:      "custom-property",
				Namespace: types.NamespaceCustom,
				Value:     "test value",
				IsLive:    false,
			},
			expectedError: "",
		},
		{
			name: "空命名空间应该使用默认命名空间",
			inputProperty: types.Property{
				Name:      "custom-property",
				Namespace: "",
				Value:     "test value",
			},
			expectedProperty: types.Property{
				Name:      "custom-property",
				Namespace: types.NamespaceCustom,
				Value:     "test value",
				IsLive:    false,
			},
			expectedError: "",
		},
		{
			name: "DAV命名空间的displayname应该被标记为活属性",
			inputProperty: types.Property{
				Name:      "displayname",
				Namespace: types.NamespaceDAV,
				Value:     "My Resource",
			},
			expectedProperty: types.Property{
				Name:      "displayname",
				Namespace: types.NamespaceDAV,
				Value:     "My Resource",
				IsLive:    true,
			},
			expectedError: "",
		},
		{
			name: "非DAV命名空间的displayname应该被标记为死属性",
			inputProperty: types.Property{
				Name:      "displayname",
				Namespace: types.NamespaceCustom,
				Value:     "My Resource",
			},
			expectedProperty: types.Property{
				Name:      "displayname",
				Namespace: types.NamespaceCustom,
				Value:     "My Resource",
				IsLive:    false,
			},
			expectedError: "",
		},
		{
			name: "getcontentlength应该被标记为活属性",
			inputProperty: types.Property{
				Name:      "getcontentlength",
				Namespace: types.NamespaceDAV,
				Value:     "1024",
			},
			expectedProperty: types.Property{
				Name:      "getcontentlength",
				Namespace: types.NamespaceDAV,
				Value:     "1024",
				IsLive:    true,
			},
			expectedError: "",
		},
		{
			name: "自定义属性应该被标记为死属性",
			inputProperty: types.Property{
				Name:      "custom-metadata",
				Namespace: types.NamespaceCustom,
				Value:     "custom value",
			},
			expectedProperty: types.Property{
				Name:      "custom-metadata",
				Namespace: types.NamespaceCustom,
				Value:     "custom value",
				IsLive:    false,
			},
			expectedError: "",
		},
		{
			name: "包含换行符的属性值应该被正确清理",
			inputProperty: types.Property{
				Name:      "description",
				Namespace: types.NamespaceCustom,
				Value:     "\n  line1  \n  line2  \n",
			},
			expectedProperty: types.Property{
				Name:      "description",
				Namespace: types.NamespaceCustom,
				Value:     "line1  \n  line2",
				IsLive:    false,
			},
			expectedError: "",
		},
	}

	validator := &DefaultPropertyValidator{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Normalize(tt.inputProperty)
			
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error: %s, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %s, got: %v", tt.expectedError, err)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
				return
			}
			
			if result.Name != tt.expectedProperty.Name {
				t.Errorf("Expected Name %s, got %s", tt.expectedProperty.Name, result.Name)
			}
			if result.Namespace != tt.expectedProperty.Namespace {
				t.Errorf("Expected Namespace %s, got %s", tt.expectedProperty.Namespace, result.Namespace)
			}
			if result.Value != tt.expectedProperty.Value {
				t.Errorf("Expected Value %s, got %s", tt.expectedProperty.Value, result.Value)
			}
			if result.IsLive != tt.expectedProperty.IsLive {
				t.Errorf("Expected IsLive %v, got %v", tt.expectedProperty.IsLive, result.IsLive)
			}
		})
	}
}

// ========================================
// StringLengthRule 测试
// ========================================

func TestStringLengthRule_Validate(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		maxLength     int
		expectedError string
	}{
		{
			name:          "值在长度限制内",
			value:         "short value",
			maxLength:     100,
			expectedError: "",
		},
		{
			name:          "值等于长度限制",
			value:         strings.Repeat("a", 50),
			maxLength:     50,
			expectedError: "",
		},
		{
			name:          "值超过长度限制",
			value:         strings.Repeat("a", 51),
			maxLength:     50,
			expectedError: "属性值超过最大长度限制",
		},
		{
			name:          "空值",
			value:         "",
			maxLength:     50,
			expectedError: "",
		},
		{
			name:          "包含换行符的长值",
			value:         "line1\n" + strings.Repeat("a", 51),
			maxLength:     50,
			expectedError: "属性值超过最大长度限制",
		},
		{
			name:          "包含中文的长值",
			value:         strings.Repeat("中", 26), // 3字节 * 26 > 50字节
			maxLength:     50,
			expectedError: "属性值超过最大长度限制",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &StringLengthRule{
				MaxLength: tt.maxLength,
				RuleName:  "string-length-rule",
			}
			
			prop := types.PropContent{
				Value: tt.value,
			}
			
			err := rule.Validate(prop)
			
			if tt.expectedError == "" {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error: %s, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %s, got: %v", tt.expectedError, err)
				}
			}
		})
	}
}

func TestStringLengthRule_GetRuleName(t *testing.T) {
	rule := &StringLengthRule{
		MaxLength: 100,
		RuleName:  "test-string-rule",
	}
	
	expected := "test-string-rule"
	result := rule.GetRuleName()
	
	if result != expected {
		t.Errorf("Expected rule name %s, got %s", expected, result)
	}
}

// ========================================
// RequiredFieldRule 测试
// ========================================

func TestRequiredFieldRule_Validate(t *testing.T) {
	tests := []struct {
		name          string
		localName     string
		expectedError string
	}{
		{
			name:          "有效的本地名称",
			localName:     "custom-property",
			expectedError: "",
		},
		{
			name:          "空本地名称",
			localName:     "",
			expectedError: "缺少必需字段",
		},
		{
			name:          "仅包含空格的本地名称",
			localName:     "   ",
			expectedError: "缺少必需字段",
		},
		{
			name:          "标准DAV属性",
			localName:     "displayname",
			expectedError: "",
		},
		{
			name:          "包含特殊字符的本地名称",
			localName:     "my:custom:property",
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &RequiredFieldRule{
				FieldName: "XMLName.Local",
				RuleName:  "required-field-rule",
			}
			
			prop := types.PropContent{
				XMLName: struct {
					Space string
					Local string
				}{
					Space: types.NamespaceCustom,
					Local: tt.localName,
				},
			}
			
			err := rule.Validate(prop)
			
			if tt.expectedError == "" {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error: %s, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %s, got: %v", tt.expectedError, err)
				}
			}
		})
	}
}

func TestRequiredFieldRule_GetRuleName(t *testing.T) {
	rule := &RequiredFieldRule{
		FieldName: "test-field",
		RuleName:  "test-required-rule",
	}
	
	expected := "test-required-rule"
	result := rule.GetRuleName()
	
	if result != expected {
		t.Errorf("Expected rule name %s, got %s", expected, result)
	}
}

// ========================================
// CompositeValidator 测试
// ========================================

func TestCompositeValidator(t *testing.T) {
	t.Run("单个规则验证成功", func(t *testing.T) {
		rule := &StringLengthRule{
			MaxLength: 100,
			RuleName:  "length-rule",
		}
		
		validator := NewCompositeValidator(rule)
		
		prop := types.PropContent{
			Value: "short value",
		}
		
		err := validator.Validate(prop)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
	
	t.Run("单个规则验证失败", func(t *testing.T) {
		rule := &StringLengthRule{
			MaxLength: 5,
			RuleName:  "length-rule",
		}
		
		validator := NewCompositeValidator(rule)
		
		prop := types.PropContent{
			Value: strings.Repeat("a", 10),
		}
		
		err := validator.Validate(prop)
		if err == nil {
			t.Errorf("Expected error, got nil")
		} else if !strings.Contains(err.Error(), "属性值超过最大长度限制") {
			t.Errorf("Expected length error, got: %v", err)
		}
	})
	
	t.Run("多个规则全部通过", func(t *testing.T) {
		rules := []ValidationRule{
			&StringLengthRule{MaxLength: 100, RuleName: "length-rule"},
			&RequiredFieldRule{FieldName: "XMLName.Local", RuleName: "required-field"},
		}
		
		validator := NewCompositeValidator(rules...)
		
		prop := types.PropContent{
			Value: "short value",
			XMLName: struct {
				Space string
				Local string
			}{
				Space: types.NamespaceCustom,
				Local: "custom-property",
			},
		}
		
		err := validator.Validate(prop)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
	
	t.Run("第一个规则失败", func(t *testing.T) {
		rules := []ValidationRule{
			&StringLengthRule{MaxLength: 5, RuleName: "length-rule"},
			&RequiredFieldRule{FieldName: "XMLName.Local", RuleName: "required-field"},
		}
		
		validator := NewCompositeValidator(rules...)
		
		prop := types.PropContent{
			Value: strings.Repeat("a", 10),
			XMLName: struct {
				Space string
				Local string
			}{
				Space: types.NamespaceCustom,
				Local: "custom-property",
			},
		}
		
		err := validator.Validate(prop)
		if err == nil {
			t.Errorf("Expected error, got nil")
		} else if !strings.Contains(err.Error(), "属性值超过最大长度限制") {
			t.Errorf("Expected length error, got: %v", err)
		}
	})
	
	t.Run("第二个规则失败", func(t *testing.T) {
		rules := []ValidationRule{
			&StringLengthRule{MaxLength: 100, RuleName: "length-rule"},
			&RequiredFieldRule{FieldName: "XMLName.Local", RuleName: "required-field"},
		}
		
		validator := NewCompositeValidator(rules...)
		
		prop := types.PropContent{
			Value: "short value",
			XMLName: struct {
				Space string
				Local string
			}{
				Space: types.NamespaceCustom,
				Local: "",
			},
		}
		
		err := validator.Validate(prop)
		if err == nil {
			t.Errorf("Expected error, got nil")
		} else if !strings.Contains(err.Error(), "缺少必需字段") {
			t.Errorf("Expected required field error, got: %v", err)
		}
	})
	
	t.Run("添加新规则", func(t *testing.T) {
		validator := NewCompositeValidator()
		
		rule1 := &StringLengthRule{MaxLength: 100, RuleName: "length-rule"}
		validator.AddRule(rule1)
		
		rule2 := &RequiredFieldRule{FieldName: "XMLName.Local", RuleName: "required-field"}
		validator.AddRule(rule2)
		
		prop := types.PropContent{
			Value: "short value",
			XMLName: struct {
				Space string
				Local string
			}{
				Space: types.NamespaceCustom,
				Local: "custom-property",
			},
		}
		
		err := validator.Validate(prop)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

// ========================================
// PropertyOperationValidator 测试
// ========================================

type mockContextProvider struct{}

func (m *mockContextProvider) GetContext() PropertyValidationContext {
	return PropertyValidationContext{
		ResourcePath: "/test/path",
		UserID:       "test-user",
		Time:         time.Now(),
	}
}

func TestPropertyOperationValidator_ValidateOperation(t *testing.T) {
	t.Run("有效的set操作", func(t *testing.T) {
		provider := &mockContextProvider{}
		validator := NewPropertyOperationValidator(provider)
		
		prop := types.PropContent{
			XMLName: struct {
				Space string
				Local string
			}{
				Space: types.NamespaceCustom,
				Local: "custom-property",
			},
			Value: "custom value",
		}
		
		err := validator.ValidateOperation("set", prop)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
	
	t.Run("尝试设置活属性应该失败", func(t *testing.T) {
		provider := &mockContextProvider{}
		validator := NewPropertyOperationValidator(provider)
		
		prop := types.PropContent{
			XMLName: struct {
				Space string
				Local string
			}{
				Space: types.NamespaceDAV,
				Local: "displayname", // 活属性
			},
			Value: "My Resource",
		}
		
		err := validator.ValidateOperation("set", prop)
		if err == nil {
			t.Errorf("Expected error when setting live property, got nil")
		} else if !strings.Contains(err.Error(), "不能直接设置活属性") {
			t.Errorf("Expected live property error, got: %v", err)
		}
	})
	
	t.Run("有效的remove操作", func(t *testing.T) {
		provider := &mockContextProvider{}
		validator := NewPropertyOperationValidator(provider)
		
		prop := types.PropContent{
			XMLName: struct {
				Space string
				Local string
			}{
				Space: types.NamespaceCustom,
				Local: "custom-property",
			},
			Value: "",
		}
		
		err := validator.ValidateOperation("remove", prop)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
	
	t.Run("尝试移除核心DAV属性应该失败", func(t *testing.T) {
		provider := &mockContextProvider{}
		validator := NewPropertyOperationValidator(provider)
		
		prop := types.PropContent{
			XMLName: struct {
				Space string
				Local string
			}{
				Space: types.NamespaceDAV,
				Local: "displayname",
			},
			Value: "",
		}
		
		err := validator.ValidateOperation("remove", prop)
		if err == nil {
			t.Errorf("Expected error when removing DAV property, got nil")
		} else if !strings.Contains(err.Error(), "不能移除核心DAV属性") {
			t.Errorf("Expected DAV property removal error, got: %v", err)
		}
	})
	
	t.Run("不支持的操作类型", func(t *testing.T) {
		provider := &mockContextProvider{}
		validator := NewPropertyOperationValidator(provider)
		
		prop := types.PropContent{
			XMLName: struct {
				Space string
				Local string
			}{
				Space: types.NamespaceCustom,
				Local: "custom-property",
			},
			Value: "value",
		}
		
		err := validator.ValidateOperation("unsupported-operation", prop)
		if err == nil {
			t.Errorf("Expected error for unsupported operation, got nil")
		} else if !strings.Contains(err.Error(), "不支持的操作类型") {
			t.Errorf("Expected unsupported operation error, got: %v", err)
		}
	})
	
	t.Run("带验证规则的set操作", func(t *testing.T) {
		provider := &mockContextProvider{}
		rule := &StringLengthRule{MaxLength: 50, RuleName: "length-rule"}
		validator := NewPropertyOperationValidator(provider, rule)
		
		prop := types.PropContent{
			XMLName: struct {
				Space string
				Local string
			}{
				Space: types.NamespaceCustom,
				Local: "custom-property",
			},
			Value: strings.Repeat("a", 51), // 超过长度限制
		}
		
		err := validator.ValidateOperation("set", prop)
		if err == nil {
			t.Errorf("Expected error for property too long, got nil")
		} else if !strings.Contains(err.Error(), "属性值超过最大长度限制") {
			t.Errorf("Expected length error, got: %v", err)
		}
	})
}

// ========================================
// NewDefaultValidator 测试
// ========================================

func TestNewDefaultValidator(t *testing.T) {
	validator := NewDefaultValidator()
	
	if validator == nil {
		t.Errorf("Expected validator instance, got nil")
	}
	
	// 检查返回的实例类型
	switch v := validator.(type) {
	case *DefaultPropertyValidator:
		// 正确类型
	default:
		t.Errorf("Expected *DefaultPropertyValidator type, got %T", v)
	}
}

// ========================================
// 边界条件测试
// ========================================

func TestPropertyValidator_EdgeCases(t *testing.T) {
	validator := &DefaultPropertyValidator{}
	
	t.Run("极长的属性名", func(t *testing.T) {
		longName := strings.Repeat("a", 1000)
		prop := types.Property{
			Name:      longName,
			Namespace: types.NamespaceCustom,
			Value:     "test",
		}
		
		err := validator.Validate(prop)
		if err != nil {
			t.Errorf("Expected no error for long property name, got: %v", err)
		}
	})
	
	t.Run("极长的命名空间", func(t *testing.T) {
		longNamespace := strings.Repeat("http://example.com/", 100)
		prop := types.Property{
			Name:      "test",
			Namespace: longNamespace,
			Value:     "test",
		}
		
		err := validator.Validate(prop)
		if err != nil {
			t.Errorf("Expected no error for long namespace, got: %v", err)
		}
	})
	
	t.Run("零长度属性值", func(t *testing.T) {
		prop := types.Property{
			Name:      "test",
			Namespace: types.NamespaceCustom,
			Value:     "",
		}
		
		err := validator.Validate(prop)
		if err != nil {
			t.Errorf("Expected no error for empty value, got: %v", err)
		}
	})
	
	t.Run("Unicode字符", func(t *testing.T) {
		prop := types.Property{
			Name:      "测试属性",
			Namespace: types.NamespaceCustom,
			Value:     "测试值",
		}
		
		err := validator.Validate(prop)
		if err != nil {
			t.Errorf("Expected no error for unicode characters, got: %v", err)
		}
		
		// 测试规范化
		normalized, err := validator.Normalize(prop)
		if err != nil {
			t.Errorf("Expected no error during normalization, got: %v", err)
		}
		if normalized.Value != "测试值" {
			t.Errorf("Expected unicode value preserved, got: %s", normalized.Value)
		}
	})
}

// ========================================
// 集成测试
// ========================================

func TestPropertyValidator_Integration(t *testing.T) {
	t.Run("完整的属性验证流程", func(t *testing.T) {
		// 创建验证器
		validator := &DefaultPropertyValidator{}
		
		// 1. 测试正常的属性
		prop := types.Property{
			Name:      "custom:metadata",
			Namespace: types.NamespaceCustom,
			Value:     "important data  ",
		}
		
		// 2. 验证属性
		err := validator.Validate(prop)
		if err != nil {
			t.Errorf("Expected no validation error, got: %v", err)
		}
		
		// 3. 规范化属性
		normalized, err := validator.Normalize(prop)
		if err != nil {
			t.Errorf("Expected no normalization error, got: %v", err)
		}
		
		// 4. 验证规范化结果
		if normalized.Name != "custom:metadata" {
			t.Errorf("Expected name preserved, got: %s", normalized.Name)
		}
		if normalized.Namespace != types.NamespaceCustom {
			t.Errorf("Expected custom namespace, got: %s", normalized.Namespace)
		}
		if normalized.Value != "important data" {
			t.Errorf("Expected value trimmed, got: %s", normalized.Value)
		}
		if normalized.IsLive != false {
			t.Errorf("Expected custom property to be dead property, got live: %v", normalized.IsLive)
		}
	})
	
	t.Run("异常情况的错误处理", func(t *testing.T) {
		validator := &DefaultPropertyValidator{}
		
		// 组合多个错误条件的属性
		prop := types.Property{
			Name:      "", // 空名称
			Namespace: "", // 空命名空间
			Value:     strings.Repeat("a", 10241), // 过长值
		}
		
		err := validator.Validate(prop)
		if err == nil {
			t.Errorf("Expected validation error, got nil")
		}
		
		// 验证错误包含第一个遇到的错误信息
		if err != nil && !strings.Contains(err.Error(), "属性名不能为空") {
			t.Errorf("Expected first error about empty name, got: %v", err)
		}
	})
}