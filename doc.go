// Package dotenv is the Go SDK for the DotEnv API.
//
// # API contract version
//
// Every outbound request carries an `X-API-Version: 1` header. Servers
// returning a different version on the response should be treated as
// incompatible.
//
// # Route format
//
// The SDK targets the short route format `/api/v1/{organization}/...`.
// The longer `/api/v1/organizations/{organization}/...` form remains
// available server-side but is deprecated (the server emits
// `Deprecation: true` and `Sunset: <date>` headers). New integrations and
// SDK contributions must use the short format.
//
// # Error handling
//
// Errors returned by service methods can be matched with errors.Is against
// the sentinel variables in errors.go (ErrClientManagedEncryption,
// ErrInvalidParameterCombination, ErrNoActiveEncryptionKey). Older typed
// errors (ErrNotFound, ErrUnauthorized, ErrForbidden, ErrValidation,
// ErrRateLimited, ErrConflict) continue to work and are populated with the
// resource type set explicitly by services rather than parsed from the URL.
package dotenv
