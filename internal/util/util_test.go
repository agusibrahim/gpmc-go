package util

import (
	"testing"
)

// TestURLSafeBase64 tests URL-safe base64 conversion
func TestURLSafeBase64(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Standard base64", "SGVsbG8gV29ybGQ=", "SGVsbG8gV29ybGQ"},
		{"With plus", "abc+def", "abc-def"},
		{"With slash", "abc/def", "abc_def"},
		{"With trailing equals", "abc/def=", "abc_def"},
		{"With multiple equals", "abc/def==", "abc_def"},
		{"Mixed", "abc+def/ghi==", "abc-def_ghi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLSafeBase64(tt.input)
			if result != tt.expected {
				t.Errorf("URLSafeBase64(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseEmail tests email extraction from auth_data
func TestParseEmail(t *testing.T) {
	tests := []struct {
		name     string
		authData string
		expected string
	}{
		{"Valid email", "Email=test@example.com&other=value", "test@example.com"},
		{"No email", "other=value", ""},
		{"Empty", "", ""},
		{"URL encoded", "Email=user%40domain.com&Token=abc", "user@domain.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseEmail(tt.authData)
			if result != tt.expected {
				t.Errorf("ParseEmail(%q) = %q; want %q", tt.authData, result, tt.expected)
			}
		})
	}
}

// TestParseLanguage tests language extraction from auth_data
func TestParseLanguage(t *testing.T) {
	tests := []struct {
		name     string
		authData string
		expected string
	}{
		{"Valid language", "lang=en_US&other=value", "en_US"},
		{"No language", "other=value", "en_US"},
		{"Empty", "", "en_US"},
		{"Indonesian", "lang=in_ID&Token=abc", "in_ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLanguage(tt.authData)
			if result != tt.expected {
				t.Errorf("ParseLanguage(%q) = %q; want %q", tt.authData, result, tt.expected)
			}
		})
	}
}

// TestInt64ToFloat tests int64 to float64 conversion
func TestInt64ToFloat(t *testing.T) {
	// Test a known value
	// IEEE 754 double representation of 123.456 is approximately 4638234784954079488 in int64
	testValue := int64(4638234784954079488)
	result := Int64ToFloat(testValue)
	expected := 123.456

	// Allow small floating point differences
	if result < expected-0.001 || result > expected+0.001 {
		t.Errorf("Int64ToFloat(%d) = %f; want %f", testValue, result, expected)
	}
}

// TestInt32ToFloat tests int32 to float32 conversion
func TestInt32ToFloat(t *testing.T) {
	// Test a known value
	testValue := int32(1094713344) // IEEE 754 single representation of 12.34
	result := Int32ToFloat(testValue)
	expected := float32(12.34)

	// Allow small floating point differences
	if result < expected-0.001 || result > expected+0.001 {
		t.Errorf("Int32ToFloat(%d) = %f; want %f", testValue, result, expected)
	}
}

// TestFixed32ToFloat tests fixed32 to float conversion
func TestFixed32ToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    int32
		expected float64
	}{
		{"Zero", 0, 0.0},
		{"One billion", 100000000, 10.0},
		{"Negative", -50000000, -5.0},
		{"Small", 1, 0.0000001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Fixed32ToFloat(tt.input)
			if result != tt.expected {
				t.Errorf("Fixed32ToFloat(%d) = %f; want %f", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSafeString tests safe string conversion
func TestSafeString(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"Empty", []byte{}, ""},
		{"ASCII", []byte{0x41, 0x42, 0x43}, "ABC"},
		{"UTF-8", []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}, "Hello"},
		{"Raw bytes", []byte{0x00, 0x01, 0x02}, "\x00\x01\x02"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeString(tt.input)
			if result != tt.expected {
				t.Errorf("SafeString(%v) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSafeBytes tests safe bytes conversion
func TestSafeBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{"Empty", "", []byte{}},
		{"ASCII", "ABC", []byte{0x41, 0x42, 0x43}},
		{"UTF-8", "Hello", []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeBytes(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("SafeBytes(%q) length = %d; want %d", tt.input, len(result), len(tt.expected))
			}
			for i := range tt.expected {
				if result[i] != tt.expected[i] {
					t.Errorf("SafeBytes(%q)[%d] = %02x; want %02x", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// TestParseInt tests integer parsing with default
func TestParseInt(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		defaultVal int
		expected  int
	}{
		{"Valid number", "42", 0, 42},
		{"Invalid", "abc", 10, 10},
		{"Negative", "-5", 0, -5},
		{"Empty", "", 99, 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseInt(tt.input, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("ParseInt(%q, %d) = %d; want %d", tt.input, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

// TestTruncateString tests string truncation
func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"Shorter than max", "Hello", 10, "Hello"},
		{"Equal to max", "Hello", 5, "Hello"},
		{"Longer than max", "Hello World", 5, "Hello"},
		{"Empty", "", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q; want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestJoinNonEmpty tests joining non-empty strings
func TestJoinNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		sep      string
		expected string
	}{
		{"All non-empty", []string{"a", "b", "c"}, ",", "a,b,c"},
		{"With empty", []string{"a", "", "c"}, ",", "a,c"},
		{"All empty", []string{"", "", ""}, ",", ""},
		{"Empty slice", []string{}, ",", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JoinNonEmpty(tt.parts, tt.sep)
			if result != tt.expected {
				t.Errorf("JoinNonEmpty(%v, %q) = %q; want %q", tt.parts, tt.sep, result, tt.expected)
			}
		})
	}
}
