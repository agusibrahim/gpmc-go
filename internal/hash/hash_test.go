package hash

import (
	"encoding/hex"
	"testing"
)

// TestConvertSHA1HashBytes tests hash conversion from bytes
func TestConvertSHA1HashBytes(t *testing.T) {
	testHash := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	expectedHashB64 := BytesToBase64(testHash)

	// Test conversion
	convertedBytes, convertedB64, err := ConvertSHA1Hash(expectedHash)
	if err != nil {
		t.Fatalf("ConvertSHA1Hash failed: %v", err)
	}

	if convertedB64 != hashB64 {
		t.Errorf("Expected hash %s, got %s", hashB64, convertedB64)
	}

	if hex.EncodeToString(convertedBytes) != expectedHash {
		t.Errorf("Expected hash bytes %x, got %x", hashBytes, convertedBytes)
	}
}

// TestConvertSHA1HashBytes tests hash conversion from bytes
func TestConvertSHA1HashBytes(t *testing.T) {
	testHash := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	hashBytes, hashB64, err := ConvertSHA1Hash(testHash)
	if err != nil {
		t.Fatalf("ConvertSHA1Hash failed: %v", err)
	}

	expectedB64 := BytesToBase64(testHash)
	if hashB64 != expectedB64 {
		t.Errorf("Expected %s, got %s", expectedB64, hashB64)
	}

	// Verify bytes match
	if len(hashBytes) != len(testHash) {
		t.Errorf("Expected bytes length %d, got %d", len(testHash), len(hashBytes))
	}
}

// TestConvertSHA1HashHex tests hash conversion from hex string
func TestConvertSHA1HashHex(t *testing.T) {
	testHashHex := "1a2b3c4d5e6f1a2b3c4d5e6f1a2b3c4d5e6f1a2b"

	hashBytes, hashB64, err := ConvertSHA1Hash(testHashHex)
	if err != nil {
		t.Fatalf("ConvertSHA1Hash failed: %v", err)
	}

	// Verify hex conversion
	convertedHex := hex.EncodeToString(hashBytes)
	if convertedHex != testHashHex {
		t.Errorf("Expected %s, got %s", testHashHex, convertedHex)
	}

	// Verify base64 is not empty
	if hashB64 == "" {
		t.Error("Expected non-empty base64 hash")
	}
}

// TestConvertSHA1HashBase64 tests hash conversion from base64 string
func TestConvertSHA1HashBase64(t *testing.T) {
	testHashBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	testHashB64 := BytesToBase64(testHashBytes)

	hashBytes, hashB64, err := ConvertSHA1Hash(testHashB64)
	if err != nil {
		t.Fatalf("ConvertSHA1Hash failed: %v", err)
	}

	if hashB64 != testHashB64 {
		t.Errorf("Expected %s, got %s", testHashB64, hashB64)
	}

	if len(hashBytes) != len(testHashBytes) {
		t.Errorf("Expected bytes length %d, got %d", len(testHashBytes), len(hashBytes))
	}
}

// TestIsHashHexadecimal tests SHA1 hex hash validation
func TestIsHashHexadecimal(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		expected bool
	}{
		{"Valid SHA1 hex", "1a2b3c4d5e6f1a2b3c4d5e6f1a2b3c4d5e6f1a2b", true},
		{"Valid SHA1 hex uppercase", "1A2B3C4D5E6F1A2B3C4D5E6F1A2B3C4D5E6F1A2B", true},
		{"Too short", "1a2b3c", false},
		{"Too long", "1a2b3c4d5e6f1a2b3c4d5e6f1a2b3c4d5e6f1a2b3c4d", false},
		{"Invalid chars", "1a2b3c4d5g6h", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHashHexadecimal(tt.hash)
			if result != tt.expected {
				t.Errorf("IsHashHexadecimal(%q) = %v; want %v", tt.hash, result, tt.expected)
			}
		})
	}
}

// TestBytesToHex tests bytes to hex conversion
func TestBytesToHex(t *testing.T) {
	testBytes := []byte{0x01, 0x02, 0x03, 0x0a, 0xff}
	expected := "0102030aff"

	result := BytesToHex(testBytes)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestBytesToBase64 tests bytes to base64 conversion
func TestBytesToBase64(t *testing.T) {
	testBytes := []byte{0x01, 0x02, 0x03}
	expected := "AQID"

	result := BytesToBase64(testBytes)
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

// TestHexToBytes tests hex to bytes conversion
func TestHexToBytes(t *testing.T) {
	testHex := "0102030aff"
	expected := []byte{0x01, 0x02, 0x03, 0x0a, 0xff}

	result, err := HexToBytes(testHex)
	if err != nil {
		t.Fatalf("HexToBytes failed: %v", err)
	}

	if len(result) != len(expected) {
		t.Fatalf("Expected length %d, got %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Byte %d: expected %02x, got %02x", i, expected[i], result[i])
		}
	}
}

// TestBase64ToBytes tests base64 to bytes conversion
func TestBase64ToBytes(t *testing.T) {
	testB64 := "AQID"
	expected := []byte{0x01, 0x02, 0x03}

	result, err := Base64ToBytes(testB64)
	if err != nil {
		t.Fatalf("Base64ToBytes failed: %v", err)
	}

	if len(result) != len(expected) {
		t.Fatalf("Expected length %d, got %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Byte %d: expected %02x, got %02x", i, expected[i], result[i])
		}
	}
}
