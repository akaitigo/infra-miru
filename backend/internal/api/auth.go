package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// jwtHeader represents the header portion of a JWT.
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// jwtClaims represents the payload portion of a JWT with standard claims.
type jwtClaims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

// JWTAuth returns middleware that validates Bearer JWT tokens using HMAC-SHA256.
// Requests without a valid token receive a 401 Unauthorized response.
func JWTAuth(secret string) func(http.Handler) http.Handler {
	secretBytes := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := extractBearerToken(r)
			if !ok {
				JSONError(w, http.StatusUnauthorized, "missing or malformed Authorization header", "AUTH_MISSING")
				return
			}

			if err := validateJWT(token, secretBytes); err != nil {
				JSONError(w, http.StatusUnauthorized, fmt.Sprintf("invalid token: %v", err), "AUTH_INVALID")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractBearerToken extracts the token from an "Authorization: Bearer <token>" header.
func extractBearerToken(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", false
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return "", false
	}

	token := strings.TrimSpace(auth[len(prefix):])
	if token == "" {
		return "", false
	}

	return token, true
}

// validateJWT verifies the JWT signature and expiration using HMAC-SHA256.
func validateJWT(token string, secret []byte) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("token must have 3 parts, got %d", len(parts))
	}

	// Decode and validate header.
	headerBytes, decErr := base64.RawURLEncoding.DecodeString(parts[0])
	if decErr != nil {
		return fmt.Errorf("decode header: %w", decErr)
	}

	var header jwtHeader
	if unmarshalErr := json.Unmarshal(headerBytes, &header); unmarshalErr != nil {
		return fmt.Errorf("parse header: %w", unmarshalErr)
	}

	if header.Alg != "HS256" {
		return fmt.Errorf("unsupported algorithm: %s", header.Alg)
	}

	// Verify HMAC-SHA256 signature.
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	actualSig, sigErr := base64.RawURLEncoding.DecodeString(parts[2])
	if sigErr != nil {
		return fmt.Errorf("decode signature: %w", sigErr)
	}

	if !hmac.Equal(expectedSig, actualSig) {
		return fmt.Errorf("signature verification failed")
	}

	// Decode and validate claims.
	claimsBytes, claimsErr := base64.RawURLEncoding.DecodeString(parts[1])
	if claimsErr != nil {
		return fmt.Errorf("decode claims: %w", claimsErr)
	}

	var claims jwtClaims
	if parseErr := json.Unmarshal(claimsBytes, &claims); parseErr != nil {
		return fmt.Errorf("parse claims: %w", parseErr)
	}

	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return fmt.Errorf("token expired")
	}

	return nil
}
