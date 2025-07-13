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

	dotenv "github.com/dotenv/sdk-go"
)

func TestOrganizationsService_List(t *testing.T) {
	tests := []struct {
		name           string
		mockOrgs       []*dotenv.Organization
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name: "successful organizations retrieval",
			mockOrgs: []*dotenv.Organization{
				{
					ID:        "org-123",
					ULID:      "01HQNWK1XQXQY1XQXQY1XQXQY1",
					Name:      "Test Organization",
					Slug:      "test-org",
					Status:    "active",
					PlanName:  "pro",
					CreatedAt: time.Now().UTC().Round(time.Second),
					UpdatedAt: time.Now().UTC().Round(time.Second),
				},
				{
					ID:        "org-456",
					ULID:      "01HQNWK1XQXQY1XQXQY1XQXQY2",
					Name:      "Another Organization",
					Slug:      "another-org",
					Status:    "active",
					PlanName:  "enterprise",
					CreatedAt: time.Now().UTC().Round(time.Second),
					UpdatedAt: time.Now().UTC().Round(time.Second),
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "empty organizations list",
			mockOrgs:       []*dotenv.Organization{},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "unauthorized",
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsUnauthorized(err))
			},
		},
		{
			name:           "forbidden - API key authentication",
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
				assert.Equal(t, "/api/v1/organizations", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)

				if tt.mockOrgs != nil && tt.mockStatusCode == http.StatusOK {
					// Create JSON:API response
					data := make([]interface{}, 0, len(tt.mockOrgs))
					for _, org := range tt.mockOrgs {
						data = append(data, map[string]interface{}{
							"type": "organizations",
							"id":   org.ID,
							"attributes": map[string]interface{}{
								"ulid":       org.ULID,
								"name":       org.Name,
								"slug":       org.Slug,
								"status":     org.Status,
								"plan_name":  org.PlanName,
								"created_at": org.CreatedAt,
								"updated_at": org.UpdatedAt,
							},
						})
					}

					response := dotenv.JSONAPIResponse{
						Data: data,
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

			orgs, resp, err := client.Organizations.List(context.Background(), nil)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, orgs)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, orgs)
				require.NotNil(t, resp)

				assert.Equal(t, len(tt.mockOrgs), len(orgs))
				for i, org := range orgs {
					assert.Equal(t, tt.mockOrgs[i].ID, org.ID)
					assert.Equal(t, tt.mockOrgs[i].ULID, org.ULID)
					assert.Equal(t, tt.mockOrgs[i].Name, org.Name)
					assert.Equal(t, tt.mockOrgs[i].Slug, org.Slug)
					assert.Equal(t, tt.mockOrgs[i].Status, org.Status)
					assert.Equal(t, tt.mockOrgs[i].PlanName, org.PlanName)
				}
			}
		})
	}
}

func TestOrganizationsService_Get(t *testing.T) {
	mockOrg := &dotenv.Organization{
		ID:        "org-123",
		ULID:      "01HQNWK1XQXQY1XQXQY1XQXQY1",
		Name:      "Test Organization",
		Slug:      "test-org",
		Status:    "active",
		PlanName:  "pro",
		CreatedAt: time.Now().UTC().Round(time.Second),
		UpdatedAt: time.Now().UTC().Round(time.Second),
	}

	tests := []struct {
		name           string
		slug           string
		mockOrg        *dotenv.Organization
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name:           "successful get",
			slug:           "test-org",
			mockOrg:        mockOrg,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "not found",
			slug:           "non-existent",
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsNotFound(err))
			},
		},
		{
			name:           "unauthorized",
			slug:           "test-org",
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsUnauthorized(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("/api/v1/organizations/%s", tt.slug), r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)

				if tt.mockOrg != nil && tt.mockStatusCode == http.StatusOK {
					response := dotenv.JSONAPIResponse{
						Data: map[string]interface{}{
							"type": "organizations",
							"id":   tt.mockOrg.ID,
							"attributes": map[string]interface{}{
								"ulid":       tt.mockOrg.ULID,
								"name":       tt.mockOrg.Name,
								"slug":       tt.mockOrg.Slug,
								"status":     tt.mockOrg.Status,
								"plan_name":  tt.mockOrg.PlanName,
								"created_at": tt.mockOrg.CreatedAt,
								"updated_at": tt.mockOrg.UpdatedAt,
							},
						},
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

			org, resp, err := client.Organizations.Get(context.Background(), tt.slug)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, org)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, org)
				require.NotNil(t, resp)

				assert.Equal(t, tt.mockOrg.ID, org.ID)
				assert.Equal(t, tt.mockOrg.Name, org.Name)
				assert.Equal(t, tt.mockOrg.Slug, org.Slug)
			}
		})
	}
}

