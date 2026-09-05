package api

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/agusibrahim/gpmc-go/internal/models"
	"google.golang.org/protobuf/encoding/protowire"
)

//go:embed sync_template.bin
var syncQueryTemplate []byte

var (
	albumMediaMask = func() []byte {
		b, _ := hex.DecodeString(
			"129a070a2e12001a0022002a00320c0a0012001a0022002a003a003a004200520062006a" +
				"0412001a007a020a00920100a2010022020a004a005a0c0a0a0a0022002a0032004a0072" +
				"240a220a1e0a00120812060a020a001a001a1022060a020a001a002a060a020a001a0012" +
				"008a01009201060a0012020a00a2010612040a001200b20100ba0100c2010012170a0412" +
				"0208011204120210011a0022020a00320512030801101a100a0812060a020a001a001200" +
				"1a00220022002a0032003a0042004a0052005a0062006a0072007a008201008a01009201" +
				"009a0100a20100b20100ba0100ca0100d20100da0100ea01002a120a0212001a0022002a" +
				"041202100132003a0032120a001204120210011a041202100122003a02120042060a0012" +
				"020a004a060a0012020a0052020a005a0062006a020a0072020a007a060a0012020a0082" +
				"01008a01009201009a0100a20100ba01060a0012020a00c201060a0012020a00ca0100da" +
				"010412020801ea01021200fa01008202008a02009202009a0200aa0200b20200ba0200c2" +
				"0200ca0200d20200e20200ea0200f20200fa02008203009203009a0300a20300aa0300ba" +
				"0300c20300ca0300d20300da0300ea0300f20300fa03008204008a04009204009a0400a2" +
				"0400aa0400ba0400c20400ca0400da0400ea0400f20400fa04008205008a05009205009a" +
				"0500a20500aa0500ba0500c20500da0500ea0500f20500fa05008206008a06009206009a" +
				"0600aa0600b20600ba0600ca0600d20600e20600ea0600f20600fa06008207008a070092" +
				"07009a0700a20700aa0700ba0700c20700ca0700d20700da0700ea0700f20700fa070082" +
				"08008a08009208009a0800a20800ba0800c20800ca0800da0800ea0800f20800fa080082" +
				"09008a09009209009a0900a20900aa0900ba0900c20900da0900",
		)
		return b
	}()

	shareField3Template, _ = hex.DecodeString(
		"1a3c100128013a080a040802100110013a080a04080210021001" +
			"3a080a040801100110013a080a040801100210013a080a040803" +
			"10011001720408011002",
	)
	shareField5Template, _ = hex.DecodeString(
		"2a88010a2e12001a0022002a00320c0a0012001a0022002a003a" +
			"003a004200520062006a0412001a007a020a00920100a2010022" +
			"020a004a005a0c0a0a0a0022002a0032004a0072240a220a1e0a" +
			"00120812060a020a001a001a1022060a020a001a002a060a020a" +
			"001a0012008a01009201060a0012020a00a2010612040a001200" +
			"b20100ba0100c20100",
	)
	shareField10Template, _ = hex.DecodeString(
		"5288010a2e12001a0022002a00320c0a0012001a0022002a003a" +
			"003a004200520062006a0412001a007a020a00920100a2010022" +
			"020a004a005a0c0a0a0a0022002a0032004a0072240a220a1e0a" +
			"00120812060a020a001a001a1022060a020a001a002a060a020a" +
			"001a0012008a01009201060a0012020a00a2010612040a001200" +
			"b20100ba0100c20100",
	)
)

// helper to build standard protobuf mobile headers
func (a *Api) mobileHeaders() (map[string]string, error) {
	token, err := a.BearerToken()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"Accept-Encoding":          "gzip",
		"Accept-Language":          a.language,
		"Content-Type":             "application/x-protobuf",
		"User-Agent":               a.userAgent,
		"Authorization":            "Bearer " + token,
		"x-goog-ext-173412678-bin": "CgcIAhClARgC",
		"x-goog-ext-174067345-bin": "CgIIAg==",
	}, nil
}

