# CRITICAL DEVELOPMENT REMINDERS - GO SDK

<critical>
- NEVER make assumptions about how ANY code works. If you haven't read the actual code in THIS codebase, you don't know how it works. Period.
</critical>

## Go SDK Architecture
- **Shared with CLI**: Core logic reused between SDK and CLI
- Package structure allows clean separation
- Public API must be stable and versioned
- Internal packages for shared utilities

## Go Best Practices
- Idiomatic error handling with wrapped context
- Interface-based design for testability
- Minimal external dependencies
- Concurrent-safe implementations

## API Client Design
- HTTP client with configurable timeouts
- Automatic retry with exponential backoff
- Context support for cancellation
- Structured logging (no sensitive data)

## Encryption Standards
- Client-side encryption: AES-256-GCM
- 32-byte keys, 12-byte IV, base64(IV + ciphertext + tag)
- Consistent with CLI implementation
- Performance-optimized for large files

## SDK Public Interface
- Fluent API design where appropriate
- Options pattern for configuration
- Clear godoc documentation
- Semantic versioning for releases