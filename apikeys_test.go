package dotenv_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dotenv "github.com/dotenvcloud/sdk-go"
)

func TestAPIKeysService_List(t *testing.T) {
	tests := []struct {
		name           string
		organization   string
		mockKeys       []dotenv.APIKey
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name:         "successful list",
			organization: "test-org",
			mockKeys: []dotenv.APIKey{
				{
					ID:          "key-123",
					Name:        "Production Key",
					TokenPrefix: "dotenv_api_",
					Abilities:   []string{"secrets:read", "projects:read"},
					CreatedAt:   time.Now().UTC().Round(time.Second),
					LastUsedAt:  timePtr(time.Now().UTC().Round(time.Second)),
				},
				{
					ID:          "key-456",
					Name:        "CI/CD Key",
					TokenPrefix: "dotenv_api_",
					Abilities:   []string{"secrets:read"},
					CreatedAt:   time.Now().UTC().Round(time.Second),
					LastUsedAt:  nil,
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "empty list",
			organization:   "test-org",
			mockKeys:       []dotenv.APIKey{},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "unauthorized",
			organization:   "test-org",
			mockKeys:       nil,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsUnauthorized(err))
			},
		},
		{
			name:           "forbidden",
			organization:   "test-org",
			mockKeys:       nil,
			mockStatusCode: http.StatusForbidden,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsForbidden(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/"+tt.organization+"/api-keys", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)

				if tt.mockStatusCode == http.StatusOK {
					// Convert mock keys to JSON:API format
					resources := make([]dotenv.APITokenResource, 0, len(tt.mockKeys))
					for _, key := range tt.mockKeys {
						resource := dotenv.APITokenResource{
							Type: "api_tokens",
							ID:   key.ID,
						}
						resource.Attributes.Name = key.Name
						resource.Attributes.TokenPrefix = key.TokenPrefix
						resource.Attributes.Abilities = key.Abilities
						resource.Attributes.ExpiresAt = key.ExpiresAt
						resource.Attributes.LastUsedAt = key.LastUsedAt
						resource.Attributes.CreatedAt = key.CreatedAt
						resource.Attributes.UpdatedAt = key.UpdatedAt
						resources = append(resources, resource)
					}

					response := map[string]interface{}{
						"data": resources,
					}
					json.NewEncoder(w).Encode(response)
				} else if tt.mockStatusCode >= 400 {
					// Send error response
					errorResp := dotenv.JSONAPIResponse{
						Errors: []dotenv.JSONAPIError{
							{
								Status: fmt.Sprintf("%d", tt.mockStatusCode),
								Title:  http.StatusText(tt.mockStatusCode),
								Detail: "Error occurred",
							},
						},
					}
					json.NewEncoder(w).Encode(errorResp)
				}
			}))
			defer server.Close()

			client := dotenv.NewClient(
				dotenv.WithAPIKey("test-key"),
				dotenv.WithBaseURL(server.URL),
			)

			keys, resp, err := client.APIKeys.List(context.Background(), tt.organization)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, keys)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, keys)
				require.NotNil(t, resp)
				assert.Equal(t, len(tt.mockKeys), len(keys))

				for i, key := range keys {
					assert.Equal(t, tt.mockKeys[i].ID, key.ID)
					assert.Equal(t, tt.mockKeys[i].Name, key.Name)
					assert.Equal(t, tt.mockKeys[i].TokenPrefix, key.TokenPrefix)
					assert.Equal(t, tt.mockKeys[i].Abilities, key.Abilities)
					assert.Equal(t, tt.mockKeys[i].CreatedAt, key.CreatedAt)
					if tt.mockKeys[i].LastUsedAt != nil {
						require.NotNil(t, key.LastUsedAt)
						assert.Equal(t, *tt.mockKeys[i].LastUsedAt, *key.LastUsedAt)
					} else {
						assert.Nil(t, key.LastUsedAt)
					}
				}
			}
		})
	}
}

