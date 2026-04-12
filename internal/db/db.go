package db

import (
	"database/sql"

	_ "modernc.org/sqlite"

	"github.com/agusibrahim/gpmc-go/internal/models"
)

// Storage handles all database operations
type Storage struct {
	db *sql.DB
}

// New creates a new Storage instance
func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Storage{db: db}, nil
}

// createTables creates the database tables if they don't exist
func createTables(db *sql.DB) error {
	// Create remote_media table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS remote_media (
			media_key TEXT PRIMARY KEY,
			file_name TEXT,
			dedup_key TEXT,
			is_canonical BOOL,
			type INTEGER,
			caption TEXT,
			collection_id TEXT,
			size_bytes INTEGER,
			quota_charged_bytes INTEGER,
			origin TEXT,
			content_version INTEGER,
			utc_timestamp INTEGER,
			server_creation_timestamp INTEGER,
			timezone_offset INTEGER,
			width INTEGER,
			height INTEGER,
			remote_url TEXT,
			upload_status INTEGER,
			trash_timestamp INTEGER,
			is_archived INTEGER,
			is_favorite INTEGER,
			is_locked INTEGER,
			is_original_quality INTEGER,
			latitude REAL,
			longitude REAL,
			location_name TEXT,
			location_id TEXT,
			is_edited INTEGER,
			make TEXT,
			model TEXT,
			aperture REAL,
			shutter_speed REAL,
			iso INTEGER,
			focal_length REAL,
			duration INTEGER,
			capture_frame_rate REAL,
			encoded_frame_rate REAL,
			is_micro_video INTEGER,
			micro_video_width INTEGER,
			micro_video_height INTEGER
		)
	`)
	if err != nil {
		return err
	}

	// Create state table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS state (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			sync_token TEXT,
			resume_token TEXT,
			init_complete INTEGER
		)
	`)
	if err != nil {
		return err
	}

	// Initialize state row
	_, err = db.Exec(`
		INSERT OR IGNORE INTO state (id, sync_token, resume_token, init_complete)
		VALUES (1, '', '', 0)
	`)

	return err
}

// Update inserts or updates multiple MediaItems in the database
func (s *Storage) Update(items []models.MediaItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO remote_media (
			media_key, file_name, dedup_key, is_canonical, type, caption, collection_id,
			size_bytes, quota_charged_bytes, origin, content_version, utc_timestamp,
			server_creation_timestamp, timezone_offset, width, height, remote_url,
			upload_status, trash_timestamp, is_archived, is_favorite, is_locked,
			is_original_quality, latitude, longitude, location_name, location_id,
			is_edited, make, model, aperture, shutter_speed, iso, focal_length,
			duration, capture_frame_rate, encoded_frame_rate, is_micro_video,
			micro_video_width, micro_video_height
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(media_key) DO UPDATE SET
			file_name = excluded.file_name,
			dedup_key = excluded.dedup_key,
			is_canonical = excluded.is_canonical,
			type = excluded.type,
			caption = excluded.caption,
			collection_id = excluded.collection_id,
			size_bytes = excluded.size_bytes,
			quota_charged_bytes = excluded.quota_charged_bytes,
			origin = excluded.origin,
			content_version = excluded.content_version,
			utc_timestamp = excluded.utc_timestamp,
			server_creation_timestamp = excluded.server_creation_timestamp,
			timezone_offset = excluded.timezone_offset,
			width = excluded.width,
			height = excluded.height,
			remote_url = excluded.remote_url,
			upload_status = excluded.upload_status,
			trash_timestamp = excluded.trash_timestamp,
			is_archived = excluded.is_archived,
			is_favorite = excluded.is_favorite,
			is_locked = excluded.is_locked,
			is_original_quality = excluded.is_original_quality,
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			location_name = excluded.location_name,
			location_id = excluded.location_id,
			is_edited = excluded.is_edited,
			make = excluded.make,
			model = excluded.model,
			aperture = excluded.aperture,
			shutter_speed = excluded.shutter_speed,
			iso = excluded.iso,
			focal_length = excluded.focal_length,
			duration = excluded.duration,
			capture_frame_rate = excluded.capture_frame_rate,
			encoded_frame_rate = excluded.encoded_frame_rate,
			is_micro_video = excluded.is_micro_video,
			micro_video_width = excluded.micro_video_width,
			micro_video_height = excluded.micro_video_height
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		_, err := stmt.Exec(
			item.MediaKey, item.FileName, item.DedupKey, item.IsCanonical, item.Type,
			item.Caption, item.CollectionID, item.SizeBytes, item.QuotaChargedBytes,
			item.Origin, item.ContentVersion, item.UTCTimestamp, item.ServerCreationTimestamp,
			item.TimezoneOffset, item.Width, item.Height, item.RemoteURL, item.UploadStatus,
			item.TrashTimestamp, item.IsArchived, item.IsFavorite, item.IsLocked,
			item.IsOriginalQuality, item.Latitude, item.Longitude, item.LocationName,
			item.LocationID, item.IsEdited, item.Make, item.Model, item.Aperture,
			item.ShutterSpeed, item.ISO, item.FocalLength, item.Duration,
			item.CaptureFrameRate, item.EncodedFrameRate, item.IsMicroVideo,
			item.MicroVideoWidth, item.MicroVideoHeight,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete removes multiple media items by their media keys
func (s *Storage) Delete(mediaKeys []string) error {
	if len(mediaKeys) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Build placeholder string
	placeholders := ""
	for i := 0; i < len(mediaKeys); i++ {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
	}

	query := "DELETE FROM remote_media WHERE media_key IN (" + placeholders + ")"
	_, err = tx.Exec(query, toInterfaceSlice(mediaKeys)...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetSyncTokens returns both sync tokens as a tuple (sync_token, resume_token)
func (s *Storage) GetSyncTokens() (string, string, error) {
	var syncToken, resumeToken string
	err := s.db.QueryRow("SELECT sync_token, resume_token FROM state WHERE id = 1").Scan(&syncToken, &resumeToken)
	if err != nil {
		return "", "", err
	}
	return syncToken, resumeToken, nil
}

// UpdateSyncTokens updates one or both sync tokens
// Pass nil for a token to leave it unchanged
func (s *Storage) UpdateSyncTokens(syncToken, resumeToken *string) error {
	if syncToken == nil && resumeToken == nil {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if syncToken != nil && resumeToken != nil {
		_, err = tx.Exec("UPDATE state SET sync_token = ?, resume_token = ? WHERE id = 1", *syncToken, *resumeToken)
	} else if syncToken != nil {
		_, err = tx.Exec("UPDATE state SET sync_token = ? WHERE id = 1", *syncToken)
	} else if resumeToken != nil {
		_, err = tx.Exec("UPDATE state SET resume_token = ? WHERE id = 1", *resumeToken)
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetInitState returns the init_complete flag
func (s *Storage) GetInitState() (bool, error) {
	var initState int
	err := s.db.QueryRow("SELECT init_complete FROM state WHERE id = 1").Scan(&initState)
	if err != nil {
		return false, err
	}
	return initState == 1, nil
}

// SetInitState sets the init_complete flag
func (s *Storage) SetInitState(state int) error {
	_, err := s.db.Exec("UPDATE state SET init_complete = ? WHERE id = 1", state)
	return err
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// Helper function to convert string slice to interface slice for SQL queries
func toInterfaceSlice(strings []string) []interface{} {
	result := make([]interface{}, len(strings))
	for i, s := range strings {
		result[i] = s
	}
	return result
}
