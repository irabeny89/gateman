package middleware

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const RatelimitCtxKey = "ratelimitPayload"
const tableName = "ratelimits"

// RateLimitModel represents a rate limit entry in the database.
type RateLimitModel struct {
	db *sql.DB
	ID int64 `json:"id"`
	// number of visits
	Count int64  `json:"count"`
	IP    string `json:"ip"`
	// request url
	URI string `json:"uri"`
	// max time allowed
	EndAt     time.Time `json:"end_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func logErr(msg, errMsg string, exit bool) {
	name := "mdw/ratelimit"
	template := "%s: %s\n- %s"
	if exit {
		log.Fatalf(template, name, msg, errMsg)
	} else {
		log.Printf(template, name, msg, errMsg)
	}
}

func InitRatelimit(db *sql.DB) error {
	tableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			count INTEGER NOT NULL DEFAULT 1,
			ip TEXT NOT NULL,
			uri TEXT NOT NULL,
			end_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`, tableName)
	triggerQuery := fmt.Sprintf(`
		CREATE TRIGGER updated_at_update
		AFTER UPDATE ON %s
		FOR EACH ROW
		BEGIN
			UPDATE %s
			SET updated_at = CURRENT_TIMESTAMP
			WHERE id = old.id;
		END;
	`, tableName, tableName)

	var (
		trx *sql.Tx
		err error
	)
	trx, err = db.Begin()
	_, err = trx.Exec(tableQuery)
	_, err = trx.Exec(triggerQuery)
	err = trx.Commit()

	return err
}
func newRatelimit(db *sql.DB) RateLimitModel {
	return RateLimitModel{db: db}
}
func (r *RateLimitModel) save(ip, uri string, endAt time.Time) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (ip, uri, end_at)
		VALUES (?, ?, ?)
		RETURNING id, ip, uri, count, end_at, created_at, updated_at;
	`, tableName)
	return r.db.QueryRow(query, ip, uri, endAt).Scan(&r.ID, &r.IP, &r.URI, &r.Count, &r.EndAt, &r.CreatedAt, &r.UpdatedAt)
}
func (r RateLimitModel) findByIP(ip string) (RateLimitModel, error) {
	query := fmt.Sprintf(`
		SELECT id, ip, uri, count, end_at, created_at, updated_at
		FROM %s
		WHERE ip = ?
	`, tableName)

	// Link the db connection so it isn't nil
	rl := RateLimitModel{db: r.db}

	err := rl.db.QueryRow(query, ip).Scan(&rl.ID, &rl.IP, &rl.URI, &rl.Count, &rl.EndAt, &rl.CreatedAt, &rl.UpdatedAt)
	return rl, err
}

func (r *RateLimitModel) switchByIP(ip string) error {
	query := fmt.Sprintf(`
		SELECT id, ip, uri, count, end_at, created_at, updated_at
		FROM %s
		WHERE ip = ?
	`, tableName)
	return r.db.QueryRow(query, ip).Scan(&r.ID, &r.IP, &r.URI, &r.Count, &r.EndAt, &r.CreatedAt, &r.UpdatedAt)
}
func (r *RateLimitModel) restart(newEndAt time.Time) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET count = 1, end_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		RETURNING count, end_at, updated_at;
	`, tableName)

	// Persist to database AND update the receiver object in one step
	return r.db.QueryRow(query, newEndAt, r.ID).Scan(&r.Count, &r.EndAt, &r.UpdatedAt)
}
func (r *RateLimitModel) incrementCount() error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET count = count + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		RETURNING count, updated_at;
	`, tableName)

	// Persist to database AND update the receiver object in one step
	return r.db.QueryRow(query, r.ID).Scan(&r.Count, &r.UpdatedAt)
}

// Ratelimit checks if the number of requests from an IP address exceeds the maximum allowed number.
// If the number of requests exceeds the maximum allowed number, it will return an http.StatusTooManyRequests error.
//
// Example:
//
//	handler := Ratelimit(
//		db,
//		10,
//		time.Minute,
//		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			fmt.Println("Handler only runs when not rate limited")
//		}),
//	)
func Ratelimit(db *sql.DB, max int, period time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			URI   = r.RequestURI
			now   = time.Now()
			endAt = now.Add(period)
		)

		IP, _, hasIP := strings.Cut(r.RemoteAddr, ":")
		if !hasIP {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		ratelimit := newRatelimit(db)
		existing, err := ratelimit.findByIP(IP)

		// SCENARIO 1: Brand new IP (Insert record)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			if err := ratelimit.save(IP, URI, endAt); err != nil {
				logErr("failed to create rate limit", err.Error(), false)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			// ratelimit was updated in-place via .save()'s RETURNING clause
			ctx := context.WithValue(r.Context(), RatelimitCtxKey, ratelimit)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// SCENARIO 2: Database Query Failure
		if err != nil {
			logErr("failed to find rate limit", err.Error(), false)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// SCENARIO 3: Rate limit window expired (Reset window and count)
		if now.After(existing.EndAt) {
			if err := existing.restart(endAt); err != nil {
				logErr("failed to restart count", err.Error(), false)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			ctx := context.WithValue(r.Context(), RatelimitCtxKey, existing)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// SCENARIO 4: Rate limit hit max allowance (Block request)
		if existing.Count >= int64(max) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// SCENARIO 5: Within limit window (Increment count)
		if err := existing.incrementCount(); err != nil {
			logErr("failed to increment count", err.Error(), false)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), RatelimitCtxKey, existing)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
