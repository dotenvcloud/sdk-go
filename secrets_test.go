package dotenv_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	dotenv "github.com/dotenvcloud/sdk-go"
)

func newSecretsTestClient(handler http.HandlerFunc) (*dotenv.Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
		dotenv.WithOrganization("test-org"),
	)
	return client, server
}

func TestStoreSecrets_RequestShape(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}

	client, server := newSecretsTestClient(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"secrets","level":"environment","source":"web","bytes":7},"message":"ok"}`))
	})
	defer server.Close()

	resp, err := client.Secrets.StoreSecrets(context.Background(), "myproj", "prod", "web", "CIPHERTEXT")
	if err != nil {
		t.Fatalf("StoreSecrets returned error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v1/test-org/secrets/store" {
		t.Errorf("path = %q, want /api/v1/test-org/secrets/store", gotPath)
	}
	for k, want := range map[string]string{"project": "myproj", "target": "prod", "environment": "web", "content": "CIPHERTEXT"} {
		if got, _ := gotBody[k].(string); got != want {
			t.Errorf("body[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestStoreSecrets_ProjectLevelOmitsEmpty(t *testing.T) {
	var gotBody map[string]interface{}
	client, server := newSecretsTestClient(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{}`))
	})
	defer server.Close()

	resp, err := client.Secrets.StoreSecrets(context.Background(), "myproj", "", "", "BLOB")
	if err != nil {
		t.Fatalf("StoreSecrets returned error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	if _, present := gotBody["target"]; present {
		t.Errorf("empty target should be omitted, got %v", gotBody["target"])
	}
	if _, present := gotBody["environment"]; present {
		t.Errorf("empty environment should be omitted, got %v", gotBody["environment"])
	}
}

func TestDeleteSecretLevel_RequestShape(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}

	client, server := newSecretsTestClient(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	resp, err := client.Secrets.DeleteSecretLevel(context.Background(), "myproj", "prod", "")
	if err != nil {
		t.Fatalf("DeleteSecretLevel returned error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v1/test-org/secrets/delete" {
		t.Errorf("path = %q, want /api/v1/test-org/secrets/delete", gotPath)
	}
	if got, _ := gotBody["project"].(string); got != "myproj" {
		t.Errorf("body[project] = %q, want myproj", got)
	}
	if got, _ := gotBody["target"].(string); got != "prod" {
		t.Errorf("body[target] = %q, want prod", got)
	}
}
