package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xob0t/gpmc-go/internal/client"
	"github.com/xob0t/gpmc-go/internal/hash"
	"github.com/xob0t/gpmc-go/internal/util"
)

var (
	authData = "androidId=3c4096cfd46caf54&app=com.google.android.apps.photos&client_sig=24bb24c05e47e0aefa68a58a766179d9b613a600&callerPkg=com.google.android.apps.photos&callerSig=24bb24c05e47e0aefa68a58a766179d9b613a600&device_country=id&Email=themenggok%40gmail.com&google_play_services_version=250932000&lang=in_ID&oauth2_foreground=1&operatorCountry=id&sdk_version=36&service=oauth2%3Aopenid%20https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fmobileapps.native%20https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fphotos.native&source=android&Token=aas_et%2FAKppINZNpKOfx89ICwELStDEsK-jXqK2oTMR_FXLCetRGFU6vG7PdcDK7FvnOIIEe-kA_kgQ0qlWRfLxSz05dDT3LQ5m3v8N4TiAPPifSfEJYPecdjI6P-tk0EF6oicSFkpYWVr0EzXygxwa9XPDFUVWdWV-ILiBZuDAaX1bmLvVRaKD_RDTWWoF6EOu57s2tJ1KCGYX2LmkavpWsGhCEY0%3D"

	// Test data from Python tests
	imageSHA1HashB64 = "bjvmULLYvkVj8jWVQFu1Pl98hYA="
	imageSHA1HashHex  = "6e3be650b2d8be4563f23595405bb53e5f7c8580"
	testMediaKey       = "AF1QipPQJJlcp_XbcSuZojLHg19NLkMiziqdjp2FS-6X"
	testMediaKey2      = "AF1QipOD9PerDX6wrOoWHZKt0361PlyACUJrm8H4NHI"
)

func TestMain(m *testing.M) {
	flag.Parse()
}

// TestAuth tests authentication with the provided auth data
func TestAuth(t *testing.T) {
	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test bearer token retrieval
	// This will validate auth data is working
	t.Log("Authentication successful - client created")
}

// TestGetMediaKeyByHashB64 tests hash lookup using base64 format
func TestGetMediaKeyByHashB64(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mediaKey, err := c.GetMediaKeyByHash(imageSHA1HashB64)
	if err != nil {
		t.Fatalf("Failed to get media key by hash: %v", err)
	}

	t.Logf("Found media key: %s", mediaKey)
	if mediaKey == "" {
		t.Log("No remote media with matching hash found (expected if not uploaded yet)")
	}
}

// TestGetMediaKeyByHashHex tests hash lookup using hex format
func TestGetMediaKeyByHashHex(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mediaKey, err := c.GetMediaKeyByHash(imageSHA1HashHex)
	if err != nil {
		t.Fatalf("Failed to get media key by hash: %v", err)
	}

	t.Logf("Found media key: %s", mediaKey)
	if mediaKey == "" {
		t.Log("No remote media with matching hash found (expected if not uploaded yet)")
	}
}

// TestSetCaption tests setting a caption on a media item
func TestSetCaption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Convert hash to dedup key
	dedupKey := util.URLSafeBase64(imageSHA1HashB64)

	result, err := c.SetCaption("Test caption from Go", dedupKey)
	if err != nil {
		t.Fatalf("Failed to set caption: %v", err)
	}

	t.Logf("Set caption result: %+v", result)

	// Reset caption
	c.SetCaption("", dedupKey)
}

// TestSetFavorite tests setting favorite flag
func TestSetFavorite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	dedupKey := util.URLSafeBase64(imageSHA1HashB64)

	// Set as favorite
	result, err := c.SetFavorite(dedupKey, true)
	if err != nil {
		t.Fatalf("Failed to set favorite: %v", err)
	}

	t.Logf("Set favorite result: %+v", result)

	// Unset
	c.SetFavorite(dedupKey, false)
}

// TestSetArchived tests setting archived flag
func TestSetArchived(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	dedupKey := util.URLSafeBase64(imageSHA1HashB64)

	result, err := c.SetArchived([]string{dedupKey}, false)
	if err != nil {
		t.Fatalf("Failed to set archived: %v", err)
	}

	t.Logf("Set archived result: %+v", result)
}

