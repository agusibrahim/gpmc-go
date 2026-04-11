package proto

// Message type definitions for Google Photos mobile API
// Converted from gpmc/message_types.py

var (
	// GET_UPLOAD_TOKEN - Simple flat structure with 5 int fields
	GetUploadTokenDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(1, FieldTypeInt, false)
		def.AddField(2, FieldTypeInt, false)
		def.AddField(3, FieldTypeInt, false)
		def.AddField(4, FieldTypeInt, false)
		def.AddField(7, FieldTypeInt, false)
		return def
	}()

	// SET_CAPTION - Simple structure with 2 string fields
	SetCaptionDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(2, FieldTypeString, false)
		def.AddField(3, FieldTypeString, false)
		return def
	}()

	// FIND_REMOTE_MEDIA_BY_HASH - Simple nested structure
	FindRemoteMediaByHashDef = func() *MessageDef {
		def := NewMessageDef()
		nested := NewMessageDef()
		nested.AddField(1, FieldTypeBytes, false)
		nested.AddField(2, FieldTypeMessage, false) // Empty message
		def.AddMessageField(1, nested, false)
		return def
	}()

	// ADD_MEDIA_TO_ALBUM - Moderate complexity
	AddMediaToAlbumDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(1, FieldTypeMessage, true) // Repeated
		def.AddField(2, FieldTypeString, false)
		def.AddField(5, FieldTypeMessage, false)
		def.AddField(6, FieldTypeMessage, false)
		def.AddField(7, FieldTypeInt, false)
		return def
	}()

	// CREATE_ALBUM - Moderate complexity with repeated field
	CreateAlbumDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(1, FieldTypeString, false)
		def.AddField(2, FieldTypeInt, false)
		def.AddField(3, FieldTypeInt, false)
		def.AddField(4, FieldTypeMessage, true) // seen_repeated
		def.AddField(6, FieldTypeMessage, false)
		def.AddField(7, FieldTypeMessage, false)
		def.AddField(8, FieldTypeMessage, false)
		return def
	}()

	// SET_FAVORITE - Moderate complexity
	SetFavoriteDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(1, FieldTypeMessage, false)
		def.AddField(2, FieldTypeMessage, false)
		def.AddField(3, FieldTypeMessage, false)
		return def
	}()

	// SET_ARCHIVED - Moderate complexity with seen_repeated
	SetArchivedDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(1, FieldTypeMessage, true) // seen_repeated
		def.AddField(3, FieldTypeInt, false)
		return def
	}()

	// GET_DOWNLOAD_URLS - Moderate complexity
	GetDownloadURLsDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(1, FieldTypeMessage, false)
		def.AddField(2, FieldTypeMessage, false)
		return def
	}()

	// RESTORE_FROM_TRASH - Moderate complexity
	RestoreFromTrashDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(2, FieldTypeInt, false)
		def.AddField(3, FieldTypeString, true) // Implicit repeated
		def.AddField(4, FieldTypeInt, false)
		def.AddField(8, FieldTypeMessage, false)
		return def
	}()

	// MOVE_TO_TRASH - Complex with nested structures
	MoveToTrashDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(2, FieldTypeInt, false)
		def.AddField(3, FieldTypeString, false) // Actually repeated in practice
		def.AddField(4, FieldTypeInt, false)
		def.AddField(8, FieldTypeMessage, false)
		def.AddField(9, FieldTypeMessage, false)
		return def
	}()

	// DELETE_PERMANENTLY - Similar to MOVE_TO_TRASH
	DeletePermanentlyDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(2, FieldTypeInt, false)
		def.AddField(3, FieldTypeString, false) // Actually repeated in practice
		def.AddField(4, FieldTypeInt, false)
		def.AddField(8, FieldTypeMessage, false)
		def.AddField(9, FieldTypeString, false)
		return def
	}()

	// COMMIT_UPLOAD - Complex with deeply nested structures
	CommitUploadDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(1, FieldTypeMessage, false)
		def.AddField(2, FieldTypeMessage, false)
		def.AddField(3, FieldTypeBytes, false)
		return def
	}()

	// LIB_STATE_RESPONSE_FIX - Used for decoding library state responses
	LibStateResponseFixDef = func() *MessageDef {
		def := NewMessageDef()
		nested := NewMessageDef()
		// Empty message structure for fixing decoded responses
		def.AddMessageField(1, nested, false)
		return def
	}()

	// GET_LIB_PAGE_INIT - Very complex deeply nested structure
	GetLibPageInitDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(1, FieldTypeMessage, false)
		def.AddField(2, FieldTypeMessage, false)
		def.AddField(3, FieldTypeMessage, false)
		return def
	}()

	// GET_LIB_STATE - Very complex deeply nested structure
	GetLibStateDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(1, FieldTypeMessage, false)
		def.AddField(2, FieldTypeMessage, false)
		def.AddField(3, FieldTypeMessage, false)
		return def
	}()

	// GET_LIB_PAGE - Very complex deeply nested structure
	GetLibPageDef = func() *MessageDef {
		def := NewMessageDef()
		def.AddField(1, FieldTypeMessage, false)
		def.AddField(2, FieldTypeMessage, false)
		def.AddField(3, FieldTypeMessage, false)
		return def
	}()
)

// Helper function to get message definition by name
func GetMessageDef(name string) *MessageDef {
	switch name {
	case "GET_UPLOAD_TOKEN":
		return GetUploadTokenDef
	case "SET_CAPTION":
		return SetCaptionDef
	case "FIND_REMOTE_MEDIA_BY_HASH":
		return FindRemoteMediaByHashDef
	case "ADD_MEDIA_TO_ALBUM":
		return AddMediaToAlbumDef
	case "CREATE_ALBUM":
		return CreateAlbumDef
	case "SET_FAVORITE":
		return SetFavoriteDef
	case "SET_ARCHIVED":
		return SetArchivedDef
	case "GET_DOWNLOAD_URLS":
		return GetDownloadURLsDef
	case "RESTORE_FROM_TRASH":
		return RestoreFromTrashDef
	case "MOVE_TO_TRASH":
		return MoveToTrashDef
	case "DELETE_PERMANENTLY":
		return DeletePermanentlyDef
	case "COMMIT_UPLOAD":
		return CommitUploadDef
	case "LIB_STATE_RESPONSE_FIX":
		return LibStateResponseFixDef
	case "GET_LIB_PAGE_INIT":
		return GetLibPageInitDef
	case "GET_LIB_STATE":
		return GetLibStateDef
	case "GET_LIB_PAGE":
		return GetLibPageDef
	default:
		return nil
	}
}
