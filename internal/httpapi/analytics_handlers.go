package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type energyPointResponse struct {
	PeriodStart     time.Time `json:"period_start"`
	EnergyKWh       float64   `json:"energy_kwh"`
	ReadingCount    int64     `json:"reading_count"`
	BackfilledCount int64     `json:"backfilled_count"`
}

type energySeriesResponse struct {
	Unit          string                `json:"unit"`
	Period        string                `json:"period"`
	Points        []energyPointResponse `json:"points"`
	CumulativeKWh float64               `json:"cumulative_kwh"`
}

func toEnergySeriesResponse(period string, series registry.EnergySeries) energySeriesResponse {
	points := make([]energyPointResponse, 0, len(series.Points))
	for _, p := range series.Points {
		points = append(points, energyPointResponse{
			PeriodStart:     p.PeriodStart,
			EnergyKWh:       p.EnergyKWh,
			ReadingCount:    p.ReadingCount,
			BackfilledCount: p.BackfilledCount,
		})
	}
	return energySeriesResponse{Unit: "kWh", Period: period, Points: points, CumulativeKWh: series.CumulativeKWh}
}

// siteEnergy is site-scoped — access enforced by RequireSiteAccess in
// router.go.
func (h *handlers) siteEnergy(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}

	series, err := h.deps.Analytics.SiteEnergy(c.Request().Context(), c.Param("site_id"), period, from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, toEnergySeriesResponse(period, series))
}

// fleetEnergy is operator-only, wired in router.go.
func (h *handlers) fleetEnergy(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}
	var cohortID *string
	if v := c.QueryParam("cohort_id"); v != "" {
		cohortID = &v
	}

	series, err := h.deps.Analytics.FleetEnergy(c.Request().Context(), cohortID, period, from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, toEnergySeriesResponse(period, series))
}

type yieldPointResponse struct {
	PeriodStart            time.Time `json:"period_start"`
	EnergyKWh              float64   `json:"energy_kwh"`
	SystemSizeKW           float64   `json:"system_size_kw"`
	SpecificYieldKWhPerKWp float64   `json:"specific_yield_kwh_per_kwp"`
}

func (h *handlers) siteSpecificYield(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}

	points, err := h.deps.Analytics.SiteSpecificYield(c.Request().Context(), c.Param("site_id"), period, from, to)
	if err != nil {
		if err == registry.ErrNoSystemSize {
			return echo.NewHTTPError(http.StatusBadRequest, "site has no system_size_kw configured — cannot compute specific yield")
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	out := make([]yieldPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, yieldPointResponse{
			PeriodStart:            p.PeriodStart,
			EnergyKWh:              p.EnergyKWh,
			SystemSizeKW:           p.SystemSizeKW,
			SpecificYieldKWhPerKWp: p.SpecificYieldKWhPerKWp,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"unit": "kWh/kWp", "period": period, "points": out})
}

type peakPointResponse struct {
	Day         time.Time  `json:"day"`
	PeakPowerKW float64    `json:"peak_power_kw"`
	OccurredAt  *time.Time `json:"occurred_at,omitempty"`
}

func (h *handlers) sitePeak(c echo.Context) error {
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}

	points, err := h.deps.Analytics.SitePeak(c.Request().Context(), c.Param("site_id"), from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	out := make([]peakPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, peakPointResponse{Day: p.Day, PeakPowerKW: p.PeakPowerKW, OccurredAt: p.OccurredAt})
	}
	return c.JSON(http.StatusOK, map[string]any{"unit": "kW", "points": out})
}

type capacityFactorPointResponse struct {
	PeriodStart       time.Time `json:"period_start"`
	EnergyKWh         float64   `json:"energy_kwh"`
	TheoreticalMaxKWh float64   `json:"theoretical_max_kwh"`
	CapacityFactorPct float64   `json:"capacity_factor_pct"`
}

const capacityFactorDefinition = "Energy actually generated divided by a theoretical maximum derived from " +
	"nameplate system size and elapsed time only — no weather/irradiance adjustment. " +
	"This is NOT Performance Ratio (PR), which compares against expected output for " +
	"prevailing conditions; PR is deferred pending a weather-data source decision."

func (h *handlers) siteCapacityFactor(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}

	points, err := h.deps.Analytics.SiteCapacityFactor(c.Request().Context(), c.Param("site_id"), period, from, to)
	if err != nil {
		if err == registry.ErrNoSystemSize {
			return echo.NewHTTPError(http.StatusBadRequest, "site has no system_size_kw configured — cannot compute capacity factor")
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	out := make([]capacityFactorPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, capacityFactorPointResponse{
			PeriodStart:       p.PeriodStart,
			EnergyKWh:         p.EnergyKWh,
			TheoreticalMaxKWh: p.TheoreticalMaxKWh,
			CapacityFactorPct: p.CapacityFactorPct,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"definition": capacityFactorDefinition,
		"period":     period,
		"points":     out,
	})
}
