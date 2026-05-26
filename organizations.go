package dotenv

import (
	"context"
	"fmt"
	"net/http"
)

// OrganizationsService handles organization operations
type OrganizationsService struct {
	client *Client
}

// List returns all organizations for the authenticated user
func (s *OrganizationsService) List(ctx context.Context, opts *ListOptions) ([]*Organization, *http.Response, error) {
	ctx = WithRequestResource(ctx, "organization", "")
	u := "/api/v1/organizations"
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

	// Parse JSON:API response
	orgs := make([]*Organization, 0)
	if data, ok := apiResp.Data.([]interface{}); ok {
		for _, item := range data {
			if orgData, ok := item.(map[string]interface{}); ok {
				org := &Organization{}
				if attrs, ok := orgData["attributes"].(map[string]interface{}); ok {
					mapToStruct(attrs, org)
					// Set ID from data
					if id, ok := orgData["id"].(string); ok {
						org.ID = id
					}
				}
				orgs = append(orgs, org)
			}
		}
	}

	return orgs, resp, nil
}

// Get returns a single organization
func (s *OrganizationsService) Get(ctx context.Context, slug string) (*Organization, *http.Response, error) {
	ctx = WithRequestResource(ctx, "organization", slug)
	u := fmt.Sprintf("/api/v1/organizations/%s", slug)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	org := new(Organization)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, org)
			// Set ID from data
			if id, ok := data["id"].(string); ok {
				org.ID = id
			}
		}
	}

	return org, resp, nil
}

// Create creates a new organization
func (s *OrganizationsService) Create(ctx context.Context, org *OrganizationCreateRequest) (*Organization, *http.Response, error) {
	u := "/api/v1/organizations"

	req, err := s.client.NewRequest(ctx, "POST", u, org)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	organization := new(Organization)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, organization)
			if id, ok := data["id"].(string); ok {
				organization.ID = id
			}
		}
	}

	return organization, resp, nil
}

// Update updates an organization
func (s *OrganizationsService) Update(ctx context.Context, id string, org *OrganizationUpdateRequest) (*Organization, *http.Response, error) {
	ctx = WithRequestResource(ctx, "organization", id)
	u := fmt.Sprintf("/api/v1/organizations/%s", id)

	req, err := s.client.NewRequest(ctx, "PATCH", u, org)
	if err != nil {
		return nil, nil, err
	}

	var apiResp JSONAPIResponse
	resp, err := s.client.Do(ctx, req, &apiResp)
	if err != nil {
		return nil, resp, err
	}

	organization := new(Organization)
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].(map[string]interface{}); ok {
			mapToStruct(attrs, organization)
			if id, ok := data["id"].(string); ok {
				organization.ID = id
			}
		}
	}

	return organization, resp, nil
}

// Delete deletes an organization
func (s *OrganizationsService) Delete(ctx context.Context, id string) (*http.Response, error) {
	ctx = WithRequestResource(ctx, "organization", id)
	u := fmt.Sprintf("/api/v1/organizations/%s", id)

	req, err := s.client.NewRequest(ctx, "DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
