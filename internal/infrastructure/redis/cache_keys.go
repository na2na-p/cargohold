// Package redis provides Redis cache key management and TTL definitions.
// All cache keys and TTLs should be defined in this file to ensure centralized management.
package redis

import (
	"fmt"
	"time"
)

// Cache Key Prefixes
// All Redis cache key prefixes are defined here to ensure consistent naming
// and centralized management across the application.
const (
	// MetadataKeyPrefix is the prefix for LFS object metadata cache keys
	// Format: lfs:meta:{oid}
	MetadataKeyPrefix = "lfs:meta:"

	// SessionKeyPrefix is the prefix for session cache keys
	// Format: lfs:session:{session_id}
	SessionKeyPrefix = "lfs:session:"

	// BatchUploadKeyPrefix is the prefix for batch upload cache keys
	// Format: lfs:batch:upload:{oid}
	BatchUploadKeyPrefix = "lfs:batch:upload:"

	// OIDCGitHubRepoKeyPrefix is the prefix for GitHub OIDC repository allowlist cache keys
	// Format: lfs:oidc:github:repo:{repository}
	OIDCGitHubRepoKeyPrefix = "lfs:oidc:github:repo:"

	// OIDCStateKeyPrefix is the prefix for OIDC state parameter cache keys
	// Format: lfs:oidc:state:{state}
	OIDCStateKeyPrefix = "lfs:oidc:state:"

	// OIDCJWKSKeyPrefix is the prefix for OIDC JWKS cache keys
	// Format: lfs:oidc:jwks:{provider}
	OIDCJWKSKeyPrefix = "lfs:oidc:jwks:"
)

// Cache TTL Definitions
// All Redis cache TTL values are defined here to ensure consistent
// expiration policies across the application.
const (
	// MetadataTTL is the TTL for LFS object metadata cache (30 minutes)
	MetadataTTL = 30 * time.Minute

	// SessionTTL is the TTL for session cache (24 hours)
	SessionTTL = 24 * time.Hour

	// OIDCGitHubRepoTTL is the TTL for GitHub OIDC repository allowlist cache (5 minutes)
	OIDCGitHubRepoTTL = 5 * time.Minute

	// OIDCStateTTL is the TTL for OIDC state parameter cache (10 minutes)
	OIDCStateTTL = 10 * time.Minute

	// OIDCJWKSTTL is the TTL for OIDC JWKS cache (24 hours)
	OIDCJWKSTTL = 24 * time.Hour
)

// Key Generation Functions
// These functions provide a consistent way to generate cache keys
// with proper prefixes.

// MetadataKey generates a cache key for LFS object metadata
func MetadataKey(oid string) string {
	return MetadataKeyPrefix + oid
}

// SessionKey generates a cache key for session data
func SessionKey(sessionID string) string {
	return SessionKeyPrefix + sessionID
}

// BatchUploadKey generates a cache key for batch upload data
func BatchUploadKey(oid string) string {
	return BatchUploadKeyPrefix + oid
}

// OIDCGitHubRepoKey generates a cache key for GitHub OIDC repository allowlist
func OIDCGitHubRepoKey(repository string) string {
	return OIDCGitHubRepoKeyPrefix + repository
}

// OIDCStateKey generates a cache key for OIDC state parameter
func OIDCStateKey(state string) string {
	return OIDCStateKeyPrefix + state
}

// OIDCJWKSKey generates a cache key for OIDC JWKS
func OIDCJWKSKey(provider string) string {
	return fmt.Sprintf("%s%s", OIDCJWKSKeyPrefix, provider)
}
