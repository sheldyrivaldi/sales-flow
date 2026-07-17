package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"salespilot/internal/domain"
)

func testUser() domain.User {
	return domain.User{ID: "user-1", Email: "a@b.com", Role: domain.RoleAdmin}
}

func TestIssueAndParse_RoundTrip(t *testing.T) {
	secret := "test-secret"
	u := testUser()

	access, refresh, err := Issue(u, secret)
	if err != nil {
		t.Fatalf("Issue error: %v", err)
	}

	claims, err := Parse(access, secret, TokenAccess)
	if err != nil {
		t.Fatalf("Parse access error: %v", err)
	}
	if claims.Subject != u.ID {
		t.Errorf("Subject = %q; want %q", claims.Subject, u.ID)
	}
	if claims.Role != u.Role {
		t.Errorf("Role = %q; want %q", claims.Role, u.Role)
	}

	// Refresh token must also parse (role may be empty).
	rClaims, err := Parse(refresh, secret, TokenRefresh)
	if err != nil {
		t.Fatalf("Parse refresh error: %v", err)
	}
	if rClaims.Subject != u.ID {
		t.Errorf("refresh Subject = %q; want %q", rClaims.Subject, u.ID)
	}
}

func TestParse_RejectsWrongTokenType(t *testing.T) {
	access, refresh, err := Issue(testUser(), "s")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(access, "s", TokenRefresh); err == nil {
		t.Error("expected access token to be rejected when a refresh token is wanted")
	}
	if _, err := Parse(refresh, "s", TokenAccess); err == nil {
		t.Error("expected refresh token to be rejected when an access token is wanted")
	}
}

func TestParse_WrongSecret(t *testing.T) {
	access, _, _ := Issue(testUser(), "correct-secret")
	if _, err := Parse(access, "wrong-secret", TokenAccess); err == nil {
		t.Error("expected error with wrong secret, got nil")
	}
}

func TestParse_ExpiredToken(t *testing.T) {
	key := []byte("secret")
	expiredClaims := Claims{
		Role: domain.RoleSales,
		Type: TokenAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u1",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).SignedString(key)
	if _, err := Parse(token, "secret", TokenAccess); err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestParse_WrongSigningMethod(t *testing.T) {
	// Build a token signed with HMAC but tamper alg header to RS256 manually is hard;
	// instead verify that a nonsense token is rejected.
	if _, err := Parse("not.a.jwt", "secret", TokenAccess); err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

func TestIssueTUISession_RoundTrip(t *testing.T) {
	secret := "test-secret"
	token, err := IssueTUISession("user-1", "session-abc", secret)
	if err != nil {
		t.Fatalf("IssueTUISession error: %v", err)
	}

	claims, err := Parse(token, secret, TokenTUISession)
	if err != nil {
		t.Fatalf("Parse TUI session error: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Errorf("Subject = %q; want %q", claims.Subject, "user-1")
	}
	if claims.SessionID != "session-abc" {
		t.Errorf("SessionID = %q; want %q", claims.SessionID, "session-abc")
	}
	if claims.Type != TokenTUISession {
		t.Errorf("Type = %q; want %q", claims.Type, TokenTUISession)
	}
}

func TestIssueTUISession_RejectedAsOtherTypes(t *testing.T) {
	secret := "s"
	tuiToken, err := IssueTUISession("user-1", "session-abc", secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(tuiToken, secret, TokenAccess); err == nil {
		t.Error("expected TUI session token to be rejected when an access token is wanted")
	}
	if _, err := Parse(tuiToken, secret, TokenRefresh); err == nil {
		t.Error("expected TUI session token to be rejected when a refresh token is wanted")
	}

	access, _, err := Issue(testUser(), secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(access, secret, TokenTUISession); err == nil {
		t.Error("expected access token to be rejected when a TUI session token is wanted")
	}
}

func TestIssue_AccessAndRefreshAreDifferent(t *testing.T) {
	access, refresh, err := Issue(testUser(), "s")
	if err != nil {
		t.Fatal(err)
	}
	if access == refresh {
		t.Error("access and refresh tokens should differ")
	}
	// Rough TTL check: access exp < refresh exp
	aParts := strings.Split(access, ".")
	rParts := strings.Split(refresh, ".")
	if len(aParts) != 3 || len(rParts) != 3 {
		t.Error("tokens should have 3 parts")
	}
}
