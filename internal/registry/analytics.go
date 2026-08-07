package registry

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
)

type Analytics struct {
	q *db.Queries
}

func NewAnalytics(q *db.Queries) *Analytics {
	return &Analytics{q: q}
}

// dailyRow is one day's generation across every device at a site (or, in
// the fleet-wide case, one device's contribution before being summed
// further by the caller) — the day-granularity building block every
// period (weekly/monthly/cumulative) view is composed from in Go, rather
// than a second physical rollup per window.
type dailyRow struct {
	Day             time.Time
	EnergyKWh       float64
	PeakPowerKW     float64
	PeakDeviceID    string
	ReadingCount    int64
	BackfilledCount int64
}

var ErrNoSystemSize = errors.New("site has no system_size_kw configured")

// loadSiteDailyRows fetches per-device-per-day rollup rows for one site and
// collapses them into one row per day, summing energy across the site's
// devices and taking the max peak (with the device that produced it, for
// the time-of-day lookup). Any device-day flagged has_reset falls back to
// a bounded raw-telemetry read rather than trusting the rollup's
// max-minus-min blindly across a counter reset.
func (a *Analytics) loadSiteDailyRows(ctx context.Context, siteID string, from, to time.Time) ([]dailyRow, error) {
	rows, err := a.q.ListSiteDailyRollup(ctx, db.ListSiteDailyRollupParams{
		SiteID: siteID,
		Day:    pgtype.Timestamptz{Time: from, Valid: true},
		Day_2:  pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	byDay := map[time.Time]*dailyRow{}
	var order []time.Time
	for _, r := range rows {
		day := r.Day.Time
		out, ok := byDay[day]
		if !ok {
			out = &dailyRow{Day: day}
			byDay[day] = out
			order = append(order, day)
		}

		energy, err := a.deviceDayEnergy(ctx, r.DeviceID, day, r.HasReset, r.EnergyStartKwh, r.EnergyEndKwh)
		if err != nil {
			return nil, err
		}
		out.EnergyKWh += energy
		out.ReadingCount += r.ReadingCount
		out.BackfilledCount += r.BackfilledCount
		if r.PeakPowerKw.Valid && r.PeakPowerKw.Float64 > out.PeakPowerKW {
			out.PeakPowerKW = r.PeakPowerKw.Float64
			out.PeakDeviceID = r.DeviceID
		}
	}

	result := make([]dailyRow, 0, len(order))
	for _, day := range order {
		result = append(result, *byDay[day])
	}
	return result, nil
}

// loadFleetDailyRows is the fleet-wide equivalent, optionally scoped to a
// cohort, collapsing per-device-per-day rows into one row per day across
// every site.
func (a *Analytics) loadFleetDailyRows(ctx context.Context, cohortID *string, from, to time.Time) ([]dailyRow, error) {
	rows, err := a.q.ListFleetDailyRollup(ctx, db.ListFleetDailyRollupParams{
		Day:      pgtype.Timestamptz{Time: from, Valid: true},
		Day_2:    pgtype.Timestamptz{Time: to, Valid: true},
		CohortID: textOrNull(cohortID),
	})
	if err != nil {
		return nil, err
	}

	byDay := map[time.Time]*dailyRow{}
	var order []time.Time
	for _, r := range rows {
		day := r.Day.Time
		out, ok := byDay[day]
		if !ok {
			out = &dailyRow{Day: day}
			byDay[day] = out
			order = append(order, day)
		}

		energy, err := a.deviceDayEnergy(ctx, r.DeviceID, day, r.HasReset, r.EnergyStartKwh, r.EnergyEndKwh)
		if err != nil {
			return nil, err
		}
		out.EnergyKWh += energy
		out.ReadingCount += r.ReadingCount
		out.BackfilledCount += r.BackfilledCount
		if r.PeakPowerKw.Valid && r.PeakPowerKw.Float64 > out.PeakPowerKW {
			out.PeakPowerKW = r.PeakPowerKw.Float64
			out.PeakDeviceID = r.DeviceID
		}
	}

	result := make([]dailyRow, 0, len(order))
	for _, day := range order {
		result = append(result, *byDay[day])
	}
	return result, nil
}

// siteCountries returns each site's own country (migrations/0010_site_country.sql),
// optionally scoped to a cohort — the lookup FleetEnergyByCountry needs to
// group fleet-wide rows by country instead of collapsing them into one
// fleet-wide total the way loadFleetDailyRows does.
func (a *Analytics) siteCountries(ctx context.Context, cohortID *string) (map[string]string, error) {
	rows, err := a.q.ListSiteCountries(ctx, textOrNull(cohortID))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.SiteID] = r.Country
	}
	return out, nil
}

