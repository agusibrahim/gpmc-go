package models

// MediaItem represents a Google Photos media item
// Based on gpmc/models.py MediaItem dataclass with 45 fields
type MediaItem struct {
	// Primary fields
	MediaKey    string `json:"media_key" db:"media_key"`
	FileName    string `json:"file_name" db:"file_name"`
	DedupKey    string `json:"dedup_key" db:"dedup_key"`
	IsCanonical bool   `json:"is_canonical" db:"is_canonical"`
	Type        int    `json:"type" db:"type"` // 1=photo, 2=video
	Caption     string `json:"caption" db:"caption"`
	CollectionID string `json:"collection_id" db:"collection_id"`

	// Size and quota
	SizeBytes        int64 `json:"size_bytes" db:"size_bytes"`
	QuotaChargedBytes int64 `json:"quota_charged_bytes" db:"quota_charged_bytes"`

	// Origin and content
	Origin          string `json:"origin" db:"origin"` // self, partner, shared
	ContentVersion  int    `json:"content_version" db:"content_version"`

	// Timestamps
	UTCTimestamp             int64 `json:"utc_timestamp" db:"utc_timestamp"`
	ServerCreationTimestamp  int64 `json:"server_creation_timestamp" db:"server_creation_timestamp"`
	TimezoneOffset           *int  `json:"timezone_offset,omitempty" db:"timezone_offset"`

	// Dimensions
	Width  *int `json:"width,omitempty" db:"width"`
	Height *int `json:"height,omitempty" db:"height"`

	// Remote and upload status
	RemoteURL    string `json:"remote_url" db:"remote_url"`
	UploadStatus *int  `json:"upload_status,omitempty" db:"upload_status"`

	// Trash and status flags
	TrashTimestamp    *int  `json:"trash_timestamp,omitempty" db:"trash_timestamp"`
	IsArchived        bool  `json:"is_archived" db:"is_archived"`
	IsFavorite        bool  `json:"is_favorite" db:"is_favorite"`
	IsLocked          bool  `json:"is_locked" db:"is_locked"`
	IsOriginalQuality bool  `json:"is_original_quality" db:"is_original_quality"`
	IsEdited          bool  `json:"is_edited" db:"is_edited"`

	// Location
	Latitude     *float64 `json:"latitude,omitempty" db:"latitude"`
	Longitude    *float64 `json:"longitude,omitempty" db:"longitude"`
	LocationName *string  `json:"location_name,omitempty" db:"location_name"`
	LocationID   *string  `json:"location_id,omitempty" db:"location_id"`

	// Camera data (photo)
	Make         *string  `json:"make,omitempty" db:"make"`
	Model        *string  `json:"model,omitempty" db:"model"`
	Aperture     *float64 `json:"aperture,omitempty" db:"aperture"`
	ShutterSpeed *float64 `json:"shutter_speed,omitempty" db:"shutter_speed"`
	ISO          *int     `json:"iso,omitempty" db:"iso"`
	FocalLength  *float64 `json:"focal_length,omitempty" db:"focal_length"`

	// Video specific
	Duration           *int    `json:"duration,omitempty" db:"duration"`
	CaptureFrameRate   *float64 `json:"capture_frame_rate,omitempty" db:"capture_frame_rate"`
	EncodedFrameRate   *float64 `json:"encoded_frame_rate,omitempty" db:"encoded_frame_rate"`
	IsMicroVideo       bool     `json:"is_micro_video" db:"is_micro_video"`
	MicroVideoWidth    *int     `json:"micro_video_width,omitempty" db:"micro_video_width"`
	MicroVideoHeight   *int     `json:"micro_video_height,omitempty" db:"micro_video_height"`
}

// IsPhoto returns true if this media item is a photo
func (m *MediaItem) IsPhoto() bool {
	return m.Type == 1
}

// IsVideo returns true if this media item is a video
func (m *MediaItem) IsVideo() bool {
	return m.Type == 2
}

// HasLocation returns true if the media item has location data
func (m *MediaItem) HasLocation() bool {
	return m.Latitude != nil && m.Longitude != nil
}

// HasCameraData returns true if the media item has camera EXIF data
func (m *MediaItem) HasCameraData() bool {
	return m.Make != nil || m.Model != nil || m.Aperture != nil ||
		m.ShutterSpeed != nil || m.ISO != nil || m.FocalLength != nil
}

// IsDeleted returns true if the media item is in trash
func (m *MediaItem) IsDeleted() bool {
	return m.TrashTimestamp != nil && *m.TrashTimestamp > 0
}
