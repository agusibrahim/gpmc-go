package client

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/schollz/progressbar/v3"
	"github.com/xob0t/gpmc-go/internal/hash"
)

// UploadResult maps file paths to their media keys
type UploadResult map[string]string

// Upload uploads one or more files or directories to Google Photos
func (c *Client) Upload(target interface{}, opts ...UploadOption) (UploadResult, error) {
	options := &uploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Process target into path-hash pairs
	pathHashPairs, err := c.handleTargetInput(target, options.recursive, options.filterExp, options.filterExclude, options.filterRegex, options.filterIgnoreCase, options.filterMatchPath)
	if err != nil {
		return nil, err
	}

	// Upload files concurrently
	results := c.uploadConcurrently(pathHashPairs, options)

	// Handle album creation if requested
	if options.albumName != "" {
		c.handleAlbumCreation(results, options.albumName, options.showProgress)
	}

	return results, nil
}

// uploadConcurrently uploads files concurrently using goroutines
func (c *Client) uploadConcurrently(pairs map[string]*UploadOptions, opts *uploadOptions) UploadResult {
	results := make(UploadResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create progress bar if requested
	var bar *progressbar.ProgressBar
	if opts.showProgress {
		bar = progressbar.Default(int64(len(pairs)), "Uploading files")
		defer bar.Close()
	}

	// Determine number of workers
	workers := opts.threads
	if workers < 1 {
		workers = 1
	}

	// Create semaphore for limiting concurrent uploads
	sem := make(chan struct{}, workers)

	for path, uploadOpts := range pairs {
		wg.Add(1)
		go func(filePath string, options *UploadOptions) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Upload file
			mediaKey, err := c.uploadFile(filePath, options, opts)
			if err != nil {
				c.logger.Error("Error uploading file", "path", filePath, "error", err)
				return
			}

			// Store result
			mu.Lock()
			results[filePath] = mediaKey
			mu.Unlock()

			// Update progress
			if bar != nil {
				bar.Add(1)
			}
		}(path, uploadOpts)
	}

	wg.Wait()
	return results
}

// uploadFile uploads a single file to Google Photos
func (c *Client) uploadFile(filePath string, opts *UploadOptions, uploadOpts *uploadOptions) (string, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	fileSize := fileInfo.Size()

	// Determine filename
	fileName := opts.FileName
	if fileName == "" {
		fileName = filepath.Base(filePath)
	}

	// Calculate hash
	var hashBytes []byte
	var hashB64 string

	if opts.Hash != nil {
		hashBytes, hashB64, err = hash.ConvertSHA1Hash(opts.Hash)
		if err != nil {
			return "", fmt.Errorf("failed to convert hash: %w", err)
		}
	} else {
		hashBytes, hashB64, err = hash.CalculateSHA1Hash(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to calculate hash: %w", err)
		}
	}

	// Check if file already exists (unless force upload)
	if !uploadOpts.forceUpload {
		if existingKey, err := c.api.FindRemoteMediaByHash(hashBytes); err == nil && existingKey != "" {
			c.logger.Info("File already exists in Google Photos", "path", filePath)
			// Delete from host if requested
			if uploadOpts.deleteFromHost {
				c.logger.Info("Deleting from host", "path", filePath)
				os.Remove(filePath)
			}
			return existingKey, nil
		}
	}

	// Get upload token
	uploadToken, err := c.api.GetUploadToken(hashB64, int(fileSize))
	if err != nil {
		return "", fmt.Errorf("failed to get upload token: %w", err)
	}

	// Open file for upload
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Upload file
	uploadResp, err := c.api.UploadFile(file, uploadToken)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Determine quality
	quality := "original"
	if uploadOpts.saver {
		quality = "saver"
	}

	// Get file modification time
	fileInfo, _ = os.Stat(filePath)
	uploadTimestamp := int(fileInfo.ModTime().Unix())

	// Commit upload
	mediaKey, err := c.api.CommitUpload(uploadResp, fileName, hashBytes, quality, uploadTimestamp)
	if err != nil {
		return "", fmt.Errorf("failed to commit upload: %w", err)
	}

	// Delete from host if requested
	if uploadOpts.deleteFromHost {
		c.logger.Info("Deleting from host", "path", filePath)
		os.Remove(filePath)
	}

	return mediaKey, nil
}

// handleTargetInput processes and validates the upload target input
func (c *Client) handleTargetInput(target interface{}, recursive bool, filterExp string, filterExclude, filterRegex, filterIgnoreCase, filterMatchPath bool) (map[string]*UploadOptions, error) {
	pairs := make(map[string]*UploadOptions)

	// Handle different input types
	switch v := target.(type) {
	case string:
		// Single path
		return c.processPath(v, recursive, filterExp, filterExclude, filterRegex, filterIgnoreCase, filterMatchPath)
	case []string:
		// Multiple paths
		for _, path := range v {
			pathPairs, err := c.processPath(path, recursive, filterExp, filterExclude, filterRegex, filterIgnoreCase, filterMatchPath)
			if err != nil {
				return nil, err
			}
			for k, v := range pathPairs {
				pairs[k] = v
			}
		}
		return pairs, nil
	case map[string]*UploadOptions:
		// Already processed
		return v, nil
	default:
		return nil, fmt.Errorf("invalid target type: %T", target)
	}

	return pairs, nil
}

// processPath processes a single file or directory path
func (c *Client) processPath(path string, recursive bool, filterExp string, filterExclude, filterRegex, filterIgnoreCase, filterMatchPath bool) (map[string]*UploadOptions, error) {
	// Check if path exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}

	pairs := make(map[string]*UploadOptions)

	if fileInfo.IsDir() {
		// Directory - scan for media files
		files, err := c.searchForMediaFiles(path, recursive)
		if err != nil {
			return nil, err
		}

		// Apply filter if specified
		if filterExp != "" {
			files = c.filterFiles(files, filterExp, filterExclude, filterRegex, filterIgnoreCase, filterMatchPath)
		}

		// Create pairs
		for _, file := range files {
			pairs[file] = &UploadOptions{}
		}
	} else {
		// Single file
		pairs[path] = &UploadOptions{}
	}

	return pairs, nil
}

// searchForMediaFiles searches for valid media files in a directory
func (c *Client) searchForMediaFiles(dirPath string, recursive bool) ([]string, error) {
	var files []string

	if recursive {
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && c.isValidMediaFile(path) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				filePath := filepath.Join(dirPath, entry.Name())
				if c.isValidMediaFile(filePath) {
					files = append(files, filePath)
				}
			}
		}
	}

	return files, nil
}

// isValidMediaFile checks if a file is a valid media file
func (c *Client) isValidMediaFile(filePath string) bool {
	// Simple extension check for now
	// In production, would use proper MIME type detection
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".bmp", ".tiff":
		return true
	case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv", ".wmv":
		return true
	}
	return false
}

// handleAlbumCreation handles album creation after upload
func (c *Client) handleAlbumCreation(results UploadResult, albumName string, showProgress bool) {
	// Collect all media keys
	var mediaKeys []string
	for _, key := range results {
		mediaKeys = append(mediaKeys, key)
	}

	if len(mediaKeys) == 0 {
		return
	}

	// Add to album
	c.addToAlbum(mediaKeys, albumName, showProgress)
}
