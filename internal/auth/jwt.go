package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"salespilot/internal/domain"
)

const (
	AccessTTL  = 15 * time.Minute
	RefreshTTL = 7 * 24 * time.Hour
)

// TokenType distinguishes access tokens from refresh tokens so one can't be
// replayed as the other — they share a signing key and are otherwise
// structurally identical.
type TokenType string

const (
	TokenAccess  TokenType = "access"
	TokenRefresh TokenType = "refresh"
)

type Claims struct {
	Role domain.Role `json:"role"`
	Type TokenType   `json:"typ"`
	jwt.RegisteredClaims
}

// Issue creates an access token and a refresh token for the given user.
func Issue(u domain.User, secret string) (access, refresh string, err error) {
	key := []byte(secret)
	now := time.Now()

	accessClaims := Claims{
		Role: u.Role,
		Type: TokenAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTTL)),
		},
	}
	access, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(key)
	if err != nil {
		return "", "", fmt.Errorf("auth.Issue access: %w", err)
	}

	refreshClaims := Claims{
		Type: TokenRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTTL)),
		},
	}
	refresh, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(key)
	if err != nil {
		return "", "", fmt.Errorf("auth.Issue refresh: %w", err)
	}

	return access, refresh, nil
}

// Parse validates a signed JWT string and returns its Claims, rejecting
// tokens whose Type does not match want (an access token cannot be used
// where a refresh token is expected, or vice versa). Rejects tokens signed
// with any method other than HMAC.
func Parse(token, secret string, want TokenType) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth.Parse: unexpected signing method %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth.Parse: %w", err)
	}
	if claims.Type != want {
		return nil, fmt.Errorf("auth.Parse: unexpected token type %q, want %q", claims.Type, want)
	}
	return claims, nil
}
