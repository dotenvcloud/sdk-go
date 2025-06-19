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
func (s *SecretsService) GetProjectSecrets(ctx context.Context, project, target, environment string) (map[string]string, *http.Response, error) {
	u := fmt.Sprintf("/api/v1/%s/secrets", project)

	// Add query parameters
	if target != "" || environment != "" {
		u = fmt.Sprintf("%s?", u)
		if target != "" {
			u = fmt.Sprintf("%starget=%s", u, target)
			if environment != "" {
				u = fmt.Sprintf("%s&environment=%s", u, environment)
			}
		} else if environment != "" {
			u = fmt.Sprintf("%senvironment=%s", u, environment)
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

	// Parse secrets from response
	secrets := make(map[string]string)
	if data, ok := apiResp.Data.([]interface{}); ok {
		for _, item := range data {
			if secretData, ok := item.(map[string]interface{}); ok {
				if attrs, ok := secretData["attributes"].(map[string]interface{}); ok {
					if key, ok := attrs["key"].(string); ok {
						if value, ok := attrs["value"].(string); ok {
							secrets[key] = value
						}
					}
				}
			}
		}
	}

	return secrets, resp, nil
}

// RetrieveSecrets fetches secrets with complex queries
func (s *SecretsService) RetrieveSecrets(ctx context.Context, params RetrieveParams) (map[string]string, *http.Response, error) {
	u := "/api/v1/secrets/retrieve"

	req, err := s.client.NewRequest(ctx, "POST", u, params)
	if err != nil {
		return nil, nil, err
	}

	var result map[string]string
	resp, err := s.client.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	return result, resp, nil
}

// PushSecrets creates or updates multiple secrets
func (s *SecretsService) PushSecrets(ctx context.Context, project string, secrets map[string]interface{}) (*http.Response, error) {
	u := fmt.Sprintf("/api/v1/%s/secrets/push", project)

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
	u := fmt.Sprintf("/api/v1/projects/%s/secrets", projectSlug)

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
	u := fmt.Sprintf("/api/v1/projects/%s/secrets/%s", projectSlug, secretKey)

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

// Create creates a new secret
func (s *SecretsService) Create(ctx context.Context, req *CreateSecretRequest) (*Secret, *http.Response, error) {
	u := "/api/v1/secrets"

	// Wrap in JSON:API format
	reqData := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "secrets",
			"attributes": map[string]interface{}{
				"project_id":     req.ProjectID,
				"target_id":      req.TargetID,
				"environment_id": req.EnvironmentID,
				"key":            req.Key,
				"value":          req.Value,
				"is_encrypted":   req.IsEncrypted,
			},
		},
	}

	httpReq, err := s.client.NewRequest(ctx, "POST", u, reqData)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, httpReq, &apiResp)
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

// Update updates an existing secret
func (s *SecretsService) Update(ctx context.Context, projectSlug, secretKey string, value string, isEncrypted bool) (*Secret, *http.Response, error) {
	u := fmt.Sprintf("/api/v1/projects/%s/secrets/%s", projectSlug, secretKey)

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
	u := fmt.Sprintf("/api/v1/projects/%s/secrets/%s", projectSlug, secretKey)

	req, err := s.client.NewRequest(ctx, "DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// BulkCreate creates multiple secrets
func (s *SecretsService) BulkCreate(ctx context.Context, req *BulkSecretsRequest) ([]*Secret, *http.Response, error) {
	u := "/api/v1/secrets/bulk"

	httpReq, err := s.client.NewRequest(ctx, "POST", u, req)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, httpReq, &apiResp)
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
