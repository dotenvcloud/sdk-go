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

func TestUserService_GetAuthenticatedUser(t *testing.T) {
	tests := []struct {
		name              string
		organization      string
		mockUser          *dotenv.User
		mockOrganizations []dotenv.UserOrganization
		mockStatusCode    int
		mockAuth          bool
		wantErr           bool
		checkError        func(t *testing.T, err error)
	}{
		{
			name:         "successful user retrieval",
			organization: "test-org",
			mockUser: &dotenv.User{
				ID:         "user-123",
				Email:      "test@example.com",
				Name:       "Test User",
				IsVerified: true,
				CreatedAt:  time.Now().UTC().Round(time.Second),
				UpdatedAt:  time.Now().UTC().Round(time.Second),
			},
			mockOrganizations: []dotenv.UserOrganization{
				{
					ID:       "org-123",
					Name:     "Test Organization",
					Slug:     "test-org",
					Role:     "owner",
					JoinedAt: time.Now().UTC().Round(time.Second),
				},
				{
					ID:       "org-456",
					Name:     "Another Org",
					Slug:     "another-org",
					Role:     "member",
					JoinedAt: time.Now().UTC().Round(time.Second),
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "unauthorized user",
			organization:   "test-org",
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsUnauthorized(err))
			},
		},
		{
			name:         "user without organizations",
			organization: "test-org",
			mockUser: &dotenv.User{
				ID:         "user-456",
				Email:      "noorg@example.com",
				Name:       "No Org User",
				IsVerified: false,
				CreatedAt:  time.Now().UTC().Round(time.Second),
				UpdatedAt:  time.Now().UTC().Round(time.Second),
			},
			mockOrganizations: []dotenv.UserOrganization{},
			mockStatusCode:    http.StatusOK,
			wantErr:           false,
		},
		{
			name:           "forbidden access",
			organization:   "test-org",
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
				assert.Equal(t, "/api/v1/user", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				// Verify authentication header
				authHeader := r.Header.Get("Authorization")
				assert.NotEmpty(t, authHeader)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)

				if tt.mockUser != nil {
					// Create a proper JSON:API response structure with included organizations
					type includeItem struct {
						Type       string `json:"type"`
						ID         string `json:"id"`
						Attributes struct {
							ULID string `json:"ulid"`
							Name string `json:"name"`
							Slug string `json:"slug"`
							Role string `json:"role"`
						} `json:"attributes"`
					}

					included := make([]includeItem, 0, len(tt.mockOrganizations))
					orgRelationships := make([]map[string]interface{}, 0, len(tt.mockOrganizations))

					for _, org := range tt.mockOrganizations {
						included = append(included, includeItem{
							Type: "organizations",
							ID:   org.ID,
							Attributes: struct {
								ULID string `json:"ulid"`
								Name string `json:"name"`
								Slug string `json:"slug"`
								Role string `json:"role"`
							}{
								ULID: org.ULID,
								Name: org.Name,
								Slug: org.Slug,
								Role: org.Role,
							},
						})

						orgRelationships = append(orgRelationships, map[string]interface{}{
							"type": "organizations",
							"id":   org.ID,
						})
					}

					response := map[string]interface{}{
						"data": map[string]interface{}{
							"type": "users",
							"id":   tt.mockUser.ID,
							"attributes": map[string]interface{}{
								"name":        tt.mockUser.Name,
								"email":       tt.mockUser.Email,
								"is_verified": tt.mockUser.IsVerified,
								"created_at":  tt.mockUser.CreatedAt,
								"updated_at":  tt.mockUser.UpdatedAt,
							},
							"relationships": map[string]interface{}{
								"organizations": map[string]interface{}{
									"data": orgRelationships,
								},
							},
						},
						"included": included,
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

			user, orgs, resp, err := client.User.GetAuthenticatedUser(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
				assert.Nil(t, orgs)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				require.NotNil(t, orgs)
				require.NotNil(t, resp)

				// Verify user data
				assert.Equal(t, tt.mockUser.ID, user.ID)
				assert.Equal(t, tt.mockUser.Email, user.Email)
				assert.Equal(t, tt.mockUser.Name, user.Name)
				assert.Equal(t, tt.mockUser.IsVerified, user.IsVerified)
				assert.WithinDuration(t, tt.mockUser.CreatedAt, user.CreatedAt, time.Second)
				assert.WithinDuration(t, tt.mockUser.UpdatedAt, user.UpdatedAt, time.Second)

				// Verify organizations
				assert.Equal(t, len(tt.mockOrganizations), len(orgs))
				for i, org := range orgs {
					assert.Equal(t, tt.mockOrganizations[i].ID, org.ID)
					assert.Equal(t, tt.mockOrganizations[i].Name, org.Name)
					assert.Equal(t, tt.mockOrganizations[i].Slug, org.Slug)
					assert.Equal(t, tt.mockOrganizations[i].Role, org.Role)
				}
			}
		})
	}
}

func TestUserService_ContextCancellation(t *testing.T) {
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

	user, orgs, _, err := client.User.GetAuthenticatedUser(ctx)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Nil(t, orgs)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestUserService_WithMultipleOrganizationRoles(t *testing.T) {
	mockUser := &dotenv.User{
		ID:         "user-789",
		Email:      "multi@example.com",
		Name:       "Multi Org User",
		IsVerified: true,
		CreatedAt:  time.Now().UTC().Round(time.Second),
		UpdatedAt:  time.Now().UTC().Round(time.Second),
	}

	mockOrganizations := []dotenv.UserOrganization{
		{
			ID:       "org-001",
			ULID:     "01HQNWK1XQXQY1XQXQY1XQXQY1",
			Name:     "Owner Organization",
			Slug:     "owner-org",
			Role:     "owner",
			JoinedAt: time.Now().Add(-365 * 24 * time.Hour).UTC().Round(time.Second),
		},
		{
			ID:       "org-002",
			ULID:     "01HQNWK1XQXQY1XQXQY1XQXQY2",
			Name:     "Admin Organization",
			Slug:     "admin-org",
			Role:     "admin",
			JoinedAt: time.Now().Add(-180 * 24 * time.Hour).UTC().Round(time.Second),
		},
		{
			ID:       "org-003",
			ULID:     "01HQNWK1XQXQY1XQXQY1XQXQY3",
			Name:     "Member Organization",
			Slug:     "member-org",
			Role:     "member",
			JoinedAt: time.Now().Add(-30 * 24 * time.Hour).UTC().Round(time.Second),
		},
		{
			ID:       "org-004",
			ULID:     "01HQNWK1XQXQY1XQXQY1XQXQY4",
			Name:     "Viewer Organization",
			Slug:     "viewer-org",
			Role:     "viewer",
			JoinedAt: time.Now().Add(-7 * 24 * time.Hour).UTC().Round(time.Second),
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create included organizations
		type includeItem struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				ULID string `json:"ulid"`
				Name string `json:"name"`
				Slug string `json:"slug"`
				Role string `json:"role"`
			} `json:"attributes"`
		}

		included := make([]includeItem, 0, len(mockOrganizations))
		orgRelationships := make([]map[string]interface{}, 0, len(mockOrganizations))

		for _, org := range mockOrganizations {
			included = append(included, includeItem{
				Type: "organizations",
				ID:   org.ID,
				Attributes: struct {
					ULID string `json:"ulid"`
					Name string `json:"name"`
					Slug string `json:"slug"`
					Role string `json:"role"`
				}{
					ULID: org.ULID,
					Name: org.Name,
					Slug: org.Slug,
					Role: org.Role,
				},
			})

			orgRelationships = append(orgRelationships, map[string]interface{}{
				"type": "organizations",
				"id":   org.ID,
			})
		}

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "users",
				"id":   mockUser.ID,
				"attributes": map[string]interface{}{
					"name":        mockUser.Name,
					"email":       mockUser.Email,
					"is_verified": mockUser.IsVerified,
					"created_at":  mockUser.CreatedAt,
					"updated_at":  mockUser.UpdatedAt,
				},
				"relationships": map[string]interface{}{
					"organizations": map[string]interface{}{
						"data": orgRelationships,
					},
				},
			},
			"included": included,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
	)

	user, orgs, resp, err := client.User.GetAuthenticatedUser(context.Background())
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, orgs)
	require.NotNil(t, resp)

	// Verify we got all organizations with different roles
	assert.Equal(t, 4, len(orgs))

	// Verify each organization and role
	roles := make(map[string]string)
	for _, org := range orgs {
		roles[org.Slug] = org.Role
	}

	assert.Equal(t, "owner", roles["owner-org"])
	assert.Equal(t, "admin", roles["admin-org"])
	assert.Equal(t, "member", roles["member-org"])
	assert.Equal(t, "viewer", roles["viewer-org"])
}
