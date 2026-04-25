package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/agusibrahim/gpmc-go/internal/hash"
	"github.com/agusibrahim/gpmc-go/internal/proto/pb"
	"github.com/schollz/progressbar/v3"
)

// UploadResult maps file paths to their media keys
type UploadResult map[string]string

type uploadTask struct {
	path string
	opts *UploadOptions
}

type manifestFile struct {
	Path      string         `json:"path"`
	Size      int64          `json:"size"`
	ModTime   int64          `json:"mod_time"`
	Hash      string         `json:"hash,omitempty"`
	MediaKey  string         `json:"media_key,omitempty"`
	Status    ProgressStatus `json:"status"`
	LastError string         `json:"last_error,omitempty"`
	Verified  bool           `json:"verified"`
}

type uploadManifest struct {
	Version int                     `json:"version"`
	Files   map[string]manifestFile `json:"files"`
}

type progressReader struct {
	r              io.Reader
	path           string
	filename       string
	fileSize       int64
	attempt        int
	batchTotal     int64
	fileSent       int64
	batchCompleted *int64
	lastEmit       time.Time
	progressChan   chan ProgressUpdate
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.fileSent += int64(n)
		now := time.Now()
		if r.progressChan != nil && (now.Sub(r.lastEmit) >= 500*time.Millisecond || r.fileSent == r.fileSize) {
			r.lastEmit = now
			r.progressChan <- ProgressUpdate{
				ID:              r.path,
				Filename:        r.filename,
				Path:            r.path,
				Status:          StatusUploading,
				Progress:        uploadProgress(r.fileSent, r.fileSize),
				BytesSent:       r.fileSent,
				BytesTotal:      r.fileSize,
				BatchBytesSent:  atomic.LoadInt64(r.batchCompleted) + r.fileSent,
				BatchBytesTotal: r.batchTotal,
				Attempt:         r.attempt,
			}
		}
	}
	return n, err
}

func (c *Client) Upload(target interface{}, opts ...UploadOption) (UploadResult, error) {
	batch, err := c.UploadBatch(target, opts...)
	if err != nil {
		return nil, err
	}
	results := make(UploadResult)
	for _, file := range batch.Files {
		if file.MediaKey != "" && file.Status != StatusError {
			results[file.Path] = file.MediaKey
		}
	}
	return results, nil
}

func (c *Client) UploadBatch(target interface{}, opts ...UploadOption) (*UploadBatchResult, error) {
	options := &uploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	pathHashPairs, err := c.handleTargetInput(target, options.recursive, options.filterExp, options.filterExclude, options.filterRegex, options.filterIgnoreCase, options.filterMatchPath)
	if err != nil {
		return nil, err
	}

	manifest := loadManifest(options.manifestPath)
	batch := c.uploadConcurrently(pathHashPairs, options, manifest)
	if options.manifestPath != "" {
		if err := saveManifest(options.manifestPath, manifest); err != nil {
			c.logger.Error("failed to save upload manifest", "path", options.manifestPath, "error", err)
		}
	}

	if options.albumName != "" {
		legacy := make(UploadResult)
		for _, file := range batch.Files {
			if file.MediaKey != "" && file.Status != StatusError {
				legacy[file.Path] = file.MediaKey
			}
		}
		c.handleAlbumCreation(legacy, options.albumName, options.showProgress)
	}

	if batch.Summary.Failed > 0 {
		return batch, fmt.Errorf("upload completed with %d failed file(s)", batch.Summary.Failed)
	}
	return batch, nil
}

