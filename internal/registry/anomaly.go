package registry

import (
	"context"
	"time"

	"github.com/timileyin42/zgnis-solar/internal/domain"
)

type Anomaly struct {
	analytics *Analytics
}

func NewAnomaly(analytics *Analytics) *Anomaly {
	return &Anomaly{analytics: analytics}
}

type AnomalyFlag struct {
	SiteID       string
	Day          time.Time
	EnergyKWh    float64
	BaselineKWh  float64
	DropFraction float64
}

// AnomalyDefinition is returned alongside every anomaly response so a
// caller never mistakes this for the concept note's fuller,
// weather/season-aware anomaly detection — see internal/domain's
// AnomalyDropThresholdFraction doc comment for why this is deliberately
// coarse.
const AnomalyDefinition = "Trailing-baseline check only: flags a day whose energy fell below " +
	"50% of its own trailing average over the prior window. Not weather- or " +
	"season-adjusted — a cloudy day can trip this without anything being wrong."

// bySite groups daily rows and evaluates the trailing-baseline check per
// site (or, for a single-site call, just that one site).
func (a *Anomaly) evaluate(rows []dailyRow, siteID string, windowDays int, asOf time.Time) []AnomalyFlag {
	if windowDays <= 0 {
		windowDays = domain.AnomalyBaselineWindowDefaultDays
	}
	asOfDay := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, asOf.Location())

	byDay := map[time.Time]float64{}
	for _, r := range rows {
		byDay[r.Day] = r.EnergyKWh
	}

	current, ok := byDay[asOfDay]
	if !ok {
		return nil
	}

	var baselineSum float64
	var baselineCount int
	for i := 1; i <= windowDays; i++ {
		day := asOfDay.AddDate(0, 0, -i)
		if v, ok := byDay[day]; ok {
			baselineSum += v
			baselineCount++
		}
	}
	if baselineCount == 0 {
		return nil // not enough history to establish a baseline
	}
	baseline := baselineSum / float64(baselineCount)
	if baseline <= 0 {
		return nil
	}

	dropFraction := (baseline - current) / baseline
	if dropFraction < domain.AnomalyDropThresholdFraction {
		return nil
	}

	return []AnomalyFlag{{
		SiteID:       siteID,
		Day:          asOfDay,
		EnergyKWh:    current,
		BaselineKWh:  baseline,
		DropFraction: dropFraction,
	}}
}

func (a *Anomaly) SiteAnomalies(ctx context.Context, siteID string, windowDays int, asOf time.Time) ([]AnomalyFlag, error) {
	asOfDay := time.Date(asOf.Year(), asOf.Month(), asOf.Day(), 0, 0, 0, 0, asOf.Location())
	if windowDays <= 0 {
		windowDays = domain.AnomalyBaselineWindowDefaultDays
	}
	from := asOfDay.AddDate(0, 0, -windowDays)
	rows, err := a.analytics.loadSiteDailyRows(ctx, siteID, from, asOfDay)
	if err != nil {
		return nil, err
	}
	return a.evaluate(rows, siteID, windowDays, asOf), nil
}

// FleetAnomalies runs the same check per site across the whole fleet.
func (a *Anomaly) FleetAnomalies(ctx context.Context, windowDays int, asOf time.Time) ([]AnomalyFlag, error) {
	sites, err := a.analytics.q.ListSitesForAnalytics(ctx)
	if err != nil {
		return nil, err
	}
	var flags []AnomalyFlag
	for _, s := range sites {
		siteFlags, err := a.SiteAnomalies(ctx, s.SiteID, windowDays, asOf)
		if err != nil {
			return nil, err
		}
		flags = append(flags, siteFlags...)
	}
	return flags, nil
}