func TestOrganizationsService_Create(t *testing.T) {
	tests := []struct {
		name           string
		createReq      *dotenv.OrganizationCreateRequest
		mockOrg        *dotenv.Organization
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name: "successful create",
			createReq: &dotenv.OrganizationCreateRequest{
				Name: "New Organization",
				Slug: "new-org",
			},
			mockOrg: &dotenv.Organization{
				ID:        "org-789",
				ULID:      "01HQNWK1XQXQY1XQXQY1XQXQY3",
				Name:      "New Organization",
				Slug:      "new-org",
				Status:    "active",
				PlanName:  "free",
				CreatedAt: time.Now().UTC().Round(time.Second),
				UpdatedAt: time.Now().UTC().Round(time.Second),
			},
			mockStatusCode: http.StatusCreated,
			wantErr:        false,
		},
		{
			name: "create without slug",
			createReq: &dotenv.OrganizationCreateRequest{
				Name: "Auto Slug Organization",
			},
			mockOrg: &dotenv.Organization{
				ID:        "org-890",
				ULID:      "01HQNWK1XQXQY1XQXQY1XQXQY4",
				Name:      "Auto Slug Organization",
				Slug:      "auto-slug-organization",
				Status:    "active",
				PlanName:  "free",
				CreatedAt: time.Now().UTC().Round(time.Second),
				UpdatedAt: time.Now().UTC().Round(time.Second),
			},
			mockStatusCode: http.StatusCreated,
			wantErr:        false,
		},
		{
			name: "conflict - duplicate slug",
			createReq: &dotenv.OrganizationCreateRequest{
				Name: "Duplicate Org",
				Slug: "existing-org",
			},
			mockStatusCode: http.StatusConflict,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsConflict(err))
			},
		},
		{
			name: "unauthorized - no permission to create",
			createReq: &dotenv.OrganizationCreateRequest{
				Name: "No Permission Org",
			},
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsUnauthorized(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/organizations", r.URL.Path)
				assert.Equal(t, "POST", r.Method)

				// Verify request body
				var reqBody dotenv.OrganizationCreateRequest
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, tt.createReq.Name, reqBody.Name)
				if tt.createReq.Slug != "" {
					assert.Equal(t, tt.createReq.Slug, reqBody.Slug)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)

				if tt.mockOrg != nil && tt.mockStatusCode == http.StatusCreated {
					response := dotenv.JSONAPIResponse{
						Data: map[string]interface{}{
							"type": "organizations",
							"id":   tt.mockOrg.ID,
							"attributes": map[string]interface{}{
								"ulid":       tt.mockOrg.ULID,
								"name":       tt.mockOrg.Name,
								"slug":       tt.mockOrg.Slug,
								"status":     tt.mockOrg.Status,
								"plan_name":  tt.mockOrg.PlanName,
								"created_at": tt.mockOrg.CreatedAt,
								"updated_at": tt.mockOrg.UpdatedAt,
							},
						},
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

			org, resp, err := client.Organizations.Create(context.Background(), tt.createReq)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, org)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, org)
				require.NotNil(t, resp)

				assert.Equal(t, tt.mockOrg.Name, org.Name)
				assert.Equal(t, tt.mockOrg.Slug, org.Slug)
				assert.Equal(t, tt.mockOrg.Status, org.Status)
			}
		})
	}
}

