package registry

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timileyin42/zgnis-solar/internal/db"
)

// testQueries connects to DATABASE_URL for integration tests that need a
// real Postgres/TimescaleDB — this project has no testcontainers/mock-DB
// layer, so these run against the same local dev database documented in
// README.md, exactly like cmd/api and cmd/ingestor do. Skips cleanly
// (not a failure) when DATABASE_URL isn't set, so `go test ./...` stays
// safe to run without a database available.
func testQueries(t *testing.T) *db.Queries {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping integration test that needs a real database")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return db.New(pool)
}

// uniqueID gives each test run its own site/device IDs so parallel or
// repeated runs never collide with a leftover row from a previous run.
func uniqueID(prefix string) string {
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// testRawPool is for test setup that db.Queries has no generated method
// for — inserting raw telemetry rows and forcing the telemetry_daily
// continuous aggregate to refresh immediately (the same manual refresh
// call README.md documents; a continuous aggregate doesn't reflect new
// rows until either its scheduled job runs or this is called). Analytics/
// Emissions tests need real (device_id, day) rollup rows to exist for
// their energy-dependent branches to be worth anything.
func testRawPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping integration test that needs a real database")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func refreshTelemetryDaily(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `CALL refresh_continuous_aggregate('telemetry_daily', NULL, NULL);`); err != nil {
		t.Fatalf("refresh telemetry_daily: %v", err)
	}
}

// createTestSite wraps Sites.Create and registers cleanup for the
// site plus anything that can reference it (telemetry, devices) —
// these tests run against the same shared local dev database a person
// may be actively looking at in the dashboard, not an isolated
// per-test database, so leaving rows behind pollutes their Sites/
// Devices lists for every run after this one.
func createTestSite(t *testing.T, ctx context.Context, sites *Sites, pool *pgxpool.Pool, in CreateSiteInput) db.Site {
	t.Helper()
	site, err := sites.Create(ctx, 1, in)
	if err != nil {
		t.Fatalf("create test site %s: %v", in.SiteID, err)
	}
	t.Cleanup(func() {
		cctx := context.Background()
		_, _ = pool.Exec(cctx, `DELETE FROM telemetry WHERE site_id = $1`, site.SiteID)
		_, _ = pool.Exec(cctx, `DELETE FROM devices WHERE site_id = $1`, site.SiteID)
		_, _ = pool.Exec(cctx, `DELETE FROM sites WHERE site_id = $1`, site.SiteID)
	})
	return site
}