func TestAPIKeysService_Create(t *testing.T) {
	tests := []struct {
		name           string
		organization   string
		createReq      dotenv.APIKeyCreateRequest
		mockResponse   *dotenv.APIKeyCreateResponse
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name:         "successful create",
			organization: "test-org",
			createReq: dotenv.APIKeyCreateRequest{
				Name:      "CI/CD Key",
				Abilities: []string{"secrets:read", "projects:read"},
				ExpiresAt: timePtr(time.Now().Add(90 * 24 * time.Hour).UTC().Round(time.Second)),
			},
			mockResponse: &dotenv.APIKeyCreateResponse{
				ID:    "key-789",
				Token: "dotenv_api_test_token_123",
				APIKey: &dotenv.APIKey{
					ID:          "key-789",
					Name:        "CI/CD Key",
					TokenPrefix: "dotenv_api_",
					Abilities:   []string{"secrets:read", "projects:read"},
					CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			mockStatusCode: http.StatusCreated,
			wantErr:        false,
		},
		{
			name:         "create without expiration",
			organization: "test-org",
			createReq: dotenv.APIKeyCreateRequest{
				Name:      "Permanent Key",
				Abilities: []string{"secrets:read"},
			},
			mockResponse: &dotenv.APIKeyCreateResponse{
				ID:    "key-999",
				Token: "dotenv_api_permanent123",
				APIKey: &dotenv.APIKey{
					ID:          "key-999",
					Name:        "Permanent Key",
					TokenPrefix: "dotenv_api_",
					Abilities:   []string{"secrets:read"},
					CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			mockStatusCode: http.StatusCreated,
			wantErr:        false,
		},
		{
			name:         "invalid abilities",
			organization: "test-org",
			createReq: dotenv.APIKeyCreateRequest{
				Name:      "Bad Key",
				Abilities: []string{"invalid:ability"},
			},
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "400")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/"+tt.organization+"/api-keys", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse request body
				var reqBody dotenv.APIKeyCreateRequest
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.createReq.Name, reqBody.Name)
				assert.Equal(t, tt.createReq.Abilities, reqBody.Abilities)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)

				if tt.mockResponse != nil {
					// Create JSON:API response
					resource := dotenv.APITokenCreationResource{
						Type: "api_tokens",
						ID:   tt.mockResponse.ID,
					}
					resource.Attributes.Name = tt.mockResponse.APIKey.Name
					resource.Attributes.Token = tt.mockResponse.Token
					resource.Attributes.TokenPrefix = tt.mockResponse.APIKey.TokenPrefix
					resource.Attributes.Abilities = tt.mockResponse.APIKey.Abilities
					resource.Attributes.ExpiresAt = tt.mockResponse.APIKey.ExpiresAt
					resource.Attributes.CreatedAt = tt.mockResponse.APIKey.CreatedAt
					resource.Attributes.UpdatedAt = tt.mockResponse.APIKey.UpdatedAt

					response := map[string]interface{}{
						"data": resource,
					}
					json.NewEncoder(w).Encode(response)
				} else if tt.mockStatusCode >= 400 {
					errorResp := dotenv.JSONAPIResponse{
						Errors: []dotenv.JSONAPIError{
							{
								Status: fmt.Sprintf("%d", tt.mockStatusCode),
								Title:  http.StatusText(tt.mockStatusCode),
								Detail: "Error occurred",
							},
						},
					}
					json.NewEncoder(w).Encode(errorResp)
				}
			}))
			defer server.Close()

			client := dotenv.NewClient(
				dotenv.WithAPIKey("test-key"),
				dotenv.WithBaseURL(server.URL),
			)

			keyResp, resp, err := client.APIKeys.Create(context.Background(), tt.organization, tt.createReq)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, keyResp)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, keyResp)
				require.NotNil(t, resp)
				assert.Equal(t, tt.mockResponse.ID, keyResp.ID)
				assert.Equal(t, tt.mockResponse.APIKey.Name, keyResp.APIKey.Name)
				assert.Equal(t, tt.mockResponse.Token, keyResp.Token)
				assert.Equal(t, tt.mockResponse.APIKey.Abilities, keyResp.APIKey.Abilities)
			}
		})
	}
}

func TestAPIKeysService_Update(t *testing.T) {
	tests := []struct {
		name           string
		organization   string
		keyID          string
		updateReq      dotenv.APIKeyUpdateRequest
		mockResponse   *dotenv.APIKey
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name:         "successful update",
			organization: "test-org",
			keyID:        "key-123",
			updateReq: dotenv.APIKeyUpdateRequest{
				Name: "Updated Key Name",
			},
			mockResponse: &dotenv.APIKey{
				ID:          "key-123",
				Name:        "Updated Key Name",
				TokenPrefix: "dotenv_api_",
				Abilities:   []string{"secrets:read"},
				CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:         "not found",
			organization: "test-org",
			keyID:        "key-nonexistent",
			updateReq: dotenv.APIKeyUpdateRequest{
				Name: "Updated Key Name",
			},
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsNotFound(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/"+tt.organization+"/api-keys/"+tt.keyID, r.URL.Path)
				assert.Equal(t, "PATCH", r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)

				if tt.mockResponse != nil {
					// Create JSON:API response
					resource := dotenv.APITokenResource{
						Type: "api_tokens",
						ID:   tt.mockResponse.ID,
					}
					resource.Attributes.Name = tt.mockResponse.Name
					resource.Attributes.TokenPrefix = tt.mockResponse.TokenPrefix
					resource.Attributes.Abilities = tt.mockResponse.Abilities
					resource.Attributes.ExpiresAt = tt.mockResponse.ExpiresAt
					resource.Attributes.LastUsedAt = tt.mockResponse.LastUsedAt
					resource.Attributes.CreatedAt = tt.mockResponse.CreatedAt
					resource.Attributes.UpdatedAt = tt.mockResponse.UpdatedAt

					response := map[string]interface{}{
						"data": resource,
					}
					json.NewEncoder(w).Encode(response)
				} else if tt.mockStatusCode >= 400 {
					errorResp := dotenv.JSONAPIResponse{
						Errors: []dotenv.JSONAPIError{
							{
								Status: fmt.Sprintf("%d", tt.mockStatusCode),
								Title:  http.StatusText(tt.mockStatusCode),
								Detail: "Error occurred",
							},
						},
					}
					json.NewEncoder(w).Encode(errorResp)
				}
			}))
			defer server.Close()

			client := dotenv.NewClient(
				dotenv.WithAPIKey("test-key"),
				dotenv.WithBaseURL(server.URL),
			)

			key, resp, err := client.APIKeys.Update(context.Background(), tt.organization, tt.keyID, tt.updateReq)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, key)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, key)
				require.NotNil(t, resp)
				assert.Equal(t, tt.mockResponse.Name, key.Name)
			}
		})
	}
}

