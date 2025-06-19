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
