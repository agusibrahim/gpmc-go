package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/agusibrahim/gpmc-go/internal/client"
)

//go:embed index.html
var staticAssets embed.FS

// WebServer handles the web UI and API for GPMC
type WebServer struct {
	client     *client.Client
	logger     *slog.Logger
	progressMu sync.RWMutex
	clients    map[chan client.ProgressUpdate]bool
	port       int
}

// NewServer creates a new WebServer instance
func NewServer(c *client.Client, port int) *WebServer {
	return &WebServer{
		client:  c,
		logger:  slog.Default(),
		clients: make(map[chan client.ProgressUpdate]bool),
		port:    port,
	}
}

// Start starts the web server
func (s *WebServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/events", s.handleEvents)

	// Album APIs
	mux.HandleFunc("/api/albums", s.handleAlbums)
	mux.HandleFunc("/api/albums/photos", s.handleAlbumPhotos)
	mux.HandleFunc("/api/albums/comment", s.handleAlbumComment)
	mux.HandleFunc("/api/albums/upload", s.handleAlbumUpload)
	mux.HandleFunc("/api/albums/share", s.handleAlbumShare)
	mux.HandleFunc("/api/albums/rename", s.handleAlbumRename)
	mux.HandleFunc("/api/albums/delete", s.handleAlbumDelete)
	mux.HandleFunc("/api/thumbnail", s.handleThumbnail)

	// AI Enhancement API
	mux.HandleFunc("/api/ai/enhance", s.handleAIEnhance)


	server := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Minute,
		WriteTimeout:      30 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	s.logger.Info("Web UI started", "url", fmt.Sprintf("http://localhost:%d", s.port))
	return server.ListenAndServe()
}

func (s *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := staticAssets.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Could not load index.html", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

func (s *WebServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		http.Error(w, "Content-Type must be multipart/form-data", http.StatusUnsupportedMediaType)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 20<<30)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	threads := 1
	if t := r.FormValue("threads"); t != "" {
		fmt.Sscanf(t, "%d", &threads)
	}
	if threads < 1 {
		threads = 1
	}
	if threads > 16 {
		threads = 16
	}
	album := r.FormValue("album")
	saver := r.FormValue("saver") == "true"

	tempDir, err := os.MkdirTemp("", "gpmc-web-*")
	if err != nil {
		http.Error(w, "Failed to create temp dir", http.StatusInternalServerError)
		return
	}

	files := r.MultipartForm.File["files"]
	filePaths, err := saveMultipartFiles(tempDir, files)
	if err != nil {
		os.RemoveAll(tempDir)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	go func() {
		defer os.RemoveAll(tempDir)

		progressChan := make(chan client.ProgressUpdate, 100)
		done := make(chan struct{})
		go func() {
			defer close(done)
			for update := range progressChan {
				s.broadcast(update)
			}
		}()

		opts := []client.UploadOption{
			client.WithThreads(threads),
			client.WithAlbum(album),
			client.WithSaver(saver),
			client.WithProgressChan(progressChan),
		}

		if _, err := s.client.UploadBatch(filePaths, opts...); err != nil {
			s.logger.Error("web upload failed", "error", err)
			s.broadcast(client.ProgressUpdate{Status: client.StatusError, Error: err.Error()})
		}
		close(progressChan)
		<-done
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "upload_started"})
}

// handleAlbums handles GET (list albums) and POST (create album)
func (s *WebServer) handleAlbums(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		albums, err := s.client.ListAlbums()
		if err != nil {
			s.logger.Warn("ListAlbums returned error", "error", err)
			// Return empty list instead of crashing UI
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		json.NewEncoder(w).Encode(albums)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Title     string   `json:"title"`
			MediaKeys []string `json:"media_keys"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			http.Error(w, "Album title is required", http.StatusBadRequest)
			return
		}

		albumKey, err := s.client.CreateAlbum(req.Title, req.MediaKeys...)
		if err != nil {
			http.Error(w, "CreateAlbum failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"album_key": albumKey,
			"title":     req.Title,
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleAlbumPhotos lists photos in an album
func (s *WebServer) handleAlbumPhotos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	albumKey := r.URL.Query().Get("album_key")
	if albumKey == "" {
		http.Error(w, "album_key parameter is required", http.StatusBadRequest)
		return
	}
	shareToken := r.URL.Query().Get("share_token")

	photos, err := s.client.ListPhotosInAlbum(albumKey, shareToken)
	if err != nil {
		http.Error(w, "Failed to list photos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(photos)
}

// handleAlbumComment adds a comment to an album
func (s *WebServer) handleAlbumComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AlbumKey   string `json:"album_key"`
		Comment    string `json:"comment"`
		ShareToken string `json:"share_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.AlbumKey == "" || strings.TrimSpace(req.Comment) == "" {
		http.Error(w, "album_key and comment text are required", http.StatusBadRequest)
		return
	}

	commentID, err := s.client.AddCommentToAlbum(req.AlbumKey, req.Comment, req.ShareToken)
	if err != nil {
		http.Error(w, "Failed to add comment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"comment_id": commentID,
		"status":     "ok",
	})
}

