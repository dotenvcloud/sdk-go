package dotenv

import (
	"context"
	"fmt"
	"net/http"
)

// SecretsService handles secret operations
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

// PushSecrets creates or updates multiple secrets
func (s *SecretsService) PushSecrets(ctx context.Context, project string, secrets map[string]interface{}) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/%s/secrets/push", s.client.organization, project)

	pushReq := PushSecretsRequest{
		Secrets: secrets,
	}

	req, err := s.client.NewRequest(ctx, "POST", u, pushReq)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// List returns all secrets for a project
func (s *SecretsService) List(ctx context.Context, projectSlug string, opts *SecretOptions) ([]*Secret, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "secret", "")
	u := fmt.Sprintf("/api/v1/%s/%s/secrets", s.client.organization, projectSlug)

	// Add query parameters
	if opts != nil {
		query := "?"
		if opts.Target != "" {
			query += fmt.Sprintf("target=%s&", opts.Target)
		}
		if opts.Environment != "" {
			query += fmt.Sprintf("environment=%s&", opts.Environment)
		}
		if opts.IncludeDecrypted {
			query += "decrypt=true&"
		}
		if opts.ResolveHierarchy {
			query += "resolve=true&"
		}
		if len(query) > 1 {
			u += query[:len(query)-1] // Remove trailing & or ?
		}
	}

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	secrets := make([]*Secret, 0)
	if data, ok := apiResp.Data.([]interface{}); ok {
		for _, item := range data {
			if secretData, ok := item.(map[string]interface{}); ok {
				secret := &Secret{}
				if attrs, ok := secretData["attributes"].(map[string]interface{}); ok {
					mapToStruct(attrs, secret)
					// Set ID from data
					if id, ok := secretData["id"].(string); ok {
						secret.ID = id
					}
				}
				secrets = append(secrets, secret)
			}
		}
	}

	return secrets, resp, nil
}

// Get returns a single secret
func (s *SecretsService) Get(ctx context.Context, projectSlug, secretKey string) (*Secret, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "secret", secretKey)
	u := fmt.Sprintf("/api/v1/%s/%s/secrets/%s", s.client.organization, projectSlug, secretKey)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	secret := new(Secret)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, secret)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				secret.ID = id
			}
		}
	}

	return secret, resp, nil
}

// Create creates a new secret using the push endpoint
// NOTE: Individual secret creation is not supported at the organization level
// Secrets must be created within a project context using PushSecrets
func (s *SecretsService) Create(ctx context.Context, req *CreateSecretRequest) (*Secret, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	if req.ProjectSlug == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"project": "project slug is required for secret creation"}}
	}

	// Use the push endpoint for creating individual secrets
	secrets := map[string]interface{}{
		req.Key: req.Value,
	}

	// Additional metadata if needed
	if req.TargetSlug != nil || req.EnvironmentSlug != nil {
		secrets = map[string]interface{}{
			req.Key: map[string]interface{}{
				"value":            req.Value,
				"target_slug":      req.TargetSlug,
				"environment_slug": req.EnvironmentSlug,
				"is_encrypted":     req.IsEncrypted,
			},
		}
	}

	resp, err := s.PushSecrets(ctx, req.ProjectSlug, secrets)
	if err != nil {
		return nil, resp, err
	}

	// Return a simple response since push doesn't return individual secret details
	// Note: We're returning slugs in the ID fields for compatibility
	secret := &Secret{
		Key:         req.Key,
		Value:       req.Value,
		IsEncrypted: req.IsEncrypted,
	}

	return secret, resp, nil
}

// Update updates an existing secret
func (s *SecretsService) Update(ctx context.Context, projectSlug, secretKey string, value string, isEncrypted bool) (*Secret, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "secret", secretKey)
	u := fmt.Sprintf("/api/v1/%s/%s/secrets/%s", s.client.organization, projectSlug, secretKey)

	// Wrap in JSON:API format
	reqData := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "secrets",
			"attributes": map[string]interface{}{
				"value":        value,
				"is_encrypted": isEncrypted,
			},
		},
	}

	req, err := s.client.NewRequest(ctx, "PATCH", u, reqData)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	secret := new(Secret)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, secret)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				secret.ID = id
			}
		}
	}

	return secret, resp, nil
}

// Delete deletes a secret
func (s *SecretsService) Delete(ctx context.Context, projectSlug, secretKey string) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "secret", secretKey)
	u := fmt.Sprintf("/api/v1/%s/%s/secrets/%s", s.client.organization, projectSlug, secretKey)

	req, err := s.client.NewRequest(ctx, "DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// BulkCreate creates multiple secrets using the push endpoint
// NOTE: Bulk secret creation is done through the project-scoped push endpoint
func (s *SecretsService) BulkCreate(ctx context.Context, req *BulkSecretsRequest) ([]*Secret, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	if req.ProjectSlug == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"project": "project slug is required for bulk secret creation"}}
	}

	// Convert bulk request to push format
	secretsMap := make(map[string]interface{})
	for _, secret := range req.Secrets {
		if secret.TargetSlug != nil || secret.EnvironmentSlug != nil {
			secretsMap[secret.Key] = map[string]interface{}{
				"value":            secret.Value,
				"target_slug":      secret.TargetSlug,
				"environment_slug": secret.EnvironmentSlug,
				"is_encrypted":     secret.IsEncrypted,
			}
		} else {
			secretsMap[secret.Key] = secret.Value
		}
	}

	resp, err := s.PushSecrets(ctx, req.ProjectSlug, secretsMap)
	if err != nil {
		return nil, resp, err
	}

	// Return the secrets as provided since push doesn't return individual details
	secrets := make([]*Secret, 0, len(req.Secrets))
	for _, s := range req.Secrets {
		secret := &Secret{
			Key:         s.Key,
			Value:       s.Value,
			IsEncrypted: s.IsEncrypted,
		}
		secrets = append(secrets, secret)
	}

	return secrets, resp, nil
}
