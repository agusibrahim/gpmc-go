package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/xob0t/gpmc-go/internal/client"
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
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
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

	// Parse multipart form
	err := r.ParseMultipartForm(100 << 20) // 100MB max in memory
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get options from form
	threads := 1
	if t := r.FormValue("threads"); t != "" {
		fmt.Sscanf(t, "%d", &threads)
	}
	album := r.FormValue("album")
	saver := r.FormValue("saver") == "true"

	// Temp directory for uploads
	tempDir, err := os.MkdirTemp("", "gpmc-web-*")
	if err != nil {
		http.Error(w, "Failed to create temp dir", http.StatusInternalServerError)
		return
	}
	// We won't defer rm here as upload is async or we wait... 
	// To keep it simple for now, we process synchronously or via goroutine.

	files := r.MultipartForm.File["files"]
	var filePaths []string

	for _, fileHeader := range files {
		// Preserve directory structure if provided in custom header or filename
		// Browsers often put path in filename for webkitdirectory
		safePath := filepath.Join(tempDir, fileHeader.Filename)
		os.MkdirAll(filepath.Dir(safePath), 0755)

		dst, err := os.Create(safePath)
		if err != nil {
			continue
		}
		src, _ := fileHeader.Open()
		io.Copy(dst, src)
		dst.Close()
		src.Close()

		filePaths = append(filePaths, safePath)
	}

	// Start upload in goroutine
	go func() {
		defer os.RemoveAll(tempDir)

		progressChan := make(chan client.ProgressUpdate, 100)
		
		// Proxy updates from client to all connected SSE clients
		go func() {
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

		s.client.Upload(filePaths, opts...)
		close(progressChan)
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "upload_started"})
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
