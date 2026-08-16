package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/irabeny89/gateman"
	"github.com/irabeny89/gateman/jwt"
)

type ctxKey string

const authPayloadKey ctxKey = "authPayload"

// RequireAuthWithJWT returns an http.Handler that requires a valid JWT token in the Authorization header.
//
// If the token is valid, the AuthPayload will be stored in the context and can be retrieved using r.Context().Value(authPayloadKey).
//
// Example:
//
//	 handler := RequireAuthWithJWT(
//			os.Getenv("JWT_SECRET"),
//			[]string{"admin"},
//			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//				fmt.Println("Admin only route")
//				authPayload := r.Context().Value(authPayloadKey).(AuthPayload)
//				fmt.Println("User ID:", authPayload.UserID)
//				fmt.Println("Email:", authPayload.Email)
//				fmt.Println("Roles:", authPayload.Roles)
//			}),
//	 )
func RequireAuthWithJWT(secret string, roles []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := jwt.ParseJWT(secret, token)
		checkRoles := func(allowList, userRoles []string) bool {
			for _, allow := range allowList {
				if slices.Contains(userRoles, allow) {
					return true
				}
			}
			return false
		}
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if len(roles) != 0 && !checkRoles(roles, claims.Roles) {
			log.Println(errors.New("Invalid roles"))
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), authPayloadKey, claims.AuthPayload)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
func Ratelimit(db *gateman.SQLite, max int, period time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		restart := func(IP string) error {
			_, err := db.Exec(`
				INSERT INTO rate_limits (ip, uri, count, end_at) 
				VALUES (?, ?, ?, ?)
				ON CONFLICT(ip, uri) DO UPDATE SET count = 1, end_at = excluded.end_at
			`, IP, r.RequestURI, 1, time.Now().Add(period))
			return err
		}
		handleErr := func(err error) {
			log.Println(err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		var (
			URI    string
			count  int
			end_at time.Time
		)
		IP, _, _ := strings.Cut(r.RemoteAddr, ":")
		rows, err := db.Query(
			`
			SELECT uri, count, end_at
			FROM rate_limits 
			WHERE ip = ?
			AND uri = ?
			LIMIT 1;
			`,
			IP, r.RequestURI,
		)
		if err != nil {
			handleErr(err)
			return
		}
		defer rows.Close()
		if rows.Next() { //* found
			rows.Scan(&URI, &count, &end_at)
			rows.Close()
			if count >= max {
				if time.Now().After(end_at) {
					if err := restart(IP); err != nil {
						handleErr(err)
						return
					}
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			_, err := db.Exec(`
				UPDATE rate_limits
				SET count = count + 1
				WHERE ip = ?
				AND uri = ?
			`, IP, r.RequestURI)
			if err != nil {
				handleErr(err)
				return
			}
			next.ServeHTTP(w, r)
		} else { // if IP not seen before, add it to the rate limit table
			rows.Close()
			if err := restart(IP); err != nil {
				handleErr(err)
				return
			}
			next.ServeHTTP(w, r)
		}
	})
}
