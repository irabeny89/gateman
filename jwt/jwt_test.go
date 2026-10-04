package jwt

import (
	"reflect"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken(t *testing.T) {
	payload := AuthPayload{
		UserID: "user-123",
		Email:  "test@example.com",
		Roles:  []string{"admin", "user"},
	}
	regClaims := NewJWTRegisteredClaims(JWTRegisteredClaimsOptions{
		ExpiresAt: 1 * time.Hour,
	})
	claims := JWTClaims{
		AuthPayload:      payload,
		RegisteredClaims: regClaims,
	}
	token, err := claims.GenerateToken("secret")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token, got empty string")
	}
}

func TestParseJWT(t *testing.T) {
	payload := AuthPayload{
		UserID: "user-123",
		Email:  "test@example.com",
		Roles:  []string{"admin", "user"},
	}
	regClaims := NewJWTRegisteredClaims(JWTRegisteredClaimsOptions{
		ExpiresAt: 1 * time.Hour,
	})
	claims := JWTClaims{
		AuthPayload:      payload,
		RegisteredClaims: regClaims,
	}
	token, err := claims.GenerateToken("secret")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	parsedClaims, err := ParseJWT("secret", token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(*parsedClaims, claims) {
		t.Fatalf("expected claims %v, got %v", claims, parsedClaims)
	}
}

func TestNewJWTRegisteredClaims(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		opt  JWTRegisteredClaimsOptions
		want jwt.RegisteredClaims
	}{
		{
			name: "with all options",
			opt: JWTRegisteredClaimsOptions{
				Aud:       []string{"my-app"},
				ExpiresAt: 24 * time.Hour,
				IssuedAt:  time.Now(),
				NotBefore: time.Now(),
				Issuer:    "my-app",
				JTI:       "1234567890",
				Subject:   "user-id",
			},
			want: jwt.RegisteredClaims{
				Audience:  []string{"my-app"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
				Issuer:    "my-app",
				ID:        "1234567890",
				Subject:   "user-id",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewJWTRegisteredClaims(tt.opt)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewJWTRegisteredClaims() = %v, want %v", got, tt.want)
			}
		})
	}
}
