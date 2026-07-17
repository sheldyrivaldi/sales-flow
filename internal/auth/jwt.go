package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"salespilot/internal/domain"
)

const (
	AccessTTL     = 15 * time.Minute
	RefreshTTL    = 7 * 24 * time.Hour
	TUISessionTTL = 4 * time.Hour
)

// TokenType distinguishes access tokens from refresh tokens (and now TUI
// session tokens) so one can't be replayed as another — they share a
// signing key and are otherwise structurally identical.
type TokenType string

const (
	TokenAccess     TokenType = "access"
	TokenRefresh    TokenType = "refresh"
	TokenTUISession TokenType = "tui_session"
)

type Claims struct {
	Role      domain.Role `json:"role"`
	Type      TokenType   `json:"typ"`
	SessionID string      `json:"sid,omitempty"`
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

// IssueTUISession creates a session token for the admin Hermes TUI feature
// (see internal/hermestui). Unlike access/refresh tokens this is carried in
// an HttpOnly cookie, not an Authorization header — browsers attach cookies
// automatically to same-origin requests, including WebSocket handshakes,
// which is the whole reason this token type exists (see plan §Authentication
// matrix). sessionID ties the token to a live entry in hermestui.Registry.
func IssueTUISession(userID, sessionID, secret string) (string, error) {
	now := time.Now()
	claims := Claims{
		Type:      TokenTUISession,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TUISessionTTL)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("auth.IssueTUISession: %w", err)
	}
	return token, nil
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
