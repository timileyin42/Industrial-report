package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type historyComparisonResponse struct {
	CurrentPeriodStart  time.Time `json:"current_period_start"`
	PreviousPeriodStart time.Time `json:"previous_period_start"`
	CurrentEnergyKWh    float64   `json:"current_energy_kwh"`
	PreviousEnergyKWh   float64   `json:"previous_energy_kwh"`
	ChangePct           *float64  `json:"change_pct,omitempty"`
}

// compareHistory is "a site against its own history" — site-scoped.
func (h *handlers) compareHistory(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	asOf, err := parseAsOf(c)
	if err != nil {
		return err
	}

	result, err := h.deps.Benchmark.CompareHistory(c.Request().Context(), c.Param("site_id"), period, asOf)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, historyComparisonResponse{
		CurrentPeriodStart:  result.CurrentPeriodStart,
		PreviousPeriodStart: result.PreviousPeriodStart,
		CurrentEnergyKWh:    result.CurrentEnergyKWh,
		PreviousEnergyKWh:   result.PreviousEnergyKWh,
		ChangePct:           result.ChangePct,
	})
}

type fleetComparisonResponse struct {
	SiteID         string  `json:"site_id"`
	SiteEnergyKWh  float64 `json:"site_energy_kwh"`
	FleetAvgKWh    float64 `json:"fleet_avg_kwh"`
	PercentileRank float64 `json:"percentile_rank"`
	SiteCount      int     `json:"site_count"`
}

// compareFleet is "a site against the fleet average / percentile rank" —
// operator-only (leaks fleet-wide distribution by construction).
func (h *handlers) compareFleet(c echo.Context) error {
	siteID := c.QueryParam("site_id")
	if siteID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "site_id query param is required")
	}
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	asOf, err := parseAsOf(c)
	if err != nil {
		return err
	}

	result, err := h.deps.Benchmark.CompareFleet(c.Request().Context(), siteID, period, asOf)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, fleetComparisonResponse{
		SiteID:         result.SiteID,
		SiteEnergyKWh:  result.SiteEnergyKWh,
		FleetAvgKWh:    result.FleetAvgKWh,
		PercentileRank: result.PercentileRank,
		SiteCount:      result.SiteCount,
	})
}

type segmentStatResponse struct {
	SegmentKey     string  `json:"segment_key"`
	SiteCount      int     `json:"site_count"`
	TotalEnergyKWh float64 `json:"total_energy_kwh"`
	AvgEnergyKWh   float64 `json:"avg_energy_kwh"`
}

const regionSegmentNote = "cohort_id is used as the closest available grouping for 'region' — " +
	"sites has no dedicated region/state column today."

// benchmarkSegments is fleet segmentation by size band / inverter model /
// cohort — operator-only.
func (h *handlers) benchmarkSegments(c echo.Context) error {
	segmentBy := c.QueryParam("segment_by")
	if segmentBy == "" {
		segmentBy = "system_size_band"
	}
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	asOf, err := parseAsOf(c)
	if err != nil {
		return err
	}

	stats, next, err := h.deps.Benchmark.Segment(c.Request().Context(), segmentBy, period, asOf, c.QueryParam("cursor"), parseLimit(c))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	out := make([]segmentStatResponse, 0, len(stats))
	for _, s := range stats {
		out = append(out, segmentStatResponse{
			SegmentKey:     s.SegmentKey,
			SiteCount:      s.SiteCount,
			TotalEnergyKWh: s.TotalEnergyKWh,
			AvgEnergyKWh:   s.AvgEnergyKWh,
		})
	}

	resp := map[string]any{"segment_by": segmentBy, "items": out}
	if next != "" {
		resp["next_cursor"] = next
	}
	if segmentBy == "region" {
		resp["note"] = regionSegmentNote
	}
	return c.JSON(http.StatusOK, resp)
}

type trendPointResponse struct {
	PeriodStart     time.Time `json:"period_start"`
	TotalCapacityKW float64   `json:"total_capacity_kw"`
	SiteCount       int       `json:"site_count"`
	TotalEnergyKWh  float64   `json:"total_energy_kwh"`
	MoMChangePct    *float64  `json:"mom_change_pct,omitempty"`
}

// fleetTrends is fleet-level growth over time — operator-only.
func (h *handlers) fleetTrends(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}

	points, err := h.deps.Benchmark.Trends(c.Request().Context(), period, from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	out := make([]trendPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, trendPointResponse{
			PeriodStart:     p.PeriodStart,
			TotalCapacityKW: p.TotalCapacityKW,
			SiteCount:       p.SiteCount,
			TotalEnergyKWh:  p.TotalEnergyKWh,
			MoMChangePct:    p.MoMChangePct,
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"period": period, "points": out})
}

// fleetCohort aggregates one cohort/project — operator-only (a cohort can
// include sites beyond the caller's own).
func (h *handlers) fleetCohort(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}

	summary, err := h.deps.Benchmark.Cohort(c.Request().Context(), c.Param("cohort_id"), period, from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"cohort_id":         summary.CohortID,
		"total_capacity_kw": summary.TotalCapacityKW,
		"site_count":        summary.SiteCount,
		"energy":            toEnergySeriesResponse(period, summary.Energy),
	})
}
