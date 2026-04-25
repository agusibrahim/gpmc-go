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