func (c *Client) uploadConcurrently(pairs map[string]*UploadOptions, opts *uploadOptions, manifest *uploadManifest) *UploadBatchResult {
	started := time.Now()
	result := &UploadBatchResult{}
	workers := opts.threads
	if workers < 1 {
		workers = 1
	}

	preflight := c.preflightUploads(pairs, opts, manifest)
	result.Files = append(result.Files, preflight.invalid...)
	result.Files = append(result.Files, preflight.resumed...)
	result.Summary.TotalBytes = preflight.totalBytes

	var batchCompleted int64
	for _, file := range result.Files {
		if file.Status == StatusSkipped || file.Status == StatusDone {
			batchCompleted += file.Size
		}
	}

	var bar *progressbar.ProgressBar
	if opts.showProgress {
		bar = progressbar.Default(int64(len(preflight.valid)+len(result.Files)), "Uploading files")
		defer bar.Close()
	}

	if opts.progressChan != nil {
		opts.progressChan <- ProgressUpdate{Status: StatusBatchMeta, TotalFiles: len(preflight.valid) + len(result.Files), BatchBytesTotal: preflight.totalBytes, BatchBytesSent: batchCompleted}
	}

	if bar != nil {
		for range result.Files {
			bar.Add(1)
		}
	}

	tasks := make(chan uploadTask)
	results := make(chan FileResult)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				results <- c.uploadFile(task.path, task.opts, opts, preflight.totalBytes, &batchCompleted, manifest)
			}
		}()
	}

	go func() {
		for _, task := range preflight.valid {
			tasks <- task
		}
		close(tasks)
		wg.Wait()
		close(results)
	}()

	for file := range results {
		result.Files = append(result.Files, file)
		if manifest != nil {
			updateManifestFile(manifest, file)
		}
		if opts.progressChan != nil {
			progress := 1.0
			if file.Status == StatusError {
				progress = 0
			}
			opts.progressChan <- ProgressUpdate{ID: file.Path, Filename: file.Filename, Path: file.Path, Status: file.Status, Progress: progress, Error: file.Error, MediaKey: file.MediaKey, BytesSent: file.Size, BytesTotal: file.Size, BatchBytesSent: atomic.LoadInt64(&batchCompleted), BatchBytesTotal: preflight.totalBytes, Attempt: file.Attempts}
		}
		if bar != nil {
			bar.Add(1)
		}
	}

	result.Summary.TotalFiles = len(result.Files)
	for _, file := range result.Files {
		switch file.Status {
		case StatusDone:
			result.Summary.Uploaded++
			result.Summary.UploadedBytes += file.Size
		case StatusSkipped:
			result.Summary.Skipped++
		case StatusError:
			result.Summary.Failed++
		}
	}
	result.Summary.Duration = time.Since(started)
	return result
}

type preflightResult struct {
	valid      []uploadTask
	invalid    []FileResult
	resumed    []FileResult
	totalBytes int64
}

