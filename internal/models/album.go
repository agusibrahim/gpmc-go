package models

// Album represents a Google Photos album
type Album struct {
	AlbumKey  string `json:"album_key"`
	Title     string `json:"title"`
	ItemCount int64  `json:"item_count"`
	CoverURL  string `json:"cover_url"`
}

// AlbumMediaItem represents a media item (photo/video) inside an album
type AlbumMediaItem struct {
	MediaKey string `json:"media_key"`
	FileName string `json:"file_name"`
}

// ShareAlbumResult holds the outcome of sharing an album
type ShareAlbumResult struct {
	SharedAlbumKey string `json:"shared_album_key"`
	ShareURL       string `json:"share_url"`
	ShareToken     string `json:"share_token"`
}

// AIEnhanceResult represents one of the AI enhanced variations returned by Google Photos Magic Editor
type AIEnhanceResult struct {
	VariationID int    `json:"variation_id"`
	URL         string `json:"url"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	DataURL     string `json:"data_url,omitempty"` // data:image/jpeg;base64,...
	Size        int    `json:"size,omitempty"`
}

