//go:build integration
// +build integration

package dotenv_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	dotenv "github.com/dotenv/sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests that test the full client flow

func TestIntegration_FullWorkflow(t *testing.T) {
	// Create a more comprehensive mock server
	server := createIntegrationServer(t)
	defer server.Close()

	client := dotenv.NewClient("test-key", dotenv.WithBaseURL(server.URL))
	ctx := context.Background()

	// Step 1: List organizations
	orgs, _, err := client.Organizations.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, orgs, 1)

	// Step 2: Get specific organization
	org, _, err := client.Organizations.Get(ctx, orgs[0].Slug)
	require.NoError(t, err)
	assert.Equal(t, orgs[0].ID, org.ID)

	// Step 3: List projects for organization
	projects, _, err := client.Projects.List(ctx, org.Slug, nil)
	require.NoError(t, err)
	require.Len(t, projects, 2)

	// Step 4: Get project details
	project, _, err := client.Projects.Get(ctx, projects[0].Slug)
	require.NoError(t, err)
	assert.Equal(t, projects[0].ID, project.ID)

	// Step 5: List environments
	envs, _, err := client.Environments.List(ctx, project.Slug, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, envs)

	// Step 6: Get secrets with encryption
	key, _, err := client.Encryption.GetEncryptionKey(ctx, project.Slug)
	require.NoError(t, err)

	secrets, _, err := client.Secrets.GetProjectSecrets(ctx, project.Slug, "production", "web")
	require.NoError(t, err)
	assert.NotEmpty(t, secrets)

	// Decrypt secrets
	decodedKey, err := dotenv.DecodeKey(key.Key)
	require.NoError(t, err)

	for k, v := range secrets {
		decrypted, err := dotenv.Decrypt(v, decodedKey)
		require.NoError(t, err)
		assert.NotEmpty(t, decrypted)
		t.Logf("Decrypted %s: %s", k, decrypted)
	}
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	server := createIntegrationServer(t)
	defer server.Close()

	client := dotenv.NewClient("test-key", dotenv.WithBaseURL(server.URL))
	ctx := context.Background()

	// Test concurrent requests
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, err := client.Organizations.List(ctx, nil)
			if err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

