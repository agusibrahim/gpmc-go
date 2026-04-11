package proto

import (
	"fmt"
	"math"
	"strconv"

	"google.golang.org/protobuf/encoding/protowire"
)

// DecodeMessage decodes protobuf bytes into a map
// data: raw protobuf bytes
// msgDef: optional MessageDef for type hints (can be nil for auto-detection)
func DecodeMessage(data []byte, msgDef *MessageDef) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for len(data) > 0 {
		// Consume field tag
		tag, wireType, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, fmt.Errorf("invalid tag: %v", protowire.ParseError(n))
		}
		data = data[n:]

		fieldNum := int(tag)

		// Get field definition if available
		var fieldDef FieldDef
		var hasDef bool
		if msgDef != nil {
			fieldDef, hasDef = msgDef.GetField(fieldNum)
		}

		fieldKey := strconv.Itoa(fieldNum)

		// Decode based on wire type
		switch wireType {
		case protowire.VarintType:
			var value uint64
			value, n = protowire.ConsumeVarint(data)
			if n < 0 {
				return nil, fmt.Errorf("invalid varint at field %d: %v", fieldNum, protowire.ParseError(n))
			}
			data = data[n:]

			// Store as int64
			result[fieldKey] = int64(value)

		case protowire.Fixed32Type:
			var value uint32
			value, n = protowire.ConsumeFixed32(data)
			if n < 0 {
				return nil, fmt.Errorf("invalid fixed32 at field %d: %v", fieldNum, protowire.ParseError(n))
			}
			data = data[n:]

			// Store as int32
			result[fieldKey] = int32(value)

		case protowire.Fixed64Type:
			var value uint64
			value, n = protowire.ConsumeFixed64(data)
			if n < 0 {
				return nil, fmt.Errorf("invalid fixed64 at field %d: %v", fieldNum, protowire.ParseError(n))
			}
			data = data[n:]

			// Store as int64
			result[fieldKey] = int64(value)

		case protowire.BytesType:
			var value []byte
			value, n = protowire.ConsumeBytes(data)
			if n < 0 {
				return nil, fmt.Errorf("invalid bytes at field %d: %v", fieldNum, protowire.ParseError(n))
			}
			data = data[n:]

			// Determine how to interpret the bytes
			if hasDef {
				switch fieldDef.Type {
				case FieldTypeString:
					result[fieldKey] = string(value)
				case FieldTypeBytes:
					result[fieldKey] = value
				case FieldTypeMessage:
					// Decode as nested message
					if fieldDef.MessageDef != nil {
						nested, err := DecodeMessage(value, fieldDef.MessageDef)
						if err != nil {
							// If nested decode fails, store as bytes
							result[fieldKey] = value
						} else {
							result[fieldKey] = nested
						}
					} else {
						// No nested definition - try to decode anyway
						nested, err := DecodeMessage(value, nil)
						if err != nil {
							result[fieldKey] = value
						} else {
							result[fieldKey] = nested
						}
					}
				default:
					// Unknown type - try to decode as message, fallback to bytes
					nested, err := DecodeMessage(value, nil)
					if err != nil {
						result[fieldKey] = value
					} else {
						result[fieldKey] = nested
					}
				}
			} else {
				// No definition - try to decode as message, fallback to string/bytes
				nested, err := DecodeMessage(value, nil)
				if err == nil && len(nested) > 0 {
					// Successfully decoded as message
					result[fieldKey] = nested
				} else {
					// Treat as string (protobuf strings are also length-delimited)
					result[fieldKey] = string(value)
				}
			}

		default:
			return nil, fmt.Errorf("unknown wire type %d at field %d", wireType, fieldNum)
		}
	}

	// Handle seen_repeated fields - wrap in array if needed
	if msgDef != nil {
		for fieldNum, fieldDef := range msgDef.Fields {
			if fieldDef.SeenRepeated {
				fieldKey := strconv.Itoa(fieldNum)
				if value, ok := result[fieldKey]; ok {
					// Check if it's already an array
					if _, isArray := value.([]interface{}); !isArray {
						result[fieldKey] = []interface{}{value}
					}
				}
			}
		}
	}

	return result, nil
}

// DecodeMessageAsFloat64 decodes a varint field as a float64
// This is used for fixed32 and fixed64 fields that represent floats
func DecodeMessageAsFloat64(data []byte) (float64, error) {
	bits, n := protowire.ConsumeFixed64(data)
	if n < 0 {
		return 0, fmt.Errorf("invalid fixed64: %v", protowire.ParseError(n))
	}
	return math.Float64frombits(bits), nil
}

// DecodeMessageAsFloat32 decodes a varint field as a float32
func DecodeMessageAsFloat32(data []byte) (float32, error) {
	bits, n := protowire.ConsumeFixed32(data)
	if n < 0 {
		return 0, fmt.Errorf("invalid fixed32: %v", protowire.ParseError(n))
	}
	return math.Float32frombits(bits), nil
}

// GetFieldAsInt gets a field value as int64, with default value
func GetFieldAsInt(data map[string]interface{}, fieldNum int, defaultValue int64) int64 {
	key := strconv.Itoa(fieldNum)
	if value, ok := data[key]; ok {
		switch v := value.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case int32:
			return int64(v)
		case uint32:
			return int64(v)
		}
	}
	return defaultValue
}

// GetFieldAsString gets a field value as string, with default value
func GetFieldAsString(data map[string]interface{}, fieldNum int, defaultValue string) string {
	key := strconv.Itoa(fieldNum)
	if value, ok := data[key]; ok {
		if s, ok := value.(string); ok {
			return s
		}
	}
	return defaultValue
}

// GetFieldAsBytes gets a field value as []byte, with default value
func GetFieldAsBytes(data map[string]interface{}, fieldNum int) []byte {
	key := strconv.Itoa(fieldNum)
	if value, ok := data[key]; ok {
		switch v := value.(type) {
		case []byte:
			return v
		case string:
			return []byte(v)
		}
	}
	return nil
}

// GetFieldAsMap gets a field value as map, with default value
func GetFieldAsMap(data map[string]interface{}, fieldNum int) map[string]interface{} {
	key := strconv.Itoa(fieldNum)
	if value, ok := data[key]; ok {
		if m, ok := value.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

// GetFieldAsArray gets a field value as array, with default value
func GetFieldAsArray(data map[string]interface{}, fieldNum int) []interface{} {
	key := strconv.Itoa(fieldNum)
	if value, ok := data[key]; ok {
		switch v := value.(type) {
		case []interface{}:
			return v
		default:
			return []interface{}{value}
		}
	}
	return nil
}

// HasField checks if a field exists in the decoded data
func HasField(data map[string]interface{}, fieldNum int) bool {
	key := strconv.Itoa(fieldNum)
	_, ok := data[key]
	return ok
}
