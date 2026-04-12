package parser

import (
	"encoding/base64"
	"strings"

	"github.com/agusibrahim/gpmc-go/internal/models"
	"github.com/agusibrahim/gpmc-go/internal/util"
)

// ParseMediaItem parses a single media item from the raw decoded protobuf data
func ParseMediaItem(data map[string]interface{}) (models.MediaItem, error) {
	item := models.MediaItem{}

	// Extract primary fields
	if v, ok := data["1"].(string); ok {
		item.MediaKey = v
	}

	// Extract field "2" which contains most of the metadata
	if field2, ok := data["2"].(map[string]interface{}); ok {
		// Extract caption from field "2" sub-fields (keys starting with "3")
		item.Caption = extractFirstMatchingString(field2, "3")

		// File name
		if v, ok := field2["4"].(string); ok {
			item.FileName = v
		}

		// Dedup key - complex extraction logic
		item.DedupKey = extractDedupKey(field2)

		// Collection ID
		if v, ok := getNestedValue(field2, []string{"1", "1"}); ok {
			if s, ok := v.(string); ok {
				item.CollectionID = s
			}
		}

		// Size and quota
		if v, ok := field2["10"].(int64); ok {
			item.SizeBytes = v
		}
		if v, ok := field2["35"].(map[string]interface{}); ok {
			if qv, ok := v["2"].(int64); ok {
				item.QuotaChargedBytes = qv
			}
			if qv, ok := v["3"].(int64); ok {
				item.IsOriginalQuality = qv == 2
			}
		}

		// Origin
		if v, ok := getNestedValue(field2, []string{"30", "1"}); ok {
			if originMap, ok := v.(int64); ok {
				item.Origin = originMapToString(int(originMap))
			}
		}

		// Content version
		if v, ok := field2["26"].(int64); ok {
			item.ContentVersion = int(v)
		}

		// Timestamps
		if v, ok := field2["7"].(int64); ok {
			item.UTCTimestamp = v
		}
		if v, ok := field2["9"].(int64); ok {
			item.ServerCreationTimestamp = v
		}
		if v, ok := field2["8"].(int64); ok {
			item.TimezoneOffset = new(int)
			*item.TimezoneOffset = int(v)
		}

		// Trash timestamp
		if v, ok := field2["16"].(map[string]interface{}); ok {
			if tv, ok := v["3"].(int64); ok {
				item.TrashTimestamp = new(int)
				*item.TrashTimestamp = int(tv)
			}
		}

		// Status flags
		if v, ok := getNestedValue(field2, []string{"29", "1"}); ok {
			if iv, ok := v.(int64); ok {
				item.IsArchived = iv == 1
			}
		}
		if v, ok := getNestedValue(field2, []string{"31", "1"}); ok {
			if iv, ok := v.(int64); ok {
				item.IsFavorite = iv == 1
			}
		}
		if v, ok := getNestedValue(field2, []string{"39", "1"}); ok {
			if iv, ok := v.(int64); ok {
				item.IsLocked = iv == 1
			}
		}
	}

	// Extract field "5" which contains type-specific data
	if field5, ok := data["5"].(map[string]interface{}); ok {
		// Type indicator
		if v, ok := field5["1"].(int64); ok {
			item.Type = int(v)
		}

		// Photo data (field "2")
		if photoData, ok := field5["2"].(map[string]interface{}); ok {
			item.IsEdited = hasKey(photoData, "4")
			if v, ok := getNestedValue(photoData, []string{"1", "1"}); ok {
				item.RemoteURL = v.(string)
			}
			if v, ok := getNestedValue(photoData, []string{"1", "9", "1"}); ok {
				if iv, ok := v.(int64); ok {
					item.Width = new(int)
					*item.Width = int(iv)
				}
			}
			if v, ok := getNestedValue(photoData, []string{"1", "9", "2"}); ok {
				if iv, ok := v.(int64); ok {
					item.Height = new(int)
					*item.Height = int(iv)
				}
			}
			// Camera EXIF data
			exifPath := []string{"1", "9", "5"}
			if make, ok := getNestedValue(photoData, append(exifPath, "1")); ok {
				if s, ok := make.(string); ok {
					item.Make = new(string)
					*item.Make = s
				}
			}
			if model, ok := getNestedValue(photoData, append(exifPath, "2")); ok {
				if s, ok := model.(string); ok {
					item.Model = new(string)
					*item.Model = s
				}
			}
			if aperture, ok := getNestedValue(photoData, append(exifPath, "4")); ok {
				if iv, ok := aperture.(int32); ok {
					item.Aperture = new(float64)
					*item.Aperture = float64(util.Int32ToFloat(iv))
				}
			}
		}

		// Video data (field "3")
		if videoData, ok := field5["3"].(map[string]interface{}); ok {
			if v, ok := getNestedValue(videoData, []string{"2", "1"}); ok {
				item.RemoteURL = v.(string)
			}
			if v, ok := getNestedValue(videoData, []string{"4", "1"}); ok {
				if iv, ok := v.(int64); ok {
					item.Duration = new(int)
					*item.Duration = int(iv)
				}
			}
			if v, ok := getNestedValue(videoData, []string{"4", "4"}); ok {
				if iv, ok := v.(int64); ok {
					item.Width = new(int)
					*item.Width = int(iv)
				}
			}
			if v, ok := getNestedValue(videoData, []string{"4", "5"}); ok {
				if iv, ok := v.(int64); ok {
					item.Height = new(int)
					*item.Height = int(iv)
				}
			}
			if v, ok := getNestedValue(videoData, []string{"6", "4"}); ok {
				if iv, ok := v.(int64); ok {
					f := util.Int64ToFloat(iv)
					item.CaptureFrameRate = new(float64)
					*item.CaptureFrameRate = f
				}
			}
			if v, ok := getNestedValue(videoData, []string{"6", "5"}); ok {
				if iv, ok := v.(int64); ok {
					f := util.Int64ToFloat(iv)
					item.EncodedFrameRate = new(float64)
					*item.EncodedFrameRate = f
				}
			}
		}
	}

	// Extract location data (field "17")
	if field17, ok := data["17"].(map[string]interface{}); ok {
		if v, ok := getNestedValue(field17, []string{"1", "1"}); ok {
			if iv, ok := v.(int32); ok {
				f := util.Fixed32ToFloat(iv)
				item.Latitude = new(float64)
				*item.Latitude = f
			}
		}
		if v, ok := getNestedValue(field17, []string{"1", "2"}); ok {
			if iv, ok := v.(int32); ok {
				f := util.Fixed32ToFloat(iv)
				item.Longitude = new(float64)
				*item.Longitude = f
			}
		}
		if v, ok := getNestedValue(field17, []string{"5", "2", "1"}); ok {
			if s, ok := v.(string); ok {
				item.LocationName = new(string)
				*item.LocationName = s
			}
		}
		if v, ok := getNestedValue(field17, []string{"5", "3"}); ok {
			if s, ok := v.(string); ok {
				item.LocationID = new(string)
				*item.LocationID = s
			}
		}
	}

	return item, nil
}

