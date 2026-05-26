package dotenv_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dotenv "github.com/lostlink/dotenv-sdk-go"
)

func TestOAuthService_ExchangeToken(t *testing.T) {
	tests := []struct {
		name           string
		request        dotenv.OAuthTokenAuthCodeRequest
		mockResponse   *dotenv.OAuthTokenResponse
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name: "successful code exchange",
			request: dotenv.OAuthTokenAuthCodeRequest{
				Code:         "test-code",
				CodeVerifier: "test-verifier",
				ClientID:     "test-client",
			},
			mockResponse: &dotenv.OAuthTokenResponse{
				AccessToken:  "test-access-token",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				RefreshToken: "test-refresh-token",
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "invalid code",
			request: dotenv.OAuthTokenAuthCodeRequest{
				Code:         "invalid-code",
				CodeVerifier: "test-verifier",
				ClientID:     "test-client",
			},
			mockResponse:   nil,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "400")
			},
		},
		{
			name: "server error",
			request: dotenv.OAuthTokenAuthCodeRequest{
				Code:         "test-code",
				CodeVerifier: "test-verifier",
				ClientID:     "test-client",
			},
			mockResponse:   nil,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "500")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/oauth/token", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse JSON body
				var reqBody dotenv.OAuthTokenAuthCodeRequest
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)

				// Verify request parameters
				assert.Equal(t, "authorization_code", reqBody.GrantType)
				assert.Equal(t, tt.request.Code, reqBody.Code)
				assert.Equal(t, tt.request.ClientID, reqBody.ClientID)
				assert.Equal(t, tt.request.CodeVerifier, reqBody.CodeVerifier)

				// Send response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					json.NewEncoder(w).Encode(tt.mockResponse)
				}
			}))
			defer server.Close()

			client := dotenv.NewClient(
				dotenv.WithBaseURL(server.URL),
			)

			token, resp, err := client.OAuth.ExchangeToken(
				context.Background(),
				tt.request,
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, token)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, token)
				require.NotNil(t, resp)
				assert.Equal(t, tt.mockResponse.AccessToken, token.AccessToken)
				assert.Equal(t, tt.mockResponse.RefreshToken, token.RefreshToken)
				assert.Equal(t, tt.mockResponse.TokenType, token.TokenType)
				assert.Equal(t, tt.mockResponse.ExpiresIn, token.ExpiresIn)
			}
		})
	}
}

func TestOAuthService_RefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		refreshToken   string
		clientID       string
		mockResponse   *dotenv.OAuthTokenResponse
		mockStatusCode int
		wantErr        bool
		checkError     func(t *testing.T, err error)
	}{
		{
			name:         "successful token refresh",
			refreshToken: "test-refresh-token",
			clientID:     "test-client",
			mockResponse: &dotenv.OAuthTokenResponse{
				AccessToken:  "new-access-token",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				RefreshToken: "new-refresh-token",
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "invalid refresh token",
			refreshToken:   "invalid-refresh-token",
			clientID:       "test-client",
			mockResponse:   nil,
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.True(t, dotenv.IsUnauthorized(err))
			},
		},
		{
			name:           "expired refresh token",
			refreshToken:   "expired-refresh-token",
			clientID:       "test-client",
			mockResponse:   nil,
			mockStatusCode: http.StatusBadRequest,
			wantErr:        true,
			checkError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "400")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/oauth/token", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Parse JSON body
				var reqBody dotenv.OAuthTokenRefreshRequest
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)

				// Verify request parameters
				assert.Equal(t, "refresh_token", reqBody.GrantType)
				assert.Equal(t, tt.refreshToken, reqBody.RefreshToken)
				assert.Equal(t, tt.clientID, reqBody.ClientID)

				// Send response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					json.NewEncoder(w).Encode(tt.mockResponse)
				}
			}))
			defer server.Close()

			client := dotenv.NewClient(
				dotenv.WithBaseURL(server.URL),
			)

			token, resp, err := client.OAuth.RefreshToken(
				context.Background(),
				tt.refreshToken,
				tt.clientID,
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, token)
				if tt.checkError != nil {
					tt.checkError(t, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, token)
				require.NotNil(t, resp)
				assert.Equal(t, tt.mockResponse.AccessToken, token.AccessToken)
				assert.Equal(t, tt.mockResponse.RefreshToken, token.RefreshToken)
				assert.Equal(t, tt.mockResponse.TokenType, token.TokenType)
				assert.Equal(t, tt.mockResponse.ExpiresIn, token.ExpiresIn)
			}
		})
	}
}

func TestOAuthService_ContextCancellation(t *testing.T) {
	// Test context cancellation for ExchangeToken
	t.Run("ExchangeToken context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow response
			<-r.Context().Done()
			return
		}))
		defer server.Close()

		client := dotenv.NewClient(
			dotenv.WithBaseURL(server.URL),
		)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		req := dotenv.OAuthTokenAuthCodeRequest{
			Code:         "code",
			CodeVerifier: "verifier",
			ClientID:     "client",
		}
		token, _, err := client.OAuth.ExchangeToken(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "context canceled")
	})

	// Test context cancellation for RefreshToken
	t.Run("RefreshToken context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow response
			<-r.Context().Done()
			return
		}))
		defer server.Close()

		client := dotenv.NewClient(
			dotenv.WithBaseURL(server.URL),
		)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		token, _, err := client.OAuth.RefreshToken(ctx, "refresh-token", "client")
		assert.Error(t, err)
		assert.Nil(t, token)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
