package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type anomalyFlagResponse struct {
	SiteID       string    `json:"site_id"`
	Day          time.Time `json:"day"`
	EnergyKWh    float64   `json:"energy_kwh"`
	BaselineKWh  float64   `json:"baseline_kwh"`
	DropFraction float64   `json:"drop_fraction"`
}

func toAnomalyFlagResponses(flags []registry.AnomalyFlag) []anomalyFlagResponse {
	out := make([]anomalyFlagResponse, 0, len(flags))
	for _, f := range flags {
		out = append(out, anomalyFlagResponse{
			SiteID:       f.SiteID,
			Day:          f.Day,
			EnergyKWh:    f.EnergyKWh,
			BaselineKWh:  f.BaselineKWh,
			DropFraction: f.DropFraction,
		})
	}
	return out
}

// siteAnomalies is site-scoped — a single site's own trailing-baseline
// check doesn't leak fleet-wide data, so restricted users may use it.
func (h *handlers) siteAnomalies(c echo.Context) error {
	asOf, err := parseAsOf(c)
	if err != nil {
		return err
	}
	windowDays := parseWindowDays(c, domain.AnomalyBaselineWindowDefaultDays)

	flags, err := h.deps.Anomaly.SiteAnomalies(c.Request().Context(), c.Param("site_id"), windowDays, asOf)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{
		"definition": registry.AnomalyDefinition,
		"flags":      toAnomalyFlagResponses(flags),
	})
}

// fleetAnomalies is operator-only.
func (h *handlers) fleetAnomalies(c echo.Context) error {
	asOf, err := parseAsOf(c)
	if err != nil {
		return err
	}
	windowDays := parseWindowDays(c, domain.AnomalyBaselineWindowDefaultDays)

	flags, err := h.deps.Anomaly.FleetAnomalies(c.Request().Context(), windowDays, asOf)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{
		"definition": registry.AnomalyDefinition,
		"flags":      toAnomalyFlagResponses(flags),
	})
}
