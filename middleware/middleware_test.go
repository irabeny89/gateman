package middleware

import (
	"database/sql"
	"fmt"
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

	validToken, err := jwt.GenerateJWT(secret, jwt.NewJWTClaims(validPayload, regClaims))
	if err != nil {
		t.Fatalf("failed to generate valid JWT: %v", err)
	}

	expiredClaims := jwt.NewJWTRegisteredClaims(jwt.JWTRegisteredClaimsOptions{
		ExpiresAt: -1 * time.Hour,
	})
	expiredToken, err := jwt.GenerateJWT(secret, jwt.NewJWTClaims(validPayload, expiredClaims))
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

func TestRatelimit(t *testing.T) {
	uri := "/api/test"
	ip := "192.168.1.1"

	setupDB := func(tb testing.TB) *sql.DB {
		tb.Helper()
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			tb.Fatalf("failed to open database: %v", err)
		}
		if err := InitRatelimit(db); err != nil {
			tb.Fatalf("failed to init schema: %v", err)
		}
		tb.Cleanup(func() {
			if err := db.Close(); err != nil {
				tb.Errorf("failed closing database: %v", err)
			}
		})
		return db
	}

	t.Run("First request inserts record and passes through", func(t *testing.T) {
		db := setupDB(t)
		req := httptest.NewRequest(http.MethodGet, uri, nil)
		req.RemoteAddr = ip + ":12345"
		rec := httptest.NewRecorder()

		var rlCtx RateLimitModel
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rlCtx = r.Context().Value(RatelimitCtxKey).(RateLimitModel)
			w.WriteHeader(http.StatusOK)
		})

		// Allow max 3 requests per minute
		Ratelimit(db, 3, time.Minute, nextHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if rlCtx.Count != 1 {
			t.Errorf("expected context count to be 1, got %d", rlCtx.Count)
		}
	})

	t.Run("Subsequent request increments count successfully", func(t *testing.T) {
		db := setupDB(t)
		req := httptest.NewRequest(http.MethodGet, uri, nil)
		req.RemoteAddr = ip + ":12345"

		handler := Ratelimit(db, 3, time.Minute, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Hit it twice
		handler.ServeHTTP(httptest.NewRecorder(), req)
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req)

		if rec2.Code != http.StatusOK {
			t.Fatalf("expected second status to be 200, got %d", rec2.Code)
		}

		rl := newRatelimit(db)
		updated, _ := rl.findByIP(ip)
		if updated.Count != 2 {
			t.Errorf("expected database count to be 2, got %d", updated.Count)
		}
	})

	t.Run("Exceeding limit returns 429 Too Many Requests", func(t *testing.T) {
		db := setupDB(t)
		req := httptest.NewRequest(http.MethodGet, uri, nil)
		req.RemoteAddr = ip + ":12345"

		handler := Ratelimit(db, 1, time.Minute, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request works (limit is 1)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		// Second request should be blocked
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req)

		if rec2.Code != http.StatusTooManyRequests {
			t.Errorf("expected status 429, got %d", rec2.Code)
		}
	})

	t.Run("Expired time window resets count back to 1", func(t *testing.T) {
		db := setupDB(t)
		req := httptest.NewRequest(http.MethodGet, uri, nil)
		req.RemoteAddr = ip + ":12345"

		var rlCtx RateLimitModel
		handler := Ratelimit(db, 3, time.Minute, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rlCtx = r.Context().Value(RatelimitCtxKey).(RateLimitModel)
			w.WriteHeader(http.StatusOK)
		}))

		// 1. Fire an initial request to create the database row (Count becomes 1)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		// 2. Manually change the database expiration to the past to simulate time passing
		pastTime := time.Now().Add(-5 * time.Minute)
		_, err := db.Exec(fmt.Sprintf("UPDATE %s SET end_at = ? WHERE ip = ?", tableName), pastTime, ip)
		if err != nil {
			t.Fatalf("failed to manually manipulate database time: %v", err)
		}

		// 3. Fire a second request. The middleware should see it's expired and reset
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req)

		if rec2.Code != http.StatusOK {
			t.Fatalf("expected status 200 after window reset, got %d", rec2.Code)
		}

		// 4. Verify context tracking reset back down to 1
		if rlCtx.Count != 1 {
			t.Errorf("expected count to reset to 1 after expiration, got %d", rlCtx.Count)
		}

		// 5. Verify database confirms the reset
		rl := newRatelimit(db)
		updated, _ := rl.findByIP(ip)
		if updated.Count != 1 {
			t.Errorf("expected database count to reset to 1, got %d", updated.Count)
		}
	})
}