// helper to read response body and automatically decompress gzip if needed
func readProtoResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
		gz, err := gzip.NewReader(bytes.NewReader(body))
		if err == nil {
			defer gz.Close()
			decompressed, err := io.ReadAll(gz)
			if err == nil {
				return decompressed, nil
			}
		}
	}
	return body, nil
}

// ListPhotosInAlbum fetches photos and videos inside an album (Endpoint /2035722585448626696)
func (a *Api) ListPhotosInAlbum(albumMediaKey string, shareToken string) ([]models.AlbumMediaItem, error) {
	// Field 1: { 1: { 1: albumMediaKey }, 5: 1, (6: shareToken) }
	var f1Sub []byte
	// 1: { 1: albumMediaKey }
	var f1Inner []byte
	f1Inner = protowire.AppendTag(f1Inner, 1, protowire.BytesType)
	f1Inner = protowire.AppendString(f1Inner, albumMediaKey)

	f1Sub = protowire.AppendTag(f1Sub, 1, protowire.BytesType)
	f1Sub = protowire.AppendBytes(f1Sub, f1Inner)

	// 5: 1 (varint)
	f1Sub = protowire.AppendTag(f1Sub, 5, protowire.VarintType)
	f1Sub = protowire.AppendVarint(f1Sub, 1)

	// Optional 6: shareToken
	if shareToken != "" {
		f1Sub = protowire.AppendTag(f1Sub, 6, protowire.BytesType)
		f1Sub = protowire.AppendString(f1Sub, shareToken)
	}

	var reqBody []byte
	reqBody = protowire.AppendTag(reqBody, 1, protowire.BytesType)
	reqBody = protowire.AppendBytes(reqBody, f1Sub)

	// Field 2: albumMediaMask (which starts with Tag 2 length-delimited)
	reqBody = append(reqBody, albumMediaMask...)

	headers, err := a.mobileHeaders()
	if err != nil {
		return nil, err
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointListPhotosInAlbum), reqBody, headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		errBody, _ := readProtoResponse(resp)
		return nil, fmt.Errorf("list photos in album failed with status %d: %s", resp.StatusCode, string(errBody))
	}

	body, err := readProtoResponse(resp)
	if err != nil {
		return nil, err
	}

	var items []models.AlbumMediaItem
	// Parse: Field 1 -> Field 4 (repeated item) -> Field 1 (media_key), Field 2 -> Field 4 (filename)
	b := body
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:]
		if num == 1 && typ == protowire.BytesType {
			v1, n2 := protowire.ConsumeBytes(b)
			if n2 >= 0 {
				b1 := v1
				for len(b1) > 0 {
					num2, typ2, n3 := protowire.ConsumeTag(b1)
					if n3 < 0 {
						break
					}
					b1 = b1[n3:]
					if num2 == 4 && typ2 == protowire.BytesType {
						v4, n4 := protowire.ConsumeBytes(b1)
						if n4 >= 0 {
							// In item: Tag 1 = media_key, Tag 2 = metadata
							item := parseMediaItemProto(v4)
							if item.MediaKey != "" {
								items = append(items, item)
							}
						}
					}
					nskip := protowire.ConsumeFieldValue(num2, typ2, b1)
					if nskip < 0 {
						break
					}
					b1 = b1[nskip:]
				}
			}
		}
		nskip := protowire.ConsumeFieldValue(num, typ, b)
		if nskip < 0 {
			break
		}
		b = b[nskip:]
	}

	return items, nil
}