// handleAlbumUpload uploads a single photo and adds it directly to the specified album
func (s *WebServer) handleAlbumUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 500<<20) // 500MB max
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	albumKey := r.FormValue("album_key")
	if albumKey == "" {
		http.Error(w, "album_key is required", http.StatusBadRequest)
		return
	}
	shareToken := r.FormValue("share_token")

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, "No file provided in 'file' field", http.StatusBadRequest)
		return
	}

	tempDir, err := os.MkdirTemp("", "gpmc-album-up-*")
	if err != nil {
		http.Error(w, "Failed to create temp dir", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	filePaths, err := saveMultipartFiles(tempDir, files)
	if err != nil || len(filePaths) == 0 {
		http.Error(w, "Failed to save file: "+err.Error(), http.StatusBadRequest)
		return
	}

	mediaKey, err := s.client.UploadPhotoToAlbum(albumKey, filePaths[0], shareToken)
	if err != nil {
		http.Error(w, "Failed to upload photo to album: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"media_key": mediaKey,
		"filename":  files[0].Filename,
		"status":    "ok",
	})
}

// handleAlbumShare generates a public share link for an album
func (s *WebServer) handleAlbumShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AlbumKey string `json:"album_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.AlbumKey == "" {
		http.Error(w, "album_key is required", http.StatusBadRequest)
		return
	}

	res, err := s.client.ShareAlbum(req.AlbumKey)
	if err != nil {
		http.Error(w, "ShareAlbum failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// handleAlbumRename renames an album
func (s *WebServer) handleAlbumRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AlbumKey string `json:"album_key"`
		Title    string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.AlbumKey == "" || strings.TrimSpace(req.Title) == "" {
		http.Error(w, "album_key and title are required", http.StatusBadRequest)
		return
	}

	if err := s.client.RenameAlbum(req.AlbumKey, req.Title); err != nil {
		http.Error(w, "RenameAlbum failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAlbumDelete deletes an album
func (s *WebServer) handleAlbumDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AlbumKey string `json:"album_key"`
		IsShared bool   `json:"is_shared"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.AlbumKey == "" {
		http.Error(w, "album_key is required", http.StatusBadRequest)
		return
	}

	if err := s.client.DeleteAlbum(req.AlbumKey, req.IsShared); err != nil {
		http.Error(w, "DeleteAlbum failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleThumbnail streams the thumbnail for a mediaKey
func (s *WebServer) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mediaKey := r.URL.Query().Get("media_key")
	if mediaKey == "" {
		http.Error(w, "media_key is required", http.StatusBadRequest)
		return
	}

	thumbBytes, err := s.client.GetThumbnail(mediaKey)
	if err != nil {
		http.Error(w, "Failed to fetch thumbnail: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(thumbBytes)
}

func saveMultipartFiles(tempDir string, files []*multipart.FileHeader) ([]string, error) {
	filePaths := make([]string, 0, len(files))
	for _, fileHeader := range files {
		cleanName, err := safeUploadName(fileHeader.Filename)
		if err != nil {
			return nil, err
		}
		safePath := filepath.Join(tempDir, cleanName)
		if err := os.MkdirAll(filepath.Dir(safePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create upload directory: %w", err)
		}

		src, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open uploaded file: %w", err)
		}
		dst, err := os.Create(safePath)
		if err != nil {
			src.Close()
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		_, copyErr := io.Copy(dst, src)
		closeErr := dst.Close()
		srcErr := src.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("failed to save uploaded file: %w", copyErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("failed to close temp file: %w", closeErr)
		}
		if srcErr != nil {
			return nil, fmt.Errorf("failed to close uploaded file: %w", srcErr)
		}
		filePaths = append(filePaths, safePath)
	}
	return filePaths, nil
}

func safeUploadName(name string) (string, error) {
	name = filepath.Clean(filepath.FromSlash(name))
	if name == "." || filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(os.PathSeparator)) || name == ".." {
		return "", fmt.Errorf("invalid upload filename")
	}
	return name, nil
}

func (s *WebServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan client.ProgressUpdate, 10)
	s.progressMu.Lock()
	s.clients[ch] = true
	s.progressMu.Unlock()

	defer func() {
		s.progressMu.Lock()
		delete(s.clients, ch)
		s.progressMu.Unlock()
		close(ch)
	}()

	for {
		select {
		case update := <-ch:
			data, _ := json.Marshal(update)
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *WebServer) broadcast(update client.ProgressUpdate) {
	s.progressMu.RLock()
	defer s.progressMu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- update:
		default:
			// Client slow, skip update
		}
	}
}

// handleAIEnhance handles photo enhancement via Google Photos AI Magic Editor Preset
func (s *WebServer) handleAIEnhance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit to 50MB
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, "No file provided in 'file' field", http.StatusBadRequest)
		return
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		http.Error(w, "Failed to open uploaded file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file content: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.logger.Info("Starting AI Photo Enhancement", "filename", fileHeader.Filename, "size", len(imgBytes))
	results, err := s.client.EnhancePhoto(imgBytes)
	if err != nil {
		s.logger.Error("AI Photo Enhancement failed", "error", err)
		http.Error(w, "AI Enhancement failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.logger.Info("AI Photo Enhancement successful", "variations", len(results))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"filename": fileHeader.Filename,
		"results":  results,
	})
}

