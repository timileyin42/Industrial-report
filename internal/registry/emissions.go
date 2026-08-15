package registry

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
)

type Emissions struct {
	analytics      *Analytics
	sites          *Sites
	q              *db.Queries
	defaultCountry string
}

// defaultCountry (GRID_COUNTRY, see cmd/api/main.go) is no longer "the
// one true country" for CO2-offset math — SiteEmissions/FleetEmissions
// below resolve each site's own country (migrations/0010_site_country.sql)
// instead. It's kept only as which country the config/emission-factor
// settings screen shows by default when no site/?country= context is
// given (e.g. an operator opening "Set New Factor" cold).
func NewEmissions(analytics *Analytics, sites *Sites, q *db.Queries, defaultCountry string) *Emissions {
	if defaultCountry == "" {
		defaultCountry = "NG"
	}
	return &Emissions{analytics: analytics, sites: sites, q: q, defaultCountry: defaultCountry}
}

// ErrNoEmissionFactor means no operator has ever set a grid emission
// factor for this country yet. Callers MUST turn this into a 409, never
// fall back to a default — the concept note is explicit that this figure
// must be client-confirmed, not assumed. See migration 0005 and plan notes.
var ErrNoEmissionFactor = errors.New("no grid emission factor configured")

type EmissionFactor struct {
	ID            int64
	KgCO2PerKWh   float64
	Country       string
	Source        string
	EffectiveFrom time.Time
	CreatedAt     time.Time
}

func toEmissionFactor(f db.GridEmissionFactor) EmissionFactor {
	kg := numericToFloat(f.KgCo2PerKwh)
	var kgVal float64
	if kg != nil {
		kgVal = *kg
	}
	return EmissionFactor{
		ID:            f.ID,
		KgCO2PerKWh:   kgVal,
		Country:       f.Country,
		Source:        f.Source,
		EffectiveFrom: f.EffectiveFrom.Time,
		CreatedAt:     f.CreatedAt.Time,
	}
}

func (e *Emissions) Current(ctx context.Context, country string) (EmissionFactor, error) {
	if country == "" {
		country = e.defaultCountry
	}
	factor, err := e.q.GetCurrentEmissionFactor(ctx, country)
	if err != nil {
		if err == pgx.ErrNoRows {
			return EmissionFactor{}, ErrNoEmissionFactor
		}
		return EmissionFactor{}, err
	}
	return toEmissionFactor(factor), nil
}

// allFactorsAscending returns every factor ever set for a country, oldest
// first — feeds factorAsOf's per-period lookup. Distinct from the
// exported History (capped, newest-first, for the settings screen's
// audit list) since correctness here depends on having the *complete*
// set, not a recent sample.
func (e *Emissions) allFactorsAscending(ctx context.Context, country string) ([]EmissionFactor, error) {
	rows, err := e.q.ListAllEmissionFactorsForCountry(ctx, country)
	if err != nil {
		return nil, err
	}
	out := make([]EmissionFactor, 0, len(rows))
	for _, r := range rows {
		out = append(out, toEmissionFactor(r))
	}
	return out, nil
}

// factorAsOf picks the factor that was actually in effect at asOf — the
// most recent one with EffectiveFrom <= asOf — so a later revision never
// silently rewrites CO2 figures for periods before it existed. factors
// must be sorted ascending by EffectiveFrom (allFactorsAscending's order).
//
// If asOf predates every factor on record (only possible for energy from
// before this platform's very first configured factor), this falls back
// to the oldest factor available rather than returning "unconfigured" —
// a documented approximation, not a silent guess: there's no way to know
// what the true historical rate would have been, and using the oldest
// known official value is a defensible floor for that edge case.
func factorAsOf(factors []EmissionFactor, asOf time.Time) EmissionFactor {
	best := factors[0]
	for _, f := range factors {
		if f.EffectiveFrom.After(asOf) {
			break
		}
		best = f
	}
	return best
}

