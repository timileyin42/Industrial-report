package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestOAuthStateRoundTrip(t *testing.T) {
	issuer := NewTokenIssuer("test-secret")

	state, err := issuer.IssueOAuthState("SELF-abc123", "elinter_csp")
	if err != nil {
		t.Fatalf("IssueOAuthState: %v", err)
	}

	claims, err := issuer.ParseOAuthState(state)
	if err != nil {
		t.Fatalf("ParseOAuthState: %v", err)
	}
	if claims.SiteID != "SELF-abc123" || claims.Provider != "elinter_csp" {
		t.Fatalf("got site=%q provider=%q, want SELF-abc123/elinter_csp", claims.SiteID, claims.Provider)
	}
}

// A real session token (same secret, same signing method) must never
// be usable as OAuth state — Purpose is what keeps the two apart.
func TestOAuthStateRejectsSessionToken(t *testing.T) {
	issuer := NewTokenIssuer("test-secret")
	siteID := "SELF-abc123"
	sessionToken, _, err := issuer.Issue(1, "restricted", &siteID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := issuer.ParseOAuthState(sessionToken); err == nil {
		t.Fatal("expected a real session token to be rejected as OAuth state, got no error")
	}
}

func TestOAuthStateRejectsExpired(t *testing.T) {
	issuer := NewTokenIssuer("test-secret")
	claims := OAuthStateClaims{
		SiteID: "SELF-abc123", Provider: "elinter_csp", Purpose: oauthStatePurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-20 * time.Minute)),
		},
	}
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(issuer.secret)
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	if _, err := issuer.ParseOAuthState(expired); err == nil {
		t.Fatal("expected an expired state to be rejected, got no error")
	}
}

func TestOAuthStateRejectsDifferentSecret(t *testing.T) {
	issuer := NewTokenIssuer("test-secret")
	other := NewTokenIssuer("a-different-secret")

	state, err := other.IssueOAuthState("SELF-abc123", "elinter_csp")
	if err != nil {
		t.Fatalf("IssueOAuthState: %v", err)
	}
	if _, err := issuer.ParseOAuthState(state); err == nil {
		t.Fatal("expected state signed with a different secret to be rejected, got no error")
	}
}
