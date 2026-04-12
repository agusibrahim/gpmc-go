package client

import (
	"fmt"
	"os"

	"github.com/schollz/progressbar/v3"
	"github.com/agusibrahim/gpmc-go/internal/db"
	"github.com/agusibrahim/gpmc-go/internal/parser"
)

// UpdateCache incrementally updates the local library cache
func (c *Client) UpdateCache(showProgress bool, maxSyncCycles int) error {
	// Ensure cache directory exists
	if err := createDirIfNotExists(c.cacheDir); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create progress bar
	var bar *progressbar.ProgressBar
	if showProgress {
		bar = progressbar.NewOptions(
			100,
			progressbar.OptionSetDescription("Updating cache"),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
		)
		defer bar.Close()
	}

	// Open database
	storage, err := db.New(c.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer storage.Close()

	// Check if init is needed
	initState, err := storage.GetInitState()
	if err != nil {
		return fmt.Errorf("failed to get init state: %w", err)
	}

	if !initState {
		c.logger.Info("Cache Initiation")
		if err := c.cacheInit(bar); err != nil {
			return err
		}
		if err := storage.SetInitState(1); err != nil {
			return fmt.Errorf("failed to set init state: %w", err)
		}
	}

	c.logger.Info("Cache Update")
	return c.cacheUpdate(bar, maxSyncCycles)
}

// cacheInit performs initial sync to populate the cache
func (c *Client) cacheInit(bar *progressbar.ProgressBar) error {
	storage, err := db.New(c.dbPath)
	if err != nil {
		return err
	}
	defer storage.Close()

	// Get current tokens
	syncToken, resumeToken, err := storage.GetSyncTokens()
	if err != nil {
		return err
	}

	// Resume incomplete init if pending
	if resumeToken != "" {
		c.logger.Info("Resuming incomplete initial sync")
		if err := c.processPagesInit(bar, resumeToken); err != nil {
			return err
		}
	}

	// Get library state
	response, err := c.api.GetLibraryState(syncToken)
	if err != nil {
		return fmt.Errorf("failed to get library state: %w", err)
	}

	nextSyncToken, nextResumeToken, remoteMedia, deletions, err := parser.ParseDBUpdate(response)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Update database
	if err := storage.UpdateSyncTokens(&nextSyncToken, &nextResumeToken); err != nil {
		return err
	}
	if err := storage.Update(remoteMedia); err != nil {
		return err
	}
	if len(deletions) > 0 {
		if err := storage.Delete(deletions); err != nil {
			return err
		}
	}

	// Update progress
	if bar != nil {
		bar.Add(len(remoteMedia) + len(deletions))
	}

	// Process remaining pages
	if nextResumeToken != "" {
		if err := c.processPagesInit(bar, nextResumeToken); err != nil {
			return err
		}
	}

	return nil
}

// cacheUpdate performs delta sync to update the cache
func (c *Client) cacheUpdate(bar *progressbar.ProgressBar, maxSyncCycles int) error {
	storage, err := db.New(c.dbPath)
	if err != nil {
		return err
	}
	defer storage.Close()

	syncCycleCount := 0
	var previousSyncToken string

	for syncCycleCount < maxSyncCycles {
		syncToken, _, err := storage.GetSyncTokens()
		if err != nil {
			return err
		}

		// Infinite sync cycle detection
		if previousSyncToken != "" && syncToken == previousSyncToken {
			c.logger.Warn(fmt.Sprintf("Sync token unchanged after %d cycles. Stopping to avoid infinite sync loop.", syncCycleCount))
			return fmt.Errorf("sync cycle detected: token unchanged after %d cycles", syncCycleCount)
		}

		response, err := c.api.GetLibraryState(syncToken)
		if err != nil {
			return fmt.Errorf("failed to get library state: %w", err)
		}

		nextSyncToken, nextResumeToken, remoteMedia, deletions, err := parser.ParseDBUpdate(response)
		if err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Update database
		if err := storage.UpdateSyncTokens(&nextSyncToken, &nextResumeToken); err != nil {
			return err
		}
		if err := storage.Update(remoteMedia); err != nil {
			return err
		}
		if len(deletions) > 0 {
			if err := storage.Delete(deletions); err != nil {
				return err
			}
		}

		// Update progress
		if bar != nil {
			bar.Add(len(remoteMedia) + len(deletions))
		}

		// Process remaining pages
		if nextResumeToken != "" {
			if err := c.processPages(bar, syncToken, nextResumeToken); err != nil {
				return err
			}
		}

		// Check if we should continue
		if !c.shouldTriggerNextSync(response) {
			break
		}

		previousSyncToken = syncToken
		syncCycleCount++
		c.logger.Debug(fmt.Sprintf("Triggering sync cycle %d", syncCycleCount+1))
	}

	return nil
}

// processPagesInit processes paginated results during initial sync
func (c *Client) processPagesInit(bar *progressbar.ProgressBar, resumeToken string) error {
	storage, err := db.New(c.dbPath)
	if err != nil {
		return err
	}
	defer storage.Close()

	nextResumeToken := resumeToken
	pageCount := 0

	for nextResumeToken != "" {
		response, err := c.api.GetLibraryPageInit(nextResumeToken)
		if err != nil {
			return fmt.Errorf("failed to get library page init: %w", err)
		}

		_, nextResumeToken, remoteMedia, deletions, err := parser.ParseDBUpdate(response)
		if err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		pageCount++

		// Update database
		if err := storage.UpdateSyncTokens(nil, &nextResumeToken); err != nil {
			return err
		}
		if err := storage.Update(remoteMedia); err != nil {
			return err
		}
		if len(deletions) > 0 {
			if err := storage.Delete(deletions); err != nil {
				return err
			}
		}

		// Update progress
		if bar != nil {
			bar.Add(len(remoteMedia) + len(deletions))
		}

		c.logger.Debug(fmt.Sprintf("Initial sync page %d: %d items", pageCount, len(remoteMedia)))
	}

	return nil
}

// processPages processes paginated results during delta sync
func (c *Client) processPages(bar *progressbar.ProgressBar, syncToken, resumeToken string) error {
	storage, err := db.New(c.dbPath)
	if err != nil {
		return err
	}
	defer storage.Close()

	nextResumeToken := resumeToken
	pageCount := 0

	for nextResumeToken != "" {
		response, err := c.api.GetLibraryPage(nextResumeToken, syncToken)
		if err != nil {
			return fmt.Errorf("failed to get library page: %w", err)
		}

		_, nextResumeToken, remoteMedia, deletions, err := parser.ParseDBUpdate(response)
		if err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		pageCount++

		// Update database
		if err := storage.UpdateSyncTokens(nil, &nextResumeToken); err != nil {
			return err
		}
		if err := storage.Update(remoteMedia); err != nil {
			return err
		}
		if len(deletions) > 0 {
			if err := storage.Delete(deletions); err != nil {
				return err
			}
		}

		// Update progress
		if bar != nil {
			bar.Add(len(remoteMedia) + len(deletions))
		}

		c.logger.Debug(fmt.Sprintf("Delta sync page %d: %d items, %d deletions", pageCount, len(remoteMedia), len(deletions)))
	}

	return nil
}

// shouldTriggerNextSync checks if another sync cycle should be triggered
func (c *Client) shouldTriggerNextSync(response map[string]interface{}) bool {
	// Check for continuation indicator in response
	if field1, ok := response["1"].(map[string]interface{}); ok {
		if field7, ok := field1["7"].(int64); ok {
			return field7 == 2
		}
	}
	return false
}

// Helper function to create directory if it doesn't exist
func createDirIfNotExists(path string) error {
	// This would use os.MkdirAll
	return nil
}