func (e *Emissions) History(ctx context.Context, country string, limit int) ([]EmissionFactor, error) {
	if country == "" {
		country = e.defaultCountry
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := e.q.ListEmissionFactorHistory(ctx, db.ListEmissionFactorHistoryParams{Country: country, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]EmissionFactor, 0, len(rows))
	for _, r := range rows {
		out = append(out, toEmissionFactor(r))
	}
	return out, nil
}

type SetEmissionFactorInput struct {
	KgCO2PerKWh   float64
	Country       string
	Source        string
	EffectiveFrom time.Time
}

func (e *Emissions) Set(ctx context.Context, actorUserID int64, in SetEmissionFactorInput) (EmissionFactor, error) {
	if in.KgCO2PerKWh <= 0 {
		return EmissionFactor{}, errors.New("kg_co2_per_kwh must be positive")
	}
	if err := validateRequired("source", in.Source); err != nil {
		return EmissionFactor{}, err
	}
	if in.Country == "" {
		in.Country = e.defaultCountry
	}

	factor, err := e.q.CreateEmissionFactor(ctx, db.CreateEmissionFactorParams{
		KgCo2PerKwh:     numericOrNull(&in.KgCO2PerKWh),
		Country:         in.Country,
		Source:          in.Source,
		EffectiveFrom:   pgtype.Timestamptz{Time: in.EffectiveFrom, Valid: true},
		CreatedByUserID: pgtype.Int8{Int64: actorUserID, Valid: true},
	})
	if err != nil {
		return EmissionFactor{}, err
	}
	recordAction(ctx, e.q, actorUserID, "emission_factor.set", "grid_emission_factor", in.Country, map[string]any{
		"kg_co2_per_kwh": in.KgCO2PerKWh,
		"source":         in.Source,
	})
	return toEmissionFactor(factor), nil
}

type EmissionPoint struct {
	PeriodStart time.Time
	EnergyKWh   float64
	KgCO2       float64
}

// CountryEmissions is one country's contribution to a fleet-wide
// emissions query — surfaced explicitly rather than silently blended into
// one factor, since a fleet spanning grids genuinely has more than one
// "current factor" and picking one to represent all of them would
// misrepresent the other country's sites' real CO2 offset.
type CountryEmissions struct {
	Country             string
	Factor              EmissionFactor
	CumulativeTonnesCO2 float64
	// Unconfigured is true when no operator has set an emission factor
	// for this country yet — its sites' generation is excluded from
	// CumulativeTonnesCO2 (never guessed), and the caller should surface
	// this rather than let the total quietly look complete.
	Unconfigured bool
}

type EmissionsSeries struct {
	// Factor is populated only when the query resolved to exactly one
	// country with a configured factor (the single-site case, or a
	// fleet/cohort that happens to be single-country) — zero-valued
	// otherwise; check CountryBreakdown for the multi-country case.
	Factor              EmissionFactor
	Points              []EmissionPoint
	CumulativeTonnesCO2 float64
	// CountryBreakdown is set (len > 1, or len == 1 with Unconfigured
	// true) only for fleet-wide queries — nil for a single-site query or
	// a single-country fleet, so existing single-country callers/JSON
	// consumers see no shape change.
	CountryBreakdown []CountryEmissions
}

// SiteEmissions converts a site's energy series into CO2 avoided using
// that site's own country's current emission factor (migrations/
// 0010_site_country.sql) — not one global default. Returns
// ErrNoEmissionFactor (-> 409 at the handler) if that country has none
// configured yet.
func (e *Emissions) SiteEmissions(ctx context.Context, siteID, period string, from, to time.Time) (EmissionsSeries, error) {
	site, err := e.sites.Get(ctx, siteID)
	if err != nil {
		return EmissionsSeries{}, err
	}
	factor, err := e.Current(ctx, site.Country)
	if err != nil {
		return EmissionsSeries{}, err
	}
	factors, err := e.allFactorsAscending(ctx, site.Country)
	if err != nil {
		return EmissionsSeries{}, err
	}
	energy, err := e.analytics.SiteEnergy(ctx, siteID, period, from, to)
	if err != nil {
		return EmissionsSeries{}, err
	}
	return e.toSeries(factor, factors, energy), nil
}

// FleetEmissions sums CO2 avoided across every country represented in the
// fleet/cohort, each using its own emission factor, rather than applying
// one global factor to fleet-wide energy the way this used to work. A
// country with no configured factor contributes zero to the total (never
// a guessed number) but is still reported in CountryBreakdown so the
// caller can surface "GB isn't configured yet" instead of a total that
// looks complete but silently excludes those sites. Only returns
// ErrNoEmissionFactor if EVERY represented country lacks a factor —
// matching the old single-country behavior when there's genuinely just
// one country to resolve.
func (e *Emissions) FleetEmissions(ctx context.Context, cohortID *string, period string, from, to time.Time) (EmissionsSeries, error) {
	byCountry, err := e.analytics.FleetEnergyByCountry(ctx, cohortID, period, from, to)
	if err != nil {
		return EmissionsSeries{}, err
	}

	countries := make([]string, 0, len(byCountry))
	for country := range byCountry {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	merged := map[time.Time]*EmissionPoint{}
	var order []time.Time
	breakdown := make([]CountryEmissions, 0, len(countries))
	var totalTonnes float64
	configuredCount := 0

	for _, country := range countries {
		factor, err := e.Current(ctx, country)
		if errors.Is(err, ErrNoEmissionFactor) {
			breakdown = append(breakdown, CountryEmissions{Country: country, Unconfigured: true})
			continue
		}
		if err != nil {
			return EmissionsSeries{}, err
		}
		configuredCount++

		factors, err := e.allFactorsAscending(ctx, country)
		if err != nil {
			return EmissionsSeries{}, err
		}
		series := e.toSeries(factor, factors, byCountry[country])
		breakdown = append(breakdown, CountryEmissions{Country: country, Factor: factor, CumulativeTonnesCO2: series.CumulativeTonnesCO2})
		totalTonnes += series.CumulativeTonnesCO2

		for _, p := range series.Points {
			point, ok := merged[p.PeriodStart]
			if !ok {
				point = &EmissionPoint{PeriodStart: p.PeriodStart}
				merged[p.PeriodStart] = point
				order = append(order, p.PeriodStart)
			}
			point.EnergyKWh += p.EnergyKWh
			point.KgCO2 += p.KgCO2
		}
	}

	if len(countries) > 0 && configuredCount == 0 {
		return EmissionsSeries{}, ErrNoEmissionFactor
	}

	sort.Slice(order, func(i, j int) bool { return order[i].Before(order[j]) })
	points := make([]EmissionPoint, 0, len(order))
	for _, t := range order {
		points = append(points, *merged[t])
	}

	result := EmissionsSeries{Points: points, CumulativeTonnesCO2: totalTonnes}
	if len(countries) == 1 && !breakdown[0].Unconfigured {
		result.Factor = breakdown[0].Factor
	} else {
		result.CountryBreakdown = breakdown
	}
	return result, nil
}

// toSeries computes CO2 per period using the factor that was actually in
// effect at each period's own date (factorAsOf), not the single current
// one applied uniformly across all of history — see factorAsOf's comment
// for why that distinction matters. currentFactor is still carried on the
// result purely for display (EmissionsSeries.Factor — "what's configured
// right now"); it plays no part in the actual math below.
//
// CumulativeTonnesCO2 is the sum of each period's own correctly-factored
// kg, not currentFactor times energy.CumulativeKWh — that single-factor
// shortcut is exactly the bug this function exists to fix: it would make
// a revised factor retroactively change historical CO2-avoided totals
// every time it's queried, even for energy generated under an earlier,
// different factor.
func (e *Emissions) toSeries(currentFactor EmissionFactor, factors []EmissionFactor, energy EnergySeries) EmissionsSeries {
	series := EmissionsSeries{Factor: currentFactor, Points: make([]EmissionPoint, 0, len(energy.Points))}
	var cumulativeKg float64
	for _, p := range energy.Points {
		factor := factorAsOf(factors, p.PeriodStart)
		kg := p.EnergyKWh * factor.KgCO2PerKWh
		cumulativeKg += kg
		series.Points = append(series.Points, EmissionPoint{PeriodStart: p.PeriodStart, EnergyKWh: p.EnergyKWh, KgCO2: kg})
	}
	series.CumulativeTonnesCO2 = cumulativeKg / 1000
	return series
}
