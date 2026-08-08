package registry

import (
	"context"
	"testing"

	"github.com/timileyin42/zgnis-solar/internal/db"
)

func TestCreateSiteRequiresValidCountry(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	sites := NewSites(q)
	ctx := context.Background()

	cases := []struct {
		name    string
		country string
		wantErr bool
	}{
		{"valid uppercase code", "NG", false},
		{"lowercase rejected", "ng", true},
		{"empty rejected", "", true},
		{"too long rejected", "NGA", true},
		{"free text rejected", "Nigeria", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			siteID := uniqueID("site-country-")
			_, err := sites.Create(ctx, 1, CreateSiteInput{
				SiteID:   siteID,
				Name:     "Country Validation Test",
				Timezone: "UTC",
				Country:  tc.country,
			})
			if err == nil {
				// Only register cleanup when creation actually succeeded —
				// the reject cases never persist a row to begin with.
				t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM sites WHERE site_id = $1`, siteID) })
			}
			if tc.wantErr && err == nil {
				t.Fatalf("expected an error for country %q, got none", tc.country)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for country %q, got %v", tc.country, err)
			}
		})
	}
}

func TestUpdateCountryCorrectsExistingSite(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	sites := NewSites(q)
	ctx := context.Background()

	siteID := uniqueID("site-update-country-")
	createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: siteID, Name: "Backfilled NG Guess", Timezone: "UTC", Country: "NG"})

	updated, err := sites.UpdateCountry(ctx, 1, siteID, "GB")
	if err != nil {
		t.Fatalf("update country: %v", err)
	}
	if updated.Country != "GB" {
		t.Fatalf("expected country GB after update, got %q", updated.Country)
	}

	fetched, err := sites.Get(ctx, siteID)
	if err != nil {
		t.Fatalf("get site: %v", err)
	}
	if fetched.Country != "GB" {
		t.Fatalf("country update did not persist: got %q", fetched.Country)
	}
}

// withRestoredPrimarySite saves whichever site is primary before the
// test (if any) and restores it afterward via t.Cleanup — these tests
// run against the same shared local dev database a person may be using
// for manual UI testing at the same time (there's no per-test-isolated
// database in this project), and "the one primary site" is genuinely
// global singleton state, unlike a test's own uniquely-named site rows.
// Blowing that away for the duration of a test run would be a real
// regression to inflict on whoever's using the app while tests run.
func withRestoredPrimarySite(t *testing.T, ctx context.Context, q *db.Queries) {
	t.Helper()
	previous, err := q.GetPrimarySite(ctx)
	hadPrevious := err == nil
	t.Cleanup(func() {
		if hadPrevious {
			_ = q.UnsetAllPrimarySites(ctx)
			_, _ = q.SetSitePrimary(ctx, previous.SiteID)
		} else {
			_ = q.UnsetAllPrimarySites(ctx)
		}
	})
}

// TestSetPrimaryIsExclusive is the regression test for the weather-widget
// bug: the dashboard used to pick "whatever site was created most
// recently and had a location" with no real concept of a default site.
// This confirms the real fix — at most one site is ever primary, and
// setting a new one atomically clears the old one — actually holds.
func TestSetPrimaryIsExclusive(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	sites := NewSites(q)
	ctx := context.Background()
	withRestoredPrimarySite(t, ctx, q)

	siteA := uniqueID("site-primary-a-")
	siteB := uniqueID("site-primary-b-")
	createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: siteA, Name: "Primary Candidate A", Timezone: "UTC", Country: "NG"})
	createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: siteB, Name: "Primary Candidate B", Timezone: "UTC", Country: "GB"})

	if _, err := sites.SetPrimary(ctx, 1, siteA); err != nil {
		t.Fatalf("set site A primary: %v", err)
	}
	primary, err := sites.PrimarySite(ctx)
	if err != nil {
		t.Fatalf("get primary site: %v", err)
	}
	if primary.SiteID != siteA {
		t.Fatalf("expected primary site %s, got %s", siteA, primary.SiteID)
	}

	if _, err := sites.SetPrimary(ctx, 1, siteB); err != nil {
		t.Fatalf("set site B primary: %v", err)
	}
	primary, err = sites.PrimarySite(ctx)
	if err != nil {
		t.Fatalf("get primary site after switch: %v", err)
	}
	if primary.SiteID != siteB {
		t.Fatalf("expected primary site to have switched to %s, got %s", siteB, primary.SiteID)
	}

	// Confirm site A was actually un-set, not left as a second primary —
	// this is the exact invariant the DB's partial unique index enforces.
	siteAFetched, err := sites.Get(ctx, siteA)
	if err != nil {
		t.Fatalf("get site A: %v", err)
	}
	if siteAFetched.IsPrimary {
		t.Fatal("site A is still marked primary after site B was set primary — exclusivity violated")
	}
}

func TestPrimarySiteReturnsErrorWhenNoneSet(t *testing.T) {
	q := testQueries(t)
	ctx := context.Background()
	sites := NewSites(q)
	withRestoredPrimarySite(t, ctx, q)

	// Clear any primary left over from a previous run/manual UI testing
	// so this test observes a clean "none set" state — the previous
	// state (if any) is restored via withRestoredPrimarySite's t.Cleanup
	// above, so this doesn't permanently affect anyone using the app.
	_ = q.UnsetAllPrimarySites(ctx)

	if _, err := sites.PrimarySite(ctx); err == nil {
		t.Fatal("expected an error when no site is marked primary, got none")
	}
}
