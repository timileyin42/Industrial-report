package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type fleetHealthTotalsResponse struct {
	TotalSites          int64   `json:"total_sites"`
	TotalDevices        int64   `json:"total_devices"`
	OnlineDevices       int64   `json:"online_devices"`
	DevicesReportingPct float64 `json:"devices_reporting_pct"`
	CoveragePct         float64 `json:"coverage_pct"`
}

type siteHealthResponse struct {
	SiteID          string     `json:"site_id"`
	SiteName        *string    `json:"site_name,omitempty"`
	TotalDevices    int64      `json:"total_devices"`
	OnlineDevices   int64      `json:"online_devices"`
	CoveragePct     float64    `json:"coverage_pct"`
	WorstLastSeenAt *time.Time `json:"worst_last_seen_at,omitempty"`
}

type fleetHealthResponse struct {
	GeneratedAt             time.Time                        `json:"generated_at"`
	OnlineThresholdMinutes  int                              `json:"online_threshold_minutes"`
	ExpectedIntervalMinutes int                              `json:"expected_interval_minutes"`
	CoverageWindowHours     int                              `json:"coverage_window_hours"`
	Fleet                   fleetHealthTotalsResponse        `json:"fleet"`
	Sites                   pageResponse[siteHealthResponse] `json:"sites"`
}

// fleetHealth is Phase 2's data-quality dashboard — deliberately a separate
// endpoint from fleetSummary (stable totals contract). Operator-only, same
// access-control pattern as /v1/fleet/summary.
func (h *handlers) fleetHealth(c echo.Context) error {
	health, err := h.deps.Fleet.Health(c.Request().Context(), c.QueryParam("cursor"), parseLimit(c))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	sites := make([]siteHealthResponse, 0, len(health.Sites))
	for _, s := range health.Sites {
		sites = append(sites, siteHealthResponse{
			SiteID:          s.SiteID,
			SiteName:        s.SiteName,
			TotalDevices:    s.TotalDevices,
			OnlineDevices:   s.OnlineDevices,
			CoveragePct:     s.CoveragePct,
			WorstLastSeenAt: s.WorstLastSeenAt,
		})
	}

	return c.JSON(http.StatusOK, fleetHealthResponse{
		GeneratedAt:             health.GeneratedAt,
		OnlineThresholdMinutes:  health.OnlineThresholdMinutes,
		ExpectedIntervalMinutes: health.ExpectedIntervalMinutes,
		CoverageWindowHours:     health.CoverageWindowHours,
		Fleet: fleetHealthTotalsResponse{
			TotalSites:          health.TotalSites,
			TotalDevices:        health.TotalDevices,
			OnlineDevices:       health.OnlineDevices,
			DevicesReportingPct: health.DevicesReportingPct,
			CoveragePct:         health.CoveragePct,
		},
		Sites: pageResponse[siteHealthResponse]{Items: sites, NextCursor: health.NextCursor},
	})
}