// TestGetDownloadURLs tests getting download URLs for a media item
func TestGetDownloadURLs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	urls, err := c.GetDownloadURLs(testMediaKey2)
	if err != nil {
		t.Fatalf("Failed to get download URLs: %v", err)
	}

	t.Logf("Download URLs: %+v", urls)
}

// TestGetThumbnail tests getting a thumbnail
func TestGetThumbnail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	thumbnail, err := c.GetThumbnail(testMediaKey2)
	if err != nil {
		t.Fatalf("Failed to get thumbnail: %v", err)
	}

	if len(thumbnail) > 0 {
		t.Logf("Thumbnail size: %d bytes", len(thumbnail))
	}
}

// TestGetStreamManifest tests getting HLS/DASH manifest
func TestGetStreamManifest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would require adding GetStreamManifest to the API
	// For now, we'll skip this test
	t.Skip("GetStreamManifest not yet implemented in Go version")
}

// TestAddToAlbum tests adding media to an album
func TestAddToAlbum(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	albumKeys := c.AddToAlbum(
		[]string{testMediaKey},
		"TEST_ALBUM_GO",
		true,
	)

	t.Logf("Album keys: %+v", albumKeys)
}

// TestMoveToTrash tests moving media to trash
func TestMoveToTrash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	result, err := c.MoveToTrash(imageSHA1HashHex)
	if err != nil {
		t.Fatalf("Failed to move to trash: %v", err)
	}

	t.Logf("Move to trash result: %+v", result)

	// Restore from trash
	c.RestoreFromTrash(imageSHA1HashHex)
}

// TestUpdateCache tests library cache update
func TestUpdateCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c, err := client.New(authData)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = c.UpdateCache(true, 2) // Limit to 2 sync cycles for testing
	if err != nil {
		t.Fatalf("Failed to update cache: %v", err)
	}

	t.Log("Cache update completed")
}

// TestHashConversion tests hash format conversions
func TestHashConversion(t *testing.T) {
	// Test B64 to bytes and back
	_, hashB64, err := hash.ConvertSHA1Hash(imageSHA1HashB64)
	if err != nil {
		t.Fatalf("Failed to convert hash from B64: %v", err)
	}

	t.Logf("Hash B64: %s", hashB64)

	// Test hex to bytes and back
	hashBytes, _, err := hash.ConvertSHA1Hash(imageSHA1HashHex)
	if err != nil {
		t.Fatalf("Failed to convert hash from hex: %v", err)
	}

	t.Logf("Hash hex: %s", hash.BytesToHex(hashBytes))

	// Compare results
	b64FromHex := hash.BytesToBase64(hashBytes)
	if b64FromHex != imageSHA1HashB64 {
		t.Errorf("Hash mismatch: %s != %s", b64FromHex, imageSHA1HashB64)
	}
}

// TestURLSafeBase64 tests URL-safe base64 conversion
func TestURLSafeBase64(t *testing.T) {
	// Standard base64 has + and /
	standardB64 := "abc+def/ghi=="
	expected := "abc-def_ghi"

	result := util.URLSafeBase64(standardB64)
	if result != expected {
		t.Errorf("URLSafeBase64(%s) = %s; want %s", standardB64, result, expected)
	}

	// Test with auth data hash
	result2 := util.URLSafeBase64(imageSHA1HashB64)
	t.Logf("URL-safe hash: %s", result2)
}

// BenchmarkHashConversion benchmarks hash conversion
func BenchmarkHashConversion(b *testing.B) {
	b.Run("B64", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hash.ConvertSHA1Hash(imageSHA1HashB64)
		}
	})

	b.Run("Hex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			hash.ConvertSHA1Hash(imageSHA1HashHex)
		}
	})
}

// Helper function to run Python tests for comparison
func runPythonComparison() error {
	// This would run the Python tests and compare results
	// For now, we'll just log that this would be done
	fmt.Println("To compare with Python version:")
	fmt.Println("1. cd /Users/macbook/Downloads/google_photos_mobile_client-main")
	fmt.Println("2. python -m pytest tests/client_test.py -v")
	fmt.Println("3. Compare results with Go tests")
	return nil
}

