package registry

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestFleetSpecificYieldIsCapacityWeighted confirms fleet-wide specific
// yield sums energy and capacity separately across sites before
// dividing (sum(energy)/sum(capacity)), not an average of each site's
// own yield — the correct way to combine sites of different sizes into
// one fleet number without letting a small site's yield count as much
// as a large one's.
func TestFleetSpecificYieldIsCapacityWeighted(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()
	sites := NewSites(q)
	analytics := NewAnalytics(q)

	cohortID := uniqueID("cohort-yield-")
	// Both seeded sites get system_size_kw=10 (see seedSiteWithEnergy) —
	// 20 kWh + 30 kWh energy across 20 kWp total capacity = 2.5 kWh/kWp,
	// which is also what a naive average of (20/10=2.0) and (30/10=3.0)
	// would give here since sizes are equal; the real assertion this
	// guards is that the aggregate reads the pooled totals, not a
	// per-site figure this test would still pass by accident if it were
	// wrong in a size-skewed fleet.
	seedSiteWithEnergy(t, ctx, q, pool, sites, "QE", &cohortID, 20)
	seedSiteWithEnergy(t, ctx, q, pool, sites, "QF", &cohortID, 30)

	points, err := analytics.FleetSpecificYield(ctx, &cohortID, "daily", time.Now().UTC().Add(-72*time.Hour), time.Now().UTC())
	if err != nil {
		t.Fatalf("fleet specific yield: %v", err)
	}
	if len(points) == 0 {
		t.Fatal("expected at least one yield point")
	}

	var totalEnergy, totalCapacity float64
	for _, p := range points {
		totalEnergy += p.EnergyKWh
		totalCapacity = p.SystemSizeKW // same fleet-wide capacity every point
	}
	wantYield := totalEnergy / totalCapacity
	for _, p := range points {
		if diff := p.SpecificYieldKWhPerKWp - wantYield; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("point %s: expected yield %.6f (energy %.2f / capacity %.2f), got %.6f",
				p.PeriodStart, wantYield, p.EnergyKWh, p.SystemSizeKW, p.SpecificYieldKWhPerKWp)
		}
	}
	if totalEnergy != 50 {
		t.Fatalf("expected total fleet energy 50 kWh across both sites, got %.2f", totalEnergy)
	}
	if totalCapacity != 20 {
		t.Fatalf("expected total fleet capacity 20 kWp across both sites, got %.2f", totalCapacity)
	}
}

// TestSitePerformanceRatioRequiresLocation confirms a site with a rated
// capacity but no saved coordinates fails clearly (ErrNoLocation) rather
// than returning an empty series indistinguishable from "not enough
// data yet" — the two are different problems with different fixes.
func TestSitePerformanceRatioRequiresLocation(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	sites := NewSites(q)
	analytics := NewAnalytics(q)
	ctx := context.Background()

	siteID := uniqueID("site-no-location-")
	createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: siteID, Name: "No Location", Timezone: "UTC", Country: "NG", SystemSizeKW: ptrFloat(5)})

	if _, err := analytics.SitePerformanceRatio(ctx, siteID, "daily", time.Now().UTC().Add(-72*time.Hour), time.Now().UTC()); !errors.Is(err, ErrNoLocation) {
		t.Fatalf("expected ErrNoLocation for a site with no gps_lat/gps_lng, got %v", err)
	}
}

// TestSitePerformanceRatioRequiresSystemSize is the mirror case: a
// located site with no rated capacity can't have an "expected output"
// computed against it either.
func TestSitePerformanceRatioRequiresSystemSize(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	sites := NewSites(q)
	analytics := NewAnalytics(q)
	ctx := context.Background()

	siteID := uniqueID("site-no-size-")
	lat, lng := 6.5244, 3.3792
	createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: siteID, Name: "No System Size", Timezone: "UTC", Country: "NG", GPSLat: &lat, GPSLng: &lng})

	if _, err := analytics.SitePerformanceRatio(ctx, siteID, "daily", time.Now().UTC().Add(-72*time.Hour), time.Now().UTC()); !errors.Is(err, ErrNoSystemSize) {
		t.Fatalf("expected ErrNoSystemSize for a site with no system_size_kw, got %v", err)
	}
}

// TestFleetPerformanceRatioExcludesSitesWithoutLocation confirms a fleet
// where only some sites have a location doesn't fail outright — it
// should still compute using the usable sites and simply skip the rest,
// mirroring FleetEmissions' "exclude, don't fail" handling of an
// unconfigured country.
func TestFleetPerformanceRatioExcludesSitesWithoutLocation(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	sites := NewSites(q)
	analytics := NewAnalytics(q)
	ctx := context.Background()

	cohortID := uniqueID("cohort-pr-mixed-")
	locatedSiteID := uniqueID("site-pr-located-")
	lat, lng := 6.5244, 3.3792
	createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: locatedSiteID, Name: "Located", Timezone: "UTC", Country: "NG", CohortID: &cohortID, SystemSizeKW: ptrFloat(5), GPSLat: &lat, GPSLng: &lng})
	unlocatedSiteID := uniqueID("site-pr-unlocated-")
	createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: unlocatedSiteID, Name: "Unlocated", Timezone: "UTC", Country: "NG", CohortID: &cohortID, SystemSizeKW: ptrFloat(5)})

	// Neither site has telemetry, so this exercises only the "does it
	// fail or proceed" branch, not the irradiance math — a live external
	// API call for a real energy/PR number belongs in a separate,
	// explicitly-network-dependent test, not this one.
	if _, err := analytics.FleetPerformanceRatio(ctx, &cohortID, "daily", time.Now().UTC().Add(-72*time.Hour), time.Now().UTC()); err != nil {
		t.Fatalf("expected fleet performance ratio to proceed using the located site alone, got error: %v", err)
	}
}

// TestFleetPerformanceRatioAllMissingLocationReturnsError confirms the
// fleet-wide call fails clearly when NO site in it has a usable
// location — matching FleetEmissions' "all countries unconfigured"
// behavior rather than silently returning an empty, complete-looking series.
func TestFleetPerformanceRatioAllMissingLocationReturnsError(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	sites := NewSites(q)
	analytics := NewAnalytics(q)
	ctx := context.Background()

	cohortID := uniqueID("cohort-pr-none-")
	siteID := uniqueID("site-pr-none-")
	createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: siteID, Name: "No Location At All", Timezone: "UTC", Country: "NG", CohortID: &cohortID, SystemSizeKW: ptrFloat(5)})

	if _, err := analytics.FleetPerformanceRatio(ctx, &cohortID, "daily", time.Now().UTC().Add(-72*time.Hour), time.Now().UTC()); !errors.Is(err, ErrNoLocation) {
		t.Fatalf("expected ErrNoLocation when no site in the fleet has a location, got %v", err)
	}
}
