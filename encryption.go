package dotenv

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// EncryptionService handles encryption key operations
type EncryptionService struct {
	client *Client
}

// GetEncryptionKey retrieves the encryption key descriptor for a project.
//
// Client-managed projects are NOT an error: the descriptor's Managed field is
// "client" and it carries the PBKDF2 proof parameters (KeyCheckSalt,
// KeyCheckIterations) instead of a key. Callers decide whether they need to
// resolve a client key based on Managed/IsClientManaged.
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
			// The key descriptor is a JSON string inside the content field.
			if content, ok := attrs["content"].(string); ok {
				var contentData map[string]interface{}
				if err := json.Unmarshal([]byte(content), &contentData); err != nil {
					return nil, resp, fmt.Errorf("failed to parse encryption key content: %w", err)
				}

				if keyData, ok := contentData["key"].(map[string]interface{}); ok {
					if managed, ok := keyData["managed"].(string); ok {
						key.Managed = managed
						key.IsClientManaged = managed == "client"
					}
					// Server-managed only.
					if keyStr, ok := keyData["key"].(string); ok {
						key.Key = keyStr
					}
					// Client-managed only: proof params (never the key).
					if salt, ok := keyData["key_check_salt"].(string); ok {
						key.KeyCheckSalt = salt
					}
					if iters, ok := keyData["key_check_iterations"].(float64); ok {
						key.KeyCheckIterations = int(iters)
					}
					if version, ok := keyData["version"].(float64); ok {
						key.Version = int(version)
					}
					if createdAt, ok := keyData["created_at"].(string); ok {
						if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
							key.CreatedAt = t
						}
					}
				}
			}
		}

		// Set ID from data if available
		if id, ok := data["id"].(string); ok {
			key.ID = id
		}
	}

	return key, resp, nil
}

// RotateClientKeys rotates a client-managed key: the client re-encrypts the
// current secrets under a new key and submits them with the new key's proof. The
// server rolls over to a new key row, preserving the old one so historical
// versions stay decryptable. Set req.HistoryPolicy to "re_encrypt" to also move
// historical versions onto the new key (drive the move via ListPendingReencrypt /
// SubmitReencryptedHistory).
func (s *EncryptionService) RotateClientKeys(ctx context.Context, project string, req ClientKeyRotationRequest) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "encryption_key", project)
	u := fmt.Sprintf("/api/v1/%s/%s/secrets/rotate-client-keys", s.client.organization, project)

	httpReq, err := s.client.NewRequest(ctx, "POST", u, req)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, httpReq, nil)
}

// RotateEncryptionKey rotates a server-managed key. Set opts.HistoryPolicy to
// "re_encrypt" to re-encrypt historical versions onto the new key server-side.
func (s *EncryptionService) RotateEncryptionKey(ctx context.Context, project string, opts *RotateOptions) (*EncryptionKey, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "encryption_key", project)
	u := fmt.Sprintf("/api/v1/%s/%s/encryption-key/rotate", s.client.organization, project)

	var body interface{}
	if opts != nil {
		body = opts
	}

	httpReq, err := s.client.NewRequest(ctx, "POST", u, body)
	if err != nil {
		return nil, nil, err
	}

	var env struct {
		Data struct {
			Key struct {
				ID         string `json:"id"`
				Attributes struct {
					EncryptionKey
					// The resource's attributes diverge from the SDK struct's
					// types: id is NUMERIC (row id) and version is a STRING.
					// Shadow both so embedding EncryptionKey doesn't fail to
					// unmarshal, then convert below.
					ID      json.RawMessage `json:"id"`
					Version string          `json:"version"`
				} `json:"attributes"`
			} `json:"key"`
		} `json:"data"`
	}
	resp, err := s.client.Do(ctx, httpReq, &env)
	if err != nil {
		return nil, resp, err
	}

	key := env.Data.Key.Attributes.EncryptionKey
	key.ID = env.Data.Key.ID
	if v, convErr := strconv.Atoi(env.Data.Key.Attributes.Version); convErr == nil {
		key.Version = v
	}

	return &key, resp, nil
}

