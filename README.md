# DotEnv Go SDK

Official Go SDK for the DotEnv API - secure environment variable management for your applications.

## Installation

```bash
go get github.com/lostlink/dotenv-sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    dotenv "github.com/lostlink/dotenv-sdk-go"
)

func main() {
    // Initialize client
    client := dotenv.NewClient("your-api-key")
    
    // List organizations
    orgs, _, err := client.Organizations.List(context.Background(), nil)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, org := range orgs {
        fmt.Printf("Organization: %s\n", org.Name)
    }
}
```

## Configuration

### Custom Base URL

```go
client := dotenv.NewClient("your-api-key", 
    dotenv.WithBaseURL("https://api.custom.dotenv.cloud"))
```

### Custom HTTP Client

```go
httpClient := &http.Client{
    Timeout: 60 * time.Second,
}
client := dotenv.NewClient("your-api-key", 
    dotenv.WithHTTPClient(httpClient))
```

### Development Mode

For local development with self-signed certificates:

```bash
export DOTENV_TLS_SKIP_VERIFY=true
export DOTENV_BASE_URL=https://dotenv.test
```

## API Coverage

### Organizations
- `List(ctx, opts)` - List all organizations
- `Get(ctx, slug)` - Get a specific organization

### Projects
- `List(ctx, orgSlug, opts)` - List projects in an organization
- `Get(ctx, projectSlug)` - Get a specific project
- `Create(ctx, orgSlug, project)` - Create a new project
- `Update(ctx, projectSlug, project)` - Update a project
- `Delete(ctx, projectSlug)` - Delete a project

### Secrets
- `GetProjectSecrets(ctx, project, target, environment)` - Get secrets for a project
- `RetrieveSecrets(ctx, params)` - Retrieve secrets with complex queries
- `PushSecrets(ctx, project, secrets)` - Push multiple secrets
- `List(ctx, projectSlug, opts)` - List all secrets
- `Get(ctx, projectSlug, secretKey)` - Get a specific secret
- `Create(ctx, request)` - Create a new secret
- `Update(ctx, projectSlug, secretKey, value, isEncrypted)` - Update a secret
- `Delete(ctx, projectSlug, secretKey)` - Delete a secret

### Targets & Environments
- `Targets.List(ctx, projectSlug, opts)` - List targets
- `Environments.List(ctx, projectSlug, targetSlug, opts)` - List environments

### Encryption
- `GetEncryptionKey(ctx, project)` - Get encryption key for a project
- `RotateClientKeys(ctx, project)` - Rotate client-side encryption keys

## Encryption

The SDK provides built-in AES-256-GCM encryption:

```go
// Generate a new key
key, err := dotenv.GenerateKey()
if err != nil {
    log.Fatal(err)
}

// Encrypt data
encrypted, err := dotenv.Encrypt("sensitive-data", key)
if err != nil {
    log.Fatal(err)
}

// Decrypt data
decrypted, err := dotenv.Decrypt(encrypted, key)
if err != nil {
    log.Fatal(err)
}
```

## Error Handling

The SDK provides typed errors for common scenarios:

```go
project, _, err := client.Projects.Get(ctx, "my-project")
if err != nil {
    if dotenv.IsNotFound(err) {
        // Handle not found
    } else if dotenv.IsUnauthorized(err) {
        // Handle unauthorized
    } else if dotenv.IsRateLimited(err) {
        rateLimitErr := err.(*dotenv.ErrRateLimited)
        // Wait for rateLimitErr.RetryAfter seconds
    } else if dotenv.IsValidation(err) {
        validationErr := err.(*dotenv.ErrValidation)
        // Handle validation errors
    }
}
```

## Advanced Usage

### Pagination

```go
opts := &dotenv.ListOptions{
    Page:    2,
    PerPage: 50,
    Sort:    "name",
}

projects, resp, err := client.Projects.List(ctx, "org-slug", opts)
// Check resp for pagination metadata
```

### Filtering

```go
opts := &dotenv.ListOptions{
    Filter: map[string]string{
        "status": "active",
        "has_secrets": "true",
    },
}
```

### Context Support

All methods support context for cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

secrets, _, err := client.Secrets.List(ctx, "project-slug", nil)
```

## Retry Logic

The SDK automatically retries failed requests with exponential backoff:
- Retries on network errors and 5xx status codes
- Respects `Retry-After` headers
- Maximum of 3 retry attempts
- Exponential backoff with jitter

## Testing

Run tests:
```bash
go test ./...
```

Run with coverage:
```bash
go test -cover ./...
```

## Examples

See the [examples](examples/) directory for more detailed usage examples.

## Contributing

Please read CONTRIBUTING.md for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.