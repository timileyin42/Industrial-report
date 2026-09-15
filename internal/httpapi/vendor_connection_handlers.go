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
	if _, ok := h.deps.ProviderRegistry.Get(req.Provider); !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "unknown provider")
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
