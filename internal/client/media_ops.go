package client

import (
	"fmt"

	"github.com/agusibrahim/gpmc-go/internal/hash"
	"github.com/agusibrahim/gpmc-go/internal/util"
)

// MoveToTrash moves remote media files to trash
func (c *Client) MoveToTrash(sha1Hashes interface{}) (map[string]interface{}, error) {
	dedupKeys, err := c.sha1HashesToDedupKeys(sha1Hashes)
	if err != nil {
		return nil, err
	}

	// Process in batches of 500
	const batchSize = 500
	var result map[string]interface{}

	for i := 0; i < len(dedupKeys); i += batchSize {
		end := i + batchSize
		if end > len(dedupKeys) {
			end = len(dedupKeys)
		}
		batch := dedupKeys[i:end]

		batchResult, err := c.api.MoveRemoteMediaToTrash(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to move items to trash: %w", err)
		}

		// Merge results
		if result == nil {
			result = batchResult
		} else {
			for k, v := range batchResult {
				result[k] = v
			}
		}
	}

	return result, nil
}

// DeletePermanently permanently deletes remote media files
func (c *Client) DeletePermanently(sha1Hashes interface{}) (map[string]interface{}, error) {
	dedupKeys, err := c.sha1HashesToDedupKeys(sha1Hashes)
	if err != nil {
		return nil, err
	}

	// Process in batches of 500
	const batchSize = 500
	var result map[string]interface{}

	for i := 0; i < len(dedupKeys); i += batchSize {
		end := i + batchSize
		if end > len(dedupKeys) {
			end = len(dedupKeys)
		}
		batch := dedupKeys[i:end]

		batchResult, err := c.api.DeleteRemoteMediaPermanently(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to delete items permanently: %w", err)
		}

		// Merge results
		if result == nil {
			result = batchResult
		} else {
			for k, v := range batchResult {
				result[k] = v
			}
		}
	}

	return result, nil
}

// RestoreFromTrash restores media files from trash
func (c *Client) RestoreFromTrash(sha1Hashes interface{}) (map[string]interface{}, error) {
	dedupKeys, err := c.sha1HashesToDedupKeys(sha1Hashes)
	if err != nil {
		return nil, err
	}

	// Process in batches of 500
	const batchSize = 500
	var result map[string]interface{}

	for i := 0; i < len(dedupKeys); i += batchSize {
		end := i + batchSize
		if end > len(dedupKeys) {
			end = len(dedupKeys)
		}
		batch := dedupKeys[i:end]

		batchResult, err := c.api.RestoreFromTrash(batch)
		if err != nil {
			return nil, fmt.Errorf("failed to restore items from trash: %w", err)
		}

		// Merge results
		if result == nil {
			result = batchResult
		} else {
			for k, v := range batchResult {
				result[k] = v
			}
		}
	}

	return result, nil
}

// SetFavorite sets or unsets the favorite flag on a media item
func (c *Client) SetFavorite(dedupKey string, isFavorite bool) (map[string]interface{}, error) {
	return c.api.SetFavorite(dedupKey, isFavorite)
}

// SetArchived sets or unsets the archived flag on media items
func (c *Client) SetArchived(dedupKeys []string, isArchived bool) (map[string]interface{}, error) {
	return c.api.SetArchived(dedupKeys, isArchived)
}

// SetCaption sets the caption for a media item
func (c *Client) SetCaption(caption, dedupKey string) (map[string]interface{}, error) {
	return c.api.SetItemCaption(caption, dedupKey)
}

// GetDownloadURLs gets download URLs for a media item
func (c *Client) GetDownloadURLs(mediaKey string) ([]string, error) {
	return c.api.GetDownloadURLs(mediaKey)
}

// sha1HashesToDedupKeys converts SHA-1 hashes to URL-safe base64 dedup keys
func (c *Client) sha1HashesToDedupKeys(sha1Hashes interface{}) ([]string, error) {
	var hashes []interface{}

	switch v := sha1Hashes.(type) {
	case string:
		hashes = []interface{}{v}
	case []byte:
		hashes = []interface{}{v}
	case []string:
		for _, h := range v {
			hashes = append(hashes, h)
		}
	case [][]byte:
		for _, h := range v {
			hashes = append(hashes, h)
		}
	default:
		return nil, fmt.Errorf("invalid hash type: %T", sha1Hashes)
	}

	var dedupKeys []string
	for _, h := range hashes {
		// Convert to bytes and base64
		hashBytes, _, err := hash.ConvertSHA1Hash(h)
		if err != nil {
			return nil, fmt.Errorf("failed to convert hash: %w", err)
		}

		// Convert to URL-safe base64
		hashB64 := hash.BytesToBase64(hashBytes)
		dedupKey := util.URLSafeBase64(hashB64)
		dedupKeys = append(dedupKeys, dedupKey)
	}

	return dedupKeys, nil
}