// GetKeyHistory lists the project's encryption key versions, newest first. For
// client-managed keys each entry carries the proof params so a caller can validate
// a user-supplied old key with VerifyKeyProof before decrypting an old version.
func (s *EncryptionService) GetKeyHistory(ctx context.Context, project string) ([]*EncryptionKeyVersion, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "encryption_key", project)
	u := fmt.Sprintf("/api/v1/%s/%s/encryption-key/history", s.client.organization, project)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var env struct {
		Data []*EncryptionKeyVersion `json:"data"`
	}
	resp, err := s.client.Do(ctx, req, &env)
	if err != nil {
		return nil, resp, err
	}

	return env.Data, resp, nil
}

// ListPendingReencrypt returns up to limit client-managed historical versions
// awaiting re-encryption onto the current key, plus the total remaining.
func (s *EncryptionService) ListPendingReencrypt(ctx context.Context, project string, limit int) ([]PendingVersion, int, *http.Response, error) {
	if s.client.organization == "" {
		return nil, 0, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "encryption_key", project)
	u := fmt.Sprintf("/api/v1/%s/%s/secrets/re-encrypt-history/pending", s.client.organization, project)

	body := map[string]interface{}{}
	if limit > 0 {
		body["limit"] = limit
	}

	req, err := s.client.NewRequest(ctx, "POST", u, body)
	if err != nil {
		return nil, 0, nil, err
	}

	var env struct {
		Data []PendingVersion `json:"data"`
		Meta struct {
			Remaining int `json:"remaining"`
		} `json:"meta"`
	}
	resp, err := s.client.Do(ctx, req, &env)
	if err != nil {
		return nil, 0, resp, err
	}

	return env.Data, env.Meta.Remaining, resp, nil
}

// SubmitReencryptedHistory submits versions re-encrypted under the current key,
// proven by keyProof, returning how many were updated and how many remain.
func (s *EncryptionService) SubmitReencryptedHistory(ctx context.Context, project string, versions []ReencryptedVersion, keyProof string) (updated, remaining int, resp *http.Response, err error) {
	if s.client.organization == "" {
		return 0, 0, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "encryption_key", project)
	u := fmt.Sprintf("/api/v1/%s/%s/secrets/re-encrypt-history", s.client.organization, project)

	body := map[string]interface{}{
		"versions":  versions,
		"key_proof": keyProof,
	}

	req, err := s.client.NewRequest(ctx, "POST", u, body)
	if err != nil {
		return 0, 0, nil, err
	}

	var env struct {
		Data struct {
			Updated   int `json:"updated"`
			Remaining int `json:"remaining"`
		} `json:"data"`
	}
	resp, err = s.client.Do(ctx, req, &env)
	if err != nil {
		return 0, 0, resp, err
	}

	return env.Data.Updated, env.Data.Remaining, resp, nil
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

// ErrEmptyKey is returned when an empty project key reaches the crypto layer.
// An empty key would otherwise pad to a publicly-known constant ('0' x32),
// encrypting secrets under a key any attacker can reproduce.
var ErrEmptyKey = errors.New("dotenv: project key must not be empty")

// EncryptWithProjectKey encrypts plaintext with a project key string using
// AES-256-GCM. Mirror of EncryptionService::encryptWithProjectKey.
func EncryptWithProjectKey(plaintext, key string) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}
	return Encrypt(plaintext, []byte(key))
}

// DecryptWithProjectKey decrypts ciphertext produced with a project key string.
// Mirror of EncryptionService::decryptWithProjectKey.
func DecryptWithProjectKey(ciphertext, key string) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}
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

// --- Key proof: client-managed key verification ---
//
// Second cross-language crypto contract (alongside padKey). For client-managed
// projects the server never holds the key, so it cannot validate a pushed key
// directly. Instead the client derives a one-way PROOF; the server stores it and
// compares on every write (like a password verifier), rejecting mismatches so a
// mistyped/wrong key can never silently corrupt the project's secrets.
//
// The proof is computed over the EFFECTIVE 32-byte AES key (padKey(key)), NOT the
// raw string — so it matches exactly when the real encryption key matches. PHP
// (hash_pbkdf2) and JS (WebCrypto deriveBits) MUST produce byte-identical output:
//
//	derived = padKey(key)                                   // 32 bytes
//	proof   = PBKDF2-HMAC-SHA256(derived, salt, iterations, dkLen=32)
//	wire    = base64(salt), base64(proof), iterations

