package dotenv_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dotenv "github.com/dotenvcloud/sdk-go"
)

func sampleEvent() dotenv.TelemetryRequest {
	return dotenv.TelemetryRequest{
		Version:     "1.2.3",
		OS:          "darwin",
		Arch:        "arm64",
		Command:     "pull",
		Duration:    1500,
		Success:     true,
		AnonymousID: "550e8400-e29b-41d4-a716-446655440000",
	}
}

func TestTelemetryService_Send(t *testing.T) {
	tests := []struct {
		name           string
		event          dotenv.TelemetryRequest
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name:           "successful flat event",
			event:          sampleEvent(),
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful error event",
			event: dotenv.TelemetryRequest{
				Version:     "1.2.3",
				OS:          "linux",
				Arch:        "amd64",
				Command:     "push",
				Duration:    0,
				Success:     false,
				ErrorType:   "authentication",
				AnonymousID: "550e8400-e29b-41d4-a716-446655440000",
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "event carrying a non-empty features list",
			event: dotenv.TelemetryRequest{
				Version:     "1.2.3",
				OS:          "darwin",
				Arch:        "arm64",
				Command:     "pull",
				Duration:    1500,
				Success:     true,
				Features:    []string{"resolve", "decrypt"},
				AnonymousID: "550e8400-e29b-41d4-a716-446655440000",
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "validation rejected",
			event:          sampleEvent(),
			mockStatusCode: http.StatusUnprocessableEntity,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "422")
			},
		},
		{
			name:           "server error",
			event:          sampleEvent(),
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "500")
			},
		},
		{
			name:           "rate limit exceeded",
			event:          sampleEvent(),
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsRateLimited(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/cli/telemetry", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Body is a single flat TelemetryRequest, not a batch wrapper.
				var got dotenv.TelemetryRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
				assert.Equal(t, tt.event, got)

				w.WriteHeader(tt.mockStatusCode)
				if tt.mockStatusCode >= 400 {
					errorResp := dotenv.JSONAPIResponse{
						Errors: []dotenv.JSONAPIError{{
							Status: fmt.Sprintf("%d", tt.mockStatusCode),
							Title:  http.StatusText(tt.mockStatusCode),
							Detail: "Error occurred",
						}},
					}
					_ = json.NewEncoder(w).Encode(errorResp)
				} else {
					_ = json.NewEncoder(w).Encode(dotenv.TelemetryResponse{Success: true})
				}
			}))
			defer server.Close()

			client := dotenv.NewClient(dotenv.WithBaseURL(server.URL))

			resp, err := client.Telemetry.Send(context.Background(), tt.event)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.mockStatusCode, resp.StatusCode)
			}
		})
	}
}

// Without a configured secret, no signature headers are attached.
func TestTelemetryService_Send_Unsigned(t *testing.T) {
	var ts, sig string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts = r.Header.Get("X-Telemetry-Timestamp")
		sig = r.Header.Get("X-Telemetry-Signature")
		_ = json.NewEncoder(w).Encode(dotenv.TelemetryResponse{Success: true})
	}))
	defer server.Close()

	client := dotenv.NewClient(dotenv.WithBaseURL(server.URL))
	_, err := client.Telemetry.Send(context.Background(), sampleEvent())
	require.NoError(t, err)

	assert.Empty(t, ts)
	assert.Empty(t, sig)
}

// With a configured secret, the signature is HMAC-SHA256 over "{timestamp}.{rawBody}"
// computed against the exact wire bytes — matching the server's check.
func TestTelemetryService_Send_Signed(t *testing.T) {
	const secret = "test-telemetry-secret"

	var verified bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts := r.Header.Get("X-Telemetry-Timestamp")
		sig := r.Header.Get("X-Telemetry-Signature")
		require.NotEmpty(t, ts)
		require.NotEmpty(t, sig)

		_, err := strconv.ParseInt(ts, 10, 64)
		require.NoError(t, err)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(ts + "." + string(body)))
		expected := hex.EncodeToString(mac.Sum(nil))

		verified = hmac.Equal([]byte(expected), []byte(sig))

		_ = json.NewEncoder(w).Encode(dotenv.TelemetryResponse{Success: true})
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithBaseURL(server.URL),
		dotenv.WithTelemetrySecret(secret),
	)
	_, err := client.Telemetry.Send(context.Background(), sampleEvent())
	require.NoError(t, err)

	assert.True(t, verified, "server-recomputed HMAC should match the sent signature")
}

// Guards the roundtrip assertion itself: a signature produced with a different
// secret must NOT match the server's recomputation. Proves TestTelemetryService_Send_Signed
// would actually catch a broken signer rather than pass trivially.
func TestTelemetryService_Send_SignatureMismatchOnWrongSecret(t *testing.T) {
	var matched bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts := r.Header.Get("X-Telemetry-Timestamp")
		sig := r.Header.Get("X-Telemetry-Signature")
		body, _ := io.ReadAll(r.Body)

		// Server verifies with a DIFFERENT secret than the client signed with.
		mac := hmac.New(sha256.New, []byte("server-side-secret"))
		mac.Write([]byte(ts + "." + string(body)))
		expected := hex.EncodeToString(mac.Sum(nil))

		matched = hmac.Equal([]byte(expected), []byte(sig))

		_ = json.NewEncoder(w).Encode(dotenv.TelemetryResponse{Success: true})
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithBaseURL(server.URL),
		dotenv.WithTelemetrySecret("client-side-secret"),
	)
	_, err := client.Telemetry.Send(context.Background(), sampleEvent())
	require.NoError(t, err)

	assert.False(t, matched, "signatures from different secrets must not match")
}

func TestTelemetryService_Send_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := dotenv.NewClient(dotenv.WithBaseURL(server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Telemetry.Send(ctx, sampleEvent())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
