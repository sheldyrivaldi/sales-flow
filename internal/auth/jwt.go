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

type Claims struct {
	Role domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// Issue creates an access token and a refresh token for the given user.
func Issue(u domain.User, secret string) (access, refresh string, err error) {
	key := []byte(secret)
	now := time.Now()

	accessClaims := Claims{
		Role: u.Role,
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

// Parse validates a signed JWT string and returns its Claims.
// Rejects tokens signed with any method other than HMAC.
func Parse(token, secret string) (*Claims, error) {
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
	return claims, nil
}
