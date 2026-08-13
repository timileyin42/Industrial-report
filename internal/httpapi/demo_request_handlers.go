package httpapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type createDemoRequestBody struct {
	Organization string `json:"organization"`
	Email        string `json:"email"`
}

// createDemoRequest is public (no auth) — see router.go's comment on
// this route group. Always responds 201 on valid input even if the
// underlying email sends failed: registry.DemoRequests.Submit logs send
// failures itself, and a prospect filling out a form has no account to
// blame a 500 on — same "don't leak infra failures to an anonymous
// caller" reasoning as password-reset's request handler.
func (h *handlers) createDemoRequest(c echo.Context) error {
	var body createDemoRequestBody
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.deps.DemoRequests.Submit(c.Request().Context(), body.Organization, body.Email); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusCreated)
}
