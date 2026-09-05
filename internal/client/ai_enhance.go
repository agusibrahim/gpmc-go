package client

import (
	"github.com/agusibrahim/gpmc-go/internal/models"
)

// EnhancePhoto sends an image to Google Photos AI Magic Editor Preset
// and returns all high-resolution AI-enhanced variations.
func (c *Client) EnhancePhoto(imgBytes []byte) ([]models.AIEnhanceResult, error) {
	return c.api.EnhancePhoto(imgBytes)
}
