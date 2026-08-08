package registry

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
)

type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityInfo     AlertSeverity = "info"
)

// Alert is a live, real-data-derived condition — never a fabricated
// historical event. There's no persisted "alerts" table in this
// platform; every alert below is computed fresh from a real, already-
// stored timestamp (last_seen_at, revoked_at, a reading's own ts, an
// anomaly-detection day), which is why this reads current/recent state
// rather than a true append-only event log.
type Alert struct {
	Type       string
	Severity   AlertSeverity
	SiteID     string
	SiteName   *string
	DeviceID   *string
	Message    string
	OccurredAt time.Time
}

type Alerts struct {
	fleet   *Fleet
	anomaly *Anomaly
	q       *db.Queries
}

func NewAlerts(fleet *Fleet, anomaly *Anomaly, q *db.Queries) *Alerts {
	return &Alerts{fleet: fleet, anomaly: anomaly, q: q}
}

// Fleet aggregates four real signals into one feed: offline/low-coverage
// sites (from the same fleet health computation Fleet Health's own page
// uses), devices whose latest reading reported a fault status, recently
// revoked devices, and recent anomaly-detection flags. Sorted newest
// first and capped at limit — this is a live "what needs attention"
// view, not a paginated historical report (that's what Audit Log and
// the Performance page's own anomaly table are for).
func (al *Alerts) Fleet(ctx context.Context, limit int) ([]Alert, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	now := time.Now().UTC()
	var alerts []Alert

	health, err := al.fleet.Health(ctx, "", 200)
	if err != nil {
		return nil, err
	}
	for _, s := range health.Sites {
		offlineCount := s.TotalDevices - s.OnlineDevices
		switch {
		case offlineCount > 0:
			occurredAt := now
			if s.WorstLastSeenAt != nil {
				occurredAt = *s.WorstLastSeenAt
			}
			alerts = append(alerts, Alert{
				Type: "device_offline", Severity: AlertSeverityCritical,
				SiteID: s.SiteID, SiteName: s.SiteName,
				Message:    fmt.Sprintf("%d of %d devices offline", offlineCount, s.TotalDevices),
				OccurredAt: occurredAt,
			})
		case s.CoveragePct < 50:
			alerts = append(alerts, Alert{
				Type: "low_coverage", Severity: AlertSeverityWarning,
				SiteID: s.SiteID, SiteName: s.SiteName,
				Message:    fmt.Sprintf("Coverage at %.0f%% over the last %dh window", s.CoveragePct, health.CoverageWindowHours),
				OccurredAt: now,
			})
		}
	}

	faultWindow := pgtype.Timestamptz{Time: now.Add(-24 * time.Hour), Valid: true}
	faults, err := al.q.ListRecentFaultReadings(ctx, faultWindow)
	if err != nil {
		return nil, err
	}
	for _, f := range faults {
		deviceID := f.DeviceID
		alerts = append(alerts, Alert{
			Type: "device_fault", Severity: AlertSeverityCritical,
			SiteID: f.SiteID, DeviceID: &deviceID,
			Message:    fmt.Sprintf("Device %s reporting fault status", deviceID),
			OccurredAt: f.Ts.Time,
		})
	}

	revokedWindow := pgtype.Timestamptz{Time: now.Add(-7 * 24 * time.Hour), Valid: true}
	revoked, err := al.q.ListRecentlyRevokedDevices(ctx, revokedWindow)
	if err != nil {
		return nil, err
	}
	for _, d := range revoked {
		deviceID := d.DeviceID
		siteID := ""
		if d.SiteID.Valid {
			siteID = d.SiteID.String
		}
		alerts = append(alerts, Alert{
			Type: "device_revoked", Severity: AlertSeverityInfo,
			SiteID: siteID, DeviceID: &deviceID,
			Message:    fmt.Sprintf("Device %s was revoked", deviceID),
			OccurredAt: d.RevokedAt.Time,
		})
	}

	// Only the last 3 days of anomaly flags — the full history already
	// has a dedicated table on the Performance page; this feed is about
	// what's recent enough to still need attention.
	anomalies, err := al.anomaly.FleetAnomalies(ctx, 7, now)
	if err != nil {
		return nil, err
	}
	recentCutoff := now.Add(-3 * 24 * time.Hour)
	for _, an := range anomalies {
		if an.Day.Before(recentCutoff) {
			continue
		}
		alerts = append(alerts, Alert{
			Type: "low_generation", Severity: AlertSeverityWarning,
			SiteID:     an.SiteID,
			Message:    fmt.Sprintf("Generation %.0f%% below its trailing average on %s", an.DropFraction*100, an.Day.Format("Jan 2")),
			OccurredAt: an.Day,
		})
	}

	sort.Slice(alerts, func(i, j int) bool { return alerts[i].OccurredAt.After(alerts[j].OccurredAt) })
	if len(alerts) > limit {
		alerts = alerts[:limit]
	}
	return alerts, nil
}