func TestIntegration_Pagination(t *testing.T) {
	server := createPaginationServer(t)
	defer server.Close()

	client := dotenv.NewClient("test-key", dotenv.WithBaseURL(server.URL))
	ctx := context.Background()

	// Test pagination
	opt := &dotenv.ListOptions{
		Page:    1,
		PerPage: 2,
	}

	allProjects := []dotenv.Project{}
	for {
		projects, resp, err := client.Projects.List(ctx, "test-org", opt)
		require.NoError(t, err)

		allProjects = append(allProjects, projects...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	assert.Len(t, allProjects, 5) // Total projects
}

func TestIntegration_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		handler       http.HandlerFunc
		expectedError func(error) bool
		retryExpected bool
	}{
		{
			name: "rate limit",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"message": "Rate limit exceeded"}`))
			},
			expectedError: dotenv.IsRateLimited,
		},
		{
			name: "server error with retry",
			handler: func() http.HandlerFunc {
				attempts := 0
				return func(w http.ResponseWriter, r *http.Request) {
					attempts++
					if attempts < 3 {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"data": []}`))
				}
			}(),
			expectedError: nil,
			retryExpected: true,
		},
		{
			name: "validation error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity)
				w.Write([]byte(`{"errors": [{"detail": "Validation failed"}]}`))
			},
			expectedError: func(err error) bool {
				return err != nil && !dotenv.IsRetryable(err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := dotenv.NewClient("test-key", dotenv.WithBaseURL(server.URL))
			_, _, err := client.Organizations.List(context.Background(), nil)

			if tt.expectedError != nil {
				assert.True(t, tt.expectedError(err))
			} else if tt.retryExpected {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIntegration_Timeout(t *testing.T) {
	// Create a server that delays responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with short timeout
	client := dotenv.NewClient("test-key",
		dotenv.WithBaseURL(server.URL),
		dotenv.WithTimeout(100*time.Millisecond),
	)

	ctx := context.Background()
	_, _, err := client.Organizations.List(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

// Helper functions

func createIntegrationServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/organizations" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": [{
					"type": "organizations",
					"id": "org-1",
					"attributes": {
						"ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
						"name": "Test Org",
						"slug": "test-org",
						"status": "active",
						"plan_name": "pro"
					}
				}]
			}`))

		case strings.Contains(r.URL.Path, "/organizations/test-org/projects"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": [
					{
						"type": "projects",
						"id": "proj-1",
						"attributes": {
							"ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
							"name": "Test Project",
							"slug": "test-project",
							"has_secrets": true,
							"secret_count": 3
						}
					},
					{
						"type": "projects",
						"id": "proj-2",
						"attributes": {
							"ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
							"name": "Another Project",
							"slug": "another-project",
							"has_secrets": true,
							"secret_count": 5
						}
					}
				]
			}`))

		case strings.Contains(r.URL.Path, "/environments"):
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"data": [
					{
						"type": "environments",
						"id": "env-1",
						"attributes": {
							"name": "production",
							"slug": "production"
						}
					},
					{
						"type": "environments",
						"id": "env-2",
						"attributes": {
							"name": "staging",
							"slug": "staging"
						}
					}
				]
			}`))

		case strings.Contains(r.URL.Path, "/encryption-key"):
			w.Header().Set("Content-Type", "application/json")
			// Real base64 encoded 32-byte key
			w.Write([]byte(`{
				"data": {
					"type": "encryption_keys",
					"id": "key-1",
					"attributes": {
						"key": "YTY0NzgyYjU4ZjU3MjM4YWQ3MjM0ZjgzYjM0ZmEzNGQ=",
						"is_active": true
					}
				}
			}`))

		case strings.Contains(r.URL.Path, "/secrets"):
			w.Header().Set("Content-Type", "application/json")
			// Return encrypted secrets
			w.Write([]byte(`{
				"data": [
					{
						"type": "secrets",
						"id": "secret-1",
						"attributes": {
							"key": "DATABASE_URL",
							"value": "encrypted_value_here"
						}
					}
				]
			}`))

		default:
			if strings.HasPrefix(r.URL.Path, "/api/v1/organizations/") {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{
					"type": "organizations",
					"id": "org-1",
					"attributes": {
						"ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
						"name": "Test Org",
						"slug": "test-org",
						"status": "active"
					}
				}`))
			} else if strings.Contains(r.URL.Path, "/projects/") {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{
					"type": "projects",
					"id": "proj-1",
					"attributes": {
						"ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
						"name": "Test Project",
						"slug": "test-project"
					}
				}`))
			} else {
				http.NotFound(w, r)
			}
		}
	}))
}

func createPaginationServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/projects") {
			page := r.URL.Query().Get("page")
			perPage := r.URL.Query().Get("per_page")

			if page == "" {
				page = "1"
			}
			if perPage == "" {
				perPage = "10"
			}

			pageNum := 1
			fmt.Sscanf(page, "%d", &pageNum)

			// Simulate 5 total projects, 2 per page
			allProjects := []map[string]interface{}{}
			for i := 1; i <= 5; i++ {
				allProjects = append(allProjects, map[string]interface{}{
					"type": "projects",
					"id":   fmt.Sprintf("proj-%d", i),
					"attributes": map[string]interface{}{
						"name": fmt.Sprintf("Project %d", i),
						"slug": fmt.Sprintf("project-%d", i),
					},
				})
			}

			// Calculate pagination
			start := (pageNum - 1) * 2
			end := start + 2
			if end > len(allProjects) {
				end = len(allProjects)
			}

			data := allProjects[start:end]

			// Set pagination headers
			if pageNum < 3 {
				w.Header().Set("X-Next-Page", fmt.Sprintf("%d", pageNum+1))
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": data,
			})
		} else {
			http.NotFound(w, r)
		}
	}))
}
