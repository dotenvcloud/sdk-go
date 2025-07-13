package dotenv

import (
	"context"
	"fmt"
	"net/http"
)

// TargetsService handles target operations
type TargetsService struct {
	client *Client
}

// List returns all targets for a project
func (s *TargetsService) List(ctx context.Context, projectSlug string, opts *ListOptions) ([]*Target, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	if projectSlug == "" {
		return nil, nil, fmt.Errorf("project identifier cannot be empty")
	}

	u := fmt.Sprintf("/api/v1/%s/%s/targets", s.client.organization, projectSlug)
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

	targets := make([]*Target, 0)
	if data, ok := apiResp.Data.([]interface{}); ok {
		for _, item := range data {
			if targetData, ok := item.(map[string]interface{}); ok {
				target := &Target{}
				if attrs, ok := targetData["attributes"].(map[string]interface{}); ok {
					mapToStruct(attrs, target)
					// Set ID from data
					if id, ok := targetData["id"].(string); ok {
						target.ID = id
					}
				}
				targets = append(targets, target)
			}
		}
	}

	return targets, resp, nil
}

// Get returns a single target
func (s *TargetsService) Get(ctx context.Context, projectSlug, targetSlug string) (*Target, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/%s/%s", s.client.organization, projectSlug, targetSlug)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	target := new(Target)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, target)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				target.ID = id
			}
		}
	}

	return target, resp, nil
}

// Create creates a new target
func (s *TargetsService) Create(ctx context.Context, projectSlug string, target *Target) (*Target, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/%s/targets", s.client.organization, projectSlug)

	// Wrap in JSON:API format
	reqData := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "targets",
			"attributes": map[string]interface{}{
				"name":        target.Name,
				"slug":        target.Slug,
				"description": target.Description,
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

	t := new(Target)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, t)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				t.ID = id
			}
		}
	}

	return t, resp, nil
}

// Update updates an existing target
func (s *TargetsService) Update(ctx context.Context, projectSlug, targetSlug string, target *Target) (*Target, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/%s/%s", s.client.organization, projectSlug, targetSlug)

	// Wrap in JSON:API format
	reqData := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "targets",
			"id":   target.ID,
			"attributes": map[string]interface{}{
				"name":        target.Name,
				"description": target.Description,
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

	t := new(Target)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, t)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				t.ID = id
			}
		}
	}

	return t, resp, nil
}

// Delete deletes a target
func (s *TargetsService) Delete(ctx context.Context, projectSlug, targetSlug string) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/%s/%s", s.client.organization, projectSlug, targetSlug)

	req, err := s.client.NewRequest(ctx, "DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
