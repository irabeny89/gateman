package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/irabeny89/gateman"
)

// JWTRegisteredClaimsOptions is a struct that holds the options for creating JWT registered claims.
//
// Example (Method 1: Using methods)
//
//	var opt1 JWTRegisteredClaimsOptions
//	opt1.WithIssuer("my-app")
//	opt1.WithSubject("user-id")
//	opt1.WithAudience([]string{"my-app"})
//	opt1.WithExpiresAt(24 * time.Hour)
//	opt1.WithIssuedAt(time.Now())
//	opt1.WithNotBefore(time.Now())
//	opt1.WithJTI("1234567890")
//
// Example (Method 2: Using struct literal)
//
//	 opt2 := JWTRegisteredClaimsOptions{
//			Issuer: "my-app",
//			Subject: "user-id",
//			Audience: []string{"my-app"},
//			ExpiresAt: 24 * time.Hour,
//			IssuedAt: time.Now(),
//			NotBefore: time.Now(),
//			JTI: "1234567890",
//		}
type JWTRegisteredClaimsOptions struct {
	Aud       []string
	ExpiresAt time.Duration
	IssuedAt  time.Time
	Issuer    string
	JTI       string
	NotBefore time.Time
	Subject   string
}

func (r *JWTRegisteredClaimsOptions) WithAudience(aud []string) *JWTRegisteredClaimsOptions {
	r.Aud = aud
	return r
}

func (r *JWTRegisteredClaimsOptions) WithExpiresAt(exp time.Duration) *JWTRegisteredClaimsOptions {
	r.ExpiresAt = exp
	return r
}

func (r *JWTRegisteredClaimsOptions) WithIssuedAt(iat time.Time) *JWTRegisteredClaimsOptions {
	r.IssuedAt = iat
	return r
}

func (r *JWTRegisteredClaimsOptions) WithIssuer(iss string) *JWTRegisteredClaimsOptions {
	r.Issuer = iss
	return r
}

func (r *JWTRegisteredClaimsOptions) WithJTI(jti string) *JWTRegisteredClaimsOptions {
	r.JTI = jti
	return r
}

func (r *JWTRegisteredClaimsOptions) WithNotBefore(nbf time.Time) *JWTRegisteredClaimsOptions {
	r.NotBefore = nbf
	return r
}

func (r *JWTRegisteredClaimsOptions) WithSubject(sub string) *JWTRegisteredClaimsOptions {
	r.Subject = sub
	return r
}

// NewJWTRegisteredClaims creates a new JWTRegisteredClaims instance with the provided options.
//
// Default values:
//
//	IssuedAt: time.Now()
//	NotBefore: time.Now()
//	ExpiresAt: 24 * time.Hour
func NewJWTRegisteredClaims(opt JWTRegisteredClaimsOptions) jwt.RegisteredClaims {
	now := time.Now()
	if opt.IssuedAt.IsZero() {
		opt.IssuedAt = now
	}
	if opt.NotBefore.IsZero() {
		opt.NotBefore = now
	}
	if opt.ExpiresAt == 0 {
		opt.ExpiresAt = 24 * time.Hour
	}
	return jwt.RegisteredClaims{
		Audience:  opt.Aud,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(opt.ExpiresAt)),
		IssuedAt:  jwt.NewNumericDate(opt.IssuedAt),
		NotBefore: jwt.NewNumericDate(opt.NotBefore),
		Issuer:    opt.Issuer,
		ID:        opt.JTI,
		Subject:   opt.Subject,
	}
}

// JWTClaims is a custom claims type that embeds AuthPayload and jwt.RegisteredClaims.
//
// Use NewJWTRegisteredClaims() to create registered claims.
type JWTClaims struct {
	gateman.AuthPayload
	jwt.RegisteredClaims
}

// NewJWTClaims creates a new JWTClaims instance with the provided payload and registered claims.
//
// Use NewJWTRegisteredClaims() to create registered claims.
func NewJWTClaims(payload gateman.AuthPayload, regClaims jwt.RegisteredClaims) JWTClaims {
	return JWTClaims{
		AuthPayload:      payload,
		RegisteredClaims: regClaims,
	}
}

// GenerateJWT creates a new JWT token with the provided claims.
//
// Example (Method 1: Using struct literal)
//
//	 payload := gateman.AuthPayload{
//			UserID: "1",
//			Email:  "[EMAIL_ADDRESS]",
//			Roles:  []string{"user"},
//		}
//		regClaims := NewJWTRegisteredClaims(
//			JWTRegisteredClaimsOptions{
//				Issuer:    "my-app",
//				Subject:   "user-id",
//				Audience:  []string{"my-app"},
//				ExpiresAt: 24 * time.Hour,
//				IssuedAt:  time.Now(),
//				NotBefore: time.Now(),
//				JTI:       "1234567890",
//			},
//		)
//		claims := NewJWTClaims(payload, regClaims)
//		token, err := GenerateJWT(os.Getenv("JWT_SECRET"), claims)
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Println(token)
//
// Example (Method 2: Using methods)
//
//	 var opt JWTRegisteredClaimsOptions
//	 opt.WithIssuer("my-app").
//			WithSubject("user-id").
//			WithAudience([]string{"my-app"}).
//			WithExpiresAt(24 * time.Hour).
//			WithIssuedAt(time.Now()).
//			WithNotBefore(time.Now()).
//			WithJTI("1234567890")
//		claims := NewJWTClaims(
//			AuthPayload{
//				UserID: "1",
//				Email:  "[EMAIL_ADDRESS]",
//				Roles:  []string{"user"},
//			},
//			NewJWTRegisteredClaims(opt),
//		)
//		token, err := GenerateJWT(os.Getenv("JWT_SECRET"), claims)
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Println(token)
func GenerateJWT(secret string, claims JWTClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseJWT parses a JWT token with the provided secret.
//
// Example:
//
//	claims, err := ParseJWT(os.Getenv("JWT_SECRET"), token)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(claims.AuthPayload)
func ParseJWT(secret string, tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}
