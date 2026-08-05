package registry

import (
	"context"
	"sort"
	"time"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/pagination"
)

type Benchmark struct {
	analytics *Analytics
}

func NewBenchmark(analytics *Analytics) *Benchmark {
	return &Benchmark{analytics: analytics}
}

// periodRangeCovering returns a [start, end] range wide enough to contain
// both the bucket asOf falls into and the immediately preceding bucket, so
// a single SiteEnergy/FleetEnergy call can be bucketed to yield both points
// at once.
func periodRangeCovering(asOf time.Time, period string) (rangeStart, currentStart, previousStart time.Time) {
	currentStart = PeriodBucket(asOf, period)
	switch period {
	case "weekly":
		previousStart = currentStart.AddDate(0, 0, -7)
	case "monthly":
		previousStart = currentStart.AddDate(0, -1, 0)
	default:
		previousStart = currentStart.AddDate(0, 0, -1)
	}
	return previousStart, currentStart, previousStart
}

type HistoryComparison struct {
	CurrentPeriodStart  time.Time
	PreviousPeriodStart time.Time
	CurrentEnergyKWh    float64
	PreviousEnergyKWh   float64
	ChangePct           *float64 // nil if there's no previous-period data to compare against
}

// CompareHistory is "a site against its own history" (concept note §10).
func (b *Benchmark) CompareHistory(ctx context.Context, siteID, period string, asOf time.Time) (HistoryComparison, error) {
	rangeStart, currentStart, previousStart := periodRangeCovering(asOf, period)
	series, err := b.analytics.SiteEnergy(ctx, siteID, period, rangeStart, asOf)
	if err != nil {
		return HistoryComparison{}, err
	}

	result := HistoryComparison{CurrentPeriodStart: currentStart, PreviousPeriodStart: previousStart}
	for _, p := range series.Points {
		switch {
		case p.PeriodStart.Equal(currentStart):
			result.CurrentEnergyKWh = p.EnergyKWh
		case p.PeriodStart.Equal(previousStart):
			result.PreviousEnergyKWh = p.EnergyKWh
		}
	}
	if result.PreviousEnergyKWh > 0 {
		pct := (result.CurrentEnergyKWh - result.PreviousEnergyKWh) / result.PreviousEnergyKWh * 100
		result.ChangePct = &pct
	}
	return result, nil
}

type FleetComparison struct {
	SiteID         string
	SiteEnergyKWh  float64
	FleetAvgKWh    float64
	PercentileRank float64
	SiteCount      int
}

// sitePeriodEnergy computes one site's energy for the single period bucket
// asOf falls into. Used by CompareFleet/Benchmark, which need a per-site
// figure rather than FleetEnergy's fleet-wide sum.
func (b *Benchmark) sitePeriodEnergy(ctx context.Context, siteID, period string, asOf time.Time) (float64, error) {
	bucketStart := PeriodBucket(asOf, period)
	series, err := b.analytics.SiteEnergy(ctx, siteID, period, bucketStart, asOf)
	if err != nil {
		return 0, err
	}
	for _, p := range series.Points {
		if p.PeriodStart.Equal(bucketStart) {
			return p.EnergyKWh, nil
		}
	}
	return 0, nil
}

// CompareFleet is "a site against the fleet average, or its percentile
// rank within the fleet" (concept note §10). Fetches every site's period
// figure individually — O(sites) queries, acceptable at this stack's
// pilot scale (AGENTS.md's own "don't pre-optimize" guidance); revisit
// with a grouped query if the fleet grows into the hundreds+.
func (b *Benchmark) CompareFleet(ctx context.Context, siteID, period string, asOf time.Time) (FleetComparison, error) {
	sites, err := b.analytics.q.ListSitesForAnalytics(ctx)
	if err != nil {
		return FleetComparison{}, err
	}

	values := make([]float64, 0, len(sites))
	var targetValue float64
	for _, s := range sites {
		v, err := b.sitePeriodEnergy(ctx, s.SiteID, period, asOf)
		if err != nil {
			return FleetComparison{}, err
		}
		values = append(values, v)
		if s.SiteID == siteID {
			targetValue = v
		}
	}

	result := FleetComparison{SiteID: siteID, SiteEnergyKWh: targetValue, SiteCount: len(values)}
	if len(values) == 0 {
		return result, nil
	}
	var sum float64
	belowOrEqual := 0
	for _, v := range values {
		sum += v
		if v <= targetValue {
			belowOrEqual++
		}
	}
	result.FleetAvgKWh = sum / float64(len(values))
	result.PercentileRank = float64(belowOrEqual) / float64(len(values)) * 100
	return result, nil
}

type SegmentStat struct {
	SegmentKey     string
	SiteCount      int
	TotalEnergyKWh float64
	AvgEnergyKWh   float64
}

