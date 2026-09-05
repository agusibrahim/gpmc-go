package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/agusibrahim/gpmc-go/internal/client"
	"github.com/agusibrahim/gpmc-go/internal/web"
)

func TestWebUIEndpoints(t *testing.T) {
	auth := os.Getenv("GP_AUTH_DATA")
	if auth == "" {
		t.Skip("GP_AUTH_DATA environment variable not set")
	}

	c, err := client.New(auth)
	if err != nil {
		t.Fatalf("Client init failed: %v", err)
	}

	port := 18999
	webSrv := web.NewServer(c, port)
	go func() {
		_ = webSrv.Start()
	}()
	time.Sleep(500 * time.Millisecond)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// 1. Test GET / (HTML)
	t.Run("GET_IndexHTML", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/")
		if err != nil {
			t.Fatalf("GET / failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
		}
		t.Log("✅ GET / returned 200 OK with Album Studio HTML")
	})

	// 2. Test GET /api/albums/photos
	t.Run("GET_AlbumPhotos", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/albums/photos?album_key=%s&share_token=%s",
			baseURL, sharedAlbumKey, sharedAlbumToken)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("GET /api/albums/photos failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
		}
		var items []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}
		t.Logf("✅ GET /api/albums/photos returned %d items", len(items))
	})

	// 3. Test POST /api/albums/comment
	t.Run("POST_AlbumComment", func(t *testing.T) {
		payload := map[string]string{
			"album_key":   sharedAlbumKey,
			"comment":     fmt.Sprintf("Web UI test comment at %s", time.Now().Format("15:04:05")),
			"share_token": sharedAlbumToken,
		}
		bodyBytes, _ := json.Marshal(payload)
		resp, err := http.Post(baseURL+"/api/albums/comment", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			t.Fatalf("POST /api/albums/comment failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
		}
		var res map[string]string
		json.NewDecoder(resp.Body).Decode(&res)
		t.Logf("✅ POST /api/albums/comment succeeded! Comment ID: %s", res["comment_id"])
	})

	// 4. Test POST /api/ai/enhance
	t.Run("POST_AIEnhance", func(t *testing.T) {
		imgPath := "../../1732015760_hd_sim.png"
		imgData, err := os.ReadFile(imgPath)
		if err != nil {
			imgPath = "../1732015760_hd_sim.png"
			imgData, err = os.ReadFile(imgPath)
		}
		if err != nil {
			t.Skipf("Test image not found: %v", err)
		}

		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		fw, err := w.CreateFormFile("file", "1732015760_hd_sim.png")
		if err != nil {
			t.Fatalf("CreateFormFile failed: %v", err)
		}
		if _, err := fw.Write(imgData); err != nil {
			t.Fatalf("Write file failed: %v", err)
		}
		w.Close()

		req, err := http.NewRequest("POST", baseURL+"/api/ai/enhance", &b)
		if err != nil {
			t.Fatalf("NewRequest failed: %v", err)
		}
		req.Header.Set("Content-Type", w.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /api/ai/enhance failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 200 OK, got %d: %s", resp.StatusCode, string(body))
		}

		var res struct {
			Status   string                   `json:"status"`
			Filename string                   `json:"filename"`
			Results  []map[string]interface{} `json:"results"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			t.Fatalf("Decode failed: %v", err)
		}

		if len(res.Results) == 0 {
			t.Fatalf("Expected at least 1 AI variation, got 0")
		}
		t.Logf("✅ POST /api/ai/enhance succeeded! Returned %d full-resolution AI variations", len(res.Results))
	})
}

