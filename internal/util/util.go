package util

import (
	"net/url"
	"strconv"
	"strings"
	"unsafe"
)

// URLSafeBase64 converts a Base64 string to URL-safe format
// Replaces + with -, / with _, and strips trailing =
func URLSafeBase64(b64 string) string {
	result := strings.ReplaceAll(b64, "+", "-")
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.TrimRight(result, "=")
	return result
}

// ParseEmail extracts the email from auth_data (URL encoded)
func ParseEmail(authData string) string {
	values, err := url.ParseQuery(authData)
	if err != nil {
		return ""
	}
	if email := values.Get("Email"); email != "" {
		return email
	}
	return ""
}

// ParseLanguage extracts the language from auth_data (URL encoded)
// Falls back to "en_US" if not found
func ParseLanguage(authData string) string {
	values, err := url.ParseQuery(authData)
	if err != nil {
		return "en_US"
	}
	if lang := values.Get("lang"); lang != "" {
		return lang
	}
	return "en_US"
}

// Int64ToFloat converts a 64-bit integer to IEEE 754 double-precision float
func Int64ToFloat(num int64) float64 {
	return *(*float64)(unsafe.Pointer(&num))
}

// Int32ToFloat converts a 32-bit integer to IEEE 754 single-precision float
func Int32ToFloat(num int32) float32 {
	return *(*float32)(unsafe.Pointer(&num))
}

// Fixed32ToFloat converts a scaled 32-bit integer (x * 10^7) to float
func Fixed32ToFloat(n int32) float64 {
	return float64(n) / 10000000.0
}

// SafeString converts bytes to string, handling non-UTF8 data
// Used for protobuf string fields that may contain raw bytes
func SafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	// Return as-is (protobuf strings can contain non-UTF8 bytes)
	return string(b)
}

// SafeBytes converts string to bytes, handling raw byte strings
func SafeBytes(s string) []byte {
	return []byte(s)
}

// ParseInt parses string to int, with fallback
func ParseInt(s string, defaultValue int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}

// TruncateString truncates a string to max length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// JoinNonEmpty joins non-empty strings with a separator
func JoinNonEmpty(parts []string, sep string) string {
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return strings.Join(result, sep)
}
