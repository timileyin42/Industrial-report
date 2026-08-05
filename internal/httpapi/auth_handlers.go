package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	Role      domain.Role `json:"role"`
	SiteID    *string     `json:"site_id,omitempty"`
}

func (h *handlers) login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	user, err := h.deps.Users.Authenticate(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		if err == registry.ErrInvalidCredentials {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	role := domain.Role(user.Role)
	siteID := textPtr(user.SiteID)

	token, expiresAt, err := h.deps.Issuer.Issue(user.ID, role, siteID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
	}

	h.deps.Users.RecordLogin(c.Request().Context(), user.ID)

	return c.JSON(http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Role:      role,
		SiteID:    siteID,
	})
}
