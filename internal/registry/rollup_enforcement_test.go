package registry

import (
	"context"
	"testing"
	"time"
)

// TestSiteEnergy_ReadsRollupsNotRawTelemetry is CLAUDE.md's fleet/
// dashboard-scale requirement made concrete: "queries must hit roll-ups/
// continuous aggregates, not scan raw telemetry for any dashboard-facing
// request." This proves it empirically rather than by reading the SQL:
// insert raw telemetry for a day, but deliberately DON'T refresh the
// telemetry_daily continuous aggregate — if SiteEnergy queried raw
// telemetry directly, the new reading would show up immediately; since it
// actually queries the rollup (internal/db/queries/analytics.sql), it
// must show zero energy for that day until the aggregate is refreshed.
// Refreshing it and re-querying is the control case, confirming the
// day's absence above was really about the rollup lagging behind, not a
// query that's silently broken/always-empty.
func TestSiteEnergy_ReadsRollupsNotRawTelemetry(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()

	siteID := uniqueID("site-rollup-")
	deviceID := uniqueID("dev-rollup-")
	site := createTestSite(t, ctx, NewSites(q), pool, CreateSiteInput{
		SiteID: siteID, Name: "Rollup Enforcement Test Site", Timezone: "UTC", Country: "NG",
	})
	if _, err := pool.Exec(ctx,
		`INSERT INTO devices (device_id, site_id, secret_hash) VALUES ($1, $2, 'test-hash')`,
		deviceID, site.SiteID,
	); err != nil {
		t.Fatalf("seed device: %v", err)
	}

	day := time.Now().UTC().Truncate(24 * time.Hour)
	from := day.Add(-1 * time.Hour)
	to := day.Add(25 * time.Hour)

	// Insert raw telemetry for "today" — energy_kwh_total climbing across
	// the day — but do NOT refresh telemetry_daily yet.
	const wantEnergyKWh = 12.5
	if _, err := pool.Exec(ctx,
		`INSERT INTO telemetry (device_id, site_id, ts, power_kw, energy_kwh_total, status) VALUES
		 ($1, $2, $3, 1.0, 0.0, 'ok'), ($1, $2, $4, 2.0, $5, 'ok')`,
		deviceID, site.SiteID, day.Add(1*time.Hour), day.Add(2*time.Hour), wantEnergyKWh,
	); err != nil {
		t.Fatalf("seed raw telemetry: %v", err)
	}

	analytics := NewAnalytics(q)

	before, err := analytics.SiteEnergy(ctx, site.SiteID, "daily", from, to)
	if err != nil {
		t.Fatalf("SiteEnergy before refresh: %v", err)
	}
	if before.CumulativeKWh != 0 {
		t.Fatalf("expected 0 kWh before the continuous aggregate is refreshed (proving this reads the rollup, not raw telemetry), got %v — "+
			"points=%+v", before.CumulativeKWh, before.Points)
	}

	refreshTelemetryDaily(t, pool)

	after, err := analytics.SiteEnergy(ctx, site.SiteID, "daily", from, to)
	if err != nil {
		t.Fatalf("SiteEnergy after refresh: %v", err)
	}
	if after.CumulativeKWh != wantEnergyKWh {
		t.Fatalf("expected %v kWh after refreshing the rollup, got %v — points=%+v", wantEnergyKWh, after.CumulativeKWh, after.Points)
	}
}
