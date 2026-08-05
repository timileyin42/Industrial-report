package httpapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type fleetSummaryResponse struct {
	TotalSites      int64    `json:"total_sites"`
	TotalDevices    int64    `json:"total_devices"`
	OnlineDevices   int64    `json:"online_devices"`
	TotalCapacityKW *float64 `json:"total_capacity_kw,omitempty"`
}

func (h *handlers) fleetSummary(c echo.Context) error {
	summary, err := h.deps.Fleet.Summary(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, fleetSummaryResponse{
		TotalSites:      summary.TotalSites,
		TotalDevices:    summary.TotalDevices,
		OnlineDevices:   summary.OnlineDevices,
		TotalCapacityKW: summary.TotalCapacityKW,
	})
}
