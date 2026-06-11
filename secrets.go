package dotenv

import (
	"context"
	"fmt"
	"net/http"
)

// SecretsService handles secret operations.
//
// Secrets are stored as one encrypted .env blob per level (project, target or
// environment) — mirroring the web app (the source of truth). Reads return the
// per-level blob; writes upsert/clear the per-level blob.
type SecretsService struct {
	client *Client
}

// GetProjectSecrets retrieves secrets for a project with optional target and environment
func (s *SecretsService) GetProjectSecrets(ctx context.Context, project, target, environment string) (*SecretsHierarchyResponse, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}

	// Build URL with path segments, NOT query parameters
	// Using the SHORT format as confirmed by curl tests
	var u string
	if environment != "" && target != "" {
		u = fmt.Sprintf("/api/v1/%s/%s/%s/%s/secrets", s.client.organization, project, target, environment)
	} else if target != "" {
		u = fmt.Sprintf("/api/v1/%s/%s/%s/secrets", s.client.organization, project, target)
	} else {
		u = fmt.Sprintf("/api/v1/%s/%s/secrets", s.client.organization, project)
	}

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var apiResp SecretsHierarchyResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	return &apiResp, resp, nil
}

// RetrieveSecrets fetches secrets with complex queries
func (s *SecretsService) RetrieveSecrets(ctx context.Context, params RetrieveParams) (map[string]string, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/secrets/retrieve", s.client.organization)

	// Default action to 'read' if not specified
	if params.Action == "" {
		params.Action = "read"
	}

	req, err := s.client.NewRequest(ctx, "POST", u, params)
	if err != nil {
		return nil, nil, err
	}

	// If raw is true, API returns simple key-value object
	if params.Raw {
		var result map[string]string
		resp, err := s.client.Do(ctx, req, &result)
		if err != nil {
			return nil, resp, err
		}
		return result, resp, nil
	}

	// Otherwise, API returns structured response
	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	// Parse secrets from structured response
	secrets := make(map[string]string)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		// Handle grouped or merged response structure
		if secretsData, ok := data["secrets"].(map[string]interface{}); ok {
			for k, v := range secretsData {
				if str, ok := v.(string); ok {
					secrets[k] = str
				}
			}
		}
	}

	return secrets, resp, nil
}

// StoreSecrets upserts the encrypted .env blob for a level (the deepest of
// project/target/environment provided; target/environment may be empty).
// content must already be encrypted — it is the inverse of the per-level
// content returned by GetProjectSecrets. keyProof is the base64 PBKDF2 proof
// for client-managed projects (empty for server-managed); the server rejects a
// mismatch so a wrong key cannot silently corrupt secrets.
func (s *SecretsService) StoreSecrets(ctx context.Context, project, target, environment, content, keyProof string) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = withDeepestResource(ctx, project, target, environment)
	u := fmt.Sprintf("/api/v1/%s/secrets/store", s.client.organization)

	body := StoreSecretsRequest{
		Project:     project,
		Target:      target,
		Environment: environment,
		Content:     content,
		KeyProof:    keyProof,
	}

	req, err := s.client.NewRequest(ctx, "POST", u, body)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// withDeepestResource tags the request with the deepest hierarchy level
// (environment > target > project). On a 404 the error then names the most
// likely missing resource — e.g. "environment 'production2' not found" — rather
// than a generic "secret not found", since these endpoints fail when the
// project/target/environment path can't be resolved.
func withDeepestResource(ctx context.Context, project, target, environment string) context.Context {
	switch {
	case environment != "":
		return WithRequestResource(ctx, "environment", environment)
	case target != "":
		return WithRequestResource(ctx, "target", target)
	default:
		return WithRequestResource(ctx, "project", project)
	}
}

// StoreSecretsWithOptions is StoreSecrets with a noBackup flag that skips writing
// a backup version for this write.
func (s *SecretsService) StoreSecretsWithOptions(ctx context.Context, project, target, environment, content, keyProof string, noBackup bool) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = withDeepestResource(ctx, project, target, environment)
	u := fmt.Sprintf("/api/v1/%s/secrets/store", s.client.organization)

	body := StoreSecretsRequest{
		Project:     project,
		Target:      target,
		Environment: environment,
		Content:     content,
		KeyProof:    keyProof,
		NoBackup:    noBackup,
	}

	req, err := s.client.NewRequest(ctx, "POST", u, body)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// DeleteSecretLevel clears (deletes) the secrets blob for the deepest provided level.
func (s *SecretsService) DeleteSecretLevel(ctx context.Context, project, target, environment string) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = withDeepestResource(ctx, project, target, environment)
	u := fmt.Sprintf("/api/v1/%s/secrets/delete", s.client.organization)

	body := DeleteSecretsRequest{
		Project:     project,
		Target:      target,
		Environment: environment,
	}

	req, err := s.client.NewRequest(ctx, "POST", u, body)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// DeleteSecretLevelWithOptions is DeleteSecretLevel with a noBackup flag. When
// noBackup is true the level's history is purged and the row hard-deleted, so
// confirmed is sent as required by the server.
func (s *SecretsService) DeleteSecretLevelWithOptions(ctx context.Context, project, target, environment string, noBackup bool) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = withDeepestResource(ctx, project, target, environment)
	u := fmt.Sprintf("/api/v1/%s/secrets/delete", s.client.organization)

	body := DeleteSecretsRequest{
		Project:     project,
		Target:      target,
		Environment: environment,
		NoBackup:    noBackup,
		Confirmed:   noBackup,
	}

	req, err := s.client.NewRequest(ctx, "POST", u, body)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
