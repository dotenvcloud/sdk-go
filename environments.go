package dotenv

import (
	"context"
	"fmt"
	"net/http"
)

// EnvironmentsService handles environment operations
type EnvironmentsService struct {
	client *Client
}

// List returns all environments for a project and target
func (s *EnvironmentsService) List(ctx context.Context, projectSlug, targetSlug string, opts *ListOptions) ([]*Environment, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	if projectSlug == "" {
		return nil, nil, fmt.Errorf("project identifier cannot be empty")
	}
	if targetSlug == "" {
		return nil, nil, fmt.Errorf("target identifier cannot be empty")
	}

	ctx = WithRequestResource(ctx, "environment", "")
	u := fmt.Sprintf("/api/v1/%s/%s/%s/environments", s.client.organization, projectSlug, targetSlug)
	u = addOptions(u, opts)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	environments := make([]*Environment, 0)
	if data, ok := apiResp.Data.([]interface{}); ok {
		for _, item := range data {
			if envData, ok := item.(map[string]interface{}); ok {
				env := &Environment{}
				if attrs, ok := envData["attributes"].(map[string]interface{}); ok {
					mapToStruct(attrs, env)
					// Set ID from data
					if id, ok := envData["id"].(string); ok {
						env.ID = id
					}
				}
				environments = append(environments, env)
			}
		}
	}

	return environments, resp, nil
}

// Get returns a single environment
func (s *EnvironmentsService) Get(ctx context.Context, projectSlug, targetSlug, environmentSlug string) (*Environment, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "environment", environmentSlug)
	u := fmt.Sprintf("/api/v1/%s/%s/%s/%s", s.client.organization, projectSlug, targetSlug, environmentSlug)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	environment := new(Environment)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, environment)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				environment.ID = id
			}
		}
	}

	return environment, resp, nil
}

// Create creates a new environment
func (s *EnvironmentsService) Create(ctx context.Context, projectSlug, targetSlug string, environment *Environment) (*Environment, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/%s/%s/environments", s.client.organization, projectSlug, targetSlug)

	// Wrap in JSON:API format
	reqData := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "environments",
			"attributes": map[string]interface{}{
				"name":        environment.Name,
				"slug":        environment.Slug,
				"description": environment.Description,
			},
		},
	}

	req, err := s.client.NewRequest(ctx, "POST", u, reqData)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	e := new(Environment)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, e)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				e.ID = id
			}
		}
	}

	return e, resp, nil
}

// Update updates an existing environment
func (s *EnvironmentsService) Update(ctx context.Context, projectSlug, targetSlug, environmentSlug string, environment *Environment) (*Environment, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "environment", environmentSlug)
	u := fmt.Sprintf("/api/v1/%s/%s/%s/%s", s.client.organization, projectSlug, targetSlug, environmentSlug)

	// Wrap in JSON:API format
	reqData := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "environments",
			"id":   environment.ID,
			"attributes": map[string]interface{}{
				"name":        environment.Name,
				"description": environment.Description,
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

	e := new(Environment)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, e)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				e.ID = id
			}
		}
	}

	return e, resp, nil
}

// Delete deletes an environment
func (s *EnvironmentsService) Delete(ctx context.Context, projectSlug, targetSlug, environmentSlug string) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "environment", environmentSlug)
	u := fmt.Sprintf("/api/v1/%s/%s/%s/%s", s.client.organization, projectSlug, targetSlug, environmentSlug)

	req, err := s.client.NewRequest(ctx, "DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
