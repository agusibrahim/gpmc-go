package client

import (
	"fmt"

	"github.com/schollz/progressbar/v3"
)

// addToAlbum adds media items to an album
func (c *Client) addToAlbum(mediaKeys []string, albumName string, showProgress bool) []string {
	const (
		albumLimit  = 20000 // Maximum items per album
		batchSize   = 500   // Items per API call
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

// AddToAlbum adds media items to an existing album (public API)
func (c *Client) AddToAlbum(mediaKeys []string, albumName string, showProgress bool) []string {
	return c.addToAlbum(mediaKeys, albumName, showProgress)
}
