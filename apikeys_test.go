package dotenv_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dotenv "github.com/dotenvcloud/sdk-go"
)

const apiKeysOrg = "01htest0000000000000000000"

func apiKeysClient(server *httptest.Server) *dotenv.Client {
	return dotenv.NewClient(dotenv.WithAPIKey("test-key"), dotenv.WithBaseURL(server.URL))
}

func TestAPIKeysService_List(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		// Flat ApiTokenResource shape (id is a string).
		_, _ = w.Write([]byte(`{"data":[
			{"id":"7","name":"CI key","abilities":["secret:read"],"created_at":"2024-01-01T00:00:00Z"}
		]}`))
	}))
	defer server.Close()

	keys, resp, err := apiKeysClient(server).APIKeys.List(context.Background(), apiKeysOrg)
	require.NoError(t, err)
	if resp != nil {
		_ = resp.Body.Close()
	}

	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "/api/v1/organizations/"+apiKeysOrg+"/api-keys", gotPath)
	require.Len(t, keys, 1)
	assert.Equal(t, "7", keys[0].ID)
	assert.Equal(t, "CI key", keys[0].Name)
	assert.Equal(t, []string{"secret:read"}, keys[0].Abilities)
}

func TestAPIKeysService_Create(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// store() returns a flat {token:{...}} object.
		_, _ = w.Write([]byte(`{"token":{"name":"CI key","token":"dotenv_abc123","abilities":["secret:read"],"expires_at":null},"message":"ok"}`))
	}))
	defer server.Close()

	out, resp, err := apiKeysClient(server).APIKeys.Create(context.Background(), apiKeysOrg, dotenv.APIKeyCreateRequest{
		Name:      "CI key",
		Abilities: []string{"secret:read"},
	})
	require.NoError(t, err)
	if resp != nil {
		_ = resp.Body.Close()
	}

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/api/v1/organizations/"+apiKeysOrg+"/api-keys", gotPath)
	assert.Equal(t, "dotenv_abc123", out.Token)
	require.NotNil(t, out.APIKey)
	assert.Equal(t, "CI key", out.Name)
	assert.Equal(t, []string{"secret:read"}, out.Abilities)
}

func TestAPIKeysService_Update(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"7","name":"Renamed","abilities":["secret:read"],"created_at":"2024-01-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	key, resp, err := apiKeysClient(server).APIKeys.Update(context.Background(), apiKeysOrg, "7", dotenv.APIKeyUpdateRequest{Name: "Renamed"})
	require.NoError(t, err)
	if resp != nil {
		_ = resp.Body.Close()
	}

	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/api/v1/organizations/"+apiKeysOrg+"/api-keys/7", gotPath)
	assert.Equal(t, "7", key.ID)
	assert.Equal(t, "Renamed", key.Name)
}

func TestAPIKeysService_Delete(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	resp, err := apiKeysClient(server).APIKeys.Delete(context.Background(), apiKeysOrg, "7")
	require.NoError(t, err)
	if resp != nil {
		_ = resp.Body.Close()
	}

	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/api/v1/organizations/"+apiKeysOrg+"/api-keys/7", gotPath)
}

func TestAPIKeysService_Rotate(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		// ApiTokenCreationResource shape: {data:{token, api_key:{...}}}.
		_, _ = w.Write([]byte(`{"data":{"token":"dotenv_new456","api_key":{"id":"7","name":"CI key","abilities":["secret:read"],"created_at":"2024-01-01T00:00:00Z"}}}`))
	}))
	defer server.Close()

	out, resp, err := apiKeysClient(server).APIKeys.Rotate(context.Background(), apiKeysOrg, "7")
	require.NoError(t, err)
	if resp != nil {
		_ = resp.Body.Close()
	}

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/api/v1/organizations/"+apiKeysOrg+"/api-keys/7/rotate", gotPath)
	assert.Equal(t, "dotenv_new456", out.Token)
	assert.Equal(t, "7", out.ID)
	require.NotNil(t, out.APIKey)
	assert.Equal(t, "CI key", out.Name)
}

func TestAPIKeysService_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"insufficient_permissions","message":"Insufficient permissions"}`))
	}))
	defer server.Close()

	_, _, err := apiKeysClient(server).APIKeys.List(context.Background(), apiKeysOrg)
	assert.Error(t, err)
}
