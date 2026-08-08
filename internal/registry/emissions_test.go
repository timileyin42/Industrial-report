package registry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timileyin42/zgnis-solar/internal/db"
)

// seedSiteWithEnergy creates a site (with a unique 2-letter test country
// so it never collides with real NG/GB emission-factor history in the
// shared dev database) plus one device and one telemetry reading, then
// forces telemetry_daily to refresh so Analytics/Emissions' energy-
// dependent methods have real rollup data to read, not an empty series.
func seedSiteWithEnergy(t *testing.T, ctx context.Context, q *db.Queries, pool *pgxpool.Pool, sites *Sites, country string, cohortID *string, energyKWh float64) string {
	t.Helper()
	siteID := uniqueID("site-" + country + "-")
	site := createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: siteID, Name: "Emissions Test " + country, Timezone: "UTC", Country: country, CohortID: cohortID, SystemSizeKW: ptrFloat(10)})
	siteID = site.SiteID
	deviceID := uniqueID("dev-" + country + "-")
	if _, err := pool.Exec(ctx, `INSERT INTO devices (device_id, site_id, secret_hash) VALUES ($1, $2, 'test-hash')`, deviceID, siteID); err != nil {
		t.Fatalf("insert device: %v", err)
	}
	// energy_kwh_total is a cumulative counter, not a per-reading value —
	// daily energy is computed as (last reading's total - first reading's
	// total) for that day. A single reading would give a trivial delta of
	// zero, so this seeds two: one at 0, one at energyKWh.
	day := time.Now().UTC().Truncate(24*time.Hour).Add(-48 * time.Hour)
	if _, err := pool.Exec(ctx,
		`INSERT INTO telemetry (device_id, site_id, ts, power_kw, energy_kwh_total) VALUES ($1, $2, $3, 0, 0)`,
		deviceID, siteID, day.Add(6*time.Hour),
	); err != nil {
		t.Fatalf("insert telemetry (start): %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO telemetry (device_id, site_id, ts, power_kw, energy_kwh_total) VALUES ($1, $2, $3, $4, $4)`,
		deviceID, siteID, day.Add(18*time.Hour), energyKWh,
	); err != nil {
		t.Fatalf("insert telemetry (end): %v", err)
	}
	refreshTelemetryDaily(t, pool)
	return siteID
}

func ptrFloat(f float64) *float64 { return &f }

// clearTestEmissionFactor deletes any existing factor rows for a
// test-only country code so "not configured yet" assertions stay valid
// across repeated test runs. grid_emission_factor is append-only BY
// DESIGN for real country factors (see migrations/0005_emission_factor.sql
// — historical reports must stay reproducible); that invariant is about
// real data, not test debris under fake 2-letter codes like "QA"/"QB"
// that will never correspond to an actual country.
func clearTestEmissionFactor(t *testing.T, ctx context.Context, pool *pgxpool.Pool, country string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `DELETE FROM grid_emission_factor WHERE country = $1`, country); err != nil {
		t.Fatalf("clear test emission factor for %s: %v", country, err)
	}
}

// TestSiteEmissionsResolvesOwnCountry is the core regression test for
// the per-site-country feature: a site's CO2 figure must come from ITS
// OWN country's factor, never a single global default — even though a
// factor for the platform's overall defaultCountry ("NG") exists too.
func TestSiteEmissionsResolvesOwnCountry(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()
	sites := NewSites(q)
	analytics := NewAnalytics(q)
	emissions := NewEmissions(analytics, sites, q, "NG")

	testCountry := "QA" // deliberately not a real ISO code — test-isolated from real NG/GB history
	clearTestEmissionFactor(t, ctx, pool, testCountry)
	t.Cleanup(func() { clearTestEmissionFactor(t, context.Background(), pool, testCountry) })
	siteID := seedSiteWithEnergy(t, ctx, q, pool, sites, testCountry, nil, 12.5)

	// No factor configured yet for this site's country — must fail even
	// though the platform-wide default country (NG) almost certainly has
	// one configured in a real environment. Falling back silently here
	// is exactly the bug this feature fixes.
	if _, err := emissions.SiteEmissions(ctx, siteID, "daily", time.Now().UTC().Add(-72*time.Hour), time.Now().UTC()); !errors.Is(err, ErrNoEmissionFactor) {
		t.Fatalf("expected ErrNoEmissionFactor before configuring %s's factor, got %v", testCountry, err)
	}

	if _, err := emissions.Set(ctx, 1, SetEmissionFactorInput{KgCO2PerKWh: 0.5, Country: testCountry, Source: "test", EffectiveFrom: time.Now().UTC().Add(-24 * time.Hour)}); err != nil {
		t.Fatalf("set emission factor: %v", err)
	}

	series, err := emissions.SiteEmissions(ctx, siteID, "daily", time.Now().UTC().Add(-72*time.Hour), time.Now().UTC())
	if err != nil {
		t.Fatalf("site emissions after configuring factor: %v", err)
	}
	if series.Factor.Country != testCountry {
		t.Fatalf("expected factor resolved for country %s, got %s", testCountry, series.Factor.Country)
	}
	wantTonnes := 12.5 * 0.5 / 1000
	if diff := series.CumulativeTonnesCO2 - wantTonnes; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("expected cumulative CO2 %.6f tonnes, got %.6f", wantTonnes, series.CumulativeTonnesCO2)
	}
}

