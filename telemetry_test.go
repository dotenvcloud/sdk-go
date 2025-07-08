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

func TestTelemetryService_SendEvent(t *testing.T) {
	tests := []struct {
		name           string
		event          dotenv.TelemetryEvent
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
		validateReq    func(t *testing.T, req dotenv.TelemetryBatchRequest)
	}{
		{
			name: "successful track basic event",
			event: dotenv.TelemetryEvent{
				Name:       "cli.command",
				Timestamp:  time.Now().UTC().Round(time.Millisecond),
				Properties: map[string]interface{}{},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful track with properties",
			event: dotenv.TelemetryEvent{
				Name:      "cli.command",
				Timestamp: time.Now().UTC().Round(time.Millisecond),
				Properties: map[string]interface{}{
					"command":       "pull",
					"project":       "myproject",
					"duration_ms":   1234,
					"success":       true,
					"version":       "1.0.0",
					"os":            "darwin",
					"architecture":  "arm64",
					"error_code":    nil,
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "successful track error event",
			event: dotenv.TelemetryEvent{
				Name:      "cli.error",
				Timestamp: time.Now().UTC().Round(time.Millisecond),
				Properties: map[string]interface{}{
					"command":     "push",
					"error_type":  "authentication",
					"error_code":  "AUTH_FAILED",
					"duration_ms": 500,
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "track with nested properties",
			event: dotenv.TelemetryEvent{
				Name:      "cli.feature_usage",
				Timestamp: time.Now().UTC().Round(time.Millisecond),
				Properties: map[string]interface{}{
					"feature": "encryption",
					"details": map[string]interface{}{
						"algorithm": "AES-256-GCM",
						"key_type":  "client",
						"success":   true,
					},
					"metrics": map[string]interface{}{
						"secrets_encrypted": 10,
						"time_ms":          250,
					},
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			validateReq: func(t *testing.T, req dotenv.TelemetryBatchRequest) {
				// Verify nested properties
				require.Len(t, req.Events, 1)
				details, ok := req.Events[0].Properties["details"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "AES-256-GCM", details["algorithm"])
				
				metrics, ok := req.Events[0].Properties["metrics"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, float64(10), metrics["secrets_encrypted"])
			},
		},
		{
			name: "invalid event name",
			event: dotenv.TelemetryEvent{
				Name:       "", // Empty event name
				Timestamp:  time.Now().UTC().Round(time.Millisecond),
				Properties: map[string]interface{}{},
			},
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "400")
			},
		},
		{
			name: "server error",
			event: dotenv.TelemetryEvent{
				Name:       "cli.command",
				Timestamp:  time.Now().UTC().Round(time.Millisecond),
				Properties: map[string]interface{}{},
			},
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "500")
			},
		},
		{
			name: "rate limit exceeded",
			event: dotenv.TelemetryEvent{
				Name:       "cli.command",
				Timestamp:  time.Now().UTC().Round(time.Millisecond),
				Properties: map[string]interface{}{},
			},
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

				// Verify request body
				var reqBody dotenv.TelemetryBatchRequest
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				
				require.Len(t, reqBody.Events, 1)
				assert.Equal(t, tt.event.Name, reqBody.Events[0].Name)
				assert.WithinDuration(t, tt.event.Timestamp, reqBody.Events[0].Timestamp, time.Second)
				
				// Validate custom request validation if provided
				if tt.validateReq != nil {
					tt.validateReq(t, reqBody)
				}

				// Send response
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockStatusCode >= 400 {
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
				} else {
					response := dotenv.TelemetryResponse{
						Success: true,
					}
					json.NewEncoder(w).Encode(response)
				}
			}))
			defer server.Close()

			client := dotenv.NewClient(
				dotenv.WithAPIKey("test-key"),
				dotenv.WithBaseURL(server.URL),
			)

			resp, err := client.Telemetry.SendEvent(context.Background(), tt.event)

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

func TestTelemetryService_SendBatch(t *testing.T) {
	// Test sending multiple events in a batch
	var capturedBatch dotenv.TelemetryBatchRequest
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&capturedBatch)
		require.NoError(t, err)
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dotenv.TelemetryResponse{Success: true})
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
	)

	// Create multiple events
	events := []dotenv.TelemetryEvent{
		{
			Name:       "cli.started",
			Timestamp:  time.Now().UTC(),
			Properties: map[string]interface{}{"version": "1.0.0"},
		},
		{
			Name:       "cli.command",
			Timestamp:  time.Now().UTC(),
			Properties: map[string]interface{}{"command": "pull"},
		},
		{
			Name:       "cli.completed",
			Timestamp:  time.Now().UTC(),
			Properties: map[string]interface{}{"duration_ms": 1500},
		},
	}

	resp, err := client.Telemetry.SendBatch(context.Background(), events)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify all events were captured
	assert.Equal(t, len(events), len(capturedBatch.Events))
	for i, event := range events {
		assert.Equal(t, event.Name, capturedBatch.Events[i].Name)
	}
}

