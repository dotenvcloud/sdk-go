package dotenv

import (
	"context"
	"fmt"
	"net/http"
)

// ProjectsService handles project operations
type ProjectsService struct {
	client *Client
}

// List returns all projects for the organization
func (s *ProjectsService) List(ctx context.Context, opts *ListOptions) ([]*Project, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}

	ctx = WithRequestResource(ctx, "project", "")
	u := fmt.Sprintf("/api/v1/%s/projects", s.client.organization)
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

	projects := make([]*Project, 0)
	if data, ok := apiResp.Data.([]interface{}); ok {
		for _, item := range data {
			if projData, ok := item.(map[string]interface{}); ok {
				proj := &Project{}
				if attrs, ok := projData["attributes"].(map[string]interface{}); ok {
					mapToStruct(attrs, proj)
					// Set ID from data
					if id, ok := projData["id"].(string); ok {
						proj.ID = id
					}
				}
				projects = append(projects, proj)
			}
		}
	}

	return projects, resp, nil
}

// Get returns a single project
func (s *ProjectsService) Get(ctx context.Context, projectSlug string) (*Project, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "project", projectSlug)
	u := fmt.Sprintf("/api/v1/%s/%s", s.client.organization, projectSlug)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	project := new(Project)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, project)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				project.ID = id
			}
		}
	}

	return project, resp, nil
}

// Create creates a new project
func (s *ProjectsService) Create(ctx context.Context, project *Project) (*Project, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	u := fmt.Sprintf("/api/v1/%s/projects", s.client.organization)

	// Flat body, matching the API's StoreProjectApiRequest.
	reqData := map[string]interface{}{
		"name": project.Name,
	}
	if project.Description != "" {
		reqData["description"] = project.Description
	}
	if project.SecretFormat != "" {
		reqData["secret_format"] = project.SecretFormat
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

	p := new(Project)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, p)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				p.ID = id
			}
		}
	}

	return p, resp, nil
}

// Update updates an existing project
func (s *ProjectsService) Update(ctx context.Context, projectSlug string, project *Project) (*Project, *http.Response, error) {
	if s.client.organization == "" {
		return nil, nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "project", projectSlug)
	u := fmt.Sprintf("/api/v1/%s/%s", s.client.organization, projectSlug)

	// Flat body, matching the API's UpdateProjectApiRequest. Only populated
	// fields are sent so partial (PATCH) updates work.
	reqData := map[string]interface{}{}
	if project.Name != "" {
		reqData["name"] = project.Name
	}
	if project.Description != "" {
		reqData["description"] = project.Description
	}
	if project.Slug != "" {
		reqData["slug"] = project.Slug
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

	p := new(Project)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, p)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				p.ID = id
			}
		}
	}

	return p, resp, nil
}

// Delete deletes a project
func (s *ProjectsService) Delete(ctx context.Context, projectSlug string) (*http.Response, error) {
	if s.client.organization == "" {
		return nil, &ErrValidation{Errors: map[string]string{"organization": "organization context is required"}}
	}
	ctx = WithRequestResource(ctx, "project", projectSlug)
	u := fmt.Sprintf("/api/v1/%s/%s", s.client.organization, projectSlug)

	req, err := s.client.NewRequest(ctx, "DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
