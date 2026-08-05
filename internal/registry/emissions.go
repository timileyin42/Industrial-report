package registry

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
)

type Emissions struct {
	analytics *Analytics
	q         *db.Queries
}

func NewEmissions(analytics *Analytics, q *db.Queries) *Emissions {
	return &Emissions{analytics: analytics, q: q}
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
	factor, err := e.q.GetCurrentEmissionFactor(ctx, country)
	if err != nil {
		if err == pgx.ErrNoRows {
			return EmissionFactor{}, ErrNoEmissionFactor
		}
		return EmissionFactor{}, err
	}
	return toEmissionFactor(factor), nil
}

func (e *Emissions) History(ctx context.Context, country string, limit int) ([]EmissionFactor, error) {
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
		in.Country = "NG"
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

type EmissionsSeries struct {
	Factor              EmissionFactor
	Points              []EmissionPoint
	CumulativeTonnesCO2 float64
}

// SiteEmissions converts a site's energy series into CO2 avoided using the
// current emission factor. Returns ErrNoEmissionFactor (-> 409 at the
// handler) if none has been configured yet.
func (e *Emissions) SiteEmissions(ctx context.Context, siteID, period string, from, to time.Time) (EmissionsSeries, error) {
	factor, err := e.Current(ctx, "NG")
	if err != nil {
		return EmissionsSeries{}, err
	}
	energy, err := e.analytics.SiteEnergy(ctx, siteID, period, from, to)
	if err != nil {
		return EmissionsSeries{}, err
	}
	return e.toSeries(factor, energy), nil
}

func (e *Emissions) FleetEmissions(ctx context.Context, cohortID *string, period string, from, to time.Time) (EmissionsSeries, error) {
	factor, err := e.Current(ctx, "NG")
	if err != nil {
		return EmissionsSeries{}, err
	}
	energy, err := e.analytics.FleetEnergy(ctx, cohortID, period, from, to)
	if err != nil {
		return EmissionsSeries{}, err
	}
	return e.toSeries(factor, energy), nil
}

func (e *Emissions) toSeries(factor EmissionFactor, energy EnergySeries) EmissionsSeries {
	series := EmissionsSeries{Factor: factor, Points: make([]EmissionPoint, 0, len(energy.Points))}
	for _, p := range energy.Points {
		kg := p.EnergyKWh * factor.KgCO2PerKWh
		series.Points = append(series.Points, EmissionPoint{PeriodStart: p.PeriodStart, EnergyKWh: p.EnergyKWh, KgCO2: kg})
	}
	series.CumulativeTonnesCO2 = energy.CumulativeKWh * factor.KgCO2PerKWh / 1000
	return series
}