// ParseDeletionItem parses a deletion item from the raw data
func ParseDeletionItem(data map[string]interface{}) (string, error) {
	if field1, ok := data["1"].(map[string]interface{}); ok {
		if v, ok := field1["1"].(int64); ok && v == 1 {
			if v, ok := getNestedValue(field1, []string{"2", "1"}); ok {
				if s, ok := v.(string); ok {
					return s, nil
				}
			}
		}
	}
	return "", nil
}

// ParseDBUpdate parses the library state update from raw protobuf data
func ParseDBUpdate(data map[string]interface{}) (syncToken string, resumeToken string, items []models.MediaItem, deletions []string, err error) {
	// Extract field "1" which contains the main response data
	if field1, ok := data["1"].(map[string]interface{}); ok {
		// Resume token (field "1")
		if v, ok := field1["1"].(string); ok {
			resumeToken = v
		}

		// Sync token (field "6")
		if v, ok := field1["6"].(string); ok {
			syncToken = v
		}

		// Media items (field "2") - can be single item or array
		if v, ok := field1["2"]; ok {
			items = parseMediaItemsList(v)
		}

		// Deletions (field "9") - can be single item or array
		if v, ok := field1["9"]; ok {
			deletions = parseDeletionsList(v)
		}
	}

	return syncToken, resumeToken, items, deletions, nil
}

// Helper functions

func extractDedupKey(field2 map[string]interface{}) string {
	// Try to get from field "21" sub-fields (keys starting with "1")
	if field21, ok := field2["21"].(map[string]interface{}); ok {
		for k, v := range field21 {
			if strings.HasPrefix(k, "1") {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
	}

	// Fallback: construct from field "13" sub-field "1"
	if field13, ok := field2["13"].(map[string]interface{}); ok {
		if v, ok := field13["1"].([]byte); ok {
			return util.URLSafeBase64(base64.StdEncoding.EncodeToString(v))
		}
	}

	return ""
}

func extractFirstMatchingString(data map[string]interface{}, prefix string) string {
	for k, v := range data {
		if strings.HasPrefix(k, prefix) {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func getNestedValue(data map[string]interface{}, path []string) (interface{}, bool) {
	current := interface{}(data)
	for _, key := range path {
		if m, ok := current.(map[string]interface{}); ok {
			var exists bool
			current, exists = m[key]
			if !exists {
				return nil, false
			}
		} else {
			return nil, false
		}
	}
	return current, true
}

func hasKey(data map[string]interface{}, key string) bool {
	_, ok := data[key]
	return ok
}

func originMapToString(origin int) string {
	switch origin {
	case 1:
		return "self"
	case 3:
		return "partner"
	case 4:
		return "shared"
	default:
		return "unknown"
	}
}

func parseMediaItemsList(v interface{}) []models.MediaItem {
	var items []models.MediaItem

	switch val := v.(type) {
	case map[string]interface{}:
		// Single item
		item, err := ParseMediaItem(val)
		if err == nil {
			items = append(items, item)
		}
	case []interface{}:
		// Array of items
		for _, item := range val {
			if itemMap, ok := item.(map[string]interface{}); ok {
				mediaItem, err := ParseMediaItem(itemMap)
				if err == nil {
					items = append(items, mediaItem)
				}
			}
		}
	}

	return items
}

func parseDeletionsList(v interface{}) []string {
	var deletions []string

	switch val := v.(type) {
	case map[string]interface{}:
		// Single deletion
		key, err := ParseDeletionItem(val)
		if err == nil && key != "" {
			deletions = append(deletions, key)
		}
	case []interface{}:
		// Array of deletions
		for _, item := range val {
			if itemMap, ok := item.(map[string]interface{}); ok {
				key, err := ParseDeletionItem(itemMap)
				if err == nil && key != "" {
					deletions = append(deletions, key)
				}
			}
		}
	}

	return deletions
}

// Fix typo in variable name
var exifPath = []string{"1", "9", "5"}
