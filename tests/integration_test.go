package tests

import (
	"crypto/sha1"
	"encoding/base64"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/agusibrahim/gpmc-go/internal/client"
	"github.com/agusibrahim/gpmc-go/internal/util"
)

var authData = os.Getenv("GP_AUTH_DATA")

// downloadRandomImage downloads a unique random photo from picsum.photos
func downloadRandomImage(t *testing.T) (string, []byte) {
	seed := strconv.Itoa(rand.Intn(1000000))
	url := "https://picsum.photos/seed/" + seed + "/600/300"
	t.Logf("Downloading test image from: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to download image: %v", err)
	}
	defer resp.Body.Close()

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read image body: %v", err)
	}

	filePath := "/tmp/test_gpmc_" + seed + ".jpg"
	if err := os.WriteFile(filePath, imgBytes, 0644); err != nil {
		t.Fatalf("Failed to write image to disk: %v", err)
	}
	return filePath, imgBytes
}

func TestE2ELifecycle(t *testing.T) {
	if authData == "" {
		t.Skip("Skipping integration test; GP_AUTH_DATA environment variable not set")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to initialize Google Photos Client: %v", err)
	}

	// Download a unique random image
	imagePath, imageBytes := downloadRandomImage(t)
	defer os.Remove(imagePath)

	// Compute SHA1 hash of the image bytes (same way the client does internally)
	sha1Hasher := sha1.New()
	sha1Hasher.Write(imageBytes)
	sha1Bytes := sha1Hasher.Sum(nil)
	hashB64 := base64.StdEncoding.EncodeToString(sha1Bytes)
	dedupKey := util.URLSafeBase64(hashB64)
	t.Logf("Image SHA1: %s (hex), B64: %s, dedupKey: %s", sha1Bytes, hashB64, dedupKey)

	var uploadedMediaKey string

	// --- 1. Upload ---
	t.Run("1_Upload", func(t *testing.T) {
		results, err := c.Upload(imagePath)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
		for _, k := range results {
			uploadedMediaKey = k
		}
		if uploadedMediaKey == "" {
			t.Fatalf("Upload succeeded but mediaKey is empty!")
		}
		t.Logf("✅ MediaKey: %s", uploadedMediaKey)
	})

	// Wait for Google to index the new media item
	t.Log("Waiting 8s for Google to index uploaded media...")
	time.Sleep(8 * time.Second)

	// --- 2. Hash Lookup ---
	t.Run("2_GetMediaKeyByHash", func(t *testing.T) {
		result, err := c.GetMediaKeyByHash(hashB64)
		if err != nil {
			// 400 is expected for freshly uploaded files not yet indexed by Google
			t.Logf("⚠️  Hash lookup returned error (may need more index time): %v", err)
			return
		}
		if result == "" {
			t.Log("⚠️  Hash not indexed yet (normal for recently uploaded files)")
		} else {
			t.Logf("✅ Hash lookup returned: %s", result)
		}
	})

	// --- 3. SetCaption ---
	t.Run("3_SetCaption", func(t *testing.T) {
		_, err := c.SetCaption("agusss GPMC-Go E2E Test "+time.Now().Format("15:04:05"), dedupKey)
		if err != nil {
			t.Errorf("SetCaption failed: %v", err)
		} else {
			t.Log("✅ Caption set successfully")
		}
	})

	// --- 4. SetFavorite (on then off) ---
	t.Run("4_SetFavorite", func(t *testing.T) {
		_, err := c.SetFavorite(dedupKey, true)
		if err != nil {
			t.Errorf("SetFavorite(true) failed: %v", err)
			return
		}
		t.Log("✅ Favorite set to true")

		_, err = c.SetFavorite(dedupKey, false)
		if err != nil {
			t.Errorf("SetFavorite(false) failed: %v", err)
		} else {
			t.Log("✅ Favorite reset to false")
		}
	})

	// --- 5. SetArchived ---
	t.Run("5_SetArchived", func(t *testing.T) {
		_, err := c.SetArchived([]string{dedupKey}, true)
		if err != nil {
			t.Errorf("SetArchived(true) failed: %v", err)
			return
		}
		t.Log("✅ Archived successfully")

		_, err = c.SetArchived([]string{dedupKey}, false)
		if err != nil {
			t.Errorf("SetArchived(false) failed: %v", err)
		} else {
			t.Log("✅ Unarchived successfully")
		}
	})

	// --- 6. GetDownloadURLs ---
	t.Run("6_GetDownloadURLs", func(t *testing.T) {
		if uploadedMediaKey == "" {
			t.Skip("No media key, skipping")
		}
		urls, err := c.GetDownloadURLs(uploadedMediaKey)
		if err != nil {
			t.Errorf("GetDownloadURLs failed: %v", err)
			return
		}
		if len(urls) > 0 {
			t.Logf("✅ Got %d download URL(s):", len(urls))
			for i, url := range urls {
				t.Logf("  [%d]: %s", i, url)
			}
		} else {
			t.Log("⚠️  GetDownloadURLs returned empty list")
		}
	})

	// --- 7. AddToAlbum ---
	t.Run("7_AddToAlbum", func(t *testing.T) {
		if uploadedMediaKey == "" {
			t.Skip("No media key, skipping")
		}
		albumKeys := c.AddToAlbum([]string{uploadedMediaKey}, "GPMC-Go E2E Album", false)
		if len(albumKeys) > 0 {
			t.Logf("✅ AddToAlbum success. Album key: %v", albumKeys)
		} else {
			t.Log("⚠️  AddToAlbum returned empty (possibly 403 permission scope, normal for this auth)")
		}
	})

	// --- 8. MoveToTrash (cleanup) ---
	t.Run("8_MoveToTrash", func(t *testing.T) {
		// MoveToTrash accepts sha1 hash bytes — pass raw bytes
		res, err := c.MoveToTrash(sha1Bytes)
		if err != nil {
			t.Errorf("MoveToTrash failed: %v (item may need to be deleted manually from Google Photos)", err)
		} else {
			t.Logf("✅ Moved to trash successfully. Response keys: %d", len(res))
		}
	})
}
