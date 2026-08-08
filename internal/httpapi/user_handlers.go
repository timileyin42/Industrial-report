package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

func parseUserIDParam(c echo.Context) (int64, error) {
	return strconv.ParseInt(c.Param("user_id"), 10, 64)
}

type createUserRequest struct {
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Role     domain.Role `json:"role"`
	SiteID   *string     `json:"site_id,omitempty"`
}

type userResponse struct {
	ID         int64      `json:"id"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	SiteID     *string    `json:"site_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	DisabledAt *time.Time `json:"disabled_at,omitempty"`
}

func toUserResponse(u db.User) userResponse {
	return userResponse{
		ID:         u.ID,
		Email:      u.Email,
		Role:       string(u.Role),
		SiteID:     textPtr(u.SiteID),
		CreatedAt:  u.CreatedAt.Time,
		DisabledAt: timestamptzPtr(u.DisabledAt),
	}
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

	return c.JSON(http.StatusCreated, toUserResponse(user))
}

func (h *handlers) listUsers(c echo.Context) error {
	users, next, err := h.deps.Users.List(c.Request().Context(), c.QueryParam("cursor"), parseLimit(c))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	items := make([]userResponse, 0, len(users))
	for _, u := range users {
		items = append(items, toUserResponse(u))
	}
	return c.JSON(http.StatusOK, pageResponse[userResponse]{Items: items, NextCursor: next})
}

type setUserDisabledRequest struct {
	Disabled bool `json:"disabled"`
}

func (h *handlers) setUserDisabled(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	var req setUserDisabledRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	targetID, err := parseUserIDParam(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	user, err := h.deps.Users.SetDisabled(c.Request().Context(), claims.UserID, targetID, req.Disabled)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	return c.JSON(http.StatusOK, toUserResponse(user))
}
