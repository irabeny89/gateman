package middleware

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
