package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/agusibrahim/gpmc-go/internal/client"
)

const (
	sharedAlbumKey   = "AF1QipOjPBEy1HY4Q2qmghWDpotCFEEjxxat8cD-DAWu9JbbhGtT9dNHbpEQR2CSRz5a0Q"
	sharedAlbumToken = "Zk5PN3ZpWXpDY2tzbTNCeHNDTlNBZG1NU09sQlNn"
)

func createDummyJPEG(path string) error {
	dummyJPEG := []byte{
		0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x01, 0x00, 0x60,
		0x00, 0x60, 0x00, 0x00, 0xff, 0xdb, 0x00, 0x43, 0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08,
		0x07, 0x07, 0x07, 0x09, 0x09, 0x08, 0x0a, 0x0c, 0x14, 0x0d, 0x0c, 0x0b, 0x0b, 0x0c, 0x19, 0x12,
		0x13, 0x0f, 0x14, 0x1d, 0x1a, 0x1f, 0x1e, 0x1d, 0x1a, 0x1c, 0x1c, 0x20, 0x24, 0x2e, 0x27, 0x20,
		0x22, 0x2c, 0x23, 0x1c, 0x1c, 0x28, 0x37, 0x29, 0x2c, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1f, 0x27,
		0x39, 0x3d, 0x38, 0x32, 0x3c, 0x2e, 0x33, 0x34, 0x32, 0xff, 0xc0, 0x00, 0x0b, 0x08, 0x00, 0x01,
		0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xff, 0xc4, 0x00, 0x1f, 0x00, 0x00, 0x01, 0x05, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0xff, 0xda, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3f,
		0x00, 0xbf, 0x00, 0xff, 0xd9,
	}
	return os.WriteFile(path, dummyJPEG, 0644)
}

func TestAlbumSharedWithToken(t *testing.T) {
	auth := os.Getenv("GP_AUTH_DATA")
	if auth == "" {
		t.Skip("GP_AUTH_DATA environment variable not set")
	}

	c, err := client.New(auth)
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// 1. List Photos in Shared Album with Token
	t.Run("ListPhotosWithToken", func(t *testing.T) {
		photos, err := c.ListPhotosInAlbum(sharedAlbumKey, sharedAlbumToken)
		if err != nil {
			t.Fatalf("ListPhotosInAlbum failed: %v", err)
		}
		t.Logf("✅ Successfully listed %d photos in shared album:", len(photos))
		for i, p := range photos {
			t.Logf("   [%d] %s (MediaKey: %s)", i+1, p.FileName, p.MediaKey)
		}
	})

	// 2. Add Comment with Token
	t.Run("AddCommentWithToken", func(t *testing.T) {
		commentText := fmt.Sprintf("Komentar gpmc-go test pada %s", time.Now().Format("15:04:05"))
		commentID, err := c.AddCommentToAlbum(sharedAlbumKey, commentText, sharedAlbumToken)
		if err != nil {
			t.Fatalf("AddCommentToAlbum with token failed: %v", err)
		}
		t.Logf("✅ Successfully added comment with token! ID: %s", commentID)
	})

	// 3. Upload Photo to Shared Album with Token
	t.Run("UploadPhotoWithToken", func(t *testing.T) {
		tmpFile := fmt.Sprintf("/tmp/gpmc_shared_test_%d.jpg", time.Now().Unix())
		if err := createDummyJPEG(tmpFile); err != nil {
			t.Fatalf("Failed to create test image: %v", err)
		}
		defer os.Remove(tmpFile)

		mediaKey, err := c.UploadPhotoToAlbum(sharedAlbumKey, tmpFile, sharedAlbumToken)
		if err != nil {
			t.Fatalf("UploadPhotoToAlbum with token failed: %v", err)
		}
		t.Logf("✅ Successfully uploaded and linked photo with token! MediaKey: %s", mediaKey)
	})
}

func TestOwnAlbumLifecycle(t *testing.T) {
	auth := os.Getenv("GP_AUTH_DATA")
	if auth == "" {
		t.Skip("GP_AUTH_DATA environment variable not set")
	}

	c, err := client.New(auth)
	if err != nil {
		t.Fatalf("Failed to initialize client: %v", err)
	}

	// 1. Upload initial photo
	tmpFile1 := fmt.Sprintf("/tmp/gpmc_own_1_%d.jpg", time.Now().Unix())
	if err := createDummyJPEG(tmpFile1); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer os.Remove(tmpFile1)

	res, err := c.Upload(tmpFile1)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	var mediaKey1 string
	for _, k := range res {
		mediaKey1 = k
		break
	}
	if mediaKey1 == "" {
		t.Fatalf("MediaKey1 is empty")
	}
	t.Logf("✅ Initial photo uploaded: %s", mediaKey1)

	// 2. Create own album
	albumTitle := fmt.Sprintf("GPMC-Go Lifecycle Test %s", time.Now().Format("15:04:05"))
	albumKey, err := c.CreateAlbum(albumTitle, mediaKey1)
	if err != nil {
		t.Fatalf("CreateAlbum failed: %v", err)
	}
	t.Logf("✅ Album created! Title: %s, Key: %s", albumTitle, albumKey)

	// Always cleanup the created album at the end
	defer func() {
		if err := c.DeleteAlbum(albumKey); err != nil {
			t.Logf("⚠️ Cleanup DeleteAlbum error: %v", err)
		} else {
			t.Logf("✅ Cleaned up and deleted album: %s", albumKey)
		}
	}()

	// 3. Share album to get shared key and share token
	shareRes, err := c.ShareAlbum(albumKey)
	if err != nil {
		t.Fatalf("ShareAlbum failed: %v", err)
	}
	t.Logf("✅ Album shared! SharedKey: %s, ShareLink: %s, Token: %s", shareRes.SharedAlbumKey, shareRes.ShareURL, shareRes.ShareToken)

	activeKey := shareRes.SharedAlbumKey
	activeToken := shareRes.ShareToken

	// 4. List photos in album
	photos, err := c.ListPhotosInAlbum(activeKey, activeToken)
	if err != nil {
		t.Fatalf("ListPhotosInAlbum failed: %v", err)
	}
	t.Logf("✅ Listed %d photo(s) in album: %s", len(photos), photos[0].FileName)

	// 5. Add comment to album
	commentID, err := c.AddCommentToAlbum(activeKey, "Test komentar akun sendiri via gpmc-go", activeToken)
	if err != nil {
		t.Fatalf("AddCommentToAlbum failed: %v", err)
	}
	t.Logf("✅ Comment added! Comment ID: %s", commentID)

	// 6. Upload second photo directly to album
	tmpFile2 := fmt.Sprintf("/tmp/gpmc_own_2_%d.jpg", time.Now().Unix())
	if err := createDummyJPEG(tmpFile2); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer os.Remove(tmpFile2)

	mediaKey2, err := c.UploadPhotoToAlbum(activeKey, tmpFile2, activeToken)
	if err != nil {
		t.Fatalf("UploadPhotoToAlbum failed: %v", err)
	}
	t.Logf("✅ Second photo uploaded directly to album! MediaKey: %s", mediaKey2)

	// 7. Rename album
	newTitle := albumTitle + " (Renamed)"
	if err := c.RenameAlbum(albumKey, newTitle); err != nil {
		t.Fatalf("RenameAlbum failed: %v", err)
	}
	t.Logf("✅ Album renamed to: %s", newTitle)
}

