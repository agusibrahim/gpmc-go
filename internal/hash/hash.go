package hash

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
)

// CalculateSHA1Hash calculates the SHA-1 hash of a file
// Returns both the raw hash bytes and the base64 encoded string
func CalculateSHA1Hash(filePath string) ([]byte, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	hashBytes := hash.Sum(nil)
	hashB64 := base64.StdEncoding.EncodeToString(hashBytes)

	return hashBytes, hashB64, nil
}

// CalculateSHA1HashFromReader calculates SHA-1 hash from an io.Reader
// Returns both the raw hash bytes and the base64 encoded string
func CalculateSHA1HashFromReader(r io.Reader) ([]byte, string, error) {
	hash := sha1.New()
	if _, err := io.Copy(hash, r); err != nil {
		return nil, "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	hashBytes := hash.Sum(nil)
	hashB64 := base64.StdEncoding.EncodeToString(hashBytes)

	return hashBytes, hashB64, nil
}

// ConvertSHA1Hash converts a SHA-1 hash from any format to bytes and base64
// Accepts: bytes, hex string (40 chars), or base64 string
func ConvertSHA1Hash(hash interface{}) ([]byte, string, error) {
	switch v := hash.(type) {
	case []byte:
		// Already bytes
		b64 := base64.StdEncoding.EncodeToString(v)
		return v, b64, nil
	case string:
		// Could be hex or base64
		if IsHashHexadecimal(v) {
			// Hex string
			bytes, err := hex.DecodeString(v)
			if err != nil {
				return nil, "", fmt.Errorf("failed to decode hex hash: %w", err)
			}
			b64 := base64.StdEncoding.EncodeToString(bytes)
			return bytes, b64, nil
		} else {
			// Base64 string
			bytes, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return nil, "", fmt.Errorf("failed to decode base64 hash: %w", err)
			}
			return bytes, v, nil
		}
	default:
		return nil, "", fmt.Errorf("invalid hash type: %T", hash)
	}
}

// IsHashHexadecimal checks if a string is a valid hexadecimal SHA-1 hash
// A valid SHA-1 hex hash is exactly 40 hexadecimal characters
func IsHashHexadecimal(s string) bool {
	if len(s) != 40 {
		return false
	}
	matched, _ := regexp.MatchString("^[0-9a-fA-F]{40}$", s)
	return matched
}

// BytesToHex converts bytes to hexadecimal string
func BytesToHex(bytes []byte) string {
	return hex.EncodeToString(bytes)
}

// BytesToBase64 converts bytes to base64 string
func BytesToBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

// HexToBytes converts hexadecimal string to bytes
func HexToBytes(hexStr string) ([]byte, error) {
	return hex.DecodeString(hexStr)
}

// Base64ToBytes converts base64 string to bytes
func Base64ToBytes(b64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(b64)
}