func (c *Client) preflightUploads(pairs map[string]*UploadOptions, opts *uploadOptions, manifest *uploadManifest) preflightResult {
	seen := make(map[string]bool)
	result := preflightResult{}
	for path, uploadOpts := range pairs {
		abs, err := filepath.Abs(path)
		if err != nil {
			result.invalid = append(result.invalid, failedFile(path, filepath.Base(path), 0, err))
			continue
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true

		info, err := os.Stat(path)
		if err != nil {
			result.invalid = append(result.invalid, failedFile(path, filepath.Base(path), 0, fmt.Errorf("stat stage failed: %w", err)))
			continue
		}
		filename := uploadOpts.FileName
		if filename == "" {
			filename = filepath.Base(path)
		}
		if info.IsDir() {
			result.invalid = append(result.invalid, failedFile(path, filename, 0, errors.New("path is a directory")))
			continue
		}
		if info.Size() < 0 {
			result.invalid = append(result.invalid, failedFile(path, filename, info.Size(), errors.New("invalid file size")))
			continue
		}
		if !c.isValidMediaFile(path) {
			result.invalid = append(result.invalid, failedFile(path, filename, info.Size(), errors.New("unsupported media extension")))
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			result.invalid = append(result.invalid, failedFile(path, filename, info.Size(), fmt.Errorf("open-file stage failed: %w", err)))
			continue
		}
		file.Close()

		result.totalBytes += info.Size()
		if opts.resume && manifest != nil {
			if entry, ok := manifest.Files[path]; ok && entry.Size == info.Size() && entry.ModTime == info.ModTime().Unix() && isResumableSuccess(entry.Status) {
				result.resumed = append(result.resumed, FileResult{Path: path, Filename: filename, Size: info.Size(), MediaKey: entry.MediaKey, Status: entry.Status, Error: entry.LastError, Hash: entry.Hash, Verified: entry.Verified, ModTime: info.ModTime().Unix()})
				continue
			}
		}
		result.valid = append(result.valid, uploadTask{path: path, opts: uploadOpts})
	}
	return result
}

func failedFile(path, filename string, size int64, err error) FileResult {
	return FileResult{Path: path, Filename: filename, Size: size, Status: StatusError, Error: err.Error()}
}

func (c *Client) uploadFile(filePath string, opts *UploadOptions, uploadOpts *uploadOptions, batchTotal int64, batchCompleted *int64, manifest *uploadManifest) FileResult {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return failedFile(filePath, filepath.Base(filePath), 0, fmt.Errorf("stat stage failed: %w", err))
	}
	fileSize := fileInfo.Size()
	fileName := opts.FileName
	if fileName == "" {
		fileName = filepath.Base(filePath)
	}
	result := FileResult{Path: filePath, Filename: fileName, Size: fileSize, ModTime: fileInfo.ModTime().Unix(), Status: StatusError}

	if uploadOpts.progressChan != nil {
		uploadOpts.progressChan <- ProgressUpdate{ID: filePath, Filename: fileName, Path: filePath, Status: StatusHashing, Progress: 0.1, BytesTotal: fileSize, BatchBytesTotal: batchTotal}
	}

	var hashBytes []byte
	var hashB64 string
	err = c.withRetry("hash", &result.Attempts, func() error {
		if opts.Hash != nil {
			var convertErr error
			hashBytes, hashB64, convertErr = hash.ConvertSHA1Hash(opts.Hash)
			return convertErr
		}
		var hashErr error
		hashBytes, hashB64, hashErr = hash.CalculateSHA1Hash(filePath)
		return hashErr
	})
	if err != nil {
		result.Error = fmt.Errorf("hash stage failed: %w", err).Error()
		return result
	}
	result.Hash = hashB64

	if !uploadOpts.forceUpload {
		var existingKey string
		err = c.withRetry("find-by-hash", &result.Attempts, func() error {
			var findErr error
			existingKey, findErr = c.api.FindRemoteMediaByHash(hashBytes)
			return findErr
		})
		if err != nil {
			result.Error = fmt.Errorf("find-by-hash stage failed: %w", err).Error()
			return result
		}
		if existingKey != "" {
			if uploadOpts.deleteFromHost {
				_ = os.Remove(filePath)
			}
			result.MediaKey = existingKey
			result.Status = StatusSkipped
			result.Verified = true
			atomic.AddInt64(batchCompleted, fileSize)
			return result
		}
	}

	var uploadToken string
	err = c.withRetry("get-upload-token", &result.Attempts, func() error {
		var tokenErr error
		uploadToken, tokenErr = c.api.GetUploadToken(hashB64, int(fileSize))
		return tokenErr
	})
	if err != nil {
		result.Error = fmt.Errorf("get-upload-token stage failed: %w", err).Error()
		return result
	}

	if uploadOpts.progressChan != nil {
		uploadOpts.progressChan <- ProgressUpdate{ID: filePath, Filename: fileName, Path: filePath, Status: StatusUploading, Progress: 0.3, BytesTotal: fileSize, BatchBytesTotal: batchTotal, Attempt: result.Attempts + 1}
	}

	file, err := os.Open(filePath)
	if err != nil {
		result.Error = fmt.Errorf("open-file stage failed: %w", err).Error()
		return result
	}
	defer file.Close()

	var uploadResp *pb.CommitUploadMessage_Field1_Field1
	err = c.withRetry("upload", &result.Attempts, func() error {
		if _, seekErr := file.Seek(0, 0); seekErr != nil {
			return seekErr
		}
		reader := &progressReader{r: file, path: filePath, filename: fileName, fileSize: fileSize, attempt: result.Attempts + 1, batchTotal: batchTotal, batchCompleted: batchCompleted, progressChan: uploadOpts.progressChan}
		resp, uploadErr := c.api.UploadFile(reader, uploadToken, fileSize)
		uploadResp = resp
		return uploadErr
	})
	if err != nil {
		result.Error = fmt.Errorf("upload stage failed for %s: %w", fileName, err).Error()
		return result
	}

	if uploadOpts.progressChan != nil {
		uploadOpts.progressChan <- ProgressUpdate{ID: filePath, Filename: fileName, Path: filePath, Status: StatusCommitting, Progress: 0.8, BytesSent: fileSize, BytesTotal: fileSize, BatchBytesSent: atomic.LoadInt64(batchCompleted) + fileSize, BatchBytesTotal: batchTotal, Attempt: result.Attempts}
	}

	quality := "original"
	if uploadOpts.saver {
		quality = "saver"
	}

	err = c.withRetry("commit", &result.Attempts, func() error {
		mediaKey, commitErr := c.api.CommitUpload(uploadResp, fileName, hashBytes, quality, int(fileInfo.ModTime().Unix()))
		result.MediaKey = mediaKey
		return commitErr
	})
	if err != nil {
		if verifiedKey := c.verifyByHash(hashBytes); verifiedKey != "" {
			result.MediaKey = verifiedKey
			result.Status = StatusDone
			result.Verified = true
			return result
		}
		result.Error = fmt.Errorf("commit stage failed: %w", err).Error()
		return result
	}

	if uploadOpts.verify {
		if c.verifyByHashWithBackoff(hashBytes) == "" {
			result.Error = "verify stage failed: uploaded media was not found by hash before timeout"
			return result
		}
	}
	result.Verified = true
	atomic.AddInt64(batchCompleted, fileSize)
	if uploadOpts.deleteFromHost {
		_ = os.Remove(filePath)
	}
	result.Status = StatusDone
	return result
}

