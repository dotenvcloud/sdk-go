package dotenv

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.dotenv.cloud"
	defaultTimeout = 30 * time.Second
	userAgent      = "dotenv-go-sdk/1.0.0"
)

// AuthType represents the authentication method
type AuthType int

const (
	// AuthTypeAPIKey uses organization API key authentication
	AuthTypeAPIKey AuthType = iota
	// AuthTypeBearer uses OAuth2 bearer token authentication
	AuthTypeBearer
)

// Client manages communication with the DotEnv API
type Client struct {
	baseURL      *url.URL
	apiKey       string   // Organization API key
	bearerToken  string   // OAuth2 access token
	authType     AuthType // Authentication method
	organization string   // Organization context (ULID)
	httpClient   *http.Client
	userAgent    string

	// Service endpoints
	Organizations *OrganizationsService
	Projects      *ProjectsService
	Targets       *TargetsService
	Environments  *EnvironmentsService
	Secrets       *SecretsService
	Encryption    *EncryptionService
	OAuth         *OAuthService
	User          *UserService
	APIKeys       *APIKeysService
	Telemetry     *TelemetryService
}

// ClientOption allows customization of the client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		u, _ := url.Parse(baseURL)
		c.baseURL = u
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithUserAgent sets a custom user agent
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithInsecureSkipVerify disables TLS certificate verification (for development only)
func WithInsecureSkipVerify() ClientOption {
	return func(c *Client) {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		c.httpClient = &http.Client{
			Timeout:   defaultTimeout,
			Transport: transport,
		}
	}
}

// WithAPIKey sets the organization API key for authentication
func WithAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.apiKey = apiKey
		c.authType = AuthTypeAPIKey
	}
}

// WithBearerToken sets the OAuth2 bearer token for authentication
func WithBearerToken(token string) ClientOption {
	return func(c *Client) {
		c.bearerToken = token
		c.authType = AuthTypeBearer
	}
}

// WithOrganization sets the organization context for API requests
func WithOrganization(organization string) ClientOption {
	return func(c *Client) {
		c.organization = organization
	}
}

// NewClient creates a new DotEnv API client with options
func NewClient(opts ...ClientOption) *Client {
	baseURL, _ := url.Parse(defaultBaseURL)

	// Create HTTP client with proper defaults
	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}

	// For development, allow insecure TLS if env var is set
	if os.Getenv("DOTENV_TLS_SKIP_VERIFY") == "true" {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	c := &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		userAgent:  userAgent,
		authType:   AuthTypeAPIKey, // Default to API key auth
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Initialize services
	c.Organizations = &OrganizationsService{client: c}
	c.Projects = &ProjectsService{client: c}
	c.Targets = &TargetsService{client: c}
	c.Environments = &EnvironmentsService{client: c}
	c.Secrets = &SecretsService{client: c}
	c.Encryption = &EncryptionService{client: c}
	c.OAuth = &OAuthService{client: c}
	c.User = &UserService{client: c}
	c.APIKeys = &APIKeysService{client: c}
	c.Telemetry = &TelemetryService{client: c}

	return c
}

// NewClientWithAPIKey creates a new client with API key authentication (backward compatibility)
func NewClientWithAPIKey(apiKey string, opts ...ClientOption) *Client {
	allOpts := append([]ClientOption{WithAPIKey(apiKey)}, opts...)
	return NewClient(allOpts...)
}

// NewRequest creates an API request
func (c *Client) NewRequest(ctx context.Context, method, urlStr string, body interface{}) (*http.Request, error) {
	u, err := c.baseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	// Set authentication header based on auth type
	switch c.authType {
	case AuthTypeBearer:
		if c.bearerToken != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.bearerToken))
		}
	case AuthTypeAPIKey:
		if c.apiKey != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		}
	}

	return req, nil
}

