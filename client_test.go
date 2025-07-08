package dotenv_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dotenv "github.com/dotenv/sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		opts    []dotenv.ClientOption
		wantErr bool
	}{
		{
			name:   "valid client",
			apiKey: "test-api-key",
		},
		{
			name:   "with custom base URL",
			apiKey: "test-api-key",
			opts: []dotenv.ClientOption{
				dotenv.WithBaseURL("https://custom.dotenv.com"),
			},
		},
		{
			name:   "with custom user agent",
			apiKey: "test-api-key",
			opts: []dotenv.ClientOption{
				dotenv.WithUserAgent("custom-agent/1.0"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := append([]dotenv.ClientOption{dotenv.WithAPIKey(tt.apiKey)}, tt.opts...)
			client := dotenv.NewClient(opts...)
			assert.NotNil(t, client)
		})
	}
}

func TestClient_Organizations_List(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/organizations", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := dotenv.JSONAPIResponse{
			Data: []interface{}{
				map[string]interface{}{
					"type": "organizations",
					"id":   "1",
					"attributes": map[string]interface{}{
						"ulid":       "01ARZ3NDEKTSV4RRFFQ69G5FAV",
						"name":       "Test Org",
						"slug":       "test-org",
						"status":     "active",
						"plan_name":  "pro",
						"created_at": time.Now().Format(time.RFC3339),
						"updated_at": time.Now().Format(time.RFC3339),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := dotenv.NewClient(dotenv.WithAPIKey("test-key"), dotenv.WithBaseURL(server.URL))

	orgs, resp, err := client.Organizations.List(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, orgs, 1)
	assert.Equal(t, "Test Org", orgs[0].Name)
	assert.Equal(t, "test-org", orgs[0].Slug)
	assert.Equal(t, "1", orgs[0].ID)
}

func TestClient_Projects_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/test-org/projects", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := dotenv.JSONAPIResponse{
			Data: []interface{}{
				map[string]interface{}{
					"type": "projects",
					"id":   "proj-1",
					"attributes": map[string]interface{}{
						"ulid":              "01ARZ3NDEKTSV4RRFFQ69G5FAV",
						"organization_id":   "org-1",
						"name":              "Test Project",
						"slug":              "test-project",
						"description":       "A test project",
						"has_secrets":       true,
						"secret_count":      5,
						"environment_count": 3,
						"target_count":      2,
						"created_at":        time.Now().Format(time.RFC3339),
						"updated_at":        time.Now().Format(time.RFC3339),
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
		dotenv.WithOrganization("test-org"),
	)

	projects, resp, err := client.Projects.List(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, projects, 1)
	assert.Equal(t, "Test Project", projects[0].Name)
	assert.Equal(t, "test-project", projects[0].Slug)
	assert.Equal(t, 5, projects[0].SecretCount)
}

func TestClient_Secrets_GetProjectSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/test-org/my-project/secrets", r.URL.Path)
		assert.Equal(t, "production", r.URL.Query().Get("target"))
		assert.Equal(t, "web", r.URL.Query().Get("environment"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := dotenv.JSONAPIResponse{
			Data: []interface{}{
				map[string]interface{}{
					"type": "secrets",
					"id":   "secret-1",
					"attributes": map[string]interface{}{
						"key":   "DATABASE_URL",
						"value": "encrypted-value-here",
					},
				},
				map[string]interface{}{
					"type": "secrets",
					"id":   "secret-2",
					"attributes": map[string]interface{}{
						"key":   "API_KEY",
						"value": "encrypted-api-key",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
		dotenv.WithOrganization("test-org"),
	)

	secrets, resp, err := client.Secrets.GetProjectSecrets(context.Background(), "my-project", "production", "web")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, secrets, 2)
	assert.Equal(t, "encrypted-value-here", secrets["DATABASE_URL"])
	assert.Equal(t, "encrypted-api-key", secrets["API_KEY"])
}

func TestClient_Encryption_GetKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/test-org/my-project/encryption-key", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := dotenv.JSONAPIResponse{
			Data: map[string]interface{}{
				"type": "encryption_keys",
				"id":   "key-1",
				"attributes": map[string]interface{}{
					"project_id":        "proj-1",
					"key":               "base64-encoded-key-here",
					"is_active":         true,
					"is_client_managed": false,
					"created_at":        time.Now().Format(time.RFC3339),
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
		dotenv.WithOrganization("test-org"),
	)

	key, resp, err := client.Encryption.GetEncryptionKey(context.Background(), "my-project")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "key-1", key.ID)
	assert.Equal(t, "base64-encoded-key-here", key.Key)
	assert.True(t, key.IsActive)
}

func TestClient_RetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := dotenv.JSONAPIResponse{
			Data: []interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := dotenv.NewClient(dotenv.WithAPIKey("test-key"), dotenv.WithBaseURL(server.URL))

	_, resp, err := client.Organizations.List(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, attempts)
}

func TestClient_RateLimiting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message": "Rate limit exceeded"}`))
	}))
	defer server.Close()

	client := dotenv.NewClient(dotenv.WithAPIKey("test-key"), dotenv.WithBaseURL(server.URL))

	_, _, err := client.Organizations.List(context.Background(), nil)
	require.Error(t, err)
	assert.True(t, dotenv.IsRateLimited(err))

	rateLimitErr, ok := err.(*dotenv.ErrRateLimited)
	assert.True(t, ok)
	assert.Equal(t, 60, rateLimitErr.RetryAfter)
}

func TestClient_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Resource not found"}`))
	}))
	defer server.Close()

	client := dotenv.NewClient(dotenv.WithAPIKey("test-key"), dotenv.WithBaseURL(server.URL))

	_, _, err := client.Projects.Get(context.Background(), "non-existent")
	require.Error(t, err)
	assert.True(t, dotenv.IsNotFound(err))
}

func TestClient_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "Invalid API key"}`))
	}))
	defer server.Close()

	client := dotenv.NewClient(dotenv.WithAPIKey("invalid-key"), dotenv.WithBaseURL(server.URL))

	_, _, err := client.Organizations.List(context.Background(), nil)
	require.Error(t, err)
	assert.True(t, dotenv.IsUnauthorized(err))
}

func TestEncryption(t *testing.T) {
	// Generate a test key
	key, err := dotenv.GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key, 32)

	// Test encryption and decryption
	plaintext := "This is a secret value"

	encrypted, err := dotenv.Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)

	// Decrypt
	decrypted, err := dotenv.Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	// Test with wrong key
	wrongKey, _ := dotenv.GenerateKey()
	_, err = dotenv.Decrypt(encrypted, wrongKey)
	assert.Error(t, err)

	// Test key encoding/decoding
	encoded := dotenv.EncodeKey(key)
	decoded, err := dotenv.DecodeKey(encoded)
	require.NoError(t, err)
	assert.Equal(t, key, decoded)
}
