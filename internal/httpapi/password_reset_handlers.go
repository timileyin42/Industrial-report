package httpapi

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type requestPasswordResetRequest struct {
	Email string `json:"email"`
}

// requestPasswordReset always returns 202 regardless of whether the email
// matched a user — see PasswordReset.Request's comment on why (never
// leak account existence via response differences).
func (h *handlers) requestPasswordReset(c echo.Context) error {
	var req requestPasswordResetRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}

	if err := h.deps.PasswordReset.Request(c.Request().Context(), req.Email); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "couldn't process the request")
	}
	return c.NoContent(http.StatusAccepted)
}

type confirmPasswordResetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *handlers) confirmPasswordReset(c echo.Context) error {
	var req confirmPasswordResetRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "token is required")
	}

	if err := h.deps.PasswordReset.Confirm(c.Request().Context(), req.Token, req.Password); err != nil {
		if err == registry.ErrInvalidOrExpiredResetToken {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