func TestAPIKeysService_Delete(t *testing.T) {
	tests := []struct {
		name           string
		organization   string
		keyID          string
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name:           "successful delete",
			organization:   "test-org",
			keyID:          "key-123",
			mockStatusCode: http.StatusNoContent,
			wantErr:        false,
		},
		{
			name:           "not found",
			organization:   "test-org",
			keyID:          "key-nonexistent",
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsNotFound(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/"+tt.organization+"/api-keys/"+tt.keyID, r.URL.Path)
				assert.Equal(t, "DELETE", r.Method)

				w.WriteHeader(tt.mockStatusCode)

				if tt.mockStatusCode >= 400 {
					errorResp := dotenv.JSONAPIResponse{
						Errors: []dotenv.JSONAPIError{
							{
								Status: fmt.Sprintf("%d", tt.mockStatusCode),
								Title:  http.StatusText(tt.mockStatusCode),
								Detail: "Error occurred",
							},
						},
					}
					json.NewEncoder(w).Encode(errorResp)
				}
			}))
			defer server.Close()

			client := dotenv.NewClient(
				dotenv.WithAPIKey("test-key"),
				dotenv.WithBaseURL(server.URL),
			)

			resp, err := client.APIKeys.Delete(context.Background(), tt.organization, tt.keyID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}

func TestAPIKeysService_Rotate(t *testing.T) {
	tests := []struct {
		name           string
		organization   string
		keyID          string
		mockResponse   *dotenv.APIKeyCreateResponse
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name:         "successful rotate",
			organization: "test-org",
			keyID:        "key-123",
			mockResponse: &dotenv.APIKeyCreateResponse{
				ID:    "key-123",
				Token: "dotenv_api_new_token_123",
				APIKey: &dotenv.APIKey{
					ID:          "key-123",
					Name:        "Rotated Key",
					TokenPrefix: "dotenv_api_",
					Abilities:   []string{"secrets:read"},
					CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "not found",
			organization:   "test-org",
			keyID:          "key-nonexistent",
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsNotFound(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/"+tt.organization+"/api-keys/"+tt.keyID+"/rotate", r.URL.Path)
				assert.Equal(t, "POST", r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)

				if tt.mockResponse != nil {
					// Create JSON:API response
					resource := dotenv.APITokenCreationResource{
						Type: "api_tokens",
						ID:   tt.mockResponse.ID,
					}
					resource.Attributes.Name = tt.mockResponse.APIKey.Name
					resource.Attributes.Token = tt.mockResponse.Token
					resource.Attributes.TokenPrefix = tt.mockResponse.APIKey.TokenPrefix
					resource.Attributes.Abilities = tt.mockResponse.APIKey.Abilities
					resource.Attributes.ExpiresAt = tt.mockResponse.APIKey.ExpiresAt
					resource.Attributes.CreatedAt = tt.mockResponse.APIKey.CreatedAt
					resource.Attributes.UpdatedAt = tt.mockResponse.APIKey.UpdatedAt

					response := map[string]interface{}{
						"data": resource,
					}
					json.NewEncoder(w).Encode(response)
				} else if tt.mockStatusCode >= 400 {
					errorResp := dotenv.JSONAPIResponse{
						Errors: []dotenv.JSONAPIError{
							{
								Status: fmt.Sprintf("%d", tt.mockStatusCode),
								Title:  http.StatusText(tt.mockStatusCode),
								Detail: "Error occurred",
							},
						},
					}
					json.NewEncoder(w).Encode(errorResp)
				}
			}))
			defer server.Close()

			client := dotenv.NewClient(
				dotenv.WithAPIKey("test-key"),
				dotenv.WithBaseURL(server.URL),
			)

			keyResp, resp, err := client.APIKeys.Rotate(context.Background(), tt.organization, tt.keyID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, keyResp)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, keyResp)
				require.NotNil(t, resp)
				assert.Equal(t, tt.mockResponse.Token, keyResp.Token)
			}
		})
	}
}

func TestAPIKeysService_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		<-r.Context().Done()
		return
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	keys, _, err := client.APIKeys.List(ctx, "test-org")
	assert.Error(t, err)
	assert.Nil(t, keys)
	assert.Contains(t, err.Error(), "context canceled")
}

func timePtr(t time.Time) *time.Time {
	return &t
}
