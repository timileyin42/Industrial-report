package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/domain"
)

type contextKey string

const claimsContextKey contextKey = "claims"

// GetClaims reads the authenticated caller's claims. Only valid on routes
// behind RequireAuth — callers elsewhere will get ok=false.
func GetClaims(c echo.Context) (*Claims, bool) {
	claims, ok := c.Get(string(claimsContextKey)).(*Claims)
	return claims, ok
}

// RequireAuth verifies the bearer JWT and attaches its claims to the
// request context. 401 on anything missing/invalid/expired.
func RequireAuth(issuer TokenIssuer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(header, prefix) {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
			}
			claims, err := issuer.Parse(strings.TrimPrefix(header, prefix))
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}
			c.Set(string(claimsContextKey), claims)
			return next(c)
		}
	}
}

// RequireRole 403s any caller whose role isn't in the allowed set. Must run
// after RequireAuth.
func RequireRole(roles ...domain.Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := GetClaims(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing claims")
			}
			for _, r := range roles {
				if claims.Role == r {
					return next(c)
				}
			}
			return echo.NewHTTPError(http.StatusForbidden, "insufficient role")
		}
	}
}

// RequireSiteAccess 403s a restricted user whose claims.SiteID doesn't match
// the site resolved by resolveSiteID for this request. Operators always
// pass. Must run after RequireAuth. Per CLAUDE.md, a mismatch is always a
// 403 — never a 404 that leaks existence, never a silently filtered 200.
func RequireSiteAccess(resolveSiteID func(c echo.Context) (string, error)) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := GetClaims(c)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing claims")
			}
			if claims.Role == domain.RoleOperator {
				return next(c)
			}
			targetSiteID, err := resolveSiteID(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusForbidden, "site access denied")
			}
			if claims.SiteID == nil || *claims.SiteID != targetSiteID {
				return echo.NewHTTPError(http.StatusForbidden, "site access denied")
			}
			return next(c)
		}
	}
}
