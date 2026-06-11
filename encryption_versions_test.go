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

func newEncVersionsTestClient(handler http.HandlerFunc) (*dotenv.Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
		dotenv.WithOrganization("test-org"),
	)
	return client, server
}

// Regression: secrets[].id is a STRING per the API contract — an int here 422s
// every client rotation on strict servers.
func TestRotateClientKeys_RequestShape(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	client, server := newEncVersionsTestClient(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"message":"ok","count":1},"message":"ok"}`))
	})
	defer server.Close()

	resp, err := client.Encryption.RotateClientKeys(context.Background(), "myproj", dotenv.ClientKeyRotationRequest{
		Secrets:            []dotenv.RotatedSecret{{ID: "42", Content: "CIPHER"}},
		KeyCheck:           "PROOF",
		KeyCheckSalt:       "U0FMVA==",
		KeyCheckIterations: 600000,
		HistoryPolicy:      "re_encrypt",
	})
	if err != nil {
		t.Fatalf("RotateClientKeys error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	if gotPath != "/api/v1/test-org/myproj/secrets/rotate-client-keys" {
		t.Errorf("path = %q", gotPath)
	}
	secrets, _ := gotBody["secrets"].([]interface{})
	if len(secrets) != 1 {
		t.Fatalf("secrets length = %d, want 1", len(secrets))
	}
	first, _ := secrets[0].(map[string]interface{})
	if id, ok := first["id"].(string); !ok || id != "42" {
		t.Errorf("secrets[0].id = %v (%T), want string \"42\"", first["id"], first["id"])
	}
	for k, want := range map[string]string{"key_check": "PROOF", "key_check_salt": "U0FMVA==", "history_policy": "re_encrypt"} {
		if got, _ := gotBody[k].(string); got != want {
			t.Errorf("body[%q] = %q, want %q", k, got, want)
		}
	}
	if iters, _ := gotBody["key_check_iterations"].(float64); int(iters) != 600000 {
		t.Errorf("key_check_iterations = %v, want 600000", gotBody["key_check_iterations"])
	}
}

func TestGetKeyHistory_ParsesDescriptors(t *testing.T) {
	client, server := newEncVersionsTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/test-org/myproj/encryption-key/history" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"version":"2","managed":"client","key_check":"P","key_check_salt":"S","key_check_iterations":600000,"is_active":true,"created_at":"2026-06-11T00:00:00Z"},{"version":"1","managed":"client","is_active":false,"rotated_at":"2026-06-11T01:00:00Z"}]}`))
	})
	defer server.Close()

	keys, resp, err := client.Encryption.GetKeyHistory(context.Background(), "myproj")
	if err != nil {
		t.Fatalf("GetKeyHistory error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	if len(keys) != 2 {
		t.Fatalf("keys length = %d, want 2", len(keys))
	}
	if keys[0].Version != "2" || !keys[0].IsActive || keys[0].KeyCheck != "P" {
		t.Errorf("unexpected first key: %+v", keys[0])
	}
	if keys[1].RotatedAt == "" {
		t.Errorf("expected rotated_at on the old key")
	}
}

func TestRotateEncryptionKey_ParsesEnvelopeAndSendsPolicy(t *testing.T) {
	var gotBody map[string]interface{}
	client, server := newEncVersionsTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/test-org/myproj/encryption-key/rotate" {
			t.Errorf("path = %q", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"ok","data":{"key":{"type":"encryption-keys","id":"7","attributes":{"id":7,"key":"NEWKEY","version":"3","is_active":true,"created_at":"2026-06-11T00:00:00Z"}}}}`))
	})
	defer server.Close()

	key, resp, err := client.Encryption.RotateEncryptionKey(context.Background(), "myproj", &dotenv.RotateOptions{HistoryPolicy: "keep"})
	if err != nil {
		t.Fatalf("RotateEncryptionKey error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	if got, _ := gotBody["history_policy"].(string); got != "keep" {
		t.Errorf("history_policy = %q, want keep", got)
	}
	if key.ID != "7" || key.Key != "NEWKEY" {
		t.Errorf("unexpected key: %+v", key)
	}
}

func TestPendingAndSubmitReencrypt_Loop(t *testing.T) {
	step := 0
	client, server := newEncVersionsTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/test-org/myproj/secrets/re-encrypt-history/pending":
			_, _ = w.Write([]byte(`{"data":[{"id":11,"content":"OLDCIPHER","key_version":"1"}],"meta":{"remaining":1}}`))
		case "/api/v1/test-org/myproj/secrets/re-encrypt-history":
			step++
			_, _ = w.Write([]byte(`{"data":{"updated":1,"remaining":0}}`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	})
	defer server.Close()

	pending, remaining, resp, err := client.Encryption.ListPendingReencrypt(context.Background(), "myproj", 50)
	if err != nil {
		t.Fatalf("ListPendingReencrypt error: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	if len(pending) != 1 || pending[0].ID != 11 || remaining != 1 {
		t.Fatalf("unexpected pending: %+v remaining=%d", pending, remaining)
	}

	updated, remaining2, resp2, err := client.Encryption.SubmitReencryptedHistory(context.Background(), "myproj",
		[]dotenv.ReencryptedVersion{{ID: 11, Content: "NEWCIPHER"}}, "PROOF")
	if err != nil {
		t.Fatalf("SubmitReencryptedHistory error: %v", err)
	}
	if resp2 != nil {
		_ = resp2.Body.Close()
	}
	if updated != 1 || remaining2 != 0 || step != 1 {
		t.Errorf("updated=%d remaining=%d step=%d", updated, remaining2, step)
	}
}
