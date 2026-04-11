package client

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/xob0t/gpmc-go/internal/api"
	"github.com/xob0t/gpmc-go/internal/util"
)

// Client represents the Google Photos client
type Client struct {
	logger     *slog.Logger
	validMimes []string
	timeout    int
	authData   string
	language   string
	api        *api.Api
	cacheDir   string
	dbPath     string
}

// UploadOptions contains options for uploading files
type UploadOptions struct {
	Hash     interface{} // bytes, string (hex/base64), or nil
	FileName string
}

// New creates a new Client instance
func New(authData string, opts ...Option) (*Client, error) {
	if authData == "" {
		// Try to get from environment
		authData = os.Getenv("GP_AUTH_DATA")
		if authData == "" {
			return nil, fmt.Errorf("auth_data is required (set GP_AUTH_DATA env var or pass as argument)")
		}
	}

	// Parse email from auth_data for logging
	email := util.ParseEmail(authData)

	// Create API client
	apiClient, err := api.New(authData)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	// Create client with defaults
	client := &Client{
		logger:     slog.New(slog.NewTextHandler(os.Stdout, nil)),
		validMimes: []string{"image/", "video/"},
		timeout:    api.DefaultTimeout,
		authData:   authData,
		language:   util.ParseLanguage(authData),
		api:        apiClient,
		cacheDir:   filepath.Join(os.Getenv("HOME"), ".gpmc", email),
		dbPath:     filepath.Join(os.Getenv("HOME"), ".gpmc", email, "storage.db"),
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	client.logger.Info("User: " + email)
	client.logger.Info("Language: " + client.language)

	// Add raw MIME types
	client.addRawMimetypes()

	return client, nil
}

// Option is a functional option for configuring the Client
type Option func(*Client)

// WithLogger sets a custom logger
func WithLogger(logger *slog.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout int) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithLanguage sets the language
func WithLanguage(language string) Option {
	return func(c *Client) {
		c.language = language
	}
}

// WithCacheDir sets the cache directory
func WithCacheDir(cacheDir string) Option {
	return func(c *Client) {
		c.cacheDir = cacheDir
		c.dbPath = filepath.Join(cacheDir, "storage.db")
	}
}

// GetMediaKeyByHash gets a Google Photos media key by media's hash
func (c *Client) GetMediaKeyByHash(sha1Hash interface{}) (string, error) {
	_, hashBytes, err := convertSHA1Hash(sha1Hash)
	if err != nil {
		return "", err
	}

	return c.api.FindRemoteMediaByHash(hashBytes)
}

// addRawMimetypes adds RAW photo file extensions to MIME types
func (c *Client) addRawMimetypes() {
	// This would add raw MIME type mappings
	// For Go, we'd use a custom MIME type detection mechanism
	// since the standard mime package doesn't support adding types
}

// convertSHA1Hash converts a SHA-1 hash from any format to bytes and base64
func convertSHA1Hash(hash interface{}) (string, []byte, error) {
	// This would be implemented using the hash package
	return "", nil, nil
}
