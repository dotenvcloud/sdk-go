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

func newVersionsTestClient(handler http.HandlerFunc) (*dotenv.Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
		dotenv.WithOrganization("test-org"),
	)
	return client, server
}

func TestSecretVersionsList_RequestShapeAndParsing(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}

	client, server := newVersionsTestClient(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"type":"secret-versions","id":"7","attributes":{"action":"update","size_bytes":42,"is_client_encrypted":true,"encryption_key_version":"2","created_at":"2026-06-11T00:00:00Z"}}],"meta":{"current_page":1,"last_page":3,"per_page":50,"total":120}}`))
	})
	defer server.Close()

	versions, meta, resp, err := client.SecretVersions.List(context.Background(), "myproj", "prod", "", &dotenv.VersionListOptions{Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v1/test-org/secrets/versions" {
		t.Errorf("path = %q", gotPath)
	}
	if got, _ := gotBody["project"].(string); got != "myproj" {
		t.Errorf("body.project = %q, want myproj", got)
	}
	if got, _ := gotBody["target"].(string); got != "prod" {
		t.Errorf("body.target = %q, want prod", got)
	}
	if _, ok := gotBody["environment"]; ok {
		t.Errorf("environment should be omitted when empty")
	}
	if len(versions) != 1 || versions[0].ID != "7" || versions[0].Action != "update" || versions[0].SizeBytes != 42 {
		t.Errorf("unexpected versions: %+v", versions)
	}
	if meta == nil || meta.LastPage != 3 || meta.Total != 120 {
		t.Errorf("unexpected meta: %+v", meta)
	}
}

func TestSecretVersionsGet_ParsesContentAndKey(t *testing.T) {
	client, server := newVersionsTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/test-org/secrets/versions/9" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"secret-versions","id":"9","attributes":{"action":"rotate","size_bytes":10},"content":"CIPHER","key":{"version":"1","managed":"client","key_check_salt":"c2FsdA==","key_check_iterations":600000}}}`))
	})
	defer server.Close()

	v, resp, err := client.SecretVersions.Get(context.Background(), "9")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	if v.ID != "9" || v.Content != "CIPHER" || v.Action != "rotate" {
		t.Errorf("unexpected version: %+v", v)
	}
	if v.Key == nil || v.Key.Managed != "client" || v.Key.KeyCheckSalt != "c2FsdA==" {
		t.Errorf("unexpected key: %+v", v.Key)
	}
}

func TestSecretVersionsRestore_SendsBody(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	client, server := newVersionsTestClient(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"secrets","secret_id":1,"restored_from":"9"},"message":"ok"}`))
	})
	defer server.Close()

	resp, err := client.SecretVersions.Restore(context.Background(), "9", &dotenv.RestoreVersionRequest{Content: "NEWCIPHER", KeyProof: "PROOF"})
	if err != nil {
		t.Fatalf("Restore returned error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	if gotPath != "/api/v1/test-org/secrets/versions/9/restore" {
		t.Errorf("path = %q", gotPath)
	}
	if got, _ := gotBody["content"].(string); got != "NEWCIPHER" {
		t.Errorf("body.content = %q", got)
	}
	if got, _ := gotBody["key_proof"].(string); got != "PROOF" {
		t.Errorf("body.key_proof = %q", got)
	}
}

func TestSecretVersionsPurge_ReturnsCount(t *testing.T) {
	client, server := newVersionsTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/test-org/secrets/versions/purge" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"purged_count":5},"message":"ok"}`))
	})
	defer server.Close()

	count, resp, err := client.SecretVersions.PurgeHistory(context.Background(), dotenv.PurgeHistoryRequest{Project: "myproj", Confirmed: true})
	if err != nil {
		t.Fatalf("PurgeHistory returned error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
}

func TestStoreSecretsWithOptions_SendsNoBackup(t *testing.T) {
	var gotBody map[string]interface{}
	client, server := newVersionsTestClient(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"secrets","level":"project","bytes":1},"message":"ok"}`))
	})
	defer server.Close()

	resp, err := client.Secrets.StoreSecretsWithOptions(context.Background(), "myproj", "", "", "CIPHER", "", true)
	if err != nil {
		t.Fatalf("StoreSecretsWithOptions error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	if nb, _ := gotBody["no_backup"].(bool); !nb {
		t.Errorf("no_backup = %v, want true", gotBody["no_backup"])
	}
}

func TestVerifyKeyProof_MatchesGeneratedProof(t *testing.T) {
	salt, proof, iterations, err := dotenv.GenerateKeyProof("my-secret-key")
	if err != nil {
		t.Fatalf("GenerateKeyProof error: %v", err)
	}

	ok, err := dotenv.VerifyKeyProof("my-secret-key", salt, iterations, proof)
	if err != nil {
		t.Fatalf("VerifyKeyProof error: %v", err)
	}
	if !ok {
		t.Error("VerifyKeyProof returned false for the correct key")
	}

	ok, err = dotenv.VerifyKeyProof("wrong-key", salt, iterations, proof)
	if err != nil {
		t.Fatalf("VerifyKeyProof error: %v", err)
	}
	if ok {
		t.Error("VerifyKeyProof returned true for the wrong key")
	}
}
