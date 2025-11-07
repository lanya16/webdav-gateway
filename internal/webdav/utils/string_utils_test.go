package utils

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

// ==================== StringUtil Tests ====================

func TestStringUtil_TrimWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "正常字符串",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "首部空格",
			input:    "  hello",
			expected: "hello",
		},
		{
			name:     "尾部空格",
			input:    "hello  ",
			expected: "hello",
		},
		{
			name:     "首尾空格",
			input:    "  hello  ",
			expected: "hello",
		},
		{
			name:     "制表符",
			input:    "\thello\t",
			expected: "hello",
		},
		{
			name:     "换行符",
			input:    "\nhello\n",
			expected: "hello",
		},
		{
			name:     "回车符",
			input:    "\rhello\r",
			expected: "hello",
		},
		{
			name:     "混合空白字符",
			input:    " \t\n\r hello \t\n\r",
			expected: "hello",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
		{
			name:     "纯空白字符串",
			input:    "   \t\n\r   ",
			expected: "",
		},
		{
			name:     "中间空格保留",
			input:    "hello   world",
			expected: "hello   world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := String.TrimWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("TrimWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStringUtil_EscapeForXML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通字符串",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "小于号转义",
			input:    "a < b",
			expected: "a &lt; b",
		},
		{
			name:     "大于号转义",
			input:    "a > b",
			expected: "a &gt; b",
		},
		{
			name:     "与号转义",
			input:    "a & b",
			expected: "a &amp; b",
		},
		{
			name:     "双引号转义",
			input:    `say "hello"`,
			expected: "say &quot;hello&quot;",
		},
		{
			name:     "单引号转义",
			input:    "say 'hello'",
			expected: "say &apos;hello&apos;",
		},
		{
			name:     "混合特殊字符",
			input:    `< > & " '`,
			expected: "&lt; &gt; &amp; &quot; &apos;",
		},
		{
			name:     "包含空白字符",
			input:    "  <tag>  ",
			expected: "&lt;tag&gt;",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
		{
			name:     "复杂XML片段",
			input:    `<element attr="value & 'test'">Content < 10</element>`,
			expected: "&lt;element attr=&quot;value &amp; &apos;test&apos;&quot;&gt;Content &lt; 10&lt;/element&gt;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := String.EscapeForXML(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeForXML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStringUtil_Slugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "普通文本",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "包含空格",
			input:    "hello world test",
			expected: "hello-world-test",
		},
		{
			name:     "下划线替换",
			input:    "hello_world",
			expected: "hello-world",
		},
		{
			name:     "点号替换",
			input:    "file.name.txt",
			expected: "file-name-txt",
		},
		{
			name:     "斜杠替换",
			input:    "path/to/file",
			expected: "path-to-file",
		},
		{
			name:     "反斜杠替换",
			input:    "path\\to\\file",
			expected: "path-to-file",
		},
		{
			name:     "特殊字符移除",
			input:    "Hello@World#123",
			expected: "hello-world123",
		},
		{
			name:     "多个连字符清理",
			input:    "hello--world--test",
			expected: "hello-world-test",
		},
		{
			name:     "首尾连字符移除",
			input:    "-hello-world-",
			expected: "hello-world",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
		{
			name:     "纯特殊字符",
			input:    "@#$%^&*()",
			expected: "",
		},
		{
			name:     "混合复杂文本",
			input:    "Hello@World! (Test-File_v1.0).txt",
			expected: "hello-world-test-file-v10-txt",
		},
		{
			name:     "中文字符",
			input:    "你好世界",
			expected: "",
		},
		{
			name:     "数字保持",
			input:    "test123",
			expected: "test123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := String.Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ==================== PropertyUtil Tests ====================

func TestPropertyUtil_GenerateKey(t *testing.T) {
	tests := []struct {
		testName string
		namespace string
		propName string
		expected string
	}{
		{
			testName: "标准命名空间和名称",
			namespace: "DAV:",
			propName: "getcontentlength",
			expected: "DAV::getcontentlength",
		},
		{
			testName: "自定义命名空间",
			namespace: "custom",
			propName: "custom-property",
			expected: "custom:custom-property",
		},
		{
			testName: "空名称",
			namespace: "DAV:",
			propName: "",
			expected: "DAV:",
		},
		{
			name:     "空命名空间",
			namespace: "",
			propName: "property",
			expected: ":property",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			result := Property.GenerateKey(tt.namespace, tt.propName)
			if result != tt.expected {
				t.Errorf("GenerateKey(%q, %q) = %q, want %q", tt.namespace, tt.propName, result, tt.expected)
			}
		})
	}
}

func TestPropertyUtil_ParseKey(t *testing.T) {
	tests := []struct {
		name            string
		key             string
		expectedNamespace string
		expectedName    string
	}{
		{
			name:            "标准格式",
			key:             "DAV::getcontentlength",
			expectedNamespace: "DAV:",
			expectedName:    "getcontentlength",
		},
		{
			name:            "自定义命名空间",
			key:             "custom:custom-property",
			expectedNamespace: "custom",
			expectedName:    "custom-property",
		},
		{
			name:            "无冒号",
			key:             "property",
			expectedNamespace: "custom",
			expectedName:    "property",
		},
		{
			name:            "空字符串",
			key:             "",
			expectedNamespace: "custom",
			expectedName:    "",
		},
		{
			name:            "多个冒号取最后一个",
			key:             "namespace:sub:name",
			expectedNamespace: "namespace:sub",
			expectedName:    "name",
		},
		{
			name:            "以冒号开头",
			key:             ":name",
			expectedNamespace: "custom",
			expectedName:    ":name",
		},
		{
			name:            "以冒号结尾",
			key:             "namespace:",
			expectedNamespace: "namespace:",
			expectedName:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, name := Property.ParseKey(tt.key)
			if namespace != tt.expectedNamespace || name != tt.expectedName {
				t.Errorf("ParseKey(%q) = (%q, %q), want (%q, %q)", tt.key, namespace, name, tt.expectedNamespace, tt.expectedName)
			}
		})
	}
}

func TestPropertyUtil_SplitNameSpace(t *testing.T) {
	tests := []struct {
		name            string
		fullName        string
		expectedNamespace string
		expectedName    string
	}{
		{
			name:            "标准格式",
			fullName:        "DAV::getcontentlength",
			expectedNamespace: "DAV:",
			expectedName:    "getcontentlength",
		},
		{
			name:            "自定义命名空间",
			fullName:        "custom:property",
			expectedNamespace: "custom",
			expectedName:    "property",
		},
		{
			name:            "无冒号",
			fullName:        "property",
			expectedNamespace: "custom",
			expectedName:    "property",
		},
		{
			name:            "空字符串",
			fullName:        "",
			expectedNamespace: "custom",
			expectedName:    "",
		},
		{
			name:            "多个冒号",
			fullName:        "ns1:ns2:property",
			expectedNamespace: "ns1",
			expectedName:    "ns2:property",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, name := Property.SplitNameSpace(tt.fullName)
			if namespace != tt.expectedNamespace || name != tt.expectedName {
				t.Errorf("SplitNameSpace(%q) = (%q, %q), want (%q, %q)", tt.fullName, namespace, name, tt.expectedNamespace, tt.expectedName)
			}
		})
	}
}

func TestPropertyUtil_CreateSuccessResponse(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		properties []interface{}
		expected   []interface{}
	}{
		{
			name:       "正常属性列表",
			path:       "/test/path",
			properties: []interface{}{"prop1", "prop2", "prop3"},
			expected:   []interface{}{"prop1", "prop2", "prop3"},
		},
		{
			name:       "nil属性",
			path:       "/test/path",
			properties: nil,
			expected:   []interface{}{},
		},
		{
			name:       "空属性列表",
			path:       "/test/path",
			properties: []interface{}{},
			expected:   []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Property.CreateSuccessResponse(tt.path, tt.properties)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("CreateSuccessResponse(%q, %v) = %v, want %v", tt.path, tt.properties, result, tt.expected)
			}
		})
	}
}

