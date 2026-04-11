package proto

import (
	"fmt"
	"math"
	"strconv"

	"google.golang.org/protobuf/encoding/protowire"
)

// EncodeMessage encodes a protobuf message body into bytes
// body: map[string]interface{} where keys are field numbers as strings
// msgDef: MessageDef defining the structure
func EncodeMessage(body map[string]interface{}, msgDef *MessageDef) []byte {
	var buf []byte

	// Determine field order
	fieldOrder := getFieldOrder(body, msgDef)

	// Encode each field
	for _, fieldNumStr := range fieldOrder {
		fieldNum, err := strconv.Atoi(fieldNumStr)
		if err != nil {
			continue // Skip invalid field numbers
		}

		value, ok := body[fieldNumStr]
		if !ok {
			continue
		}

		fieldDef, hasDef := msgDef.GetField(fieldNum)

		// Handle different value types
		switch v := value.(type) {
		case int:
			buf = encodeField(buf, fieldNum, int64(v), fieldDef, hasDef)
		case int8:
			buf = encodeField(buf, fieldNum, int64(v), fieldDef, hasDef)
		case int16:
			buf = encodeField(buf, fieldNum, int64(v), fieldDef, hasDef)
		case int32:
			buf = encodeField(buf, fieldNum, int64(v), fieldDef, hasDef)
		case int64:
			buf = encodeField(buf, fieldNum, v, fieldDef, hasDef)
		case uint:
			buf = encodeField(buf, fieldNum, int64(v), fieldDef, hasDef)
		case uint8:
			buf = encodeField(buf, fieldNum, int64(v), fieldDef, hasDef)
		case uint16:
			buf = encodeField(buf, fieldNum, int64(v), fieldDef, hasDef)
		case uint32:
			buf = encodeField(buf, fieldNum, int64(v), fieldDef, hasDef)
		case uint64:
			if v <= math.MaxInt64 {
				buf = encodeField(buf, fieldNum, int64(v), fieldDef, hasDef)
			}
		case float32:
			// Encode as int32 then reinterpret as float
			buf = encodeField(buf, fieldNum, int64(math.Float32bits(v)), fieldDef, hasDef)
		case float64:
			// Encode as int64 then reinterpret as float
			buf = encodeField(buf, fieldNum, int64(math.Float64bits(v)), fieldDef, hasDef)
		case string:
			buf = encodeStringField(buf, fieldNum, v, fieldDef, hasDef)
		case []byte:
			buf = encodeBytesField(buf, fieldNum, v, fieldDef, hasDef)
		case map[string]interface{}:
			buf = encodeMessageField(buf, fieldNum, v, fieldDef, hasDef)
		case []interface{}:
			// Repeated field - encode each element
			for _, elem := range v {
				switch elemVal := elem.(type) {
				case int:
					buf = encodeField(buf, fieldNum, int64(elemVal), fieldDef, hasDef)
				case int32:
					buf = encodeField(buf, fieldNum, int64(elemVal), fieldDef, hasDef)
				case int64:
					buf = encodeField(buf, fieldNum, elemVal, fieldDef, hasDef)
				case string:
					buf = encodeStringField(buf, fieldNum, elemVal, fieldDef, hasDef)
				case []byte:
					buf = encodeBytesField(buf, fieldNum, elemVal, fieldDef, hasDef)
				case map[string]interface{}:
					buf = encodeMessageField(buf, fieldNum, elemVal, fieldDef, hasDef)
				}
			}
		case bool:
			// Encode bool as varint (0 or 1)
			var val int64
			if v {
				val = 1
			}
			buf = encodeField(buf, fieldNum, val, fieldDef, hasDef)
		case nil:
			// Skip nil values
		default:
			// Unknown type - try to handle as string
			buf = encodeStringField(buf, fieldNum, fmt.Sprintf("%v", v), fieldDef, hasDef)
		}
	}

	return buf
}

// encodeField encodes a varint field
func encodeField(buf []byte, fieldNum int, value int64, fieldDef FieldDef, hasDef bool) []byte {
	tag := protowire.EncodeTag(protowire.Number(fieldNum), protowire.VarintType)
	buf = protowire.AppendVarint(buf, tag)
	buf = protowire.AppendVarint(buf, uint64(value))
	return buf
}

// encodeStringField encodes a string field (length-delimited)
func encodeStringField(buf []byte, fieldNum int, value string, fieldDef FieldDef, hasDef bool) []byte {
	tag := protowire.EncodeTag(protowire.Number(fieldNum), protowire.BytesType)
	buf = protowire.AppendVarint(buf, tag)
	buf = protowire.AppendString(buf, value)
	return buf
}

// encodeBytesField encodes a bytes field (length-delimited)
func encodeBytesField(buf []byte, fieldNum int, value []byte, fieldDef FieldDef, hasDef bool) []byte {
	tag := protowire.EncodeTag(protowire.Number(fieldNum), protowire.BytesType)
	buf = protowire.AppendVarint(buf, tag)
	buf = protowire.AppendBytes(buf, value)
	return buf
}

// encodeMessageField encodes a nested message field
func encodeMessageField(buf []byte, fieldNum int, value map[string]interface{}, fieldDef FieldDef, hasDef bool) []byte {
	var nestedMsg []byte

	// If we have a field definition for the nested message, use it
	if hasDef && fieldDef.Type == FieldTypeMessage && fieldDef.MessageDef != nil {
		nestedMsg = EncodeMessage(value, fieldDef.MessageDef)
	} else {
		// No definition - try to encode as best effort
		nestedMsg = EncodeMessage(value, &MessageDef{Fields: make(map[int]FieldDef)})
	}

	tag := protowire.EncodeTag(protowire.Number(fieldNum), protowire.BytesType)
	buf = protowire.AppendVarint(buf, tag)
	buf = protowire.AppendVarint(buf, uint64(len(nestedMsg)))
	buf = append(buf, nestedMsg...)
	return buf
}

// getFieldOrder returns the field numbers in encoding order
// Uses field_order from definition if available, otherwise uses natural order
func getFieldOrder(body map[string]interface{}, msgDef *MessageDef) []string {
	// Collect all field numbers from the body
	var fieldNums []string
	for key := range body {
		fieldNums = append(fieldNums, key)
	}

	// If there's a field_order specification, use it
	// This is a simplified implementation - the full version would
	// need to handle the complex field_order patterns from the Python code
	return fieldNums
}
