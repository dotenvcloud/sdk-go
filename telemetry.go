package dotenv

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// TelemetryService handles telemetry operations.
type TelemetryService struct {
	client *Client
}

// TelemetryRequest is a single CLI telemetry event. It mirrors the
// `TelemetryRequest` schema in the OpenAPI spec (the contract of record): one
// flat event per request. Batching is intentionally unsupported — reintroducing
// it must be a deliberate spec change, not a silent divergence.
type TelemetryRequest struct {
	Version     string   `json:"version"`
	OS          string   `json:"os"`
	Arch        string   `json:"arch"`
	Command     string   `json:"command"`
	Duration    int64    `json:"duration"`
	Success     bool     `json:"success"`
	ErrorType   string   `json:"error_type,omitempty"`
	Features    []string `json:"features,omitempty"`
	AnonymousID string   `json:"anonymous_id"`
}

// TelemetryResponse is the server's acknowledgement.
type TelemetryResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Send submits a single telemetry event to POST /api/v1/cli/telemetry,
// HMAC-signing the request when the client was configured with a telemetry
// secret. Telemetry is best-effort; callers typically ignore the error.
func (s *TelemetryService) Send(ctx context.Context, event TelemetryRequest) (*http.Response, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	// Build via the shared request path (base URL, default headers, auth), then
	// attach the exact body bytes we sign so the server's HMAC over the raw body
	// matches byte-for-byte (NewRequest's json.Encoder would append a newline).
	req, err := s.client.NewRequest(ctx, http.MethodPost, "/api/v1/cli/telemetry", nil)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.Header.Set("Content-Type", "application/json")

	s.client.signTelemetry(req, body)

	var resp TelemetryResponse
	return s.client.Do(ctx, req, &resp)
}
