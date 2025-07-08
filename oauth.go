package dotenv

import (
	"context"
	"fmt"
	"net/http"
)

// OAuthService handles OAuth token operations
type OAuthService struct {
	client *Client
}

// OAuthTokenAuthCodeRequest represents the authorization code exchange request
type OAuthTokenAuthCodeRequest struct {
	GrantType    string `json:"grant_type"`    // "authorization_code"
	Code         string `json:"code"`          // Authorization code
	CodeVerifier string `json:"code_verifier"` // PKCE code verifier
	ClientID     string `json:"client_id"`     // OAuth client ID
}

// OAuthTokenRefreshRequest represents the token refresh request
type OAuthTokenRefreshRequest struct {
	GrantType    string `json:"grant_type"`    // "refresh_token"
	RefreshToken string `json:"refresh_token"` // Refresh token
	ClientID     string `json:"client_id"`     // OAuth client ID
}

// OAuthTokenResponse represents the OAuth token response
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OAuthErrorResponse represents an OAuth error response
type OAuthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// ExchangeToken exchanges an authorization code for access token
func (s *OAuthService) ExchangeToken(ctx context.Context, req OAuthTokenAuthCodeRequest) (*OAuthTokenResponse, *http.Response, error) {
	u := "/api/v1/oauth/token"

	// Set grant type
	req.GrantType = "authorization_code"

	httpReq, err := s.client.NewRequest(ctx, "POST", u, req)
	if err != nil {
		return nil, nil, err
	}

	// OAuth token endpoint doesn't require authentication
	httpReq.Header.Del("Authorization")

	var tokenResp OAuthTokenResponse
	resp, err := s.client.Do(ctx, httpReq, &tokenResp)
	if err != nil {
		// Check if it's an OAuth error
		if resp != nil && resp.StatusCode >= 400 {
			var oauthErr OAuthErrorResponse
			if jsonErr := parseJSONResponse(resp, &oauthErr); jsonErr == nil && oauthErr.Error != "" {
				return nil, resp, fmt.Errorf("oauth error: %s - %s", oauthErr.Error, oauthErr.ErrorDescription)
			}
		}
		return nil, resp, err
	}

	return &tokenResp, resp, nil
}

// RefreshToken uses a refresh token to get a new access token
func (s *OAuthService) RefreshToken(ctx context.Context, refreshToken string, clientID string) (*OAuthTokenResponse, *http.Response, error) {
	u := "/api/v1/oauth/token"

	req := OAuthTokenRefreshRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		ClientID:     clientID,
	}

	httpReq, err := s.client.NewRequest(ctx, "POST", u, req)
	if err != nil {
		return nil, nil, err
	}

	// OAuth token endpoint doesn't require authentication
	httpReq.Header.Del("Authorization")

	var tokenResp OAuthTokenResponse
	resp, err := s.client.Do(ctx, httpReq, &tokenResp)
	if err != nil {
		// Check if it's an OAuth error
		if resp != nil && resp.StatusCode >= 400 {
			var oauthErr OAuthErrorResponse
			if jsonErr := parseJSONResponse(resp, &oauthErr); jsonErr == nil && oauthErr.Error != "" {
				return nil, resp, fmt.Errorf("oauth error: %s - %s", oauthErr.Error, oauthErr.ErrorDescription)
			}
		}
		return nil, resp, err
	}

	return &tokenResp, resp, nil
}