// ==================== TimeUtil Tests ====================

func TestTimeUtil_UnixToTime(t *testing.T) {
	tests := []struct {
		name     string
		unix     int64
		expected time.Time
	}{
		{
			name:     "Unix时间戳0",
			unix:     0,
			expected: time.Unix(0, 0),
		},
		{
			name:     "正数时间戳",
			unix:     1640995200, // 2022-01-01 00:00:00 UTC
			expected: time.Unix(1640995200, 0),
		},
		{
			name:     "负数时间戳",
			unix:     -1,
			expected: time.Unix(-1, 0),
		},
		{
			name:     "当前时间戳",
			unix:     time.Now().Unix(),
			expected: time.Now().Truncate(time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Time.UnixToTime(tt.unix)
			if !result.Equal(tt.expected) {
				t.Errorf("UnixToTime(%d) = %v, want %v", tt.unix, result, tt.expected)
			}
		})
	}
}

func TestTimeUtil_TimeToUnix(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected int64
	}{
		{
			name:     "Unix纪元时间",
			input:    time.Unix(0, 0),
			expected: 0,
		},
		{
			name:     "指定时间",
			input:    time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: 1640995200,
		},
		{
			name:     "当前时间",
			input:    time.Now(),
			expected: time.Now().Unix(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Time.TimeToUnix(tt.input)
			if result != tt.expected {
				t.Errorf("TimeToUnix(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimeUtil_NowUnix(t *testing.T) {
	t.Run("返回当前Unix时间戳", func(t *testing.T) {
		before := time.Now().Unix()
		result := Time.NowUnix()
		after := time.Now().Unix()
		
		if result < before || result > after {
			t.Errorf("NowUnix() = %d, expected between %d and %d", result, before, after)
		}
	})
}

func TestTimeUtil_FormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "秒级持续时间",
			duration: 30 * time.Second,
			expected: "30s",
		},
		{
			name:     "分钟级持续时间",
			duration: 5 * time.Minute,
			expected: "5m",
		},
		{
			name:     "小时级持续时间",
			duration: 2 * time.Hour,
			expected: "2h",
		},
		{
			name:     "小于1秒",
			duration: 500 * time.Millisecond,
			expected: "0s",
		},
		{
			name:     "接近1分钟",
			duration: 59 * time.Second,
			expected: "59s",
		},
		{
			name:     "接近1小时",
			duration: 59 * time.Minute,
			expected: "59m",
		},
		{
			name:     "复合时间",
			duration: 90 * time.Minute,
			expected: "90m", // 不够1小时，但会显示分钟
		},
		{
			name:     "零持续时间",
			duration: 0,
			expected: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Time.FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

// ==================== Generic Functions Tests ====================

func TestMergeSlices(t *testing.T) {
	tests := []struct {
		name     string
		slices   [][]int
		expected []int
	}{
		{
			name:     "合并两个切片",
			slices:   [][]int{{1, 2}, {3, 4}},
			expected: []int{1, 2, 3, 4},
		},
		{
			name:     "合并多个切片",
			slices:   [][]int{{1}, {2, 3}, {4, 5, 6}},
			expected: []int{1, 2, 3, 4, 5, 6},
		},
		{
			name:     "空切片",
			slices:   [][]int{},
			expected: nil,
		},
		{
			name:     "包含空切片",
			slices:   [][]int{{1, 2}, {}, {3, 4}},
			expected: []int{1, 2, 3, 4},
		},
		{
			name:     "单个切片",
			slices:   [][]int{{1, 2, 3}},
			expected: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeSlices(tt.slices...)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("MergeSlices(%v) = %v, want %v", tt.slices, result, tt.expected)
			}
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "无重复元素",
			input:    []int{1, 2, 3, 4, 5},
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "有重复元素",
			input:    []int{1, 2, 2, 3, 3, 3, 4},
			expected: []int{1, 2, 3, 4},
		},
		{
			name:     "空切片",
			input:    []int{},
			expected: []int{},
		},
		{
			name:     "全相同元素",
			input:    []int{1, 1, 1, 1},
			expected: []int{1},
		},
		{
			name:     "字符串切片",
			input:    []int{},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveDuplicates(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("RemoveDuplicates(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMapSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		fn       func(int) string
		expected []string
	}{
		{
			name:     "数字转字符串",
			input:    []int{1, 2, 3},
			fn:       func(i int) string { return fmt.Sprintf("%d", i) },
			expected: []string{"1", "2", "3"},
		},
		{
			name:     "空切片",
			input:    []int{},
			fn:       func(i int) string { return fmt.Sprintf("%d", i) },
			expected: []string{},
		},
		{
			name:     "nil切片",
			input:    nil,
			fn:       func(i int) string { return fmt.Sprintf("%d", i) },
			expected: nil,
		},
		{
			name:     "复杂转换",
			input:    []int{1, 2, 3},
			fn:       func(i int) string { return fmt.Sprintf("num_%d", i) },
			expected: []string{"num_1", "num_2", "num_3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapSlice(tt.input, tt.fn)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("MapSlice(%v, fn) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFilterSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		fn       func(int) bool
		expected []int
	}{
		{
			name:     "过滤偶数",
			input:    []int{1, 2, 3, 4, 5, 6},
			fn:       func(i int) bool { return i%2 == 0 },
			expected: []int{2, 4, 6},
		},
		{
			name:     "空切片",
			input:    []int{},
			fn:       func(i int) bool { return i%2 == 0 },
			expected: []int{},
		},
		{
			name:     "nil切片",
			input:    nil,
			fn:       func(i int) bool { return i%2 == 0 },
			expected: nil,
		},
		{
			name:     "过滤所有元素",
			input:    []int{1, 3, 5},
			fn:       func(i int) bool { return i%2 == 0 },
			expected: []int{},
		},
		{
			name:     "不过滤任何元素",
			input:    []int{2, 4, 6},
			fn:       func(i int) bool { return i%2 == 0 },
			expected: []int{2, 4, 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterSlice(tt.input, tt.fn)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FilterSlice(%v, fn) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReduceSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		initial  int
		fn       func(int, int) int
		expected int
	}{
		{
			name:     "求和",
			input:    []int{1, 2, 3, 4},
			initial:  0,
			fn:       func(acc, val int) int { return acc + val },
			expected: 10,
		},
		{
			name:     "求积",
			input:    []int{2, 3, 4},
			initial:  1,
			fn:       func(acc, val int) int { return acc * val },
			expected: 24,
		},
		{
			name:     "空切片",
			input:    []int{},
			initial:  5,
			fn:       func(acc, val int) int { return acc + val },
			expected: 5,
		},
		{
			name:     "nil切片",
			input:    nil,
			initial:  10,
			fn:       func(acc, val int) int { return acc + val },
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReduceSlice(tt.input, tt.initial, tt.fn)
			if result != tt.expected {
				t.Errorf("ReduceSlice(%v, %d, fn) = %d, want %d", tt.input, tt.initial, result, tt.expected)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected bool
	}{
		{
			name:     "nil切片",
			input:    nil,
			expected: true,
		},
		{
			name:     "空切片",
			input:    []int{},
			expected: true,
		},
		{
			name:     "非空切片",
			input:    []int{1, 2, 3},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmpty(tt.input)
			if result != tt.expected {
				t.Errorf("IsEmpty(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsNotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected bool
	}{
		{
			name:     "nil切片",
			input:    nil,
			expected: false,
		},
		{
			name:     "空切片",
			input:    []int{},
			expected: false,
		},
		{
			name:     "非空切片",
			input:    []int{1, 2, 3},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotEmpty(tt.input)
			if result != tt.expected {
				t.Errorf("IsNotEmpty(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		item     int
		expected bool
	}{
		{
			name:     "包含元素",
			input:    []int{1, 2, 3, 4, 5},
			item:     3,
			expected: true,
		},
		{
			name:     "不包含元素",
			input:    []int{1, 2, 3, 4, 5},
			item:     6,
			expected: false,
		},
		{
			name:     "空切片",
			input:    []int{},
			item:     1,
			expected: false,
		},
		{
			name:     "nil切片",
			input:    nil,
			item:     1,
			expected: false,
		},
		{
			name:     "字符串切片",
			input:    []int{},
			item:     1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.input, tt.item)
			if result != tt.expected {
				t.Errorf("Contains(%v, %d) = %v, want %v", tt.input, tt.item, result, tt.expected)
			}
		})
	}
}

func TestMapContains(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int
		key      string
		expected bool
	}{
		{
			name:     "包含键",
			input:    map[string]int{"a": 1, "b": 2, "c": 3},
			key:      "b",
			expected: true,
		},
		{
			name:     "不包含键",
			input:    map[string]int{"a": 1, "b": 2, "c": 3},
			key:      "d",
			expected: false,
		},
		{
			name:     "空映射",
			input:    map[string]int{},
			key:      "a",
			expected: false,
		},
		{
			name:     "nil映射",
			input:    nil,
			key:      "a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapContains(tt.input, tt.key)
			if result != tt.expected {
				t.Errorf("MapContains(%v, %q) = %v, want %v", tt.input, tt.key, result, tt.expected)
			}
		})
	}
}

func TestSafeMapGet(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int
		key      string
		expectedValue int
		expectedExists bool
	}{
		{
			name:            "存在的键",
			input:           map[string]int{"a": 1, "b": 2},
			key:             "a",
			expectedValue:   1,
			expectedExists:  true,
		},
		{
			name:            "不存在的键",
			input:           map[string]int{"a": 1, "b": 2},
			key:             "c",
			expectedValue:   0,
			expectedExists:  false,
		},
		{
			name:            "nil映射",
			input:           nil,
			key:             "a",
			expectedValue:   0,
			expectedExists:  false,
		},
		{
			name:            "空映射",
			input:           map[string]int{},
			key:             "a",
			expectedValue:   0,
			expectedExists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, exists := SafeMapGet(tt.input, tt.key)
			if value != tt.expectedValue || exists != tt.expectedExists {
				t.Errorf("SafeMapGet(%v, %q) = (%v, %v), want (%v, %v)", tt.input, tt.key, value, exists, tt.expectedValue, tt.expectedExists)
			}
		})
	}
}

func TestWithDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		defaultValue int
		expected int
	}{
		{
			name:     "零值使用默认值",
			value:    0,
			defaultValue: 10,
			expected: 10,
		},
		{
			name:     "非零值使用原值",
			value:    5,
			defaultValue: 10,
			expected: 5,
		},
		{
			name:     "负数使用原值",
			value:    -1,
			defaultValue: 10,
			expected: -1,
		},
		{
			name:     "字符串类型",
			value:    0,
			defaultValue: 0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithDefault(tt.value, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("WithDefault(%v, %v) = %v, want %v", tt.value, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestSafeSliceAccess(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		index    int
		expectedValue int
		expectedExists bool
	}{
		{
			name:            "有效索引",
			input:           []int{10, 20, 30},
			index:           1,
			expectedValue:   20,
			expectedExists:  true,
		},
		{
			name:            "第一个元素",
			input:           []int{10, 20, 30},
			index:           0,
			expectedValue:   10,
			expectedExists:  true,
		},
		{
			name:            "最后一个元素",
			input:           []int{10, 20, 30},
			index:           2,
			expectedValue:   30,
			expectedExists:  true,
		},
		{
			name:            "负索引",
			input:           []int{10, 20, 30},
			index:           -1,
			expectedValue:   0,
			expectedExists:  false,
		},
		{
			name:            "超出范围索引",
			input:           []int{10, 20, 30},
			index:           5,
			expectedValue:   0,
			expectedExists:  false,
		},
		{
			name:            "空切片",
			input:           []int{},
			index:           0,
			expectedValue:   0,
			expectedExists:  false,
		},
		{
			name:            "nil切片",
			input:           nil,
			index:           0,
			expectedValue:   0,
			expectedExists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, exists := SafeSliceAccess(tt.input, tt.index)
			if value != tt.expectedValue || exists != tt.expectedExists {
				t.Errorf("SafeSliceAccess(%v, %d) = (%v, %v), want (%v, %v)", tt.input, tt.index, value, exists, tt.expectedValue, tt.expectedExists)
			}
		})
	}
}

// ==================== Edge Cases and Boundary Tests ====================

func TestStringUtil_BoundaryConditions(t *testing.T) {
	t.Run("极长字符串", func(t *testing.T) {
		longStr := string(make([]byte, 10000)) // 创建10000字节的字符串
		result := String.TrimWhitespace(longStr)
		if len(result) != 10000 {
			t.Errorf("极长字符串处理失败，期望长度10000，实际长度%d", len(result))
		}
	})

	t.Run("包含所有空白字符", func(t *testing.T) {
		input := " \t\n\r\x00\x0B\x0C\x1C\x1D\x1E\x1F hello world \x1F\x1E\x1D\x1C\x0C\x0B\x00\n\r\t "
		result := String.TrimWhitespace(input)
		if result != "hello world" {
			t.Errorf("包含所有空白字符的字符串处理失败，期望'hello world'，实际%q", result)
		}
	})
}

func TestPropertyUtil_EdgeCases(t *testing.T) {
	t.Run("非常长的键名", func(t *testing.T) {
		longKey := "namespace:" + string(make([]byte, 1000))
		ns, name := Property.ParseKey(longKey)
		if ns != "namespace" || name != string(make([]byte, 1000)) {
			t.Errorf("长键名解析失败")
		}
	})

	t.Run("包含特殊字符的键名", func(t *testing.T) {
		specialKey := "ns:with:special@chars#here"
		ns, name := Property.ParseKey(specialKey)
		if ns != "ns:with:special" || name != "special@chars#here" {
			t.Errorf("特殊字符键名解析失败")
		}
	})
}

func TestTimeUtil_EdgeCases(t *testing.T) {
	t.Run("极大Unix时间戳", func(t *testing.T) {
		maxInt64 := int64(9223372036854775807)
		result := Time.UnixToTime(maxInt64)
		expected := time.Unix(maxInt64, 0)
		if !result.Equal(expected) {
			t.Errorf("极大时间戳处理失败")
		}
	})

	t.Run("极小Unix时间戳", func(t *testing.T) {
		minInt64 := int64(-9223372036854775808)
		result := Time.UnixToTime(minInt64)
		expected := time.Unix(minInt64, 0)
		if !result.Equal(expected) {
			t.Errorf("极小时间戳处理失败")
		}
	})

	t.Run("FormatDuration处理极大时长", func(t *testing.T) {
		largeDuration := 365 * 24 * time.Hour * 100 // 100年
		result := Time.FormatDuration(largeDuration)
		if result != fmt.Sprintf("%dh", 365*24*100) {
			t.Errorf("极大时长格式化失败，期望包含365*24*100，实际%q", result)
		}
	})
}