func (c *Client) withRetry(stage string, attempts *int, fn func() error) error {
	var err error
	for i := 0; i < 3; i++ {
		*attempts = *attempts + 1
		err = fn()
		if err == nil || !isTransient(err) {
			return err
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return fmt.Errorf("%s failed after retries: %w", stage, err)
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "temporary") || strings.Contains(msg, "connection reset") || strings.Contains(msg, "status 429") || strings.Contains(msg, "status 502") || strings.Contains(msg, "status 503") || strings.Contains(msg, "status 504")
}

func (c *Client) verifyByHash(hashBytes []byte) string {
	key, err := c.api.FindRemoteMediaByHash(hashBytes)
	if err != nil {
		return ""
	}
	return key
}

func (c *Client) verifyByHashWithBackoff(hashBytes []byte) string {
	for i := 0; i < 5; i++ {
		if key := c.verifyByHash(hashBytes); key != "" {
			return key
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return ""
}

func uploadProgress(sent, total int64) float64 {
	if total <= 0 {
		return 0.3
	}
	return 0.3 + (float64(sent)/float64(total))*0.5
}

func isResumableSuccess(status ProgressStatus) bool {
	return status == StatusDone || status == StatusSkipped
}

func loadManifest(path string) *uploadManifest {
	manifest := &uploadManifest{Version: 1, Files: make(map[string]manifestFile)}
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest
	}
	if json.Unmarshal(data, manifest) != nil || manifest.Files == nil {
		manifest.Version = 1
		manifest.Files = make(map[string]manifestFile)
	}
	return manifest
}

func saveManifest(path string, manifest *uploadManifest) error {
	if manifest == nil || path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func updateManifestFile(manifest *uploadManifest, file FileResult) {
	if manifest == nil {
		return
	}
	manifest.Files[file.Path] = manifestFile{Path: file.Path, Size: file.Size, ModTime: file.ModTime, Hash: file.Hash, MediaKey: file.MediaKey, Status: file.Status, LastError: file.Error, Verified: file.Verified}
}

// handleTargetInput processes and validates the upload target input
func (c *Client) handleTargetInput(target interface{}, recursive bool, filterExp string, filterExclude, filterRegex, filterIgnoreCase, filterMatchPath bool) (map[string]*UploadOptions, error) {
	pairs := make(map[string]*UploadOptions)
	switch v := target.(type) {
	case string:
		return c.processPath(v, recursive, filterExp, filterExclude, filterRegex, filterIgnoreCase, filterMatchPath)
	case []string:
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
		return v, nil
	default:
		return nil, fmt.Errorf("invalid target type: %T", target)
	}
}

func (c *Client) processPath(path string, recursive bool, filterExp string, filterExclude, filterRegex, filterIgnoreCase, filterMatchPath bool) (map[string]*UploadOptions, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}
	pairs := make(map[string]*UploadOptions)
	if fileInfo.IsDir() {
		files, err := c.searchForMediaFiles(path, recursive)
		if err != nil {
			return nil, err
		}
		if filterExp != "" {
			files = c.filterFiles(files, filterExp, filterExclude, filterRegex, filterIgnoreCase, filterMatchPath)
		}
		for _, file := range files {
			pairs[file] = &UploadOptions{}
		}
	} else {
		pairs[path] = &UploadOptions{}
	}
	return pairs, nil
}

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

func (c *Client) isValidMediaFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".bmp", ".tiff":
		return true
	case ".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv", ".wmv":
		return true
	}
	return false
}

func (c *Client) handleAlbumCreation(results UploadResult, albumName string, showProgress bool) {
	var mediaKeys []string
	for _, key := range results {
		mediaKeys = append(mediaKeys, key)
	}
	if len(mediaKeys) == 0 {
		return
	}
	c.addToAlbum(mediaKeys, albumName, showProgress)
}