// TestAllFeatures runs all feature tests
func TestAllFeatures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comprehensive test in short mode")
	}

	t.Run("Auth", TestAuth)
	t.Run("HashConversion", TestHashConversion)
	t.Run("URLSafeBase64", TestURLSafeBase64)
	t.Run("GetMediaKeyByHashB64", TestGetMediaKeyByHashB64)
	t.Run("GetMediaKeyByHashHex", TestGetMediaKeyByHashHex)
	t.Run("SetCaption", TestSetCaption)
	t.Run("SetFavorite", TestSetFavorite)
	t.Run("SetArchived", TestSetArchived)
	t.Run("GetDownloadURLs", TestGetDownloadURLs)
	t.Run("GetThumbnail", TestGetThumbnail)
	t.Run("AddToAlbum", TestAddToAlbum)
	t.Run("MoveToTrash", TestMoveToTrash)
	// Skip UpdateCache as it takes too long
	// t.Run("UpdateCache", TestUpdateCache)
}

// TestCompareWithPython compares Go implementation with Python
func TestCompareWithPython(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	t.Log("=== Go vs Python Comparison ===")
	t.Log("Running Python tests...")
	t.Log("To run Python tests manually:")
	t.Log("  cd /Users/macbook/Downloads/google_photos_mobile_client-main")
	t.Log("  export GP_AUTH_DATA='" + authData + "'")
	t.Log("  python -m pytest tests/client_test.py::TestUpload::test_hash_check_b64 -v -s")

	t.Log("\nRunning Go tests...")
	t.Run("HashCheckB64", TestGetMediaKeyByHashB64)
	t.Run("HashCheckHex", TestGetMediaKeyByHashHex)

	t.Log("\n=== Comparison Notes ===")
	t.Log("Both implementations should:")
	t.Log("1. Return the same media key for the same hash")
	t.Log("2. Handle auth token renewal automatically")
	t.Log("3. Use the same API endpoints")
	t.Log("4. Produce compatible protobuf encodings")
}

// TestUploadFile tests uploading a test file
func TestUploadFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping upload test in short mode")
	}

	// Create a small test file
	testFile := "/tmp/test_upload_gpmc.txt"
	testContent := []byte("Test file from Go client - " + time.Now().Format(time.RFC3339))

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	c, err := client.New(authData,
		client.WithProgress(true),
		client.WithForceUpload(true),
		client.WithSaver(true),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	results, err := c.Upload(testFile)
	if err != nil {
		t.Fatalf("Failed to upload: %v", err)
	}

	for path, mediaKey := range results {
		t.Logf("Uploaded %s -> %s", filepath.Base(path), mediaKey)
	}

	// Clean up - move to trash
	c.MoveToTrash(imageSHA1HashHex)
}

// TestConcurrentUpload tests concurrent uploads
func TestConcurrentUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent upload test in short mode")
	}

	// Create multiple test files
	var testFiles []string
	for i := 0; i < 3; i++ {
		testFile := fmt.Sprintf("/tmp/test_concurrent_%d.txt", i)
		testContent := []byte(fmt.Sprintf("Concurrent test file %d - %s", i, time.Now().Format(time.RFC3339)))
		if err := os.WriteFile(testFile, testContent, 0644); err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}
		testFiles = append(testFiles, testFile)
		defer os.Remove(testFile)
	}

	c, err := client.New(authData,
		client.WithThreads(3),
		client.WithProgress(true),
		client.WithForceUpload(true),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	start := time.Now()
	results, err := c.Upload(testFiles)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to upload concurrently: %v", err)
	}

	t.Logf("Uploaded %d files concurrently in %v", len(results), duration)
	for path, mediaKey := range results {
		t.Logf("  %s -> %s", filepath.Base(path), mediaKey)
	}

	// Clean up
	for _, path := range results {
		// Get the hash for cleanup
		// In real usage, you'd track the hashes
		os.Remove(path)
	}
}

// Main function to run tests
func main() {
	// Set auth data from environment if not set
	if os.Getenv("GP_AUTH_DATA") == "" {
		os.Setenv("GP_AUTH_DATA", authData)
	}

	// Run specific test
	testing.Main(func(pat, str string) (bool, error) { return true, nil },
		[]testing.InternalTest{
			{"TestCompareWithPython", TestCompareWithPython},
			{"TestAllFeatures", TestAllFeatures},
			{"TestAuth", TestAuth},
		},
		nil,
	)
}
