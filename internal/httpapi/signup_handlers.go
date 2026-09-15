package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signupResponse struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	Role      domain.Role `json:"role"`
	SiteID    string      `json:"site_id"`
}

// signup is the public self-service registration endpoint — distinct
// from the existing operator-only POST /v1/users. Creates a restricted
// user and a fresh site for them, then logs them in immediately (same
// response shape as login) so the frontend can go straight from signup
// into the Connect-Your-Inverter step without a second round trip.
func (h *handlers) signup(c echo.Context) error {
	var req signupRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	result, err := h.deps.Signup.SelfSignup(c.Request().Context(), registry.SelfSignupInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	siteID := result.Site.SiteID
	token, expiresAt, err := h.deps.Issuer.Issue(result.User.ID, domain.RoleRestricted, &siteID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
	}

	return c.JSON(http.StatusCreated, signupResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Role:      domain.RoleRestricted,
		SiteID:    siteID,
	})
}
