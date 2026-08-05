package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/pagination"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type createSiteRequest struct {
	SiteID            string   `json:"site_id"`
	Name              string   `json:"name"`
	Address           *string  `json:"address,omitempty"`
	GPSLat            *float64 `json:"gps_lat,omitempty"`
	GPSLng            *float64 `json:"gps_lng,omitempty"`
	InverterMakeModel *string  `json:"inverter_make_model,omitempty"`
	SystemSizeKW      *float64 `json:"system_size_kw,omitempty"`
	Timezone          string   `json:"timezone"`
	CohortID          *string  `json:"cohort_id,omitempty"`
}

type pageResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

func (h *handlers) createSite(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	var req createSiteRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Timezone == "" {
		req.Timezone = "Africa/Lagos"
	}

	site, err := h.deps.Sites.Create(c.Request().Context(), claims.UserID, registry.CreateSiteInput{
		SiteID:            req.SiteID,
		Name:              req.Name,
		Address:           req.Address,
		GPSLat:            req.GPSLat,
		GPSLng:            req.GPSLng,
		InverterMakeModel: req.InverterMakeModel,
		SystemSizeKW:      req.SystemSizeKW,
		Timezone:          req.Timezone,
		CohortID:          req.CohortID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, toSiteResponse(site))
}

func (h *handlers) getSite(c echo.Context) error {
	site, err := h.deps.Sites.Get(c.Request().Context(), c.Param("site_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "site not found")
	}
	return c.JSON(http.StatusOK, toSiteResponse(site))
}

func (h *handlers) listSites(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	var siteFilter *string
	if claims.Role == domain.RoleRestricted {
		siteFilter = claims.SiteID
	}

	limit := parseLimit(c)
	sites, next, err := h.deps.Sites.List(c.Request().Context(), siteFilter, c.QueryParam("cursor"), limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	items := make([]siteResponse, 0, len(sites))
	for _, s := range sites {
		items = append(items, toSiteResponse(s))
	}
	return c.JSON(http.StatusOK, pageResponse[siteResponse]{Items: items, NextCursor: next})
}

func parseLimit(c echo.Context) int {
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return pagination.DefaultPageLimit
}

func parseOptionalTime(c echo.Context, param string) (*time.Time, error) {
	v := c.QueryParam(param)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