// FleetEnergyByCountry is loadFleetDailyRows' country-partitioned sibling
// — grouped by (day, site's country) rather than just day, so
// Emissions.FleetEmissions can apply each country's own grid emission
// factor instead of one global default (see internal/registry/emissions.go).
func (a *Analytics) FleetEnergyByCountry(ctx context.Context, cohortID *string, period string, from, to time.Time) (map[string]EnergySeries, error) {
	rows, err := a.q.ListFleetDailyRollup(ctx, db.ListFleetDailyRollupParams{
		Day:      pgtype.Timestamptz{Time: from, Valid: true},
		Day_2:    pgtype.Timestamptz{Time: to, Valid: true},
		CohortID: textOrNull(cohortID),
	})
	if err != nil {
		return nil, err
	}

	countryBySite, err := a.siteCountries(ctx, cohortID)
	if err != nil {
		return nil, err
	}

	byCountry := map[string]map[time.Time]*dailyRow{}
	orderByCountry := map[string][]time.Time{}
	for _, r := range rows {
		country := countryBySite[r.SiteID]
		if country == "" {
			// Shouldn't happen — country is NOT NULL on sites — but a
			// device whose site was deleted out from under it (if that
			// ever becomes possible) falls in an honestly-labeled bucket
			// rather than silently joining someone else's country total.
			country = "unknown"
		}
		dayMap, ok := byCountry[country]
		if !ok {
			dayMap = map[time.Time]*dailyRow{}
			byCountry[country] = dayMap
		}

		day := r.Day.Time
		out, ok := dayMap[day]
		if !ok {
			out = &dailyRow{Day: day}
			dayMap[day] = out
			orderByCountry[country] = append(orderByCountry[country], day)
		}

		energy, err := a.deviceDayEnergy(ctx, r.DeviceID, day, r.HasReset, r.EnergyStartKwh, r.EnergyEndKwh)
		if err != nil {
			return nil, err
		}
		out.EnergyKWh += energy
		out.ReadingCount += r.ReadingCount
		out.BackfilledCount += r.BackfilledCount
		if r.PeakPowerKw.Valid && r.PeakPowerKw.Float64 > out.PeakPowerKW {
			out.PeakPowerKW = r.PeakPowerKw.Float64
			out.PeakDeviceID = r.DeviceID
		}
	}

	result := make(map[string]EnergySeries, len(byCountry))
	for country, dayMap := range byCountry {
		rowsForCountry := make([]dailyRow, 0, len(dayMap))
		for _, day := range orderByCountry[country] {
			rowsForCountry = append(rowsForCountry, *dayMap[day])
		}
		result[country] = bucketDailyRows(rowsForCountry, period)
	}
	return result, nil
}

