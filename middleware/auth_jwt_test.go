package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/irabeny89/gateman/jwt"
	_ "github.com/mattn/go-sqlite3"
)

func TestRequireAuthWithJWT(t *testing.T) {
	secret := "super-secret-key"

	validPayload := jwt.AuthPayload{
		UserID: "user-123",
		Email:  "test@example.com",
		Roles:  []string{"admin", "user"},
	}
	regClaims := jwt.NewJWTRegisteredClaims(jwt.JWTRegisteredClaimsOptions{
		ExpiresAt: 1 * time.Hour,
	})
	claims := jwt.JWTClaims{
		AuthPayload:      validPayload,
		RegisteredClaims: regClaims,
	}
	validToken, err := claims.GenerateToken(secret)
	if err != nil {
		t.Fatalf("failed to generate valid JWT: %v", err)
	}

	expiredRegClaims := jwt.NewJWTRegisteredClaims(jwt.JWTRegisteredClaimsOptions{
		ExpiresAt: -1 * time.Hour,
	})
	expClaims := jwt.JWTClaims{
		AuthPayload:      validPayload,
		RegisteredClaims: expiredRegClaims,
	}
	expiredToken, err := expClaims.GenerateToken(secret)
	if err != nil {
		t.Fatalf("failed to generate expired JWT: %v", err)
	}

	tests := []struct {
		name           string
		secret         string
		requiredRoles  []string
		authHeader     string
		expectedStatus int
		verifyCtx      bool
	}{
		{
			name:           "Valid token with matching role",
			secret:         secret,
			requiredRoles:  []string{"admin"},
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			verifyCtx:      true,
		},
		{
			name:           "Valid token without role restriction",
			secret:         secret,
			requiredRoles:  []string{},
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			verifyCtx:      true,
		},
		{
			name:           "Valid token with non-matching role",
			secret:         secret,
			requiredRoles:  []string{"superadmin"},
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Invalid secret token",
			secret:         "wrong-secret",
			requiredRoles:  []string{"admin"},
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Expired token",
			secret:         secret,
			requiredRoles:  []string{"admin"},
			authHeader:     "Bearer " + expiredToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing authorization header",
			secret:         secret,
			requiredRoles:  []string{"admin"},
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Malformed authorization header",
			secret:         secret,
			requiredRoles:  []string{"admin"},
			authHeader:     "InvalidHeaderFormat",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var contextPayload jwt.AuthPayload
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if val := r.Context().Value(AuthJWTCtxKey); val != nil {
					if payload, ok := val.(jwt.AuthPayload); ok {
						contextPayload = payload
					}
				}
				w.WriteHeader(http.StatusOK)
			})

			handler := RequireAuthWithJWT(tt.secret, tt.requiredRoles, nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status code %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.verifyCtx {
				if contextPayload.UserID != validPayload.UserID {
					t.Errorf("expected context UserID %s, got %s", validPayload.UserID, contextPayload.UserID)
				}
				if contextPayload.Email != validPayload.Email {
					t.Errorf("expected context Email %s, got %s", validPayload.Email, contextPayload.Email)
				}
			}
		})
	}
}