// TestFleetEmissionsExcludesUnconfiguredCountry confirms a fleet
// spanning two countries — one configured, one not — reports the
// configured country's real CO2 in the total while excluding the
// unconfigured one (never guessing its factor), and surfaces both in
// the breakdown so the gap is visible rather than silently absorbed.
func TestFleetEmissionsExcludesUnconfiguredCountry(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()
	sites := NewSites(q)
	analytics := NewAnalytics(q)
	emissions := NewEmissions(analytics, sites, q, "NG")

	cohortID := uniqueID("cohort-mixed-")
	configuredCountry := "QB"
	unconfiguredCountry := "QC"
	clearTestEmissionFactor(t, ctx, pool, unconfiguredCountry)
	t.Cleanup(func() {
		clearTestEmissionFactor(t, context.Background(), pool, configuredCountry)
		clearTestEmissionFactor(t, context.Background(), pool, unconfiguredCountry)
	})

	seedSiteWithEnergy(t, ctx, q, pool, sites, configuredCountry, &cohortID, 20)
	seedSiteWithEnergy(t, ctx, q, pool, sites, unconfiguredCountry, &cohortID, 20)

	if _, err := emissions.Set(ctx, 1, SetEmissionFactorInput{KgCO2PerKWh: 0.4, Country: configuredCountry, Source: "test", EffectiveFrom: time.Now().UTC().Add(-24 * time.Hour)}); err != nil {
		t.Fatalf("set emission factor for %s: %v", configuredCountry, err)
	}
	// Deliberately never configure a factor for unconfiguredCountry.

	series, err := emissions.FleetEmissions(ctx, &cohortID, "daily", time.Now().UTC().Add(-72*time.Hour), time.Now().UTC())
	if err != nil {
		t.Fatalf("fleet emissions with one configured country: %v", err)
	}
	if series.CountryBreakdown == nil {
		t.Fatal("expected a country breakdown for a fleet spanning two countries, got nil")
	}

	var sawConfigured, sawUnconfigured bool
	for _, c := range series.CountryBreakdown {
		switch c.Country {
		case configuredCountry:
			sawConfigured = true
			if c.Unconfigured {
				t.Errorf("%s should not be reported as unconfigured", configuredCountry)
			}
			wantTonnes := 20 * 0.4 / 1000
			if diff := c.CumulativeTonnesCO2 - wantTonnes; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("expected %s's contribution to be %.6f tonnes, got %.6f", configuredCountry, wantTonnes, c.CumulativeTonnesCO2)
			}
		case unconfiguredCountry:
			sawUnconfigured = true
			if !c.Unconfigured {
				t.Errorf("%s should be reported as unconfigured", unconfiguredCountry)
			}
			if c.CumulativeTonnesCO2 != 0 {
				t.Errorf("unconfigured country must contribute 0 to its own reported figure, got %.6f", c.CumulativeTonnesCO2)
			}
		}
	}
	if !sawConfigured || !sawUnconfigured {
		t.Fatalf("expected both countries in breakdown, sawConfigured=%v sawUnconfigured=%v", sawConfigured, sawUnconfigured)
	}

	// The total must reflect ONLY the configured country's real energy —
	// never zero (that would silently drop real generation) and never
	// inflated by guessing a factor for the unconfigured one.
	wantTotal := 20 * 0.4 / 1000
	if diff := series.CumulativeTonnesCO2 - wantTotal; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("expected total CO2 %.6f tonnes (configured country only), got %.6f", wantTotal, series.CumulativeTonnesCO2)
	}
}

// TestFleetEmissionsAllUnconfiguredReturnsError confirms the "single
// country" 409-on-missing-factor behavior still holds when a fleet
// happens to span countries that are ALL unconfigured — this must not
// silently succeed with a zero total that looks like a real, complete
// answer.
func TestFleetEmissionsAllUnconfiguredReturnsError(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()
	sites := NewSites(q)
	analytics := NewAnalytics(q)
	emissions := NewEmissions(analytics, sites, q, "NG")

	cohortID := uniqueID("cohort-allunconf-")
	clearTestEmissionFactor(t, ctx, pool, "QD")
	seedSiteWithEnergy(t, ctx, q, pool, sites, "QD", &cohortID, 15)

	if _, err := emissions.FleetEmissions(ctx, &cohortID, "daily", time.Now().UTC().Add(-72*time.Hour), time.Now().UTC()); !errors.Is(err, ErrNoEmissionFactor) {
		t.Fatalf("expected ErrNoEmissionFactor when every country in the fleet is unconfigured, got %v", err)
	}
}
