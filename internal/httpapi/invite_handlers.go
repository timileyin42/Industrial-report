package httpapi

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type createInviteRequest struct {
	Email  string      `json:"email"`
	Role   domain.Role `json:"role"`
	SiteID *string     `json:"site_id,omitempty"`
}

// createInvite is the alternative to createUser (POST /v1/users, still
// unchanged) — operator supplies role/site, the invitee sets their own
// password via acceptInvite instead of the operator choosing one.
func (h *handlers) createInvite(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	var req createInviteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	user, err := h.deps.Invites.Create(c.Request().Context(), claims.UserID, registry.CreateInviteInput{
		Email:  req.Email,
		Role:   req.Role,
		SiteID: req.SiteID,
	})
	if err != nil {
		if err == registry.ErrUnknownSite {
			return echo.NewHTTPError(http.StatusBadRequest, "site does not exist")
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, userResponse{
		Email:  user.Email,
		Role:   domain.Role(user.Role),
		SiteID: textPtr(user.SiteID),
	})
}

type acceptInviteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// acceptInvite is public (no auth) by necessity — the invitee has no
// session yet, only the token from their email link. loginLimiter-style
// rate limiting is applied at the route (router.go), same reasoning as
// login/password-reset.
func (h *handlers) acceptInvite(c echo.Context) error {
	var req acceptInviteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "token is required")
	}

	if err := h.deps.Invites.Accept(c.Request().Context(), req.Token, req.Password); err != nil {
		if err == registry.ErrInvalidOrExpiredInvite {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