// Benchmark segments the fleet by system-size band, inverter model, or
// cohort, and reports each segment's period energy. "Region" per concept
// note §10 isn't an available field on `sites` today (no dedicated
// region/state column) — cohort_id is used as the closest available
// grouping, documented here rather than inventing a region value from
// free-text `address`.
func (b *Benchmark) Segment(ctx context.Context, segmentBy, period string, asOf time.Time, cursorToken string, limit int) ([]SegmentStat, string, error) {
	if limit <= 0 || limit > 200 {
		limit = pagination.DefaultPageLimit
	}

	sites, err := b.analytics.q.ListSitesForAnalytics(ctx)
	if err != nil {
		return nil, "", err
	}

	totals := map[string]*SegmentStat{}
	var keys []string
	for _, s := range sites {
		key := segmentKey(segmentBy, s)
		if key == "" {
			continue
		}
		v, err := b.sitePeriodEnergy(ctx, s.SiteID, period, asOf)
		if err != nil {
			return nil, "", err
		}
		stat, ok := totals[key]
		if !ok {
			stat = &SegmentStat{SegmentKey: key}
			totals[key] = stat
			keys = append(keys, key)
		}
		stat.SiteCount++
		stat.TotalEnergyKWh += v
	}
	sort.Strings(keys)

	startIdx := 0
	if cursorToken != "" {
		c, err := pagination.Decode(cursorToken)
		if err != nil {
			return nil, "", err
		}
		for i, k := range keys {
			if k > c.Tiebreak {
				startIdx = i
				break
			}
			startIdx = i + 1
		}
	}

	endIdx := startIdx + limit
	if endIdx > len(keys) {
		endIdx = len(keys)
	}

	out := make([]SegmentStat, 0, endIdx-startIdx)
	for _, k := range keys[startIdx:endIdx] {
		stat := *totals[k]
		if stat.SiteCount > 0 {
			stat.AvgEnergyKWh = stat.TotalEnergyKWh / float64(stat.SiteCount)
		}
		out = append(out, stat)
	}

	next := ""
	if endIdx < len(keys) {
		next, err = pagination.Encode(pagination.Cursor{Time: asOf, Tiebreak: keys[endIdx-1]})
		if err != nil {
			return nil, "", err
		}
	}
	return out, next, nil
}

func segmentKey(segmentBy string, s db.ListSitesForAnalyticsRow) string {
	switch segmentBy {
	case "inverter_make_model":
		if s.InverterMakeModel.Valid {
			return s.InverterMakeModel.String
		}
		return ""
	case "cohort", "region":
		if s.CohortID.Valid {
			return s.CohortID.String
		}
		return ""
	default: // "system_size_band"
		size := numericToFloat(s.SystemSizeKw)
		if size == nil {
			return ""
		}
		switch {
		case *size < 5:
			return "0-5kW"
		case *size < 10:
			return "5-10kW"
		case *size < 20:
			return "10-20kW"
		default:
			return "20kW+"
		}
	}
}

type TrendPoint struct {
	PeriodStart     time.Time
	TotalCapacityKW float64
	SiteCount       int
	TotalEnergyKWh  float64
	MoMChangePct    *float64
}

// Trends is fleet-level growth over time (concept note §10): installed
// capacity, energy, and month-on-month movement.
func (b *Benchmark) Trends(ctx context.Context, period string, from, to time.Time) ([]TrendPoint, error) {
	sites, err := b.analytics.q.ListSitesForAnalytics(ctx)
	if err != nil {
		return nil, err
	}
	energy, err := b.analytics.FleetEnergy(ctx, nil, period, from, to)
	if err != nil {
		return nil, err
	}

	points := make([]TrendPoint, 0, len(energy.Points))
	var prevEnergy *float64
	for _, ep := range energy.Points {
		periodEnd := periodBucketEnd(ep.PeriodStart, period)
		var capacity float64
		count := 0
		for _, s := range sites {
			if s.CreatedAt.Valid && !s.CreatedAt.Time.Before(periodEnd) {
				continue // site didn't exist during any part of this period
			}
			if size := numericToFloat(s.SystemSizeKw); size != nil {
				capacity += *size
			}
			count++
		}
		tp := TrendPoint{PeriodStart: ep.PeriodStart, TotalCapacityKW: capacity, SiteCount: count, TotalEnergyKWh: ep.EnergyKWh}
		if prevEnergy != nil && *prevEnergy > 0 {
			pct := (ep.EnergyKWh - *prevEnergy) / *prevEnergy * 100
			tp.MoMChangePct = &pct
		}
		e := ep.EnergyKWh
		prevEnergy = &e
		points = append(points, tp)
	}
	return points, nil
}

type CohortSummary struct {
	CohortID        string
	TotalCapacityKW float64
	SiteCount       int
	Energy          EnergySeries
}

// Cohort aggregates energy + capacity for one project/cohort grouping
// (concept note §10: "energy and avoided emissions can be aggregated and
// reported per cohort as well as fleet-wide"). Operator-only at the
// handler level — a cohort can include sites beyond the caller's own.
func (b *Benchmark) Cohort(ctx context.Context, cohortID, period string, from, to time.Time) (CohortSummary, error) {
	sites, err := b.analytics.q.ListSitesForAnalytics(ctx)
	if err != nil {
		return CohortSummary{}, err
	}

	var capacity float64
	count := 0
	for _, s := range sites {
		if !s.CohortID.Valid || s.CohortID.String != cohortID {
			continue
		}
		count++
		if size := numericToFloat(s.SystemSizeKw); size != nil {
			capacity += *size
		}
	}

	c := cohortID
	energy, err := b.analytics.FleetEnergy(ctx, &c, period, from, to)
	if err != nil {
		return CohortSummary{}, err
	}

	return CohortSummary{CohortID: cohortID, TotalCapacityKW: capacity, SiteCount: count, Energy: energy}, nil
}