func (a *Analytics) deviceDayEnergy(ctx context.Context, deviceID string, day time.Time, hasReset bool, startKwh, endKwh pgtype.Float8) (float64, error) {
	if !hasReset {
		if !startKwh.Valid || !endKwh.Valid {
			return 0, nil
		}
		return endKwh.Float64 - startKwh.Float64, nil
	}

	rows, err := a.q.GetRawEnergyReadingsForDeviceDay(ctx, db.GetRawEnergyReadingsForDeviceDayParams{
		DeviceID: deviceID,
		Ts:       pgtype.Timestamptz{Time: day, Valid: true},
		Ts_2:     pgtype.Timestamptz{Time: day.Add(24 * time.Hour), Valid: true},
	})
	if err != nil {
		return 0, err
	}
	readings := make([]float64, len(rows))
	for i, r := range rows {
		readings[i] = r.EnergyKwhTotal
	}
	return domain.DailyEnergyFromReadings(readings), nil
}

// PeriodBucket returns the start of the daily/weekly/monthly bucket a day
// falls into. Weekly buckets start Monday (ISO-style); monthly buckets
// start on the 1st. An unrecognized period defaults to daily (no bucketing).
func PeriodBucket(day time.Time, period string) time.Time {
	switch period {
	case "weekly":
		offset := (int(day.Weekday()) + 6) % 7 // Monday=0
		return day.AddDate(0, 0, -offset)
	case "monthly":
		return time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, day.Location())
	default:
		return day
	}
}

// periodBucketEnd returns the exclusive end of the bucket a PeriodBucket
// start belongs to — used to decide whether a site existed during ANY part
// of a period (not just at its exact start), e.g. for Trends' capacity
// count.
func periodBucketEnd(bucketStart time.Time, period string) time.Time {
	switch period {
	case "weekly":
		return bucketStart.AddDate(0, 0, 7)
	case "monthly":
		return bucketStart.AddDate(0, 1, 0)
	default:
		return bucketStart.AddDate(0, 0, 1)
	}
}

type EnergyPoint struct {
	PeriodStart     time.Time
	EnergyKWh       float64
	ReadingCount    int64
	BackfilledCount int64
}

type EnergySeries struct {
	Points []EnergyPoint
	// CumulativeKWh sums every point in the requested range — not a
	// device's full lifetime total, which would need an unbounded scan
	// back to install date. Labeled as such in the API response.
	CumulativeKWh float64
}

func bucketDailyRows(rows []dailyRow, period string) EnergySeries {
	byBucket := map[time.Time]*EnergyPoint{}
	var order []time.Time
	for _, r := range rows {
		bucket := PeriodBucket(r.Day, period)
		p, ok := byBucket[bucket]
		if !ok {
			p = &EnergyPoint{PeriodStart: bucket}
			byBucket[bucket] = p
			order = append(order, bucket)
		}
		p.EnergyKWh += r.EnergyKWh
		p.ReadingCount += r.ReadingCount
		p.BackfilledCount += r.BackfilledCount
	}

	series := EnergySeries{Points: make([]EnergyPoint, 0, len(order))}
	for _, b := range order {
		point := *byBucket[b]
		series.Points = append(series.Points, point)
		series.CumulativeKWh += point.EnergyKWh
	}
	return series
}

func (a *Analytics) SiteEnergy(ctx context.Context, siteID, period string, from, to time.Time) (EnergySeries, error) {
	rows, err := a.loadSiteDailyRows(ctx, siteID, from, to)
	if err != nil {
		return EnergySeries{}, err
	}
	return bucketDailyRows(rows, period), nil
}

func (a *Analytics) FleetEnergy(ctx context.Context, cohortID *string, period string, from, to time.Time) (EnergySeries, error) {
	rows, err := a.loadFleetDailyRows(ctx, cohortID, from, to)
	if err != nil {
		return EnergySeries{}, err
	}
	return bucketDailyRows(rows, period), nil
}

type YieldPoint struct {
	PeriodStart            time.Time
	EnergyKWh              float64
	SystemSizeKW           float64
	SpecificYieldKWhPerKWp float64
}