// Do executes an API request
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*http.Response, error) {
	// Apply retry logic
	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for API errors
	if err := checkResponse(resp); err != nil {
		return resp, err
	}

	if v != nil && resp.StatusCode != http.StatusNoContent {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
		}
	}

	return resp, err
}

// checkResponse checks for API errors
func checkResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err := io.ReadAll(r.Body)
	if err == nil && data != nil {
		json.Unmarshal(data, errorResponse)
	}

	// Handle specific error types
	switch r.StatusCode {
	case http.StatusUnauthorized:
		return &ErrUnauthorized{Message: errorResponse.Message}
	case http.StatusForbidden:
		// Extract resource info from URL
		resource, id := extractResourceFromURL(r.Request.URL)
		return &ErrForbidden{
			Resource: resource,
			ID:       id,
			Action:   "access",
		}
	case http.StatusNotFound:
		// Extract resource info from URL
		resource, id := extractResourceFromURL(r.Request.URL)
		return &ErrNotFound{Resource: resource, ID: id}
	case http.StatusTooManyRequests:
		retryAfter := 60 // default to 60 seconds
		if ra := r.Header.Get("Retry-After"); ra != "" {
			fmt.Sscanf(ra, "%d", &retryAfter)
		}
		return &ErrRateLimited{RetryAfter: retryAfter}
	case http.StatusBadRequest:
		if errorResponse.Errors != nil {
			return &ErrValidation{Errors: errorResponse.Errors}
		}
	case http.StatusConflict:
		resource, _ := extractResourceFromURL(r.Request.URL)
		return &ErrConflict{Resource: resource}
	}

	return errorResponse
}

// APIKey returns the client's API key
func (c *Client) APIKey() string {
	return c.apiKey
}

// BearerToken returns the client's bearer token
func (c *Client) BearerToken() string {
	return c.bearerToken
}

// AuthType returns the client's authentication type
func (c *Client) AuthType() AuthType {
	return c.authType
}

// IsUsingAPIKey returns true if the client is using API key authentication
func (c *Client) IsUsingAPIKey() bool {
	return c.authType == AuthTypeAPIKey
}

// IsUsingBearer returns true if the client is using bearer token authentication
func (c *Client) IsUsingBearer() bool {
	return c.authType == AuthTypeBearer
}

// SetTLSSkipVerify enables or disables TLS certificate verification
func (c *Client) SetTLSSkipVerify(skip bool) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skip},
	}
	c.httpClient = &http.Client{
		Timeout:   defaultTimeout,
		Transport: transport,
	}
}

// Organization returns the client's organization context
func (c *Client) Organization() string {
	return c.organization
}

// SetOrganization updates the client's organization context
func (c *Client) SetOrganization(organization string) {
	c.organization = organization
}

// extractResourceFromURL attempts to extract resource type and ID from API URL
func extractResourceFromURL(u *url.URL) (resource string, id string) {
	path := u.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Common patterns:
	// /api/v1/organizations/{org} -> organization, {org}
	// /api/v1/organizations/{org}/projects/{project} -> project, {project}
	// /api/v1/organizations/{org}/projects/{project}/secrets -> secrets, {project}

	for i := 0; i < len(parts)-1; i++ {
		switch parts[i] {
		case "organizations":
			if i+1 < len(parts) {
				return "organization", parts[i+1]
			}
		case "projects":
			if i+1 < len(parts) {
				return "project", parts[i+1]
			}
		case "targets":
			if i+1 < len(parts) {
				return "target", parts[i+1]
			}
		case "environments":
			if i+1 < len(parts) {
				return "environment", parts[i+1]
			}
		case "secrets":
			if i+1 < len(parts) && parts[i+1] != "bulk" && parts[i+1] != "retrieve" {
				return "secret", parts[i+1]
			}
		}
	}

	// Check last part for resource type
	lastPart := parts[len(parts)-1]
	switch lastPart {
	case "secrets":
		return "secrets", ""
	case "projects":
		return "projects", ""
	case "organizations":
		return "organizations", ""
	}

	return "resource", ""
}
