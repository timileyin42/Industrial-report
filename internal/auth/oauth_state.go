package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// oauthStateTTL is short — this only needs to survive one redirect
// round trip to the vendor's own consent screen and back, not a real
// session.
const oauthStateTTL = 10 * time.Minute

const oauthStatePurpose = "vendor_oauth_state"

// OAuthStateClaims binds an OAuth callback back to the site and
// provider that started it. The vendor's own redirect is a plain
// browser navigation, not an API call — it carries no Authorization
// header — so this signed value is the only thing the callback
// handler can trust to know which site's connection this is for.
//
// Purpose distinguishes this from a real session token minted by the
// same TokenIssuer/secret (see jwt.go's Claims), so one can never be
// replayed as the other even though both are HS256 JWTs sharing a key.
type OAuthStateClaims struct {
	SiteID   string `json:"site_id"`
	Provider string `json:"provider"`
	Purpose  string `json:"purpose"`
	jwt.RegisteredClaims
}

// IssueOAuthState mints the short-lived, signed state parameter for
// one Connect-Your-Inverter OAuth attempt.
func (t TokenIssuer) IssueOAuthState(siteID, provider string) (string, error) {
	claims := OAuthStateClaims{
		SiteID: siteID, Provider: provider, Purpose: oauthStatePurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(oauthStateTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.secret)
}

// ParseOAuthState verifies a callback's state parameter and returns
// which site/provider it belongs to. Rejects an expired state, a bad
// signature, and — critically — a valid session token presented here
// instead (Purpose won't match), since both are signed with the same
// secret.
func (t TokenIssuer) ParseOAuthState(state string) (*OAuthStateClaims, error) {
	claims := &OAuthStateClaims{}
	token, err := jwt.ParseWithClaims(state, claims, func(tok *jwt.Token) (interface{}, error) {
		return t.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.Purpose != oauthStatePurpose {
		return nil, errors.New("invalid or expired oauth state")
	}
	return claims, nil
}
