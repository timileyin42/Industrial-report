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

func (h *handlers) fleetSpecificYield(c echo.Context) error {
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

	points, err := h.deps.Analytics.FleetSpecificYield(c.Request().Context(), cohortID, period, from, to)
	if err != nil {
		if err == registry.ErrNoSystemSize {
			return echo.NewHTTPError(http.StatusBadRequest, "no site in this fleet has system_size_kw configured — cannot compute specific yield")
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
	"prevailing conditions — see GET .../analytics/performance-ratio for that."

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

type performanceRatioPointResponse struct {
	PeriodStart         time.Time `json:"period_start"`
	EnergyKWh           float64   `json:"energy_kwh"`
	ExpectedEnergyKWh   float64   `json:"expected_energy_kwh"`
	PerformanceRatioPct float64   `json:"performance_ratio_pct"`
}

const performanceRatioDefinition = "Energy actually generated divided by the expected output for the " +
	"sunlight this site actually received (historical irradiance, not a forecast) — the weather-adjusted " +
	"metric Capacity Factor deliberately isn't. A healthy system should sit fairly steady here across " +
	"different weather; a drop usually means a real fault (soiling, shading, degradation), not just a cloudy day."

func performanceRatioErrorToHTTP(err error) error {
	switch err {
	case registry.ErrNoSystemSize:
		return echo.NewHTTPError(http.StatusBadRequest, "no system_size_kw configured — cannot compute performance ratio")
	case registry.ErrNoLocation:
		return echo.NewHTTPError(http.StatusBadRequest, "no location (gps_lat/gps_lng) set — performance ratio needs a location to look up historical sunlight")
	default:
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
}

func (h *handlers) sitePerformanceRatio(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}

	points, err := h.deps.Analytics.SitePerformanceRatio(c.Request().Context(), c.Param("site_id"), period, from, to)
	if err != nil {
		return performanceRatioErrorToHTTP(err)
	}

	out := make([]performanceRatioPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, performanceRatioPointResponse{
			PeriodStart:         p.PeriodStart,
			EnergyKWh:           p.EnergyKWh,
			ExpectedEnergyKWh:   p.ExpectedEnergyKWh,
			PerformanceRatioPct: p.PerformanceRatioPct,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"definition": performanceRatioDefinition,
		"period":     period,
		"points":     out,
	})
}

func (h *handlers) fleetPerformanceRatio(c echo.Context) error {
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

	points, err := h.deps.Analytics.FleetPerformanceRatio(c.Request().Context(), cohortID, period, from, to)
	if err != nil {
		return performanceRatioErrorToHTTP(err)
	}

	out := make([]performanceRatioPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, performanceRatioPointResponse{
			PeriodStart:         p.PeriodStart,
			EnergyKWh:           p.EnergyKWh,
			ExpectedEnergyKWh:   p.ExpectedEnergyKWh,
			PerformanceRatioPct: p.PerformanceRatioPct,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"definition": performanceRatioDefinition,
		"period":     period,
		"points":     out,
	})
}

type powerCurvePointResponse struct {
	Bucket     time.Time `json:"bucket"`
	AvgPowerKW float64   `json:"avg_power_kw"`
}

// fleetPowerCurve is the intraday sibling of fleetEnergy — plots
// average power over 5-minute buckets across the requested window
// (defaulting to the trailing 24h), rather than one energy total per
// calendar day. Feeds the Dashboard's "Generation Overview" Day view.
func (h *handlers) fleetPowerCurve(c echo.Context) error {
	from, to, err := parseIntradayRange(c)
	if err != nil {
		return err
	}
	var cohortID *string
	if v := c.QueryParam("cohort_id"); v != "" {
		cohortID = &v
	}

	points, err := h.deps.Analytics.FleetPowerCurve(c.Request().Context(), cohortID, from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	out := make([]powerCurvePointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, powerCurvePointResponse{Bucket: p.Bucket, AvgPowerKW: p.AvgPowerKW})
	}
	return c.JSON(http.StatusOK, map[string]any{"unit": "kW", "points": out})
}

func (h *handlers) currentGeneration(c echo.Context) error {
	kw, err := h.deps.Fleet.CurrentGeneration(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"current_power_kw": kw})
}

type topSiteResponse struct {
	SiteID                 string   `json:"site_id"`
	Name                   *string  `json:"name,omitempty"`
	EnergyKWh              float64  `json:"energy_kwh"`
	SystemSizeKW           *float64 `json:"system_size_kw,omitempty"`
	SpecificYieldKWhPerKWp float64  `json:"specific_yield_kwh_per_kwp"`
}

func (h *handlers) topSitesToday(c echo.Context) error {
	sites, err := h.deps.Analytics.TopSitesToday(c.Request().Context(), parseLimit(c))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	out := make([]topSiteResponse, 0, len(sites))
	for _, s := range sites {
		out = append(out, topSiteResponse{
			SiteID: s.SiteID, Name: s.Name, EnergyKWh: s.EnergyKWh,
			SystemSizeKW: s.SystemSizeKW, SpecificYieldKWhPerKWp: s.SpecificYieldKWhPerKWp,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": out})
}
