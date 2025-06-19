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

// List returns all projects for an organization
func (s *ProjectsService) List(ctx context.Context, organizationSlug string, opts *ListOptions) ([]*Project, *http.Response, error) {
	u := fmt.Sprintf("/api/v1/organizations/%s/projects", organizationSlug)
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
	u := fmt.Sprintf("/api/v1/projects/%s", projectSlug)

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
func (s *ProjectsService) Create(ctx context.Context, organizationSlug string, project *Project) (*Project, *http.Response, error) {
	u := fmt.Sprintf("/api/v1/organizations/%s/projects", organizationSlug)

	// Wrap in JSON:API format
	reqData := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "projects",
			"attributes": map[string]interface{}{
				"name":        project.Name,
				"slug":        project.Slug,
				"description": project.Description,
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
	u := fmt.Sprintf("/api/v1/projects/%s", projectSlug)

	// Wrap in JSON:API format
	reqData := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "projects",
			"id":   project.ID,
			"attributes": map[string]interface{}{
				"name":        project.Name,
				"description": project.Description,
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
	u := fmt.Sprintf("/api/v1/projects/%s", projectSlug)

	req, err := s.client.NewRequest(ctx, "DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
