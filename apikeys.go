package dotenv

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// APIKeysService handles API key operations
type APIKeysService struct {
	client *Client
}

// APIKey represents an organization API key
type APIKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	TokenPrefix string     `json:"token_prefix"` // First few characters of the token
	Abilities   []string   `json:"abilities"`    // Permissions/scopes
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// APIKeyCreateRequest represents a request to create an API key
type APIKeyCreateRequest struct {
	Name      string     `json:"name"`
	Abilities []string   `json:"abilities"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// APIKeyUpdateRequest represents a request to update an API key
type APIKeyUpdateRequest struct {
	Name string `json:"name"`
}

// APIKeyCreateResponse represents the response when creating an API key
type APIKeyCreateResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"` // Full token - only shown once
	*APIKey
}

// APITokenResource represents the JSON:API resource for API tokens
type APITokenResource struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes struct {
		Name        string     `json:"name"`
		TokenPrefix string     `json:"token_prefix"`
		Abilities   []string   `json:"abilities"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
		LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
		CreatedAt   time.Time  `json:"created_at"`
		UpdatedAt   time.Time  `json:"updated_at"`
	} `json:"attributes"`
}

// APITokenCreationResource represents the response when creating an API token
type APITokenCreationResource struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes struct {
		Name        string     `json:"name"`
		Token       string     `json:"token"` // Full token - only shown once
		TokenPrefix string     `json:"token_prefix"`
		Abilities   []string   `json:"abilities"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
		CreatedAt   time.Time  `json:"created_at"`
		UpdatedAt   time.Time  `json:"updated_at"`
	} `json:"attributes"`
}

// List returns all API keys for an organization
func (s *APIKeysService) List(ctx context.Context, organization string) ([]*APIKey, *http.Response, error) {
	ctx = WithRequestResource(ctx, "api_key", "")
	u := fmt.Sprintf("/api/v1/%s/api-keys", organization)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Data []APITokenResource `json:"data"`
	}
	resp, err := s.client.Do(ctx, req, &response)
	if err != nil {
		return nil, resp, err
	}

	// Convert to APIKey structs
	keys := make([]*APIKey, 0, len(response.Data))
	for _, resource := range response.Data {
		key := &APIKey{
			ID:          resource.ID,
			Name:        resource.Attributes.Name,
			TokenPrefix: resource.Attributes.TokenPrefix,
			Abilities:   resource.Attributes.Abilities,
			ExpiresAt:   resource.Attributes.ExpiresAt,
			LastUsedAt:  resource.Attributes.LastUsedAt,
			CreatedAt:   resource.Attributes.CreatedAt,
			UpdatedAt:   resource.Attributes.UpdatedAt,
		}
		keys = append(keys, key)
	}

	return keys, resp, nil
}

// Create creates a new API key
func (s *APIKeysService) Create(ctx context.Context, organization string, createReq APIKeyCreateRequest) (*APIKeyCreateResponse, *http.Response, error) {
	u := fmt.Sprintf("/api/v1/%s/api-keys", organization)

	req, err := s.client.NewRequest(ctx, "POST", u, createReq)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Data APITokenCreationResource `json:"data"`
	}
	resp, err := s.client.Do(ctx, req, &response)
	if err != nil {
		return nil, resp, err
	}

	// Convert to APIKeyCreateResponse
	result := &APIKeyCreateResponse{
		ID:    response.Data.ID,
		Token: response.Data.Attributes.Token,
		APIKey: &APIKey{
			ID:          response.Data.ID,
			Name:        response.Data.Attributes.Name,
			TokenPrefix: response.Data.Attributes.TokenPrefix,
			Abilities:   response.Data.Attributes.Abilities,
			ExpiresAt:   response.Data.Attributes.ExpiresAt,
			CreatedAt:   response.Data.Attributes.CreatedAt,
			UpdatedAt:   response.Data.Attributes.UpdatedAt,
		},
	}

	return result, resp, nil
}

// Update updates an existing API key
func (s *APIKeysService) Update(ctx context.Context, organization, keyID string, updateReq APIKeyUpdateRequest) (*APIKey, *http.Response, error) {
	ctx = WithRequestResource(ctx, "api_key", keyID)
	u := fmt.Sprintf("/api/v1/%s/api-keys/%s", organization, keyID)

	req, err := s.client.NewRequest(ctx, "PATCH", u, updateReq)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Data APITokenResource `json:"data"`
	}
	resp, err := s.client.Do(ctx, req, &response)
	if err != nil {
		return nil, resp, err
	}

	// Convert to APIKey
	key := &APIKey{
		ID:          response.Data.ID,
		Name:        response.Data.Attributes.Name,
		TokenPrefix: response.Data.Attributes.TokenPrefix,
		Abilities:   response.Data.Attributes.Abilities,
		ExpiresAt:   response.Data.Attributes.ExpiresAt,
		LastUsedAt:  response.Data.Attributes.LastUsedAt,
		CreatedAt:   response.Data.Attributes.CreatedAt,
		UpdatedAt:   response.Data.Attributes.UpdatedAt,
	}

	return key, resp, nil
}

// Delete deletes an API key
func (s *APIKeysService) Delete(ctx context.Context, organization, keyID string) (*http.Response, error) {
	ctx = WithRequestResource(ctx, "api_key", keyID)
	u := fmt.Sprintf("/api/v1/%s/api-keys/%s", organization, keyID)

	req, err := s.client.NewRequest(ctx, "DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Rotate rotates an API key, generating a new token
func (s *APIKeysService) Rotate(ctx context.Context, organization, keyID string) (*APIKeyCreateResponse, *http.Response, error) {
	u := fmt.Sprintf("/api/v1/%s/api-keys/%s/rotate", organization, keyID)

	req, err := s.client.NewRequest(ctx, "POST", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Data APITokenCreationResource `json:"data"`
	}
	resp, err := s.client.Do(ctx, req, &response)
	if err != nil {
		return nil, resp, err
	}

	// Convert to APIKeyCreateResponse
	result := &APIKeyCreateResponse{
		ID:    response.Data.ID,
		Token: response.Data.Attributes.Token,
		APIKey: &APIKey{
			ID:          response.Data.ID,
			Name:        response.Data.Attributes.Name,
			TokenPrefix: response.Data.Attributes.TokenPrefix,
			Abilities:   response.Data.Attributes.Abilities,
			ExpiresAt:   response.Data.Attributes.ExpiresAt,
			CreatedAt:   response.Data.Attributes.CreatedAt,
			UpdatedAt:   response.Data.Attributes.UpdatedAt,
		},
	}

	return result, resp, nil
}
