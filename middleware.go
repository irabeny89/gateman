package gateman

import (
	"context"
	"net/http"
	"slices"
	"strings"
)

type ctxKey string

const authPayloadKey ctxKey = "authPayload"

// checkRoles check if a user role is in an allow list.
// Return true if in allow list and false otherwise.
func checkRoles(allowList, userRoles []string) bool {
	for _, allow := range allowList {
		if slices.Contains(userRoles, allow) {
			return true
		}
	}
	return false
}

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
		claims, err := ParseJWT(secret, token)

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if len(roles) != 0 && !checkRoles(roles, claims.Roles) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), authPayloadKey, claims.AuthPayload)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
