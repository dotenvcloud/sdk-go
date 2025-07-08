package dotenv

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// TelemetryService handles telemetry operations
type TelemetryService struct {
	client *Client
}

// TelemetryEvent represents a single telemetry event
type TelemetryEvent struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Context    TelemetryContext       `json:"context"`
	Timestamp  time.Time              `json:"timestamp"`
}

// TelemetryContext contains context information for telemetry
type TelemetryContext struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Version     string `json:"version"`
	CI          bool   `json:"ci"`
	SessionID   string `json:"session_id"`
	AnalyticsID string `json:"analytics_id,omitempty"`
}

// TelemetryBatchRequest represents a batch of telemetry events
type TelemetryBatchRequest struct {
	Events []TelemetryEvent `json:"events"`
}

// TelemetryResponse represents the response from telemetry submission
type TelemetryResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// SendEvent sends a single telemetry event
func (s *TelemetryService) SendEvent(ctx context.Context, event TelemetryEvent) (*http.Response, error) {
	return s.SendBatch(ctx, []TelemetryEvent{event})
}

// SendBatch sends a batch of telemetry events
func (s *TelemetryService) SendBatch(ctx context.Context, events []TelemetryEvent) (*http.Response, error) {
	u := "/api/v1/cli/telemetry"

	req, err := s.client.NewRequest(ctx, "POST", u, TelemetryBatchRequest{Events: events})
	if err != nil {
		return nil, err
	}

	// Add CLI version header if available
	if version := ctx.Value("cli-version"); version != nil {
		req.Header.Set("X-CLI-Version", fmt.Sprintf("%v", version))
	}

	var telemetryResp TelemetryResponse
	resp, err := s.client.Do(ctx, req, &telemetryResp)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// TrackCommand tracks a CLI command execution
func (s *TelemetryService) TrackCommand(ctx context.Context, command string, duration time.Duration, success bool, properties map[string]interface{}) (*http.Response, error) {
	if properties == nil {
		properties = make(map[string]interface{})
	}
	
	properties["command"] = command
	properties["duration"] = duration.Milliseconds()
	properties["success"] = success

	event := TelemetryEvent{
		Name:       "cli.command",
		Properties: properties,
		Timestamp:  time.Now(),
	}

	return s.SendEvent(ctx, event)
}

// TrackError tracks a CLI error
func (s *TelemetryService) TrackError(ctx context.Context, command string, errorType string, properties map[string]interface{}) (*http.Response, error) {
	if properties == nil {
		properties = make(map[string]interface{})
	}
	
	properties["command"] = command
	properties["error_type"] = errorType

	event := TelemetryEvent{
		Name:       "cli.error",
		Properties: properties,
		Timestamp:  time.Now(),
	}

	return s.SendEvent(ctx, event)
}

// TrackFeatureUsage tracks feature usage
func (s *TelemetryService) TrackFeatureUsage(ctx context.Context, feature string, properties map[string]interface{}) (*http.Response, error) {
	if properties == nil {
		properties = make(map[string]interface{})
	}
	
	properties["feature"] = feature

	event := TelemetryEvent{
		Name:       "cli.feature_usage",
		Properties: properties,
		Timestamp:  time.Now(),
	}

	return s.SendEvent(ctx, event)
}