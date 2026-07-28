package gateman

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT claims data
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Auth handles JWT authentication
type Auth struct {
	secret   []byte
	repo     UserRepository
	tokenTTL time.Duration
}

// New creates a new Auth instance
func New(secret string, repo UserRepository) *Auth {
	return &Auth{
		secret:   []byte(secret),
		repo:     repo,
		tokenTTL: 24 * time.Hour,
	}
}

// GenerateToken generates a JWT token for the given user
func (a *Auth) GenerateToken(user *User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secret)
}
