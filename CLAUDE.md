# DotEnv Go SDK - AI Assistant Context

This is the Go SDK for the DotEnv platform, providing a native Go client library for interacting with the DotEnv API. This SDK is also used as the core library for the DotEnv CLI.

## Overview

The Go SDK is designed to:
- Provide idiomatic Go interface to the DotEnv API
- Share core functionality with the CLI tool
- Handle encryption/decryption with native Go crypto
- Support context-based cancellation
- Offer comprehensive error handling

## Tech Stack

- **Language**: Go 1.21+
- **HTTP Client**: Standard library net/http
- **Encryption**: crypto/aes with GCM mode
- **Testing**: Standard testing package with testify
- **Dependencies**: Minimal external dependencies

## Key Features

### 1. Client Configuration
```go
client := dotenv.NewClient(
    dotenv.WithAPIKey("your-api-key"),
    dotenv.WithOrganization("org-id"),
    dotenv.WithBaseURL("https://api.dotenv.cloud"),
)
```

### 2. Secret Management
```go
// List secrets
secrets, err := client.Secrets.List(ctx, "project-id")

// Get specific secret
secret, err := client.Secrets.Get(ctx, "project-id", "secret-key")

// Set secret
err := client.Secrets.Set(ctx, "project-id", "secret-key", "value")

// Delete secret
err := client.Secrets.Delete(ctx, "project-id", "secret-key")
```

### 3. Encryption
```go
// Client-side encryption
encrypted, err := client.Encryption.Encrypt("sensitive-data", key)

// Decryption
decrypted, err := client.Encryption.Decrypt(encrypted, key)
```

### 4. Error Handling
```go
if err != nil {
    switch e := err.(type) {
    case *dotenv.AuthError:
        // Handle authentication error
    case *dotenv.NotFoundError:
        // Handle not found error
    case *dotenv.RateLimitError:
        // Handle rate limit with retry after e.RetryAfter
    }
}
```

## Project Structure

```
packages/sdk-go/
├── client.go           # Main client implementation
├── secrets.go          # Secrets resource
├── projects.go         # Projects resource
├── organizations.go    # Organizations resource
├── encryption.go       # Encryption utilities
├── errors.go          # Error types
├── options.go         # Client options
├── transport.go       # HTTP transport
├── retry.go           # Retry logic
├── examples/          # Usage examples
├── internal/          # Internal packages
├── go.mod
├── go.sum
└── README.md
```

## Development Guidelines

### Code Style
- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `golangci-lint` for linting
- Keep interfaces small and focused

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...
```

### Performance
- Use connection pooling
- Implement proper timeout handling
- Minimize allocations in hot paths
- Use sync.Pool for frequently allocated objects

## API Integration

The SDK implements these API operations:
- Secret CRUD operations
- Project management
- Organization management
- Encryption key rotation
- Audit log retrieval

## Shared CLI Code

The CLI reuses these SDK components:
- HTTP client and transport
- Encryption/decryption logic
- Error types and handling
- API response models

## Building

```bash
# Download dependencies
go mod download

# Build the package
go build ./...

# Run tests
go test ./...

# Generate mocks (if needed)
go generate ./...
```

## Security Considerations

1. **Zero Trust**: Always validate server certificates
2. **Key Storage**: Never log or expose encryption keys
3. **Memory Safety**: Clear sensitive data from memory
4. **Rate Limiting**: Implement exponential backoff

## Context Support

All API methods support context for cancellation:
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

secrets, err := client.Secrets.List(ctx, "project-id")
```

## Connection Management

```go
// Custom HTTP client
httpClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
    },
}

client := dotenv.NewClient(
    dotenv.WithHTTPClient(httpClient),
)
```

## Common Patterns

### Retry Logic
```go
err := retry.Do(func() error {
    return client.Secrets.Set(ctx, "project", "key", "value")
}, retry.Attempts(3), retry.Delay(time.Second))
```

### Bulk Operations
```go
// Batch secret updates
batch := client.Secrets.Batch()
batch.Set("key1", "value1")
batch.Set("key2", "value2")
batch.Delete("key3")
err := batch.Execute(ctx, "project-id")
```

## Debugging

Enable debug logging:
```go
client := dotenv.NewClient(
    dotenv.WithDebug(true),
    dotenv.WithLogger(log.Default()),
)