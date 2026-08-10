package registry

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestTelemetryList_PaginatesExhaustivelyWithoutLossOrDuplication is the
// real regression CLAUDE.md's pagination requirement is about: not just
// that a cursor token round-trips (pagination package's own tests cover
// that), but that repeatedly following next_cursor across a full list
// endpoint actually walks every row exactly once — no row skipped at a
// page boundary, none duplicated across pages, and the last page reports
// no next_cursor. Scoped to one freshly-created site (Telemetry.List
// always filters by site_id) so this is isolated from whatever else
// already lives in the shared dev database — no risk of paginating
// through months of other tests' leftover rows.
func TestTelemetryList_PaginatesExhaustivelyWithoutLossOrDuplication(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()

	siteID := uniqueID("site-page-")
	deviceID := uniqueID("dev-page-")
	site := createTestSite(t, ctx, NewSites(q), pool, CreateSiteInput{
		SiteID: siteID, Name: "Pagination Test Site", Timezone: "UTC", Country: "NG",
	})
	if _, err := pool.Exec(ctx,
		`INSERT INTO devices (device_id, site_id, secret_hash) VALUES ($1, $2, 'test-hash')`,
		deviceID, site.SiteID,
	); err != nil {
		t.Fatalf("seed device: %v", err)
	}

	const totalRows = 11
	const pageSize = 3
	base := time.Now().UTC().Truncate(time.Second)
	wantTimestamps := make(map[string]bool, totalRows)
	for i := 0; i < totalRows; i++ {
		ts := base.Add(time.Duration(i) * time.Minute)
		if _, err := pool.Exec(ctx,
			`INSERT INTO telemetry (device_id, site_id, ts, power_kw, energy_kwh_total, status) VALUES ($1, $2, $3, 1.0, 10.0, 'ok')`,
			deviceID, site.SiteID, ts,
		); err != nil {
			t.Fatalf("seed telemetry row %d: %v", i, err)
		}
		wantTimestamps[ts.Format(time.RFC3339)] = true
	}

	telemetry := NewTelemetry(q)
	seen := make(map[string]int) // ts -> number of times seen across pages
	cursor := ""
	pages := 0
	for {
		pages++
		if pages > totalRows+2 {
			t.Fatalf("paginated %d times without terminating — cursor likely stuck in a loop", pages)
		}
		rows, next, err := telemetry.List(ctx, ListTelemetryInput{SiteID: site.SiteID, CursorToken: cursor, Limit: pageSize})
		if err != nil {
			t.Fatalf("List (page %d): %v", pages, err)
		}
		if pages < (totalRows+pageSize-1)/pageSize && len(rows) != pageSize {
			t.Fatalf("page %d: expected a full page of %d rows, got %d", pages, pageSize, len(rows))
		}
		for _, r := range rows {
			key := r.Ts.Time.UTC().Format(time.RFC3339)
			seen[key]++
		}
		if next == "" {
			break
		}
		cursor = next
	}

	if len(seen) != totalRows {
		t.Fatalf("expected %d distinct rows across all pages, saw %d distinct: %v", totalRows, len(seen), seen)
	}
	for ts, count := range seen {
		if count != 1 {
			t.Errorf("row with ts=%s was returned %d times across pages, expected exactly once", ts, count)
		}
		if !wantTimestamps[ts] {
			t.Errorf("saw an unexpected row with ts=%s that this test never inserted", ts)
		}
	}
	if got := len(seen); got != totalRows {
		t.Fatalf("row-count mismatch after pagination: got %d, want %d (%s)", got, totalRows, fmt.Sprintf("%d pages", pages))
	}
}
