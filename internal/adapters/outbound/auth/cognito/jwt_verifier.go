package cognito

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSet representa o conjunto de chaves públicas do Cognito
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWTVerifier verifica tokens JWT usando JWKS do Cognito
type JWTVerifier struct {
	jwksURL string
	keys    map[string]*rsa.PublicKey
}

// NewJWTVerifier cria um novo verificador de JWT
func NewJWTVerifier(region, userPoolID string) *JWTVerifier {
	return &JWTVerifier{
		jwksURL: fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, userPoolID),
		keys:    make(map[string]*rsa.PublicKey),
	}
}

// VerifyToken verifica a assinatura e validade do token
func (v *JWTVerifier) VerifyToken(ctx context.Context, tokenString string) (string, error) {
	// Parse o token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verifica o método de assinatura
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Pega o kid do header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found in token header")
		}

		// Busca a chave pública
		key, err := v.getPublicKey(ctx, kid)
		if err != nil {
			return nil, err
		}

		return key, nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to verify token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	// Extrai o sub (Cognito User ID)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("missing sub claim")
	}

	return sub, nil
}

// getPublicKey busca a chave pública do JWKS
func (v *JWTVerifier) getPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Verifica se já temos a chave em cache
	if key, exists := v.keys[kid]; exists {
		return key, nil
	}

	// Busca as chaves do JWKS
	if err := v.fetchJWKS(ctx); err != nil {
		return nil, err
	}

	// Tenta novamente após buscar
	key, exists := v.keys[kid]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", kid)
	}

	return key, nil
}

// fetchJWKS busca as chaves públicas do Cognito
func (v *JWTVerifier) fetchJWKS(ctx context.Context) error {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", v.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS: status %d", resp.StatusCode)
	}

	var jwks JWKSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Converte JWK para chaves RSA públicas
	for _, jwk := range jwks.Keys {
		key, err := v.jwkToPublicKey(jwk)
		if err != nil {
			continue
		}
		v.keys[jwk.Kid] = key
	}

	return nil
}

// jwkToPublicKey converte JWK para *rsa.PublicKey
func (v *JWTVerifier) jwkToPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode N (modulus)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, err
	}

	// Decode E (exponent)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)

	// Converte E para int
	var e int
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}