func TestTelemetryService_TrackCommand(t *testing.T) {
	var capturedEvent dotenv.TelemetryEvent

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var batch dotenv.TelemetryBatchRequest
		err := json.NewDecoder(r.Body).Decode(&batch)
		require.NoError(t, err)
		require.Len(t, batch.Events, 1)
		
		capturedEvent = batch.Events[0]
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dotenv.TelemetryResponse{Success: true})
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
	)

	// Track a command
	props := map[string]interface{}{
		"project": "test-project",
		"target":  "production",
	}
	
	resp, err := client.Telemetry.TrackCommand(
		context.Background(),
		"pull",
		1500*time.Millisecond,
		true,
		props,
	)
	
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify captured event
	assert.Equal(t, "cli.command", capturedEvent.Name)
	assert.Equal(t, "pull", capturedEvent.Properties["command"])
	assert.Equal(t, float64(1500), capturedEvent.Properties["duration"])
	assert.Equal(t, true, capturedEvent.Properties["success"])
	assert.Equal(t, "test-project", capturedEvent.Properties["project"])
	assert.Equal(t, "production", capturedEvent.Properties["target"])
}

func TestTelemetryService_TrackError(t *testing.T) {
	var capturedEvent dotenv.TelemetryEvent

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var batch dotenv.TelemetryBatchRequest
		err := json.NewDecoder(r.Body).Decode(&batch)
		require.NoError(t, err)
		require.Len(t, batch.Events, 1)
		
		capturedEvent = batch.Events[0]
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dotenv.TelemetryResponse{Success: true})
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
	)

	// Track an error
	props := map[string]interface{}{
		"project":    "test-project",
		"error_code": "AUTH_001",
	}
	
	resp, err := client.Telemetry.TrackError(
		context.Background(),
		"push",
		"authentication_failed",
		props,
	)
	
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify captured event
	assert.Equal(t, "cli.error", capturedEvent.Name)
	assert.Equal(t, "push", capturedEvent.Properties["command"])
	assert.Equal(t, "authentication_failed", capturedEvent.Properties["error_type"])
	assert.Equal(t, "test-project", capturedEvent.Properties["project"])
	assert.Equal(t, "AUTH_001", capturedEvent.Properties["error_code"])
}

func TestTelemetryService_TrackFeatureUsage(t *testing.T) {
	var capturedEvent dotenv.TelemetryEvent

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var batch dotenv.TelemetryBatchRequest
		err := json.NewDecoder(r.Body).Decode(&batch)
		require.NoError(t, err)
		require.Len(t, batch.Events, 1)
		
		capturedEvent = batch.Events[0]
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dotenv.TelemetryResponse{Success: true})
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
	)

	// Track feature usage
	props := map[string]interface{}{
		"encryption_type": "client-side",
		"key_source":      "user-provided",
	}
	
	resp, err := client.Telemetry.TrackFeatureUsage(
		context.Background(),
		"encryption",
		props,
	)
	
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify captured event
	assert.Equal(t, "cli.feature_usage", capturedEvent.Name)
	assert.Equal(t, "encryption", capturedEvent.Properties["feature"])
	assert.Equal(t, "client-side", capturedEvent.Properties["encryption_type"])
	assert.Equal(t, "user-provided", capturedEvent.Properties["key_source"])
}

func TestTelemetryService_ContextCancellation(t *testing.T) {
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

	event := dotenv.TelemetryEvent{
		Name:       "cli.test.cancelled",
		Timestamp:  time.Now().UTC(),
		Properties: map[string]interface{}{},
	}

	_, err := client.Telemetry.SendEvent(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestTelemetryService_SpecialCharactersInProperties(t *testing.T) {
	// Test properties with special characters and various data types
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var batch dotenv.TelemetryBatchRequest
		err := json.NewDecoder(r.Body).Decode(&batch)
		require.NoError(t, err)
		require.Len(t, batch.Events, 1)
		
		event := batch.Events[0]
		
		// Verify special characters are preserved
		assert.Equal(t, "value with \"quotes\" and 'apostrophes'", event.Properties["special_string"])
		assert.Equal(t, "path/with/slashes\\and\\backslashes", event.Properties["path"])
		assert.Equal(t, "unicode: 你好世界 🌍", event.Properties["unicode"])
		
		// Verify numeric types
		assert.Equal(t, float64(42), event.Properties["integer"])
		assert.Equal(t, 3.14159, event.Properties["float"])
		assert.Equal(t, true, event.Properties["boolean"])
		assert.Nil(t, event.Properties["null_value"])
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dotenv.TelemetryResponse{Success: true})
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
	)

	event := dotenv.TelemetryEvent{
		Name:      "cli.test.special_chars",
		Timestamp: time.Now().UTC(),
		Properties: map[string]interface{}{
			"special_string": "value with \"quotes\" and 'apostrophes'",
			"path":          "path/with/slashes\\and\\backslashes",
			"unicode":       "unicode: 你好世界 🌍",
			"integer":       42,
			"float":         3.14159,
			"boolean":       true,
			"null_value":    nil,
			"array":         []string{"one", "two", "three"},
		},
	}

	resp, err := client.Telemetry.SendEvent(context.Background(), event)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestTelemetryService_CLIVersionHeader(t *testing.T) {
	var capturedVersion string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedVersion = r.Header.Get("X-CLI-Version")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(dotenv.TelemetryResponse{Success: true})
	}))
	defer server.Close()

	client := dotenv.NewClient(
		dotenv.WithAPIKey("test-key"),
		dotenv.WithBaseURL(server.URL),
	)

	// Create context with CLI version
	ctx := context.WithValue(context.Background(), "cli-version", "1.2.3")

	event := dotenv.TelemetryEvent{
		Name:       "cli.test",
		Timestamp:  time.Now().UTC(),
		Properties: map[string]interface{}{},
	}

	_, err := client.Telemetry.SendEvent(ctx, event)
	require.NoError(t, err)
	
	assert.Equal(t, "1.2.3", capturedVersion)
}