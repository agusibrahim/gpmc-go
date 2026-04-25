package client

import (
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestProgressReaderReportsInFlightBytesWithoutCommittingThem(t *testing.T) {
	var batchCompleted int64 = 10
	progressChan := make(chan ProgressUpdate, 10)
	reader := &progressReader{
		r:              io.NopCloser(&limitedReader{remaining: 5}),
		path:           "file.jpg",
		filename:       "file.jpg",
		fileSize:       5,
		batchTotal:     15,
		batchCompleted: &batchCompleted,
		progressChan:   progressChan,
	}

	_, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if got := atomic.LoadInt64(&batchCompleted); got != 10 {
		t.Fatalf("batchCompleted = %d; want 10 before successful upload accounting", got)
	}
	var last ProgressUpdate
	for len(progressChan) > 0 {
		last = <-progressChan
	}
	if last.BatchBytesSent != 15 {
		t.Fatalf("BatchBytesSent = %d; want 15 including in-flight bytes", last.BatchBytesSent)
	}
}

func TestPreflightDoesNotResumeFailedManifestEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(path, []byte("fake jpg"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	manifest := &uploadManifest{Version: 1, Files: map[string]manifestFile{
		path: {Path: path, Size: info.Size(), ModTime: info.ModTime().Unix(), Status: StatusError, LastError: "previous failure"},
	}}
	client := &Client{}
	result := client.preflightUploads(map[string]*UploadOptions{path: {}}, &uploadOptions{resume: true}, manifest)

	if len(result.resumed) != 0 {
		t.Fatalf("resumed failed entries = %d; want 0", len(result.resumed))
	}
	if len(result.valid) != 1 {
		t.Fatalf("valid entries = %d; want 1", len(result.valid))
	}
}

func TestPreflightResumesSuccessfulManifestEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(path, []byte("fake jpg"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	manifest := &uploadManifest{Version: 1, Files: map[string]manifestFile{
		path: {Path: path, Size: info.Size(), ModTime: info.ModTime().Unix(), Status: StatusDone, MediaKey: "media-key", Verified: true},
	}}
	client := &Client{}
	result := client.preflightUploads(map[string]*UploadOptions{path: {}}, &uploadOptions{resume: true}, manifest)

	if len(result.valid) != 0 {
		t.Fatalf("valid entries = %d; want 0", len(result.valid))
	}
	if len(result.resumed) != 1 {
		t.Fatalf("resumed entries = %d; want 1", len(result.resumed))
	}
	if result.resumed[0].MediaKey != "media-key" {
		t.Fatalf("resumed media key = %q; want media-key", result.resumed[0].MediaKey)
	}
}

func TestUpdateManifestFileDoesNotRestatSource(t *testing.T) {
	manifest := &uploadManifest{Version: 1, Files: map[string]manifestFile{}}
	path := filepath.Join(t.TempDir(), "deleted.jpg")
	file := FileResult{Path: path, Filename: "deleted.jpg", Size: 123, ModTime: 456, Status: StatusDone, MediaKey: "media-key", Hash: "hash", Verified: true}

	updateManifestFile(manifest, file)

	entry, ok := manifest.Files[path]
	if !ok {
		t.Fatal("manifest entry was not written")
	}
	if entry.Size != 123 || entry.ModTime != 456 || entry.MediaKey != "media-key" {
		t.Fatalf("manifest entry = %+v", entry)
	}
}

type limitedReader struct {
	remaining int
}

func (r *limitedReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if len(p) > r.remaining {
		p = p[:r.remaining]
	}
	for i := range p {
		p[i] = 'x'
	}
	r.remaining -= len(p)
	return len(p), nil
}
