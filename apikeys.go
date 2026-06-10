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

// APIKey represents an organization API key. Fields mirror the API's flat
// ApiTokenResource (id is a string per the contract).
type APIKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	TokenPrefix string     `json:"token_prefix"` // First few characters of the token (when provided)
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

// APIKeyCreateResponse represents the response when creating or rotating an API key
type APIKeyCreateResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"` // Full token - only shown once
	*APIKey
}

// List returns all API keys for an organization
func (s *APIKeysService) List(ctx context.Context, organization string) ([]*APIKey, *http.Response, error) {
	ctx = WithRequestResource(ctx, "api_key", "")
	u := fmt.Sprintf("/api/v1/organizations/%s/api-keys", organization)

	req, err := s.client.NewRequest(ctx, "GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Data []APIKey `json:"data"`
	}
	resp, err := s.client.Do(ctx, req, &response)
	if err != nil {
		return nil, resp, err
	}

	keys := make([]*APIKey, 0, len(response.Data))
	for i := range response.Data {
		keys = append(keys, &response.Data[i])
	}

	return keys, resp, nil
}

// Create creates a new API key. The full token is only returned here, once.
func (s *APIKeysService) Create(ctx context.Context, organization string, createReq APIKeyCreateRequest) (*APIKeyCreateResponse, *http.Response, error) {
	u := fmt.Sprintf("/api/v1/organizations/%s/api-keys", organization)

	req, err := s.client.NewRequest(ctx, "POST", u, createReq)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Token struct {
			Name      string     `json:"name"`
			Token     string     `json:"token"`
			Abilities []string   `json:"abilities"`
			ExpiresAt *time.Time `json:"expires_at"`
		} `json:"token"`
		Message string `json:"message"`
	}
	resp, err := s.client.Do(ctx, req, &response)
	if err != nil {
		return nil, resp, err
	}

	result := &APIKeyCreateResponse{
		Token: response.Token.Token,
		APIKey: &APIKey{
			Name:      response.Token.Name,
			Abilities: response.Token.Abilities,
			ExpiresAt: response.Token.ExpiresAt,
		},
	}

	return result, resp, nil
}

// Update updates an existing API key's name.
func (s *APIKeysService) Update(ctx context.Context, organization, keyID string, updateReq APIKeyUpdateRequest) (*APIKey, *http.Response, error) {
	ctx = WithRequestResource(ctx, "api_key", keyID)
	u := fmt.Sprintf("/api/v1/organizations/%s/api-keys/%s", organization, keyID)

	req, err := s.client.NewRequest(ctx, "PUT", u, updateReq)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Data APIKey `json:"data"`
	}
	resp, err := s.client.Do(ctx, req, &response)
	if err != nil {
		return nil, resp, err
	}

	key := response.Data
	return &key, resp, nil
}

// Delete deletes an API key
func (s *APIKeysService) Delete(ctx context.Context, organization, keyID string) (*http.Response, error) {
	ctx = WithRequestResource(ctx, "api_key", keyID)
	u := fmt.Sprintf("/api/v1/organizations/%s/api-keys/%s", organization, keyID)

	req, err := s.client.NewRequest(ctx, "DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Rotate rotates an API key, generating a new token while keeping its config.
func (s *APIKeysService) Rotate(ctx context.Context, organization, keyID string) (*APIKeyCreateResponse, *http.Response, error) {
	u := fmt.Sprintf("/api/v1/organizations/%s/api-keys/%s/rotate", organization, keyID)

	req, err := s.client.NewRequest(ctx, "POST", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var response struct {
		Data struct {
			Token  string `json:"token"`
			APIKey APIKey `json:"api_key"`
		} `json:"data"`
	}
	resp, err := s.client.Do(ctx, req, &response)
	if err != nil {
		return nil, resp, err
	}

	key := response.Data.APIKey
	result := &APIKeyCreateResponse{
		ID:     key.ID,
		Token:  response.Data.Token,
		APIKey: &key,
	}

	return result, resp, nil
}
