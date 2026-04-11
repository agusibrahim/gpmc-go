package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/xob0t/gpmc-go/internal/proto"
)

// GetUploadToken obtains an upload token for a file
func (a *Api) GetUploadToken(shaHashB64 string, fileSize int) (string, error) {
	protoBody := map[string]interface{}{
		"1": int64(2),
		"2": int64(2),
		"3": int64(1),
		"4": int64(3),
		"7": int64(fileSize),
	}

	serializedData := proto.EncodeMessage(protoBody, proto.GetUploadTokenDef)

	token, err := a.BearerToken()
	if err != nil {
		return "", err
	}

	headers := map[string]string{
		"Accept-Encoding":    "gzip",
		"Accept-Language":    a.language,
		"Content-Type":       "application/x-protobuf",
		"User-Agent":         a.userAgent,
		"Authorization":      "Bearer " + token,
		"X-Goog-Hash":        "sha1=" + shaHashB64,
		"X-Upload-Content-Length": strconv.Itoa(fileSize),
	}

	resp, err := a.makeRequest("POST", GetUploadURL(), serializedData, headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload token request failed with status %d", resp.StatusCode)
	}

	return resp.Header.Get("X-GUploader-UploadID"), nil
}

// FindRemoteMediaByHash checks if a file with the given hash exists in Google Photos
func (a *Api) FindRemoteMediaByHash(sha1Hash []byte) (string, error) {
	protoBody := map[string]interface{}{
		"1": map[string]interface{}{
			"1": map[string]interface{}{
				"1": sha1Hash,
			},
			"2": map[string]interface{}{},
		},
	}

	serializedData := proto.EncodeMessage(protoBody, proto.FindRemoteMediaByHashDef)

	token, err := a.BearerToken()
	if err != nil {
		return "", err
	}

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": a.language,
		"Content-Type":    "application/x-protobuf",
		"User-Agent":      a.userAgent,
		"Authorization":   "Bearer " + token,
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointFindByHash), serializedData, headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("find by hash request failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.FindRemoteMediaByHashDef)
	if err != nil {
		return "", err
	}

	// Extract media key from nested response
	if field1, ok := decodedMessage["1"].(map[string]interface{}); ok {
		if field2, ok := field1["2"].(map[string]interface{}); ok {
			if field1, ok := field2["2"].(map[string]interface{}); ok {
				if mediaKey, ok := field1["1"].(string); ok {
					return mediaKey, nil
				}
			}
		}
	}

	return "", nil // Not found
}

