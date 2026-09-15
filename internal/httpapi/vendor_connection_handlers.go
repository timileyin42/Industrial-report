package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/registry"
	"github.com/timileyin42/zgnis-solar/internal/syncengine"
)

type vendorProviderResponse struct {
	Name         string                  `json:"name"`
	DisplayName  string                  `json:"display_name"`
	AuthType     string                  `json:"auth_type"`
	Capabilities syncengine.Capabilities `json:"capabilities"`
}

// listVendorProviders feeds the Connect-Your-Inverter vendor picker —
// public (no login needed to see what's supported), never exposes
// anything credential-related.
func (h *handlers) listVendorProviders(c echo.Context) error {
	providers := h.deps.ProviderRegistry.List()
	out := make([]vendorProviderResponse, 0, len(providers))
	for _, p := range providers {
		out = append(out, vendorProviderResponse{
			Name: p.Name(), DisplayName: p.DisplayName(),
			AuthType: p.AuthType(), Capabilities: p.Capabilities(),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"providers": out})
}

type createVendorConnectionRequest struct {
	Provider string `json:"provider"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type vendorConnectionResponse struct {
	ID           int64      `json:"id"`
	Provider     string     `json:"provider"`
	Status       string     `json:"status"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	LastError    *string    `json:"last_error,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func toVendorConnectionResponse(v registry.VendorConnection) vendorConnectionResponse {
	return vendorConnectionResponse{
		ID: v.ID, Provider: v.Provider, Status: v.Status,
		LastSyncedAt: v.LastSyncedAt, LastError: v.LastError, CreatedAt: v.CreatedAt,
	}
}

// createVendorConnection is the "Connect Your Inverter" submit — site-
// scoped, not operator-only: this is the customer connecting *their
// own* account to the site they just signed up with, enforced by
// RequireSiteAccess exactly like every other site-scoped endpoint.
func (h *handlers) createVendorConnection(c echo.Context) error {
	claims, _ := auth.GetClaims(c)
	siteID := c.Param("site_id")

	var req createVendorConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	provider, ok := h.deps.ProviderRegistry.Get(req.Provider)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "unknown provider")
	}
	if provider.AuthType() != syncengine.AuthTypePassword {
		return echo.NewHTTPError(http.StatusBadRequest, "this vendor requires the OAuth connect flow, not a password — see POST .../vendor-connections/oauth/start")
	}

	conn, err := h.deps.VendorConnections.Create(c.Request().Context(), claims.UserID, registry.CreateConnectionInput{
		SiteID: siteID, Provider: req.Provider, Email: req.Email, Password: req.Password,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, toVendorConnectionResponse(conn))
}

func (h *handlers) listVendorConnections(c echo.Context) error {
	conns, err := h.deps.VendorConnections.ListForSite(c.Request().Context(), c.Param("site_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	out := make([]vendorConnectionResponse, 0, len(conns))
	for _, v := range conns {
		out = append(out, toVendorConnectionResponse(v))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": out})
}

type startOAuthRequest struct {
	Provider string `json:"provider"`
}
type startOAuthResponse struct {
	AuthorizationURL string `json:"authorization_url"`
}

// startVendorOAuth begins the OAuth connect flow for a vendor with no
// password-login API (see syncengine.OAuthProvider's doc comment) —
// the OAuth counterpart to createVendorConnection. Site-scoped like
// every other vendor-connections route: the customer is starting a
// connection for *their own* site.
func (h *handlers) startVendorOAuth(c echo.Context) error {
	siteID := c.Param("site_id")
	var req startOAuthRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	provider, ok := h.deps.ProviderRegistry.Get(req.Provider)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "unknown provider")
	}
	oauthProvider, ok := provider.(syncengine.OAuthProvider)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "this vendor doesn't use OAuth — see POST .../vendor-connections")
	}
	state, err := h.deps.Issuer.IssueOAuthState(siteID, provider.Name())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "couldn't start the connection, try again")
	}
	return c.JSON(http.StatusOK, startOAuthResponse{
		AuthorizationURL: oauthProvider.AuthorizationURL(state, oauthRedirectURI(h.deps.APIPublicBaseURL, provider.Name())),
	})
}

// vendorOAuthCallback is where a vendor's own consent screen redirects
// the customer's browser back to, after they approve or deny access.
// Public — this is a plain browser navigation from the vendor's own
// domain, never called by this app's SPA, so it carries no
// Authorization header at all. The signed state parameter (see
// auth.OAuthStateClaims) is the only thing tying this request back to
// a site — the :provider path segment alone is never trusted for that.
//
// Always ends in a redirect back into the SPA (never a JSON error
// response) — there's no script on this page to read one; the
// frontend reads the outcome from the query string it lands on
// instead, same as e.g. AcceptInvitePage reads its token from one.
func (h *handlers) vendorOAuthCallback(c echo.Context) error {
	providerName := c.Param("provider")
	frontendURL := h.deps.AppBaseURL + "/connect-inverter"

	if errParam := c.QueryParam("error"); errParam != "" {
		// The customer declined consent on the vendor's own screen —
		// not a bug on either side.
		return c.Redirect(http.StatusFound, frontendURL+"?oauth_error=denied")
	}

	claims, err := h.deps.Issuer.ParseOAuthState(c.QueryParam("state"))
	if err != nil || claims.Provider != providerName {
		return c.Redirect(http.StatusFound, frontendURL+"?oauth_error=invalid_state")
	}

	provider, ok := h.deps.ProviderRegistry.Get(providerName)
	if !ok {
		return c.Redirect(http.StatusFound, frontendURL+"?oauth_error=unknown_provider")
	}
	oauthProvider, ok := provider.(syncengine.OAuthProvider)
	if !ok {
		return c.Redirect(http.StatusFound, frontendURL+"?oauth_error=unknown_provider")
	}

	ctx := c.Request().Context()
	token, err := oauthProvider.ExchangeCode(ctx, c.QueryParam("code"), oauthRedirectURI(h.deps.APIPublicBaseURL, providerName))
	if err != nil {
		return c.Redirect(http.StatusFound, frontendURL+"?oauth_error=exchange_failed")
	}
	if _, err := h.deps.VendorConnections.CreateFromOAuth(ctx, claims.SiteID, providerName, token); err != nil {
		return c.Redirect(http.StatusFound, frontendURL+"?oauth_error=store_failed")
	}
	return c.Redirect(http.StatusFound, frontendURL+"?connected=1")
}

// oauthRedirectURI must be byte-for-byte identical between the start
// call (AuthorizationURL) and the exchange call (ExchangeCode) — most
// vendors' OAuth2 implementations require that — and must also be
// byte-for-byte what's registered in that vendor's own developer
// console, which is only possible to get exactly right once a real
// vendor app exists to register it with.
func oauthRedirectURI(apiPublicBaseURL, providerName string) string {
	return apiPublicBaseURL + "/v1/vendor-connections/oauth/callback/" + providerName
}

func (h *handlers) revokeVendorConnection(c echo.Context) error {
	claims, _ := auth.GetClaims(c)
	id, err := parseInt64Param(c, "connection_id")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid connection_id")
	}
	if err := h.deps.VendorConnections.Revoke(c.Request().Context(), claims.UserID, c.Param("site_id"), id); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
