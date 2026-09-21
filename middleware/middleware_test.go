package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/irabeny89/gateman/jwt"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "gateman_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp db file: %v", err)
	}
	dbPath := f.Name()
	f.Close()

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		os.Remove(dbPath)
		t.Fatalf("failed to open sqlite database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE rate_limits (
			ip TEXT,
			uri TEXT,
			count INTEGER,
			end_at DATETIME,
			PRIMARY KEY (ip, uri)
		);
	`)
	if err != nil {
		db.Close()
		os.Remove(dbPath)
		t.Fatalf("failed to create rate_limits table: %v", err)
	}

	return db, func() {
		db.Close()
		os.Remove(dbPath)
	}
}

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
				if val := r.Context().Value(authPayloadKey); val != nil {
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
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("First request from new IP inserts record and succeeds", func(t *testing.T) {
		sqlite, cleanup := setupTestDB(t)
		defer cleanup()

		handler := Ratelimit(sqlite, 3, time.Minute, nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var count int
		err := sqlite.QueryRow("SELECT count FROM rate_limits WHERE ip = ? AND uri = ?", "192.168.1.1", "/api/test").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query rate_limits table: %v", err)
		}
		if count != 1 {
			t.Errorf("expected initial count 1, got %d", count)
		}
	})

	t.Run("Subsequent requests increment count", func(t *testing.T) {
		sqlite, cleanup := setupTestDB(t)
		defer cleanup()

		handler := Ratelimit(sqlite, 3, time.Minute, nextHandler)

		for i := 1; i <= 3; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.RemoteAddr = "192.168.1.2:12345"
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("request %d: expected status 200, got %d", i, rec.Code)
			}
		}

		var count int
		err := sqlite.QueryRow("SELECT count FROM rate_limits WHERE ip = ? AND uri = ?", "192.168.1.2", "/api/test").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query rate_limits table: %v", err)
		}
		if count != 3 {
			t.Errorf("expected count 3 after 3 requests, got %d", count)
		}
	})

	t.Run("Database query error returns 500", func(t *testing.T) {
		sqlite, cleanup := setupTestDB(t)
		defer cleanup()

		_, err := sqlite.Exec("DROP TABLE rate_limits;")
		if err != nil {
			t.Fatalf("failed to drop table for test: %v", err)
		}

		handler := Ratelimit(sqlite, 3, time.Minute, nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.RemoteAddr = "192.168.1.3:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500 on db error, got %d", rec.Code)
		}
	})

	t.Run("Rate limit restart when window expires", func(t *testing.T) {
		sqlite, cleanup := setupTestDB(t)
		defer cleanup()

		// Pre-populate an expired record with max reached count
		pastEndAt := time.Now().Add(-10 * time.Minute)
		_, err := sqlite.Exec(`
			INSERT INTO rate_limits (ip, uri, count, end_at)
			VALUES (?, ?, ?, ?)
		`, "192.168.1.4", "/api/test", 5, pastEndAt)
		if err != nil {
			t.Fatalf("failed to insert initial expired record: %v", err)
		}

		handler := Ratelimit(sqlite, 5, 1*time.Minute, nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.RemoteAddr = "192.168.1.4:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200 after window reset, got %d", rec.Code)
		}
	})

	t.Run("Exceeding rate limit blocks request with 429", func(t *testing.T) {
		sqlite, cleanup := setupTestDB(t)
		defer cleanup()

		handler := Ratelimit(sqlite, 2, 1*time.Minute, nextHandler)

		// First request (count 1) -> 200
		req1 := httptest.NewRequest(http.MethodGet, "/api/limit", nil)
		req1.RemoteAddr = "192.168.1.5:12345"
		rec1 := httptest.NewRecorder()
		handler.ServeHTTP(rec1, req1)
		if rec1.Code != http.StatusOK {
			t.Errorf("req1: expected status 200, got %d", rec1.Code)
		}

		// Second request (count 2) -> 200
		req2 := httptest.NewRequest(http.MethodGet, "/api/limit", nil)
		req2.RemoteAddr = "192.168.1.5:12345"
		rec2 := httptest.NewRecorder()
		handler.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusOK {
			t.Errorf("req2: expected status 200, got %d", rec2.Code)
		}

		// Third request (count 3 > max 2) -> 429
		req3 := httptest.NewRequest(http.MethodGet, "/api/limit", nil)
		req3.RemoteAddr = "192.168.1.5:12345"
		rec3 := httptest.NewRecorder()
		handler.ServeHTTP(rec3, req3)
		if rec3.Code != http.StatusTooManyRequests {
			t.Errorf("req3: expected status 429 (TooManyRequests), got %d", rec3.Code)
		}
	})
}