func TestOrganizationsService_Update(t *testing.T) {
	newName := "Updated Organization"
	newSlug := "updated-org"

	tests := []struct {
		name           string
		orgID          string
		updateReq      *dotenv.OrganizationUpdateRequest
		mockOrg        *dotenv.Organization
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name:  "successful update - name only",
			orgID: "org-123",
			updateReq: &dotenv.OrganizationUpdateRequest{
				Name: &newName,
			},
			mockOrg: &dotenv.Organization{
				ID:        "org-123",
				ULID:      "01HQNWK1XQXQY1XQXQY1XQXQY1",
				Name:      newName,
				Slug:      "test-org",
				Status:    "active",
				PlanName:  "pro",
				CreatedAt: time.Now().UTC().Round(time.Second),
				UpdatedAt: time.Now().UTC().Round(time.Second),
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:  "successful update - both name and slug",
			orgID: "org-123",
			updateReq: &dotenv.OrganizationUpdateRequest{
				Name: &newName,
				Slug: &newSlug,
			},
			mockOrg: &dotenv.Organization{
				ID:        "org-123",
				ULID:      "01HQNWK1XQXQY1XQXQY1XQXQY1",
				Name:      newName,
				Slug:      newSlug,
				Status:    "active",
				PlanName:  "pro",
				CreatedAt: time.Now().UTC().Round(time.Second),
				UpdatedAt: time.Now().UTC().Round(time.Second),
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:  "not found",
			orgID: "non-existent",
			updateReq: &dotenv.OrganizationUpdateRequest{
				Name: &newName,
			},
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsNotFound(err))
			},
		},
		{
			name:  "forbidden - no permission",
			orgID: "org-456",
			updateReq: &dotenv.OrganizationUpdateRequest{
				Name: &newName,
			},
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
				assert.Equal(t, fmt.Sprintf("/api/v1/organizations/%s", tt.orgID), r.URL.Path)
				assert.Equal(t, "PATCH", r.Method)

				// Verify request body
				var reqBody dotenv.OrganizationUpdateRequest
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				if tt.updateReq.Name != nil {
					assert.Equal(t, *tt.updateReq.Name, *reqBody.Name)
				}
				if tt.updateReq.Slug != nil {
					assert.Equal(t, *tt.updateReq.Slug, *reqBody.Slug)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)

				if tt.mockOrg != nil && tt.mockStatusCode == http.StatusOK {
					response := dotenv.JSONAPIResponse{
						Data: map[string]interface{}{
							"type": "organizations",
							"id":   tt.mockOrg.ID,
							"attributes": map[string]interface{}{
								"ulid":       tt.mockOrg.ULID,
								"name":       tt.mockOrg.Name,
								"slug":       tt.mockOrg.Slug,
								"status":     tt.mockOrg.Status,
								"plan_name":  tt.mockOrg.PlanName,
								"created_at": tt.mockOrg.CreatedAt,
								"updated_at": tt.mockOrg.UpdatedAt,
							},
						},
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

			org, resp, err := client.Organizations.Update(context.Background(), tt.orgID, tt.updateReq)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, org)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, org)
				require.NotNil(t, resp)

				assert.Equal(t, tt.mockOrg.ID, org.ID)
				assert.Equal(t, tt.mockOrg.Name, org.Name)
				assert.Equal(t, tt.mockOrg.Slug, org.Slug)
			}
		})
	}
}

func TestOrganizationsService_Delete(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name:           "successful delete",
			orgID:          "org-123",
			mockStatusCode: http.StatusNoContent,
			wantErr:        false,
		},
		{
			name:           "not found",
			orgID:          "non-existent",
			mockStatusCode: http.StatusNotFound,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsNotFound(err))
			},
		},
		{
			name:           "forbidden - no permission",
			orgID:          "org-456",
			mockStatusCode: http.StatusForbidden,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsForbidden(err))
			},
		},
		{
			name:           "unauthorized",
			orgID:          "org-789",
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsUnauthorized(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("/api/v1/organizations/%s", tt.orgID), r.URL.Path)
				assert.Equal(t, "DELETE", r.Method)

				w.Header().Set("Content-Type", "application/json")
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

			resp, err := client.Organizations.Delete(context.Background(), tt.orgID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
			}
		})
	}
}

func TestOrganizationsService_ContextCancellation(t *testing.T) {
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

	// Test List
	orgs, _, err := client.Organizations.List(ctx, nil)
	assert.Error(t, err)
	assert.Nil(t, orgs)
	assert.Contains(t, err.Error(), "context canceled")

	// Test Get
	org, _, err := client.Organizations.Get(ctx, "test-org")
	assert.Error(t, err)
	assert.Nil(t, org)

	// Test Create
	createReq := &dotenv.OrganizationCreateRequest{Name: "Test"}
	org, _, err = client.Organizations.Create(ctx, createReq)
	assert.Error(t, err)
	assert.Nil(t, org)

	// Test Update
	updateReq := &dotenv.OrganizationUpdateRequest{Name: &createReq.Name}
	org, _, err = client.Organizations.Update(ctx, "org-123", updateReq)
	assert.Error(t, err)
	assert.Nil(t, org)

	// Test Delete
	_, err = client.Organizations.Delete(ctx, "org-123")
	assert.Error(t, err)
}
