package dotenv

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EncryptionService handles encryption key operations
type EncryptionService struct {
	client *Client
}

// GetEncryptionKey retrieves the encryption key for a project
func (s *EncryptionService) GetEncryptionKey(ctx context.Context, project string) (*EncryptionKey, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "encryption_key", project)
	u := fmt.Sprintf("/api/v1/%s/%s/encryption-key", s.client.organization, project)

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
			// The key is now in a JSON string inside the content field
			if content, ok := attrs["content"].(string); ok {
				// Parse the JSON content
				var contentData map[string]interface{}
				if err := json.Unmarshal([]byte(content), &contentData); err != nil {
					return nil, resp, fmt.Errorf("failed to parse encryption key content: %w", err)
				}

				// Extract the key information
				if keyData, ok := contentData["key"].(map[string]interface{}); ok {
					if keyStr, ok := keyData["key"].(string); ok {
						key.Key = keyStr
					}
					if version, ok := keyData["version"].(float64); ok {
						// Store version if needed
						key.ID = fmt.Sprintf("v%d", int(version))
					}
					if createdAt, ok := keyData["created_at"].(string); ok {
						// Parse created_at if needed
						if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
							key.CreatedAt = t
						}
					}
				}
			}

			// Other attributes are available in attrs if needed:
			// - format: attrs["format"].(string)
			// - encrypted: attrs["encrypted"].(bool)
		}

		// Set ID from data if available
		if id, ok := data["id"].(string); ok {
			key.ID = id
		}
	}

	return key, resp, nil
}

// RotateClientKeys initiates client-side key rotation
func (s *EncryptionService) RotateClientKeys(ctx context.Context, project string) (*EncryptionKey, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "encryption_key", project)
	u := fmt.Sprintf("/api/v1/%s/%s/secrets/rotate-client-keys", s.client.organization, project)

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

// Encrypt encrypts data using AES-256-GCM.
//
// Keys shorter than 32 bytes are silently padded with '0' bytes — this
// preserves parity with the web application but halves the effective
// entropy. Prefer EncryptWithStrictKey for new integrations.
func Encrypt(plaintext string, key []byte) (string, error) {
	// Apply key padding to ensure key is exactly 32 bytes
	paddedKey := padKey(key)

	block, err := aes.NewCipher(paddedKey)
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

// ErrKeyTooShort is returned by EncryptWithStrictKey when the supplied key is
// shorter than the 32 bytes required for AES-256.
var ErrKeyTooShort = errors.New("dotenv: encryption key must be at least 32 bytes")

// EncryptWithStrictKey encrypts data using AES-256-GCM and rejects keys
// shorter than 32 bytes. Unlike Encrypt this does not silently pad — short
// keys are a programmer error worth surfacing.
func EncryptWithStrictKey(plaintext string, key []byte) (string, error) {
	if len(key) < 32 {
		return "", ErrKeyTooShort
	}

	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	combined := append(nonce, ciphertext...)

	return base64.StdEncoding.EncodeToString(combined), nil
}

// Decrypt decrypts data encrypted with AES-256-GCM
func Decrypt(ciphertext string, key []byte) (string, error) {
	// Apply key padding to ensure key is exactly 32 bytes
	paddedKey := padKey(key)

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

	block, err := aes.NewCipher(paddedKey)
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

// --- Canonical project-key crypto (mirrors the web app's EncryptionService) ---
//
// The web app (apps/web, App\Services\EncryptionService) is the platform's
// source of truth for the crypto contract; this SDK mirrors it. A project key
// is a RAW STRING — it is NEVER hex/base64-decoded. It is used as ASCII bytes,
// padded with '0' (0x30) or truncated to 32 bytes for AES-256-GCM; wire format
// is base64(IV[12] + ciphertext + tag[16]). Decoding the key (e.g. hex) would
// yield different key bytes and fail GCM authentication against data written by
// the web or JS, so all consumers (the CLI included) must derive keys this way.

// DeriveProjectKey converts a project key string into the 32-byte AES-256 key,
// mirroring EncryptionService::padKey (str_pad($k, 32, '0') then substr(0, 32)).
func DeriveProjectKey(key string) []byte {
	return padKey([]byte(key))
}

// EncryptWithProjectKey encrypts plaintext with a project key string using
// AES-256-GCM. Mirror of EncryptionService::encryptWithProjectKey.
func EncryptWithProjectKey(plaintext, key string) (string, error) {
	return Encrypt(plaintext, []byte(key))
}

// DecryptWithProjectKey decrypts ciphertext produced with a project key string.
// Mirror of EncryptionService::decryptWithProjectKey.
func DecryptWithProjectKey(ciphertext, key string) (string, error) {
	return Decrypt(ciphertext, []byte(key))
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

// padKey pads or truncates a key to exactly 32 bytes for AES-256
// This matches the web application's key padding behavior
func padKey(key []byte) []byte {
	if len(key) >= 32 {
		// Key is 32 bytes or longer, truncate to 32 bytes
		return key[:32]
	}

	// Key is shorter than 32 bytes, pad with '0' bytes
	padded := make([]byte, 32)
	copy(padded, key)
	for i := len(key); i < 32; i++ {
		padded[i] = '0'
	}
	return padded
}
