package dotenv

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
)

// EncryptionService handles encryption key operations
type EncryptionService struct {
	client *Client
}

// GetEncryptionKey retrieves the encryption key for a project
func (s *EncryptionService) GetEncryptionKey(ctx context.Context, project string) (*EncryptionKey, *http.Response, error) {
	u := fmt.Sprintf("/api/v1/%s/encryption-key", project)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	key := new(EncryptionKey)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, key)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				key.ID = id
			}
		}
	}

	return key, resp, nil
}

// RotateClientKeys initiates client-side key rotation
func (s *EncryptionService) RotateClientKeys(ctx context.Context, project string) (*EncryptionKey, *http.Response, error) {
	u := fmt.Sprintf("/api/v1/%s/secrets/rotate-client-keys", project)

	req, err := s.client.NewRequest(ctx, "POST", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	key := new(EncryptionKey)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, key)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				key.ID = id
			}
		}
	}

	return key, resp, nil
}

// Encrypt encrypts data using AES-256-GCM
func Encrypt(plaintext string, key []byte) (string, error) {
	// Ensure key is 32 bytes for AES-256
	if len(key) != 32 {
		return "", fmt.Errorf("encryption key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a nonce. GCM standard recommends 12 bytes
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Combine nonce + ciphertext for storage
	// Format: base64(nonce + ciphertext + tag)
	// Note: GCM's Seal already appends the tag to ciphertext
	combined := append(nonce, ciphertext...)

	return base64.StdEncoding.EncodeToString(combined), nil
}

// Decrypt decrypts data encrypted with AES-256-GCM
func Decrypt(ciphertext string, key []byte) (string, error) {
	// Ensure key is 32 bytes for AES-256
	if len(key) != 32 {
		return "", fmt.Errorf("decryption key must be 32 bytes for AES-256")
	}

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	// Minimum length check (12 byte nonce + at least 16 byte tag)
	if len(data) < 28 {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce := data[:12]
	ciphertextWithTag := data[12:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertextWithTag, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// GenerateKey generates a new 32-byte encryption key
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncodeKey encodes a key to base64
func EncodeKey(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// DecodeKey decodes a base64 encoded key
func DecodeKey(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