func (a *Analytics) SiteSpecificYield(ctx context.Context, siteID, period string, from, to time.Time) ([]YieldPoint, error) {
	site, err := a.q.GetSite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	systemSizeKW := numericToFloat(site.SystemSizeKw)
	if systemSizeKW == nil || *systemSizeKW <= 0 {
		return nil, ErrNoSystemSize
	}

	series, err := a.SiteEnergy(ctx, siteID, period, from, to)
	if err != nil {
		return nil, err
	}

	points := make([]YieldPoint, 0, len(series.Points))
	for _, p := range series.Points {
		points = append(points, YieldPoint{
			PeriodStart:            p.PeriodStart,
			EnergyKWh:              p.EnergyKWh,
			SystemSizeKW:           *systemSizeKW,
			SpecificYieldKWhPerKWp: p.EnergyKWh / *systemSizeKW,
		})
	}
	return points, nil
}

type PeakPoint struct {
	Day         time.Time
	PeakPowerKW float64
	OccurredAt  *time.Time
}

// SitePeak is always day-granularity (peak-time-of-day isn't meaningful
// bucketed into a week/month), regardless of what a caller might pass as
// a period elsewhere.
func (a *Analytics) SitePeak(ctx context.Context, siteID string, from, to time.Time) ([]PeakPoint, error) {
	rows, err := a.loadSiteDailyRows(ctx, siteID, from, to)
	if err != nil {
		return nil, err
	}

	points := make([]PeakPoint, 0, len(rows))
	for _, r := range rows {
		point := PeakPoint{Day: r.Day, PeakPowerKW: r.PeakPowerKW}
		if r.PeakDeviceID != "" && r.PeakPowerKW > 0 {
			ts, err := a.q.GetPeakReadingTimeForDeviceDay(ctx, db.GetPeakReadingTimeForDeviceDayParams{
				DeviceID: r.PeakDeviceID,
				Ts:       pgtype.Timestamptz{Time: r.Day, Valid: true},
				Ts_2:     pgtype.Timestamptz{Time: r.Day.Add(24 * time.Hour), Valid: true},
				PowerKw:  r.PeakPowerKW,
			})
			if err == nil && ts.Valid {
				point.OccurredAt = &ts.Time
			}
		}
		points = append(points, point)
	}
	return points, nil
}

type CapacityFactorPoint struct {
	PeriodStart       time.Time
	EnergyKWh         float64
	TheoreticalMaxKWh float64
	CapacityFactorPct float64
}

// SiteCapacityFactor is deliberately NOT Performance Ratio: it compares
// actual output to a theoretical max derived purely from nameplate size
// and elapsed time, with no weather/irradiance adjustment. PR (actual vs.
// what the system should produce for prevailing conditions) is explicitly
// deferred — see plan notes.
func (a *Analytics) SiteCapacityFactor(ctx context.Context, siteID, period string, from, to time.Time) ([]CapacityFactorPoint, error) {
	site, err := a.q.GetSite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	systemSizeKW := numericToFloat(site.SystemSizeKw)
	if systemSizeKW == nil || *systemSizeKW <= 0 {
		return nil, ErrNoSystemSize
	}

	rows, err := a.loadSiteDailyRows(ctx, siteID, from, to)
	if err != nil {
		return nil, err
	}

	byBucket := map[time.Time]*CapacityFactorPoint{}
	var order []time.Time
	for _, r := range rows {
		bucket := PeriodBucket(r.Day, period)
		p, ok := byBucket[bucket]
		if !ok {
			p = &CapacityFactorPoint{PeriodStart: bucket}
			byBucket[bucket] = p
			order = append(order, bucket)
		}
		p.EnergyKWh += r.EnergyKWh
		p.TheoreticalMaxKWh += *systemSizeKW * 24 // one day's worth of hours
	}

	points := make([]CapacityFactorPoint, 0, len(order))
	for _, b := range order {
		p := *byBucket[b]
		if p.TheoreticalMaxKWh > 0 {
			p.CapacityFactorPct = p.EnergyKWh / p.TheoreticalMaxKWh * 100
		}
		points = append(points, p)
	}
	return points, nil
}