func parseMediaItemProto(b []byte) models.AlbumMediaItem {
	var item models.AlbumMediaItem
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:]
		if num == 1 && typ == protowire.BytesType {
			v, n2 := protowire.ConsumeBytes(b)
			if n2 >= 0 {
				item.MediaKey = string(v)
			}
		} else if num == 2 && typ == protowire.BytesType {
			v, n2 := protowire.ConsumeBytes(b)
			if n2 >= 0 {
				// Parse metadata: Field 4 = filename
				b2 := v
				for len(b2) > 0 {
					numSub, typSub, nSub := protowire.ConsumeTag(b2)
					if nSub < 0 {
						break
					}
					b2 = b2[nSub:]
					if numSub == 4 && typSub == protowire.BytesType {
						vf, nf := protowire.ConsumeBytes(b2)
						if nf >= 0 {
							item.FileName = string(vf)
						}
					}
					nskip := protowire.ConsumeFieldValue(numSub, typSub, b2)
					if nskip < 0 {
						break
					}
					b2 = b2[nskip:]
				}
			}
		}
		nskip := protowire.ConsumeFieldValue(num, typ, b)
		if nskip < 0 {
			break
		}
		b = b[nskip:]
	}
	return item
}

// AddCommentToAlbum posts a comment to an album (Endpoint /15273438921077897261)
func (a *Api) AddCommentToAlbum(albumMediaKey, commentText, shareToken string) (string, error) {
	// Field 1: { 1: albumMediaKey }
	var f1Inner []byte
	f1Inner = protowire.AppendTag(f1Inner, 1, protowire.BytesType)
	f1Inner = protowire.AppendString(f1Inner, albumMediaKey)

	var reqBody []byte
	reqBody = protowire.AppendTag(reqBody, 1, protowire.BytesType)
	reqBody = protowire.AppendBytes(reqBody, f1Inner)

	// Field 2: { 1: { 1: 0, 2: commentText } }
	var cInner []byte
	cInner = protowire.AppendTag(cInner, 1, protowire.VarintType)
	cInner = protowire.AppendVarint(cInner, 0)
	cInner = protowire.AppendTag(cInner, 2, protowire.BytesType)
	cInner = protowire.AppendString(cInner, commentText)

	var f2Inner []byte
	f2Inner = protowire.AppendTag(f2Inner, 1, protowire.BytesType)
	f2Inner = protowire.AppendBytes(f2Inner, cInner)

	reqBody = protowire.AppendTag(reqBody, 2, protowire.BytesType)
	reqBody = protowire.AppendBytes(reqBody, f2Inner)

	// Field 3: shareToken (if present)
	if shareToken != "" {
		reqBody = protowire.AppendTag(reqBody, 3, protowire.BytesType)
		reqBody = protowire.AppendString(reqBody, shareToken)
	}

	// Field 6: raw bytes 32060a0012020a001a00
	tag6Bytes, _ := hex.DecodeString("32060a0012020a001a00")
	reqBody = append(reqBody, tag6Bytes...)

	// Field 7: timestamp_ms (varint)
	reqBody = protowire.AppendTag(reqBody, 7, protowire.VarintType)
	reqBody = protowire.AppendVarint(reqBody, uint64(time.Now().UnixNano()/int64(time.Millisecond)))

	headers, err := a.mobileHeaders()
	if err != nil {
		return "", err
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointCommentAlbum), reqBody, headers)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		errBody, _ := readProtoResponse(resp)
		return "", fmt.Errorf("add comment failed with status %d: %s", resp.StatusCode, string(errBody))
	}

	body, err := readProtoResponse(resp)
	if err != nil {
		return "", err
	}

	// Extract comment ID from Field 1
	b := body
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:]
		if num == 1 && typ == protowire.BytesType {
			v, n2 := protowire.ConsumeBytes(b)
			if n2 >= 0 {
				return string(v), nil
			}
		}
		nskip := protowire.ConsumeFieldValue(num, typ, b)
		if nskip < 0 {
			break
		}
		b = b[nskip:]
	}

	return "", fmt.Errorf("comment ID not found in response")
}

