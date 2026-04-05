package api_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/akaitigo/infra-miru/backend/internal/api"
)

const testJWTSecret = "test-secret-key-for-unit-tests"

// buildJWT constructs a minimal HS256 JWT for testing.
func buildJWT(t *testing.T, secret string, exp int64) string {
	t.Helper()

	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}

	claims, err := json.Marshal(map[string]any{
		"sub": "test-user",
		"iat": time.Now().Unix(),
		"exp": exp,
	})
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(header)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claims)

	signingInput := headerB64 + "." + claimsB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return headerB64 + "." + claimsB64 + "." + sig
}

func TestJWTAuth(t *testing.T) {
	t.Parallel()

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "valid token passes",
			authHeader: "Bearer " + buildJWT(t, testJWTSecret, time.Now().Add(time.Hour).Unix()),
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing Authorization header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "AUTH_MISSING",
		},
		{
			name:       "malformed header without Bearer prefix",
			authHeader: "Token abc",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "AUTH_MISSING",
		},
		{
			name:       "empty Bearer token",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "AUTH_MISSING",
		},
		{
			name:       "expired token",
			authHeader: "Bearer " + buildJWT(t, testJWTSecret, time.Now().Add(-time.Hour).Unix()),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "AUTH_INVALID",
		},
		{
			name:       "wrong secret",
			authHeader: "Bearer " + buildJWT(t, "wrong-secret", time.Now().Add(time.Hour).Unix()),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "AUTH_INVALID",
		},
		{
			name:       "garbage token",
			authHeader: "Bearer not.a.valid.jwt",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "AUTH_INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			middleware := api.JWTAuth(testJWTSecret)
			handler := middleware(okHandler)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantCode != "" {
				var errResp api.ErrorResponse
				if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp.Code != tt.wantCode {
					t.Errorf("error code = %q, want %q", errResp.Code, tt.wantCode)
				}
			}
		})
	}
}
