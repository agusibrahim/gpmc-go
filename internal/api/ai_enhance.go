package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/agusibrahim/gpmc-go/internal/models"
	"google.golang.org/protobuf/encoding/protowire"
)

// EnhancePhoto sends an image to Google Photos AI Magic Editor Preset (Effect 4)
// and returns all high-resolution AI-enhanced variations.
func (a *Api) EnhancePhoto(imgBytes []byte) ([]models.AIEnhanceResult, error) {
	if len(imgBytes) == 0 {
		return nil, fmt.Errorf("empty image payload")
	}

	// 1. Ensure image is in JPEG format
	jpegBytes, err := ensureJPEG(imgBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare JPEG image: %w", err)
	}
	fileSize := len(jpegBytes)

	token, err := a.BearerToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get bearer token: %w", err)
	}

	// 2. Step 1: Initialize interactive upload session
	var initProto []byte
	initProto = protowire.AppendTag(initProto, 1, protowire.VarintType)
	initProto = protowire.AppendVarint(initProto, 4)
	initProto = protowire.AppendTag(initProto, 2, protowire.VarintType)
	initProto = protowire.AppendVarint(initProto, 1)
	initProto = protowire.AppendTag(initProto, 7, protowire.VarintType)
	initProto = protowire.AppendVarint(initProto, uint64(fileSize))

	initReq, err := http.NewRequest("POST", BaseUploadURL, bytes.NewReader(initProto))
	if err != nil {
		return nil, err
	}
	initReq.Header.Set("Content-Type", "application/x-protobuf")
	initReq.Header.Set("X-Goog-Upload-File-Name", "enchilada_upload.jpg")
	initReq.Header.Set("X-Upload-Content-Length", strconv.Itoa(fileSize))
	initReq.Header.Set("Authorization", "Bearer "+token)
	initReq.Header.Set("User-Agent", a.userAgent)

	client := a.newUploadSession()
	initResp, err := client.Do(initReq)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize upload session: %w", err)
	}
	defer initResp.Body.Close()

	if initResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(initResp.Body)
		return nil, fmt.Errorf("upload init failed with status %d: %s", initResp.StatusCode, string(body))
	}

	uploadID := initResp.Header.Get("X-GUploader-UploadID")
	if uploadID == "" {
		if loc := initResp.Header.Get("Location"); loc != "" {
			if parsed, err := url.Parse(loc); err == nil {
				uploadID = parsed.Query().Get("upload_id")
			}
		}
	}
	if uploadID == "" {
		return nil, fmt.Errorf("server did not return X-GUploader-UploadID")
	}

	// 3. Step 2: PUT raw JPEG binary
	putURL := BaseUploadURL + "?upload_id=" + uploadID
	putReq, err := http.NewRequest("PUT", putURL, bytes.NewReader(jpegBytes))
	if err != nil {
		return nil, err
	}
	putReq.Header.Set("Content-Type", "image/jpeg")
	putReq.Header.Set("Content-Length", strconv.Itoa(fileSize))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("User-Agent", a.userAgent)

	putResp, err := client.Do(putReq)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image binary: %w", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		return nil, fmt.Errorf("image upload failed with status %d: %s", putResp.StatusCode, string(body))
	}

	uploadTokenBytes, err := io.ReadAll(putResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload response token: %w", err)
	}

	// 4. Step 3: PhotosGenerateMagicEditorPresetEffect RPC
	// Construct Protobuf Request Payload:
	// Field 1: { 2: uploadTokenBytes, 3: "" }
	var innerF1 []byte
	innerF1 = protowire.AppendTag(innerF1, 2, protowire.BytesType)
	innerF1 = protowire.AppendBytes(innerF1, uploadTokenBytes)
	innerF1 = protowire.AppendTag(innerF1, 3, protowire.BytesType)
	innerF1 = protowire.AppendBytes(innerF1, []byte{})

	var rpcReqBytes []byte
	rpcReqBytes = protowire.AppendTag(rpcReqBytes, 1, protowire.BytesType)
	rpcReqBytes = protowire.AppendBytes(rpcReqBytes, innerF1)

	// Field 2: { 18: "" }
	var innerF2 []byte
	innerF2 = protowire.AppendTag(innerF2, 18, protowire.BytesType)
	innerF2 = protowire.AppendBytes(innerF2, []byte{})
	rpcReqBytes = protowire.AppendTag(rpcReqBytes, 2, protowire.BytesType)
	rpcReqBytes = protowire.AppendBytes(rpcReqBytes, innerF2)

	// Field 3: { 1: 4, 2: 1 } -> Effect Type 4 (Magic Editor Preset)
	var innerF3 []byte
	innerF3 = protowire.AppendTag(innerF3, 1, protowire.VarintType)
	innerF3 = protowire.AppendVarint(innerF3, 4)
	innerF3 = protowire.AppendTag(innerF3, 2, protowire.VarintType)
	innerF3 = protowire.AppendVarint(innerF3, 1)
	rpcReqBytes = protowire.AppendTag(rpcReqBytes, 3, protowire.BytesType)
	rpcReqBytes = protowire.AppendBytes(rpcReqBytes, innerF3)

	rpcReq, err := http.NewRequest("POST", URLMagicEditorPresetEffect, bytes.NewReader(rpcReqBytes))
	if err != nil {
		return nil, err
	}
	rpcReq.Header.Set("Content-Type", "application/x-protobuf")
	rpcReq.Header.Set("Authorization", "Bearer "+token)
	rpcReq.Header.Set("User-Agent", a.userAgent)
	rpcReq.Header.Set("x-goog-ext-173412678-bin", "CgcIAhClARgC")
	rpcReq.Header.Set("x-goog-ext-174067345-bin", "CgIIAg==")

	rpcResp, err := client.Do(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("AI enhancement RPC failed: %w", err)
	}
	defer rpcResp.Body.Close()

	if rpcResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rpcResp.Body)
		return nil, fmt.Errorf("AI enhancement RPC returned status %d: %s", rpcResp.StatusCode, string(body))
	}

	rpcRespBytes, err := readProtoResponse(rpcResp)
	if err != nil {
		return nil, fmt.Errorf("failed to read AI response: %w", err)
	}

	// 5. Parse AI Response to extract URLs
	urls := parseMagicEditorURLs(rpcRespBytes)
	if len(urls) == 0 {
		return nil, fmt.Errorf("no AI-enhanced variations returned by server")
	}

	// 6. Download full-res variations & prepare base64 data for instantaneous web rendering
	results := make([]models.AIEnhanceResult, 0, len(urls))
	for idx, rawURL := range urls {
		fullResURL := rawURL
		if strings.Contains(fullResURL, "=cp2") {
			fullResURL = strings.Replace(fullResURL, "=cp2", "=s0-k-no-rj-gd-ft", 1)
		} else if !strings.Contains(fullResURL, "=s0") {
			fullResURL = fullResURL + "=s0-k-no-rj-gd-ft"
		}
		fullResURL = strings.Replace(fullResURL, "lh3.googleusercontent.com", "ap2.googleusercontent.com", 1)

		dlReq, err := http.NewRequest("GET", fullResURL, nil)
		if err != nil {
			continue
		}
		dlReq.Header.Set("Authorization", "Bearer "+token)
		dlReq.Header.Set("User-Agent", a.userAgent)

		dlResp, err := client.Do(dlReq)
		if err != nil {
			continue
		}

		imgData, err := io.ReadAll(dlResp.Body)
		dlResp.Body.Close()
		if err != nil || len(imgData) == 0 {
			continue
		}

		cfg, _, _ := image.DecodeConfig(bytes.NewReader(imgData))

		b64Data := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(imgData)
		results = append(results, models.AIEnhanceResult{
			VariationID: idx + 1,
			URL:         fullResURL,
			Width:       cfg.Width,
			Height:      cfg.Height,
			DataURL:     b64Data,
			Size:        len(imgData),
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("failed to download enhanced image variations")
	}

	return results, nil
}

// ensureJPEG verifies if the image is JPEG, or decodes and re-encodes to high quality JPEG.
func ensureJPEG(imgBytes []byte) ([]byte, error) {
	// Check JPEG magic bytes FF D8 FF
	if len(imgBytes) >= 3 && imgBytes[0] == 0xFF && imgBytes[1] == 0xD8 && imgBytes[2] == 0xFF {
		return imgBytes, nil
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("unsupported image format: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return buf.Bytes(), nil
}

// parseMagicEditorURLs parses the repeated Field 1 -> Field 1 (string URL) from protobuf response
func parseMagicEditorURLs(buf []byte) []string {
	var urls []string
	b := buf
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:]

		if typ == protowire.BytesType && num == 1 {
			val, n := protowire.ConsumeBytes(b)
			if n < 0 {
				break
			}
			b = b[n:]

			// Inside Field 1: search for Field 1 string
			subB := val
			for len(subB) > 0 {
				sNum, sTyp, sn := protowire.ConsumeTag(subB)
				if sn < 0 {
					break
				}
				subB = subB[sn:]

				if sTyp == protowire.BytesType && sNum == 1 {
					sVal, sn := protowire.ConsumeBytes(subB)
					if sn < 0 {
						break
					}
					subB = subB[sn:]
					strURL := string(sVal)
					if strings.HasPrefix(strURL, "http") {
						urls = append(urls, strURL)
					}
				} else {
					m := protowire.ConsumeFieldValue(sNum, sTyp, subB)
					if m < 0 {
						break
					}
					subB = subB[m:]
				}
			}
		} else {
			m := protowire.ConsumeFieldValue(num, typ, b)
			if m < 0 {
				break
			}
			b = b[m:]
		}
	}
	return urls
}
