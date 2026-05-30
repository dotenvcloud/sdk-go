package dotenv_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	dotenv "github.com/dotenvcloud/sdk-go"
)

func BenchmarkClient_Organizations_List(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [{"type": "organizations", "id": "1", "attributes": {"name": "Test Org", "slug": "test-org"}}]}`))
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
		dotenv.WithOrganization("test-org"),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := client.Organizations.List(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncryption(b *testing.B) {
	key, _ := dotenv.GenerateKey()
	plaintext := "This is a test secret value that needs to be encrypted"

	b.Run("Encrypt", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := dotenv.Encrypt(plaintext, key)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Decrypt", func(b *testing.B) {
		encrypted, _ := dotenv.Encrypt(plaintext, key)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := dotenv.Decrypt(encrypted, key)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("EncryptDecrypt", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encrypted, err := dotenv.Encrypt(plaintext, key)
			if err != nil {
				b.Fatal(err)
			}
			_, err = dotenv.Decrypt(encrypted, key)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkKeyOperations(b *testing.B) {
	b.Run("GenerateKey", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := dotenv.GenerateKey()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("EncodeDecodeKey", func(b *testing.B) {
		key, _ := dotenv.GenerateKey()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encoded := dotenv.EncodeKey(key)
			_, err := dotenv.DecodeKey(encoded)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkJSONParsing(b *testing.B) {
	// Large response payload
	largeResponse := `{
		"data": [
`
	for i := 0; i < 100; i++ {
		if i > 0 {
			largeResponse += ","
		}
		largeResponse += `
			{
				"type": "secrets",
				"id": "secret-` + string(rune(i)) + `",
				"attributes": {
					"key": "SECRET_` + string(rune(i)) + `",
					"value": "encrypted_value_here_with_some_long_content_to_simulate_real_data"
				}
			}`
	}
	largeResponse += `
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(largeResponse))
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
		dotenv.WithOrganization("test-org"),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := client.Secrets.GetProjectSecrets(ctx, "test-project", "", "")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentRequests(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
		dotenv.WithOrganization("test-org"),
	)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := client.Organizations.List(ctx, nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
