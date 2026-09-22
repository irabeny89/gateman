package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/irabeny89/gateman/jwt"
)

const AuthJWTCtxKey = "authJWTPayload"
// RequireAuthWithJWT returns an http.Handler that requires a valid JWT token in the Authorization header.
//
// Roles can be added for access control.
//
// If the token is valid, the auth payload will be stored in the context and can be retrieved using r.Context().Value(authPayloadKey).
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
	checkRoles := func(allowList, userRoles []string) bool {
		for _, allow := range allowList {
			if slices.Contains(userRoles, allow) {
				return true
			}
		}
		return false
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := jwt.ParseJWT(secret, token)
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
		ctx := context.WithValue(r.Context(), AuthJWTCtxKey, claims.AuthPayload)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