// AddMediaListToAlbum links media keys into an existing album (Endpoint /4733640162746355126)
func (a *Api) AddMediaListToAlbum(albumMediaKey string, mediaKeys []string, shareToken string) error {
	// Field 1: { 1: albumMediaKey }
	var f1Inner []byte
	f1Inner = protowire.AppendTag(f1Inner, 1, protowire.BytesType)
	f1Inner = protowire.AppendString(f1Inner, albumMediaKey)

	var reqBody []byte
	reqBody = protowire.AppendTag(reqBody, 1, protowire.BytesType)
	reqBody = protowire.AppendBytes(reqBody, f1Inner)

	// Field 2:
	//   1: 2
	//   3: repeated { 1: { 1: mediaKey } }
	//   7: 1
	var f2 []byte
	f2 = protowire.AppendTag(f2, 1, protowire.VarintType)
	f2 = protowire.AppendVarint(f2, 2)

	for _, k := range mediaKeys {
		var f3_1_1 []byte
		f3_1_1 = protowire.AppendTag(f3_1_1, 1, protowire.BytesType)
		f3_1_1 = protowire.AppendString(f3_1_1, k)

		var f3_1 []byte
		f3_1 = protowire.AppendTag(f3_1, 1, protowire.BytesType)
		f3_1 = protowire.AppendBytes(f3_1, f3_1_1)

		f2 = protowire.AppendTag(f2, 3, protowire.BytesType)
		f2 = protowire.AppendBytes(f2, f3_1)
	}

	f2 = protowire.AppendTag(f2, 7, protowire.VarintType)
	f2 = protowire.AppendVarint(f2, 1)

	reqBody = protowire.AppendTag(reqBody, 2, protowire.BytesType)
	reqBody = protowire.AppendBytes(reqBody, f2)

	if shareToken != "" {
		reqBody = protowire.AppendTag(reqBody, 3, protowire.BytesType)
		reqBody = protowire.AppendString(reqBody, shareToken)
	}

	headers, err := a.mobileHeaders()
	if err != nil {
		return err
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointAddToAlbumV2), reqBody, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add media to album failed with status %d", resp.StatusCode)
	}

	return nil
}

// RenameAlbum changes the title of an existing album (Endpoint /15456209598285517170)
func (a *Api) RenameAlbum(albumMediaKey, newTitle string) error {
	var reqBody []byte
	reqBody = protowire.AppendTag(reqBody, 1, protowire.BytesType)
	reqBody = protowire.AppendString(reqBody, albumMediaKey)

	reqBody = protowire.AppendTag(reqBody, 2, protowire.BytesType)
	reqBody = protowire.AppendString(reqBody, newTitle)

	reqBody = protowire.AppendTag(reqBody, 3, protowire.VarintType)
	reqBody = protowire.AppendVarint(reqBody, 1)

	headers, err := a.mobileHeaders()
	if err != nil {
		return err
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointRenameAlbum), reqBody, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rename album failed with status %d", resp.StatusCode)
	}

	return nil
}

// SetAlbumDescription sets narrative description of an album (Endpoint /4733640162746355126)
func (a *Api) SetAlbumDescription(albumMediaKey, description string) error {
	var reqBody []byte
	reqBody = protowire.AppendTag(reqBody, 2, protowire.BytesType)
	reqBody = protowire.AppendString(reqBody, description)

	f3Prefix := []byte("\n\x12Add description\xe2\x80\xa6\x10\x01\".\n,")
	f3Content := append(f3Prefix, []byte(albumMediaKey)...)
	reqBody = protowire.AppendTag(reqBody, 3, protowire.BytesType)
	reqBody = protowire.AppendBytes(reqBody, f3Content)

	var f4Inner []byte
	f4Inner = protowire.AppendTag(f4Inner, 1, protowire.BytesType)
	f4Inner = protowire.AppendString(f4Inner, albumMediaKey)
	reqBody = protowire.AppendTag(reqBody, 4, protowire.BytesType)
	reqBody = protowire.AppendBytes(reqBody, f4Inner)

	headers, err := a.mobileHeaders()
	if err != nil {
		return err
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointAddToAlbumV2), reqBody, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("set album description failed with status %d", resp.StatusCode)
	}

	return nil
}

