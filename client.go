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
	"time"
)

const (
	defaultBaseURL = "https://api.dotenv.com"
	defaultTimeout = 30 * time.Second
	userAgent      = "dotenv-go-sdk/1.0.0"
)

// Client manages communication with the DotEnv API
type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	userAgent  string

	// Service endpoints
	Organizations *OrganizationsService
	Projects      *ProjectsService
	Targets       *TargetsService
	Environments  *EnvironmentsService
	Secrets       *SecretsService
	Encryption    *EncryptionService
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

// NewClient creates a new DotEnv API client
func NewClient(apiKey string, opts ...ClientOption) *Client {
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
		apiKey:     apiKey,
		httpClient: httpClient,
		userAgent:  userAgent,
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

	return c
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

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
	case http.StatusNotFound:
		return &ErrNotFound{Resource: "resource", ID: ""}
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
	}

	return errorResponse
}

// APIKey returns the client's API key
func (c *Client) APIKey() string {
	return c.apiKey
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