// UploadFile uploads file data to Google Photos
func (a *Api) UploadFile(file io.Reader, uploadToken string) (map[string]interface{}, error) {
	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": a.language,
		"User-Agent":      a.userAgent,
		"Authorization":   "Bearer " + token,
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	resp, err := a.makeRequest("PUT", GetUploadURLWithToken(uploadToken), data, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("file upload failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, nil)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}

// CommitUpload commits an uploaded file to Google Photos
func (a *Api) CommitUpload(uploadResp map[string]interface{}, fileName string, sha1Hash []byte, quality string, uploadTimestamp int) (string, error) {
	qualityMap := map[string]int{
		"saver":    1,
		"original": 3,
	}

	if uploadTimestamp == 0 {
		uploadTimestamp = int(time.Now().Unix())
	}

	protoBody := map[string]interface{}{
		"1": map[string]interface{}{
			"1": uploadResp,
			"2": fileName,
			"3": sha1Hash,
			"4": map[string]interface{}{
				"1": int64(uploadTimestamp),
				"2": int64(46000000),
			},
			"7": int64(qualityMap[quality]),
			"10": int64(1),
		},
		"2": map[string]interface{}{
			"3": a.model,
			"4": a.make,
			"5": int64(a.androidAPIVersion),
		},
		"3": []byte{0x01, 0x03},
	}

	serializedData := proto.EncodeMessage(protoBody, proto.CommitUploadDef)

	token, err := a.BearerToken()
	if err != nil {
		return "", err
	}

	headers := map[string]string{
		"Accept-Encoding":           "gzip",
		"Accept-Language":           a.language,
		"Content-Type":              "application/x-protobuf",
		"User-Agent":                a.userAgent,
		"Authorization":             "Bearer " + token,
		"x-goog-ext-173412678-bin":  "CgcIAhClARgC",
		"x-goog-ext-174067345-bin":  "CgIIAg==",
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointCommitUpload), serializedData, headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("commit upload failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.CommitUploadDef)
	if err != nil {
		return "", fmt.Errorf("upload rejected by API: %w", err)
	}

	// Extract media key from response
	if field1, ok := decodedMessage["1"].(map[string]interface{}); ok {
		if field3, ok := field1["3"].(map[string]interface{}); ok {
			if mediaKey, ok := field3["1"].(string); ok {
				return mediaKey, nil
			}
		}
	}

	return "", fmt.Errorf("could not extract media key from response")
}

// CreateAlbum creates a new album with the given media items
func (a *Api) CreateAlbum(albumName string, mediaKeys []string) (string, error) {
	// Build media keys array for field "4"
	mediaKeyItems := make([]map[string]interface{}, len(mediaKeys))
	for i, key := range mediaKeys {
		mediaKeyItems[i] = map[string]interface{}{
			"1": map[string]interface{}{
				"1": key,
			},
		}
	}

	protoBody := map[string]interface{}{
		"1": albumName,
		"2": int64(time.Now().Unix()),
		"3": int64(1),
		"4": mediaKeyItems,
		"6": map[string]interface{}{},
		"7": map[string]interface{}{
			"1": int64(3),
		},
		"8": map[string]interface{}{
			"3": a.model,
			"4": a.make,
			"5": int64(a.androidAPIVersion),
		},
	}

	serializedData := proto.EncodeMessage(protoBody, proto.CreateAlbumDef)

	token, err := a.BearerToken()
	if err != nil {
		return "", err
	}

	headers := map[string]string{
		"Accept-Encoding":          "gzip",
		"Accept-Language":          a.language,
		"Content-Type":             "application/x-protobuf",
		"User-Agent":               a.userAgent,
		"Authorization":            "Bearer " + token,
		"x-goog-ext-173412678-bin": "CgcIAhClARgC",
		"x-goog-ext-174067345-bin": "CgIIAg==",
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointCreateAlbum), serializedData, headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create album failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.CreateAlbumDef)
	if err != nil {
		return "", err
	}

	// Extract album media key
	if field1, ok := decodedMessage["1"].(map[string]interface{}); ok {
		if albumKey, ok := field1["1"].(string); ok {
			return albumKey, nil
		}
	}

	return "", fmt.Errorf("could not extract album key from response")
}

// AddMediaToAlbum adds media items to an existing album
func (a *Api) AddMediaToAlbum(albumMediaKey string, mediaKeys []string) (map[string]interface{}, error) {
	protoBody := map[string]interface{}{
		"1": mediaKeys,
		"2": albumMediaKey,
		"5": map[string]interface{}{
			"1": int64(2),
		},
		"6": map[string]interface{}{
			"3": a.model,
			"4": a.make,
			"5": int64(a.androidAPIVersion),
		},
		"7": int64(time.Now().Unix()),
	}

	serializedData := proto.EncodeMessage(protoBody, proto.AddMediaToAlbumDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding":          "gzip",
		"Accept-Language":          a.language,
		"Content-Type":             "application/x-protobuf",
		"User-Agent":               a.userAgent,
		"Authorization":            "Bearer " + token,
		"x-goog-ext-173412678-bin": "CgcIAhClARgC",
		"x-goog-ext-174067345-bin": "CgIIAg==",
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointAddToAlbum), serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("add to album failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.AddMediaToAlbumDef)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}

// MoveRemoteMediaToTrash moves media items to trash
func (a *Api) MoveRemoteMediaToTrash(dedupKeys []string) (map[string]interface{}, error) {
	protoBody := map[string]interface{}{
		"2": int64(1),
		"3": dedupKeys,
		"4": int64(1),
		"8": map[string]interface{}{
			"4": map[string]interface{}{
				"2": map[string]interface{}{},
				"3": map[string]interface{}{
					"1": map[string]interface{}{},
				},
				"4": map[string]interface{}{},
				"5": map[string]interface{}{
					"1": map[string]interface{}{},
				},
			},
		},
		"9": map[string]interface{}{
			"1": int64(5),
			"2": map[string]interface{}{
				"1": int64(a.clientVersionCode),
				"2": strconv.Itoa(a.androidAPIVersion),
			},
		},
	}

	serializedData := proto.EncodeMessage(protoBody, proto.MoveToTrashDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": a.language,
		"Content-Type":    "application/x-protobuf",
		"User-Agent":      a.userAgent,
		"Authorization":   "Bearer " + token,
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointTrashDelete), serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("move to trash failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.MoveToTrashDef)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}

// DeleteRemoteMediaPermanently permanently deletes media items
func (a *Api) DeleteRemoteMediaPermanently(dedupKeys []string) (map[string]interface{}, error) {
	protoBody := map[string]interface{}{
		"2": int64(2),
		"3": dedupKeys,
		"4": int64(2),
		"8": map[string]interface{}{
			"4": map[string]interface{}{
				"2": map[string]interface{}{},
				"3": map[string]interface{}{
					"1": map[string]interface{}{},
				},
				"4": map[string]interface{}{},
				"5": map[string]interface{}{
					"1": map[string]interface{}{},
				},
			},
		},
		"9": "",
	}

	serializedData := proto.EncodeMessage(protoBody, proto.DeletePermanentlyDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": a.language,
		"Content-Type":    "application/x-protobuf",
		"User-Agent":      a.userAgent,
		"Authorization":   "Bearer " + token,
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointTrashDelete), serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("delete permanently failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.DeletePermanentlyDef)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}

// SetItemCaption sets the caption for a media item
func (a *Api) SetItemCaption(caption, dedupKey string) (map[string]interface{}, error) {
	protoBody := map[string]interface{}{
		"2": caption,
		"3": dedupKey,
	}

	serializedData := proto.EncodeMessage(protoBody, proto.SetCaptionDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": a.language,
		"Content-Type":    "application/x-protobuf",
		"User-Agent":      a.userAgent,
		"Authorization":   "Bearer " + token,
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointSetCaption), serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("set caption failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.SetCaptionDef)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}

// SetFavorite sets or unsets the favorite flag on a media item
func (a *Api) SetFavorite(dedupKey string, isFavorite bool) (map[string]interface{}, error) {
	actionMap := map[bool]int64{false: 2, true: 1}
	protoBody := map[string]interface{}{
		"1": map[string]interface{}{
			"2": dedupKey,
		},
		"2": map[string]interface{}{
			"1": actionMap[isFavorite],
		},
		"3": map[string]interface{}{
			"1": map[string]interface{}{
				"19": map[string]interface{}{},
			},
		},
	}

	serializedData := proto.EncodeMessage(protoBody, proto.SetFavoriteDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": a.language,
		"Content-Type":    "application/x-protobuf",
		"User-Agent":      a.userAgent,
		"Authorization":   "Bearer " + token,
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointSetFavorite), serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("set favorite failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.SetFavoriteDef)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}

// SetArchived sets or unsets the archived flag on media items
func (a *Api) SetArchived(dedupKeys []string, isArchived bool) (map[string]interface{}, error) {
	actionMap := map[bool]int64{false: 2, true: 1}
	items := make([]map[string]interface{}, len(dedupKeys))
	for i, key := range dedupKeys {
		items[i] = map[string]interface{}{
			"1": key,
			"2": map[string]interface{}{
				"1": actionMap[isArchived],
			},
		}
	}

	protoBody := map[string]interface{}{
		"1": items,
		"3": int64(1),
	}

	serializedData := proto.EncodeMessage(protoBody, proto.SetArchivedDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": a.language,
		"Content-Type":    "application/x-protobuf",
		"User-Agent":      a.userAgent,
		"Authorization":   "Bearer " + token,
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointSetArchived), serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("set archived failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.SetArchivedDef)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}

// GetDownloadURLs gets download URLs for a media item
func (a *Api) GetDownloadURLs(mediaKey string) ([]string, error) {
	protoBody := map[string]interface{}{
		"1": map[string]interface{}{
			"1": map[string]interface{}{
				"1": mediaKey,
			},
		},
		"2": map[string]interface{}{
			"1": map[string]interface{}{
				"7": map[string]interface{}{
					"2": map[string]interface{}{},
				},
			},
			"5": map[string]interface{}{
				"2": map[string]interface{}{},
				"3": map[string]interface{}{},
				"5": map[string]interface{}{
					"1": map[string]interface{}{},
					"3": int64(0),
				},
			},
		},
	}

	serializedData := proto.EncodeMessage(protoBody, proto.GetDownloadURLsDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": a.language,
		"Content-Type":    "application/x-protobuf",
		"User-Agent":      a.userAgent,
		"Authorization":   "Bearer " + token,
	}

	resp, err := a.makeRequest("POST", DownloadURL, serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get download URLs failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.GetDownloadURLsDef)
	if err != nil {
		return nil, err
	}

	// Extract URLs from response
	// This is a simplified extraction - the real implementation would need to handle the nested structure
	var urls []string
	if field1, ok := decodedMessage["1"].(map[string]interface{}); ok {
		if field2, ok := field1["2"].(map[string]interface{}); ok {
			// Extract URLs from nested structure
			for _, v := range field2 {
				if urlMap, ok := v.(map[string]interface{}); ok {
					if url, ok := urlMap["1"].(string); ok {
						urls = append(urls, url)
					}
				}
			}
		}
	}

	return urls, nil
}

// RestoreFromTrash restores media items from trash
func (a *Api) RestoreFromTrash(dedupKeys []string) (map[string]interface{}, error) {
	protoBody := map[string]interface{}{
		"2": int64(1),
		"3": dedupKeys,
		"4": int64(1),
		"8": map[string]interface{}{
			"4": map[string]interface{}{
				"2": map[string]interface{}{},
				"3": map[string]interface{}{
					"1": map[string]interface{}{},
				},
				"4": map[string]interface{}{},
				"5": map[string]interface{}{
					"1": map[string]interface{}{},
				},
			},
		},
		"9": map[string]interface{}{
			"1": int64(5),
			"2": map[string]interface{}{
				"1": int64(a.clientVersionCode),
				"2": strconv.Itoa(a.androidAPIVersion),
			},
		},
	}

	serializedData := proto.EncodeMessage(protoBody, proto.RestoreFromTrashDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": a.language,
		"Content-Type":    "application/x-protobuf",
		"User-Agent":      a.userAgent,
		"Authorization":   "Bearer " + token,
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointTrashDelete), serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("restore from trash failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.RestoreFromTrashDef)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}

// GetThumbnail gets a thumbnail image for a media item
func (a *Api) GetThumbnail(mediaKey string) ([]byte, error) {
	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept-Language": a.language,
		"User-Agent":      a.userAgent,
		"Authorization":   "Bearer " + token,
	}

	resp, err := a.makeRequest("GET", GetThumbnailURL(mediaKey), nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get thumbnail failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// makeRequest is a helper function to make HTTP requests
func (a *Api) makeRequest(method, url string, data []byte, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if data != nil {
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Make request
	client := a.newSession()
	return client.Do(req)
}

// Helper function to convert SHA1 bytes to base64 for API calls
func sha1BytesToBase64(sha1Hash []byte) string {
	return base64.StdEncoding.EncodeToString(sha1Hash)
}

// GetLibraryState gets the current library state
func (a *Api) GetLibraryState(syncToken string) (map[string]interface{}, error) {
	protoBody := map[string]interface{}{
		"1": map[string]interface{}{
			"1": map[string]interface{}{
				"1": map[string]interface{}{
					"1": map[string]interface{}{},
					"3": map[string]interface{}{},
					"4": map[string]interface{}{},
					"5": map[string]interface{}{
						"1":  map[string]interface{}{},
						"2":  map[string]interface{}{},
						"3":  map[string]interface{}{},
						"4":  map[string]interface{}{},
						"5":  map[string]interface{}{},
						"7":  map[string]interface{}{},
					},
					"6":  map[string]interface{}{},
					"7":  map[string]interface{}{
						"2": map[string]interface{}{},
					},
					"15": map[string]interface{}{},
					"16": map[string]interface{}{},
					"17": map[string]interface{}{},
					"19": map[string]interface{}{},
					"20": map[string]interface{}{},
					"21": map[string]interface{}{
						"5": map[string]interface{}{
							"3": map[string]interface{}{},
						},
						"6": map[string]interface{}{},
					},
					"25": map[string]interface{}{},
					"30": map[string]interface{}{
						"2": map[string]interface{}{},
					},
					"31": map[string]interface{}{},
					"32": map[string]interface{}{},
					"33": map[string]interface{}{
						"1": map[string]interface{}{},
					},
					"34": map[string]interface{}{},
					"36": map[string]interface{}{},
					"37": map[string]interface{}{},
					"38": map[string]interface{}{},
					"39": map[string]interface{}{},
					"40": map[string]interface{}{},
					"41": map[string]interface{}{},
				},
			},
		},
		"2": map[string]interface{}{
			"1": map[string]interface{}{
				"1": map[string]interface{}{
					"2": map[string]interface{}{},
					"3": map[string]interface{}{
						"2": map[string]interface{}{
							"3": map[string]interface{}{},
						},
						"4": map[string]interface{}{
							"2": map[string]interface{}{},
						},
					},
					"4": int64(1),
					"5": map[string]interface{}{
						"2": map[string]interface{}{
							"2": map[string]interface{}{},
							"3": map[string]interface{}{
								"2": int64(1),
							},
						},
					},
					"6": int64(1),
				},
			},
		},
		"3": map[string]interface{}{
			"2": map[string]interface{}{},
			"3": map[string]interface{}{
				"7": map[string]interface{}{},
				"8": map[string]interface{}{},
				"14": map[string]interface{}{
					"1": map[string]interface{}{},
				},
			},
		},
	}

	serializedData := proto.EncodeMessage(protoBody, proto.GetLibStateDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding":          "gzip",
		"Accept-Language":          a.language,
		"Content-Type":             "application/x-protobuf",
		"User-Agent":               a.userAgent,
		"Authorization":            "Bearer " + token,
		"x-goog-ext-173412678-bin": "CgcIAhClARgC",
		"x-goog-ext-174067345-bin": "CgIIAg==",
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointLibraryState), serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get library state failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.LibStateResponseFixDef)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}

// GetLibraryPageInit gets the initial library page during first sync
func (a *Api) GetLibraryPageInit(resumeToken string) (map[string]interface{}, error) {
	protoBody := map[string]interface{}{
		"1": map[string]interface{}{
			"1": map[string]interface{}{
				"1": map[string]interface{}{
					"1": map[string]interface{}{},
					"3": map[string]interface{}{},
					"4": map[string]interface{}{},
					"5": map[string]interface{}{
						"1":  map[string]interface{}{},
						"2":  map[string]interface{}{},
						"3":  map[string]interface{}{},
						"4":  map[string]interface{}{},
						"5":  map[string]interface{}{},
						"7":  map[string]interface{}{},
					},
					"6":  map[string]interface{}{},
					"7":  map[string]interface{}{
						"2": map[string]interface{}{},
					},
					"15": map[string]interface{}{},
					"16": map[string]interface{}{},
					"17": map[string]interface{}{},
					"19": map[string]interface{}{},
					"20": map[string]interface{}{},
					"21": map[string]interface{}{
						"5": map[string]interface{}{
							"3": map[string]interface{}{},
						},
						"6": map[string]interface{}{},
					},
					"25": map[string]interface{}{},
					"30": map[string]interface{}{
						"2": map[string]interface{}{},
					},
					"31": map[string]interface{}{},
					"32": map[string]interface{}{},
					"33": map[string]interface{}{
						"1": map[string]interface{}{},
					},
					"34": map[string]interface{}{},
					"36": map[string]interface{}{},
					"37": map[string]interface{}{},
					"38": map[string]interface{}{},
					"39": map[string]interface{}{},
					"40": map[string]interface{}{},
					"41": map[string]interface{}{},
				},
			},
			"4": resumeToken,
		},
	}

	serializedData := proto.EncodeMessage(protoBody, proto.GetLibPageInitDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding":          "gzip",
		"Accept-Language":          a.language,
		"Content-Type":             "application/x-protobuf",
		"User-Agent":               a.userAgent,
		"Authorization":            "Bearer " + token,
		"x-goog-ext-173412678-bin": "CgcIAhClARgC",
		"x-goog-ext-174067345-bin": "CgIIAg==",
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointLibraryState), serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get library page init failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.LibStateResponseFixDef)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}

// GetLibraryPage gets a library page during delta sync
func (a *Api) GetLibraryPage(resumeToken, syncToken string) (map[string]interface{}, error) {
	protoBody := map[string]interface{}{
		"1": map[string]interface{}{
			"1": map[string]interface{}{
				"1": map[string]interface{}{},
				"3": map[string]interface{}{},
				"4": map[string]interface{}{},
				"5": map[string]interface{}{
					"1":  map[string]interface{}{},
					"2":  map[string]interface{}{},
					"3":  map[string]interface{}{},
					"4":  map[string]interface{}{},
					"5":  map[string]interface{}{},
					"7":  map[string]interface{}{},
				},
				"6":  map[string]interface{}{},
				"7":  map[string]interface{}{
					"2": map[string]interface{}{},
				},
				"15": map[string]interface{}{},
				"16": map[string]interface{}{},
				"17": map[string]interface{}{},
				"19": map[string]interface{}{},
				"20": map[string]interface{}{},
				"21": map[string]interface{}{
					"5": map[string]interface{}{
						"3": map[string]interface{}{},
					},
					"6": map[string]interface{}{},
				},
				"25": map[string]interface{}{},
				"30": map[string]interface{}{
					"2": map[string]interface{}{},
				},
				"31": map[string]interface{}{},
				"32": map[string]interface{}{},
				"33": map[string]interface{}{
					"1": map[string]interface{}{},
				},
				"34": map[string]interface{}{},
				"36": map[string]interface{}{},
				"37": map[string]interface{}{},
				"38": map[string]interface{}{},
				"39": map[string]interface{}{},
				"40": map[string]interface{}{},
				"41": map[string]interface{}{},
			},
			"4": resumeToken,
			"6": syncToken,
		},
	}

	serializedData := proto.EncodeMessage(protoBody, proto.GetLibPageDef)

	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept-Encoding":          "gzip",
		"Accept-Language":          a.language,
		"Content-Type":             "application/x-protobuf",
		"User-Agent":               a.userAgent,
		"Authorization":            "Bearer " + token,
		"x-goog-ext-173412678-bin": "CgcIAhClARgC",
		"x-goog-ext-174067345-bin": "CgIIAg==",
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointLibraryState), serializedData, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get library page failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	decodedMessage, err := proto.DecodeMessage(body, proto.LibStateResponseFixDef)
	if err != nil {
		return nil, err
	}

	return decodedMessage, nil
}