// ShareAlbum generates a public share link for an album (Endpoint /11822839958055627768)
func (a *Api) ShareAlbum(albumMediaKey string) (*models.ShareAlbumResult, error) {
	var reqBody []byte
	// Field 3
	reqBody = append(reqBody, shareField3Template...)

	// Field 4: { 1: 1, 2: { 1: { 1: albumMediaKey } }, 7: 1, 9: 2 }
	var f4 []byte
	f4 = protowire.AppendTag(f4, 1, protowire.VarintType)
	f4 = protowire.AppendVarint(f4, 1)

	var f1Inner []byte
	f1Inner = protowire.AppendTag(f1Inner, 1, protowire.BytesType)
	f1Inner = protowire.AppendString(f1Inner, albumMediaKey)

	var f1Outer []byte
	f1Outer = protowire.AppendTag(f1Outer, 1, protowire.BytesType)
	f1Outer = protowire.AppendBytes(f1Outer, f1Inner)

	var f2 []byte
	f2 = protowire.AppendTag(f2, 2, protowire.BytesType)
	f2 = protowire.AppendBytes(f2, f1Outer)

	f4 = append(f4, f2...)

	f4 = protowire.AppendTag(f4, 7, protowire.VarintType)
	f4 = protowire.AppendVarint(f4, 1)

	f4 = protowire.AppendTag(f4, 9, protowire.VarintType)
	f4 = protowire.AppendVarint(f4, 2)

	reqBody = protowire.AppendTag(reqBody, 4, protowire.BytesType)
	reqBody = protowire.AppendBytes(reqBody, f4)

	// Field 5
	reqBody = append(reqBody, shareField5Template...)

	// Field 8: timestamp_ms
	reqBody = protowire.AppendTag(reqBody, 8, protowire.VarintType)
	reqBody = protowire.AppendVarint(reqBody, uint64(time.Now().UnixNano()/int64(time.Millisecond)))

	// Field 9: flags [3, 1, 2, 5, 6]
	for _, flag := range []uint64{3, 1, 2, 5, 6} {
		reqBody = protowire.AppendTag(reqBody, 9, protowire.VarintType)
		reqBody = protowire.AppendVarint(reqBody, flag)
	}

	// Field 10
	reqBody = append(reqBody, shareField10Template...)

	headers, err := a.mobileHeaders()
	if err != nil {
		return nil, err
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointShareAlbum), reqBody, headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		errBody, _ := readProtoResponse(resp)
		return nil, fmt.Errorf("share album failed with status %d: %s", resp.StatusCode, string(errBody))
	}

	body, err := readProtoResponse(resp)
	if err != nil {
		return nil, err
	}

	res := &models.ShareAlbumResult{}
	b := body
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:]
		if typ == protowire.BytesType {
			v, n2 := protowire.ConsumeBytes(b)
			if n2 >= 0 {
				switch num {
				case 1:
					res.SharedAlbumKey = string(v)
				case 2:
					res.ShareURL = string(v)
				case 5:
					res.ShareToken = string(v)
				}
			}
		}
		nskip := protowire.ConsumeFieldValue(num, typ, b)
		if nskip < 0 {
			break
		}
		b = b[nskip:]
	}

	return res, nil
}

// DeleteAlbum deletes either a regular or shared album
func (a *Api) DeleteAlbum(albumMediaKey string, isShared bool) error {
	headers, err := a.mobileHeaders()
	if err != nil {
		return err
	}

	if isShared || len(albumMediaKey) > 55 {
		// Shared album: Endpoint /12338166548769389427, fields: 1: key, 2: 0, 3: 3
		var reqBody []byte
		reqBody = protowire.AppendTag(reqBody, 1, protowire.BytesType)
		reqBody = protowire.AppendString(reqBody, albumMediaKey)
		reqBody = protowire.AppendTag(reqBody, 2, protowire.VarintType)
		reqBody = protowire.AppendVarint(reqBody, 0)
		reqBody = protowire.AppendTag(reqBody, 3, protowire.VarintType)
		reqBody = protowire.AppendVarint(reqBody, 3)

		resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointDeleteSharedAlbum), reqBody, headers)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("delete shared album failed with status %d", resp.StatusCode)
		}
		return nil
	}

	// Regular album: Endpoint /16147285624795325881, field: 1: key
	var reqBody []byte
	reqBody = protowire.AppendTag(reqBody, 1, protowire.BytesType)
	reqBody = protowire.AppendString(reqBody, albumMediaKey)

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointDeleteRegularAlbum), reqBody, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If 404, fallback to delete shared album
		if resp.StatusCode == http.StatusNotFound {
			return a.DeleteAlbum(albumMediaKey, true)
		}
		return fmt.Errorf("delete album failed with status %d", resp.StatusCode)
	}

	return nil
}

