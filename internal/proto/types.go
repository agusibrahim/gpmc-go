package proto

// FieldType represents the protobuf field type
type FieldType int

const (
	FieldTypeInt    FieldType = iota // varint wire type
	FieldTypeString                   // length-delimited, UTF-8 string
	FieldTypeBytes                    // length-delimited, raw bytes
	FieldTypeMessage                  // length-delimited, nested message
)

// FieldDef defines a protobuf field
type FieldDef struct {
	Type         FieldType    // Field type
	MessageDef   *MessageDef  // Nested message definition (for FieldTypeMessage only)
	SeenRepeated bool         // True if field is marked as repeated
	FieldOrder   []int        // Explicit field ordering for encoding
}

// MessageDef defines a protobuf message type
type MessageDef struct {
	Fields map[int]FieldDef // Field number -> Field definition
}

// NewMessageDef creates a new MessageDef
func NewMessageDef() *MessageDef {
	return &MessageDef{
		Fields: make(map[int]FieldDef),
	}
}

// AddField adds a field to the message definition
func (m *MessageDef) AddField(fieldNumber int, fieldType FieldType, seenRepeated bool) {
	m.Fields[fieldNumber] = FieldDef{
		Type:         fieldType,
		SeenRepeated: seenRepeated,
	}
}

// AddMessageField adds a nested message field to the message definition
func (m *MessageDef) AddMessageField(fieldNumber int, msgDef *MessageDef, seenRepeated bool) {
	m.Fields[fieldNumber] = FieldDef{
		Type:         FieldTypeMessage,
		MessageDef:   msgDef,
		SeenRepeated: seenRepeated,
	}
}

// AddFieldWithOrder adds a field with explicit field ordering
func (m *MessageDef) AddFieldWithOrder(fieldNumber int, fieldType FieldType, seenRepeated bool, fieldOrder []int) {
	m.Fields[fieldNumber] = FieldDef{
		Type:         fieldType,
		SeenRepeated: seenRepeated,
		FieldOrder:   fieldOrder,
	}
}

// AddMessageFieldWithOrder adds a nested message field with explicit field ordering
func (m *MessageDef) AddMessageFieldWithOrder(fieldNumber int, msgDef *MessageDef, seenRepeated bool, fieldOrder []int) {
	m.Fields[fieldNumber] = FieldDef{
		Type:         FieldTypeMessage,
		MessageDef:   msgDef,
		SeenRepeated: seenRepeated,
		FieldOrder:   fieldOrder,
	}
}

// GetField returns the field definition for a given field number
func (m *MessageDef) GetField(fieldNumber int) (FieldDef, bool) {
	def, ok := m.Fields[fieldNumber]
	return def, ok
}

// HasField checks if a field number exists in the message definition
func (m *MessageDef) HasField(fieldNumber int) bool {
	_, ok := m.Fields[fieldNumber]
	return ok
}
