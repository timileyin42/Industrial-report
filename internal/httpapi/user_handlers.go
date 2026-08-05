package httpapi

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type createUserRequest struct {
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Role     domain.Role `json:"role"`
	SiteID   *string     `json:"site_id,omitempty"`
}

type userResponse struct {
	Email  string      `json:"email"`
	Role   domain.Role `json:"role"`
	SiteID *string     `json:"site_id,omitempty"`
}

func (h *handlers) createUser(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	var req createUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	user, err := h.deps.Users.Create(c.Request().Context(), claims.UserID, registry.CreateUserInput{
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
		SiteID:   req.SiteID,
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