// ListAlbums fetches all albums for the user (Endpoint /18047484249733410717)
func (a *Api) ListAlbums() ([]models.Album, error) {
	headers, err := a.mobileHeaders()
	if err != nil {
		return nil, err
	}

	resp, err := a.makeRequest("POST", GetPhotosDataURL(EndpointLibraryState), syncQueryTemplate, headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		errBody, _ := readProtoResponse(resp)
		return nil, fmt.Errorf("list albums failed with status %d: %s", resp.StatusCode, string(errBody))
	}

	body, err := readProtoResponse(resp)
	if err != nil {
		return nil, err
	}

	var albums []models.Album
	// Parse: Field 1 -> Field 3 (collections/albums)
	b := body
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:]
		if num == 1 && typ == protowire.BytesType {
			v1, n2 := protowire.ConsumeBytes(b)
			if n2 >= 0 {
				b1 := v1
				for len(b1) > 0 {
					num2, typ2, n3 := protowire.ConsumeTag(b1)
					if n3 < 0 {
						break
					}
					b1 = b1[n3:]
					if num2 == 3 && typ2 == protowire.BytesType {
						v3, n4 := protowire.ConsumeBytes(b1)
						if n4 >= 0 {
							album := parseAlbumEntry(v3)
							if album.AlbumKey != "" {
								albums = append(albums, album)
							}
						}
					}
					nskip := protowire.ConsumeFieldValue(num2, typ2, b1)
					if nskip < 0 {
						break
					}
					b1 = b1[nskip:]
				}
			}
		}
		nskip := protowire.ConsumeFieldValue(num, typ, b)
		if nskip < 0 {
			break
		}
		b = b[nskip:]
	}

	return albums, nil
}

func parseAlbumEntry(b []byte) models.Album {
	var album models.Album
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:]
		if num == 1 && typ == protowire.BytesType {
			v, n2 := protowire.ConsumeBytes(b)
			if n2 >= 0 {
				album.AlbumKey = string(v)
			}
		} else if num == 2 && typ == protowire.BytesType {
			v, n2 := protowire.ConsumeBytes(b)
			if n2 >= 0 {
				b2 := v
				for len(b2) > 0 {
					num2, typ2, n3 := protowire.ConsumeTag(b2)
					if n3 < 0 {
						break
					}
					b2 = b2[n3:]
					if num2 == 5 && typ2 == protowire.BytesType {
						vt, nt := protowire.ConsumeBytes(b2)
						if nt >= 0 {
							album.Title = string(vt)
						}
					} else if num2 == 8 && typ2 == protowire.VarintType {
						vc, nc := protowire.ConsumeVarint(b2)
						if nc >= 0 {
							album.ItemCount = int64(vc)
						}
					} else if num2 == 15 && typ2 == protowire.BytesType {
						vu, nu := protowire.ConsumeBytes(b2)
						if nu >= 0 {
							album.CoverURL = string(vu)
						}
					}
					nskip := protowire.ConsumeFieldValue(num2, typ2, b2)
					if nskip < 0 {
						break
					}
					b2 = b2[nskip:]
				}
			}
		}
		nskip := protowire.ConsumeFieldValue(num, typ, b)
		if nskip < 0 {
			break
		}
		b = b[nskip:]
	}
	return album
}
