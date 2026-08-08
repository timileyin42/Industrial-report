package registry

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/pagination"
)

type Fleet struct {
	sites            *Sites
	devices          *Devices
	onlineThreshold  time.Duration
	expectedInterval time.Duration
	coverageWindow   time.Duration
}

func NewFleet(sites *Sites, devices *Devices, onlineThreshold, expectedInterval, coverageWindow time.Duration) *Fleet {
	return &Fleet{
		sites:            sites,
		devices:          devices,
		onlineThreshold:  onlineThreshold,
		expectedInterval: expectedInterval,
		coverageWindow:   coverageWindow,
	}
}

type FleetSummary struct {
	TotalSites      int64
	TotalDevices    int64
	OnlineDevices   int64
	TotalCapacityKW *float64
}

func (f *Fleet) Summary(ctx context.Context) (FleetSummary, error) {
	siteTotals, err := f.sites.q.FleetTotals(ctx)
	if err != nil {
		return FleetSummary{}, err
	}
	totalDevices, err := f.devices.q.CountDevices(ctx)
	if err != nil {
		return FleetSummary{}, err
	}
	cutoff := time.Now().UTC().Add(-f.onlineThreshold)
	online, err := f.devices.q.CountOnlineDevices(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
	if err != nil {
		return FleetSummary{}, err
	}
	return FleetSummary{
		TotalSites:      siteTotals.TotalSites,
		TotalDevices:    totalDevices,
		OnlineDevices:   online,
		TotalCapacityKW: numericToFloat(siteTotals.TotalCapacityKw),
	}, nil
}

type FleetHealth struct {
	GeneratedAt             time.Time
	OnlineThresholdMinutes  int
	ExpectedIntervalMinutes int
	CoverageWindowHours     int
	TotalSites              int64
	TotalDevices            int64
	OnlineDevices           int64
	DevicesReportingPct     float64
	CoveragePct             float64
	Sites                   []SiteHealth
	NextCursor              string
}

type SiteHealth struct {
	SiteID        string
	SiteName      *string
	SiteCreatedAt time.Time
	TotalDevices  int64
	OnlineDevices int64
	CoveragePct   float64
	// WorstLastSeenAt is nil when every device at this site has never
	// reported even once — SiteCreatedAt (always real) is the fallback
	// anchor for that case, used by Alerts.Fleet.
	WorstLastSeenAt *time.Time
}

// Health is the fleet-wide data-quality view (coverage, per-site
// breakdown) — deliberately a separate endpoint/method from Summary
// (stable totals contract) rather than an extension of it. See plan notes.
func (f *Fleet) Health(ctx context.Context, cursorToken string, limit int) (FleetHealth, error) {
	if limit <= 0 || limit > 200 {
		limit = pagination.DefaultPageLimit
	}

	now := time.Now().UTC()
	windowStart := now.Add(-f.coverageWindow)
	onlineCutoff := now.Add(-f.onlineThreshold)
	expectedIntervalSeconds := f.expectedInterval.Seconds()

	totals, err := f.sites.q.FleetHealthTotals(ctx, db.FleetHealthTotalsParams{
		OnlineCutoff:            pgtype.Timestamptz{Time: onlineCutoff, Valid: true},
		WindowStart:             pgtype.Timestamptz{Time: windowStart, Valid: true},
		Now:                     pgtype.Timestamptz{Time: now, Valid: true},
		ExpectedIntervalSeconds: expectedIntervalSeconds,
	})
	if err != nil {
		return FleetHealth{}, err
	}

	var cursorSiteID pgtype.Text
	if cursorToken != "" {
		c, err := pagination.Decode(cursorToken)
		if err != nil {
			return FleetHealth{}, err
		}
		cursorSiteID = pgtype.Text{String: c.Tiebreak, Valid: true}
	}

	rows, err := f.sites.q.ListSiteHealth(ctx, db.ListSiteHealthParams{
		OnlineCutoff:            pgtype.Timestamptz{Time: onlineCutoff, Valid: true},
		CursorSiteID:            cursorSiteID,
		PageLimit:               int32(limit),
		WindowStart:             pgtype.Timestamptz{Time: windowStart, Valid: true},
		Now:                     pgtype.Timestamptz{Time: now, Valid: true},
		ExpectedIntervalSeconds: expectedIntervalSeconds,
	})
	if err != nil {
		return FleetHealth{}, err
	}

	sites := make([]SiteHealth, 0, len(rows))
	for _, r := range rows {
		sites = append(sites, SiteHealth{
			SiteID:          r.SiteID,
			SiteName:        textPtr(r.SiteName),
			SiteCreatedAt:   r.SiteCreatedAt.Time,
			TotalDevices:    r.TotalDevices,
			OnlineDevices:   r.OnlineDevices,
			CoveragePct:     coveragePct(r.ActualReadings, r.ExpectedReadings),
			WorstLastSeenAt: timestamptzPtr(r.WorstLastSeenAt),
		})
	}

	next := ""
	if len(rows) == limit {
		last := rows[len(rows)-1]
		next, err = pagination.Encode(pagination.Cursor{Time: now, Tiebreak: last.SiteID})
		if err != nil {
			return FleetHealth{}, err
		}
	}

	var devicesReportingPct float64
	if totals.TotalDevices > 0 {
		devicesReportingPct = float64(totals.OnlineDevices) / float64(totals.TotalDevices) * 100
	}

	return FleetHealth{
		GeneratedAt:             now,
		OnlineThresholdMinutes:  int(f.onlineThreshold.Minutes()),
		ExpectedIntervalMinutes: int(f.expectedInterval.Minutes()),
		CoverageWindowHours:     int(f.coverageWindow.Hours()),
		TotalSites:              totals.TotalSites,
		TotalDevices:            totals.TotalDevices,
		OnlineDevices:           totals.OnlineDevices,
		DevicesReportingPct:     devicesReportingPct,
		CoveragePct:             coveragePct(totals.TotalActualReadings, totals.TotalExpectedReadings),
		Sites:                   sites,
		NextCursor:              next,
	}, nil
}

// CurrentGeneration is a live "how much power right now" figure — the
// most recent reading from every currently-online device, summed. This
// is intentionally NOT a rollup; it reflects this exact moment, unlike
// every other energy figure on this platform (which is historical/
// cumulative by design).
func (f *Fleet) CurrentGeneration(ctx context.Context) (float64, error) {
	cutoff := time.Now().UTC().Add(-f.onlineThreshold)
	return f.devices.q.CurrentFleetGeneration(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
}

func coveragePct(actual int64, expected float64) float64 {
	if expected <= 0 {
		return 0
	}
	pct := float64(actual) / expected * 100
	if pct > 100 {
		pct = 100
	}
	return pct
}
