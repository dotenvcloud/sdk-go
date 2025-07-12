package dotenv

import (
	"time"
)

// Organization represents a DotEnv organization
type Organization struct {
	ID        string    `json:"id"`
	ULID      string    `json:"ulid"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	PlanName  string    `json:"plan_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Project represents a DotEnv project
type Project struct {
	ID               string    `json:"id"`
	ULID             string    `json:"ulid"`
	OrganizationID   string    `json:"organization_id"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	Description      string    `json:"description"`
	SecretFormat     string    `json:"secret_format"`
	EncryptionType   string    `json:"encryption_type"` // "server" or "client"
	HasSecrets       bool      `json:"has_secrets"`
	SecretCount      int       `json:"secret_count"`
	EnvironmentCount int       `json:"environment_count"`
	TargetCount      int       `json:"target_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Target represents a deployment target
type Target struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"project_id"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	Description      string    `json:"description"`
	EnvironmentCount int       `json:"environment_count"`
	SecretCount      int       `json:"secret_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Environment represents an environment
type Environment struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	TargetID    string    `json:"target_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Secret represents an encrypted secret
type Secret struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"project_id"`
	TargetID        *string   `json:"target_id,omitempty"`
	EnvironmentID   *string   `json:"environment_id,omitempty"`
	Key             string    `json:"key"`
	Value           string    `json:"value"` // Encrypted value
	IsEncrypted     bool      `json:"is_encrypted"`
	EncryptionKeyID string    `json:"encryption_key_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// EncryptionKey represents an encryption key
type EncryptionKey struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"project_id"`
	Key             string    `json:"key"` // Base64 encoded
	IsActive        bool      `json:"is_active"`
	IsClientManaged bool      `json:"is_client_managed"`
	CreatedAt       time.Time `json:"created_at"`
}

// API Response Types for JSON:API format
type JSONAPIResponse struct {
	Data     interface{}    `json:"data"`
	Meta     *Meta          `json:"meta,omitempty"`
	Links    *Links         `json:"links,omitempty"`
	Included []interface{}  `json:"included,omitempty"`
	Errors   []JSONAPIError `json:"errors,omitempty"`
}

// JSONAPIData represents a JSON:API data element
type JSONAPIData struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id"`
	Attributes    interface{}            `json:"attributes"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
}

// JSONAPIError represents a JSON:API error
type JSONAPIError struct {
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// Meta contains pagination metadata
type Meta struct {
	CurrentPage int `json:"current_page"`
	From        int `json:"from"`
	LastPage    int `json:"last_page"`
	PerPage     int `json:"per_page"`
	To          int `json:"to"`
	Total       int `json:"total"`
}

// Links contains pagination links
type Links struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Prev  string `json:"prev"`
	Next  string `json:"next"`
}

// Request types
type CreateSecretRequest struct {
	ProjectSlug     string  `json:"project_slug"`
	TargetSlug      *string `json:"target_slug,omitempty"`
	EnvironmentSlug *string `json:"environment_slug,omitempty"`
	Key             string  `json:"key"`
	Value           string  `json:"value"`
	IsEncrypted     bool    `json:"is_encrypted"`
}

type BulkSecretsRequest struct {
	ProjectSlug string              `json:"project_slug"`
	Secrets     []BulkSecretItem   `json:"secrets"`
}

type BulkSecretItem struct {
	Key             string  `json:"key"`
	Value           string  `json:"value"`
	TargetSlug      *string `json:"target_slug,omitempty"`
	EnvironmentSlug *string `json:"environment_slug,omitempty"`
	IsEncrypted     bool    `json:"is_encrypted"`
}

// RetrieveParams represents parameters for retrieving secrets
type RetrieveParams struct {
	Project     string `json:"project"`
	Target      string `json:"target,omitempty"`
	Environment string `json:"environment,omitempty"`
	Action      string `json:"action,omitempty"` // read, decrypt, key:retrieve (default: read)
	Merge       string `json:"merge,omitempty"`  // 'true' or 'false' (default: 'false')
	Raw         bool   `json:"raw,omitempty"`    // Simple key-value output
	Filters     struct {
		Names  []string `json:"names,omitempty"`
		Tags   []string `json:"tags,omitempty"`
		Search string   `json:"search,omitempty"`
	} `json:"filters,omitempty"`
}

// PushSecretsRequest represents a request to push multiple secrets
type PushSecretsRequest struct {
	Secrets     map[string]interface{} `json:"secrets"`
	Target      string                 `json:"target,omitempty"`
	Environment string                 `json:"environment,omitempty"`
}

// Options for API calls
type ListOptions struct {
	Page    int
	PerPage int
	Sort    string
	Filter  map[string]string
}

type SecretOptions struct {
	IncludeDecrypted bool
	ResolveHierarchy bool
	Target           string
	Environment      string
}

// EncryptionMode represents the encryption mode
type EncryptionMode string

const (
	EncryptionModeServerManaged EncryptionMode = "server_managed"
	EncryptionModeClientManaged EncryptionMode = "client_managed"
	EncryptionModeHybrid        EncryptionMode = "hybrid"
)

// SecretsHierarchyResponse represents the hierarchical secrets response from the API
type SecretsHierarchyResponse struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Encrypted bool                     `json:"encrypted"`
			Format    string                   `json:"format"`
			Levels    map[string]SecretLevel   `json:"levels"`
		} `json:"attributes"`
	} `json:"data"`
	Meta struct {
		APIPath   string `json:"api_path"`
		Format    string `json:"format"`
		Merged    string `json:"merged"`
		Timestamp string `json:"timestamp"`
		Hierarchy struct {
			Project     string  `json:"project"`
			Target      *string `json:"target"`
			Environment *string `json:"environment"`
		} `json:"hierarchy"`
		Action string `json:"action"`
	} `json:"meta"`
}

// SecretLevel represents a single level of secrets in the hierarchy
type SecretLevel struct {
	Encrypted bool   `json:"encrypted"`
	Content   string `json:"content"`
	Source    string `json:"source"`
}
