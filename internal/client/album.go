package client

import (
	"fmt"

	"github.com/agusibrahim/gpmc-go/internal/models"
	"github.com/schollz/progressbar/v3"
)

// addToAlbum adds media items to an album
func (c *Client) addToAlbum(mediaKeys []string, albumName string, showProgress bool) []string {
	const (
		albumLimit = 20000 // Maximum items per album
		batchSize  = 500   // Items per API call
	)

	var albumKeys []string

	if len(mediaKeys) > albumLimit {
		c.logger.Warn(fmt.Sprintf("%d items exceed album limit of %d. They will be split into multiple albums.", len(mediaKeys), albumLimit))
	}

	// Create progress bar
	var bar *progressbar.ProgressBar
	if showProgress {
		bar = progressbar.Default(int64(len(mediaKeys)), "Adding to album")
		defer bar.Close()
	}

	// Process in album-sized batches
	for i := 0; i < len(mediaKeys); i += albumLimit {
		end := i + albumLimit
		if end > len(mediaKeys) {
			end = len(mediaKeys)
		}
		albumBatch := mediaKeys[i:end]

		// Add suffix if more than one album
		currentAlbumName := albumName
		if len(mediaKeys) > albumLimit {
			currentAlbumName = fmt.Sprintf("%s %d", albumName, (i/albumLimit)+1)
		}

		// Create album with first batch, then add remaining batches
		var currentAlbumKey string
		for j := 0; j < len(albumBatch); j += batchSize {
			batchEnd := j + batchSize
			if batchEnd > len(albumBatch) {
				batchEnd = len(albumBatch)
			}
			batch := albumBatch[j:batchEnd]

			if currentAlbumKey == "" {
				// Create album with first batch
				key, err := c.api.CreateAlbum(currentAlbumName, batch)
				if err != nil {
					c.logger.Error("Failed to create album", "error", err)
					continue
				}
				currentAlbumKey = key
				albumKeys = append(albumKeys, key)
			} else {
				// Add to existing album
				_, err := c.api.AddMediaToAlbum(currentAlbumKey, batch)
				if err != nil {
					c.logger.Error("Failed to add to album", "error", err)
					continue
				}
			}

			// Update progress
			if bar != nil {
				bar.Add(len(batch))
			}
		}
	}

	return albumKeys
}

// AddToAlbum adds media items to an existing album (legacy API)
func (c *Client) AddToAlbum(mediaKeys []string, albumName string, showProgress bool) []string {
	return c.addToAlbum(mediaKeys, albumName, showProgress)
}

// ListAlbums fetches all albums for the authenticated user
func (c *Client) ListAlbums() ([]models.Album, error) {
	return c.api.ListAlbums()
}

// ListPhotosInAlbum fetches all photos and videos inside an album
func (c *Client) ListPhotosInAlbum(albumKey string, shareToken ...string) ([]models.AlbumMediaItem, error) {
	token := ""
	if len(shareToken) > 0 {
		token = shareToken[0]
	}
	return c.api.ListPhotosInAlbum(albumKey, token)
}

// AddCommentToAlbum adds a comment to an album
func (c *Client) AddCommentToAlbum(albumKey, commentText string, shareToken ...string) (string, error) {
	token := ""
	if len(shareToken) > 0 {
		token = shareToken[0]
	}
	return c.api.AddCommentToAlbum(albumKey, commentText, token)
}

// AddPhotoToAlbum links a media key to an existing album
func (c *Client) AddPhotoToAlbum(albumKey, mediaKey string, shareToken ...string) error {
	token := ""
	if len(shareToken) > 0 {
		token = shareToken[0]
	}
	return c.api.AddMediaListToAlbum(albumKey, []string{mediaKey}, token)
}

// UploadPhotoToAlbum uploads a photo and attaches it to the specified album
func (c *Client) UploadPhotoToAlbum(albumKey, filePath string, shareToken ...string) (string, error) {
	results, err := c.Upload(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to upload photo: %w", err)
	}

	mediaKey := ""
	for _, k := range results {
		if k != "" {
			mediaKey = k
			break
		}
	}

	if mediaKey == "" {
		return "", fmt.Errorf("upload finished but no mediaKey was returned")
	}

	token := ""
	if len(shareToken) > 0 {
		token = shareToken[0]
	}

	if err := c.api.AddMediaListToAlbum(albumKey, []string{mediaKey}, token); err != nil {
		return mediaKey, fmt.Errorf("photo uploaded (mediaKey: %s) but failed to link to album: %w", mediaKey, err)
	}

	return mediaKey, nil
}

// CreateAlbum creates a new album with optional media keys
func (c *Client) CreateAlbum(albumName string, mediaKeys ...string) (string, error) {
	return c.api.CreateAlbum(albumName, mediaKeys)
}

// RenameAlbum changes the title of an album
func (c *Client) RenameAlbum(albumKey, newTitle string) error {
	return c.api.RenameAlbum(albumKey, newTitle)
}

// SetAlbumDescription updates the narrative / description text of an album
func (c *Client) SetAlbumDescription(albumKey, description string) error {
	return c.api.SetAlbumDescription(albumKey, description)
}

// ShareAlbum generates a public share link and share token for an album
func (c *Client) ShareAlbum(albumKey string) (*models.ShareAlbumResult, error) {
	return c.api.ShareAlbum(albumKey)
}

// DeleteAlbum deletes either a regular or shared album
func (c *Client) DeleteAlbum(albumKey string, isShared ...bool) error {
	shared := false
	if len(isShared) > 0 {
		shared = isShared[0]
	}
	return c.api.DeleteAlbum(albumKey, shared)
}