// KeyProofIterations is the platform-fixed PBKDF2 iteration count for key proofs.
const KeyProofIterations = 600000

const (
	keyProofSaltLen = 16
	keyProofDKLen   = 32
)

// DeriveKeyProof computes the base64 key proof for a key string given a
// base64-encoded salt and an iteration count. A zero/negative iteration count
// falls back to KeyProofIterations. Used when VERIFYING a key before a push.
func DeriveKeyProof(key, saltB64 string, iterations int) (string, error) {
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return "", fmt.Errorf("invalid key proof salt: %w", err)
	}
	if iterations <= 0 {
		iterations = KeyProofIterations
	}
	proof, err := pbkdf2.Key(sha256.New, string(padKey([]byte(key))), salt, iterations, keyProofDKLen)
	if err != nil {
		return "", fmt.Errorf("derive key proof: %w", err)
	}
	return base64.StdEncoding.EncodeToString(proof), nil
}

// dataKeyDKLen is the AES-256 key length (bytes) derived for client-managed
// projects.
const dataKeyDKLen = 32

// DeriveDataKey derives the 32-byte AES-256 data key for a client-managed
// project from a passphrase, using PBKDF2-HMAC-SHA256 over the project's salt
// and iteration count.
//
// This replaces the legacy padKey derivation (raw passphrase '0'-padded to 32
// bytes, with no salt and no stretching) for client-managed encryption. padKey
// made a weak/short passphrase trivially brute-forceable offline against stolen
// ciphertext and reused the same AES key across projects with the same
// passphrase. PBKDF2 salts (per project) and stretches (iterations) the
// passphrase, raising offline brute force to `iterations` rounds per guess.
//
// Cross-language contract — MUST match the browser's EncryptionUtils.deriveDataKey
// byte-for-byte:
//
//	salt   = base64decode(saltB64)
//	aesKey = PBKDF2-HMAC-SHA256(passphrase, salt, iterations, dkLen=32)
//
// The passphrase is used as raw UTF-8 bytes — NOT padKey'd; that padding is the
// weakness being removed. A zero/negative iteration count falls back to
// KeyProofIterations. The salt/iterations are the project's existing
// key_check_salt / key_check_iterations, so no new key material is needed.
func DeriveDataKey(passphrase, saltB64 string, iterations int) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, fmt.Errorf("invalid data key salt: %w", err)
	}
	if iterations <= 0 {
		iterations = KeyProofIterations
	}
	key, err := pbkdf2.Key(sha256.New, passphrase, salt, iterations, dataKeyDKLen)
	if err != nil {
		return nil, fmt.Errorf("derive data key: %w", err)
	}
	return key, nil
}

// GenerateKeyProof creates a fresh random salt and the matching proof for a key
// string, returning base64 salt, base64 proof and the iteration count. Used when
// ESTABLISHING a client-managed key (project create / key rotation).
func GenerateKeyProof(key string) (saltB64, proofB64 string, iterations int, err error) {
	salt := make([]byte, keyProofSaltLen)
	if _, err = io.ReadFull(rand.Reader, salt); err != nil {
		return "", "", 0, err
	}
	proof, err := pbkdf2.Key(sha256.New, string(padKey([]byte(key))), salt, KeyProofIterations, keyProofDKLen)
	if err != nil {
		return "", "", 0, err
	}
	return base64.StdEncoding.EncodeToString(salt), base64.StdEncoding.EncodeToString(proof), KeyProofIterations, nil
}

// VerifyKeyProof reports whether key matches the stored proof for the given salt
// and iteration count, using a constant-time comparison. Used to validate a
// user-supplied old key locally (against a key-history descriptor) before
// attempting to decrypt an old version — a wrong key is rejected up front.
func VerifyKeyProof(key, saltB64 string, iterations int, expectedProofB64 string) (bool, error) {
	got, err := DeriveKeyProof(key, saltB64, iterations)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expectedProofB64)) == 1, nil
}
