// Package social provides JWT validation for Apple and Google ID tokens
// received from mobile apps during social sign-in.
package social

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	appleJWKSURL  = "https://appleid.apple.com/auth/keys"
	googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

	appleIssuer   = "https://appleid.apple.com"
	googleIssuer1 = "accounts.google.com"
	googleIssuer2 = "https://accounts.google.com"

	jwksCacheTTL = 1 * time.Hour
	fetchTimeout = 10 * time.Second
)

// TokenClaims holds the validated claims extracted from a social ID token.
type TokenClaims struct {
	Sub   string
	Email string
	Name  string
}

// ValidateAppleToken validates an Apple ID token JWT and returns its claims.
// The email field may be empty for repeat Apple sign-ins (Apple omits it after
// the first authentication); callers should handle that case explicitly.
func ValidateAppleToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	keys, err := getJWKS(ctx, appleJWKSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Apple public keys: %w", err)
	}

	token, err := jwt.Parse(tokenString, rsaKeyFunc(ctx, appleJWKSURL, keys),
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(appleIssuer),
		jwt.WithAudience("careers.hirefy.app"),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid Apple token: %w", err)
	}

	return extractClaims(token, "apple")
}

// ValidateGoogleToken validates a Google ID token JWT and returns its claims.
func ValidateGoogleToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	keys, err := getJWKS(ctx, googleJWKSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google public keys: %w", err)
	}

	token, err := jwt.Parse(tokenString, rsaKeyFunc(ctx, googleJWKSURL, keys),
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid Google token: %w", err)
	}

	// Validate issuer manually – Google uses two different issuer strings.
	mc, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid Google token claims type")
	}
	iss, _ := mc["iss"].(string)
	if iss != googleIssuer1 && iss != googleIssuer2 {
		return nil, fmt.Errorf("invalid Google token issuer: %q", iss)
	}

	return extractClaims(token, "google")
}

// rsaKeyFunc returns a jwt.Keyfunc that resolves the signing key by kid.
// On cache miss it re-fetches the JWKS once to handle key rotation.
func rsaKeyFunc(ctx context.Context, jwksURL string, keys map[string]*rsa.PublicKey) jwt.Keyfunc {
	return func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		if key, ok := keys[kid]; ok {
			return key, nil
		}
		// Key not found – the JWKS may have rotated; invalidate cache and retry.
		invalidateCache(jwksURL)
		freshKeys, err := getJWKS(ctx, jwksURL)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh JWKS: %w", err)
		}
		if key, ok := freshKeys[kid]; ok {
			return key, nil
		}
		return nil, fmt.Errorf("signing key %q not found in JWKS", kid)
	}
}

// extractClaims pulls sub, email, and name out of a validated JWT token.
func extractClaims(token *jwt.Token, provider string) (*TokenClaims, error) {
	mc, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unexpected claims type in %s token", provider)
	}
	sub, _ := mc["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("missing sub claim in %s token", provider)
	}
	email, _ := mc["email"].(string)
	name, _ := mc["name"].(string)
	return &TokenClaims{Sub: sub, Email: email, Name: name}, nil
}

// ── JWKS in-memory cache ──────────────────────────────────────────────────────

type jwkEntry struct {
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

var (
	cacheOnce sync.Once
	jwksCache *jwksStore
)

type jwksStore struct {
	mu      sync.RWMutex
	entries map[string]*jwkEntry
}

func getCache() *jwksStore {
	cacheOnce.Do(func() {
		jwksCache = &jwksStore{entries: make(map[string]*jwkEntry)}
	})
	return jwksCache
}

func getJWKS(ctx context.Context, url string) (map[string]*rsa.PublicKey, error) {
	c := getCache()

	c.mu.RLock()
	e, ok := c.entries[url]
	c.mu.RUnlock()
	if ok && time.Since(e.fetched) < jwksCacheTTL {
		return e.keys, nil
	}

	keys, err := fetchJWKS(ctx, url)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.entries[url] = &jwkEntry{keys: keys, fetched: time.Now()}
	c.mu.Unlock()

	return keys, nil
}

func invalidateCache(url string) {
	c := getCache()
	c.mu.Lock()
	delete(c.entries, url)
	c.mu.Unlock()
}

// ── JWKS fetching & RSA key parsing ──────────────────────────────────────────

type jwksJSON struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func fetchJWKS(ctx context.Context, url string) (map[string]*rsa.PublicKey, error) {
	reqCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build JWKS request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("JWKS fetch error for %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint %s returned HTTP %d", url, resp.StatusCode)
	}

	var raw jwksJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS response: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(raw.Keys))
	for _, k := range raw.Keys {
		if k.Kty != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("failed to decode modulus for key %q: %w", k.Kid, err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("failed to decode exponent for key %q: %w", k.Kid, err)
		}
		n := new(big.Int).SetBytes(nBytes)
		e := int(new(big.Int).SetBytes(eBytes).Int64())
		keys[k.Kid] = &rsa.PublicKey{N: n, E: e}
	}
	return keys, nil
}
