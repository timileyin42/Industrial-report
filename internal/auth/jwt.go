package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/timileyin42/zgnis-solar/internal/domain"
)

const tokenTTL = 24 * time.Hour

// Claims is what every authenticated request carries: who the caller is,
// their role, and — for a restricted user — the one site they're scoped to.
type Claims struct {
	UserID int64       `json:"user_id"`
	Role   domain.Role `json:"role"`
	SiteID *string     `json:"site_id,omitempty"`
	jwt.RegisteredClaims
}

type TokenIssuer struct {
	secret []byte
}

func NewTokenIssuer(secret string) TokenIssuer {
	return TokenIssuer{secret: []byte(secret)}
}

func (t TokenIssuer) Issue(userID int64, role domain.Role, siteID *string) (string, time.Time, error) {
	expiresAt := time.Now().Add(tokenTTL)
	claims := Claims{
		UserID: userID,
		Role:   role,
		SiteID: siteID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secret)
	return signed, expiresAt, err
}

func (t TokenIssuer) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(tok *jwt.Token) (interface{}, error) {
		return t.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
