package proto

import (
	"testing"
)

// TestRoundtripGetUploadToken tests encoding and decoding GET_UPLOAD_TOKEN
func TestRoundtripGetUploadToken(t *testing.T) {
	original := map[string]interface{}{
		"1": int64(2),
		"2": int64(2),
		"3": int64(1),
		"4": int64(3),
		"7": int64(12345),
	}

	// Encode
	encoded := EncodeMessage(original, GetUploadTokenDef)

	// Decode
	decoded, err := DecodeMessage(encoded, GetUploadTokenDef)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	// Verify
	if v, ok := decoded["1"].(int64); !ok || v != 2 {
		t.Errorf("Expected field 1 to be 2, got %v", v)
	}
	if v, ok := decoded["7"].(int64); !ok || v != 12345 {
		t.Errorf("Expected field 7 to be 12345, got %v", v)
	}
}

// TestRoundtripSetCaption tests encoding and decoding SET_CAPTION
func TestRoundtripSetCaption(t *testing.T) {
	original := map[string]interface{}{
		"2": "Test caption",
		"3": "dedup_key_123",
	}

	// Encode
	encoded := EncodeMessage(original, SetCaptionDef)

	// Decode
	decoded, err := DecodeMessage(encoded, SetCaptionDef)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	// Verify
	if v, ok := decoded["2"].(string); !ok || v != "Test caption" {
		t.Errorf("Expected field 2 to be 'Test caption', got %v", v)
	}
	if v, ok := decoded["3"].(string); !ok || v != "dedup_key_123" {
		t.Errorf("Expected field 3 to be 'dedup_key_123', got %v", v)
	}
}

// TestEncodeEmptyMap tests encoding an empty map
func TestEncodeEmptyMap(t *testing.T) {
	original := map[string]interface{}{}

	// Encode - should not panic; an empty top-level message encodes to zero bytes.
	encoded := EncodeMessage(original, GetUploadTokenDef)

	if len(encoded) != 0 {
		t.Errorf("EncodeMessage(empty) length = %d; want 0", len(encoded))
	}
}

// TestEncodeNestedMessage tests encoding a nested message
func TestEncodeNestedMessage(t *testing.T) {
	original := map[string]interface{}{
		"1": map[string]interface{}{
			"1": []byte{0x01, 0x02, 0x03},
			"2": map[string]interface{}{},
		},
	}

	// Encode
	encoded := EncodeMessage(original, FindRemoteMediaByHashDef)

	if len(encoded) == 0 {
		t.Error("Encoded data should not be empty")
	}

	// Decode to verify structure
	decoded, err := DecodeMessage(encoded, FindRemoteMediaByHashDef)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	if _, ok := decoded["1"]; !ok {
		t.Error("Expected field 1 to exist")
	}
}

// TestEncodeWithBytesField tests encoding a bytes field
func TestEncodeWithBytesField(t *testing.T) {
	original := map[string]interface{}{
		"3": []byte{0x01, 0x03},
	}

	// Encode
	encoded := EncodeMessage(original, CommitUploadDef)

	if len(encoded) == 0 {
		t.Error("Encoded data should not be empty")
	}
}

// TestEncodeWithRepeatedField tests encoding a repeated field
func TestEncodeWithRepeatedField(t *testing.T) {
	original := map[string]interface{}{
		"1": []interface{}{"key1", "key2", "key3"},
	}

	// Encode
	encoded := EncodeMessage(original, AddMediaToAlbumDef)

	if len(encoded) == 0 {
		t.Error("Encoded data should not be empty")
	}
}

// TestDecodeWithoutDefinition tests decoding without a message definition
func TestDecodeWithoutDefinition(t *testing.T) {
	// First encode with definition
	original := map[string]interface{}{
		"1": int64(42),
		"2": "test string",
	}

	encoded := EncodeMessage(original, GetUploadTokenDef)

	// Decode without definition
	decoded, err := DecodeMessage(encoded, nil)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	// Should still be able to decode
	if _, ok := decoded["1"]; !ok {
		t.Error("Expected field 1 to exist")
	}
}
