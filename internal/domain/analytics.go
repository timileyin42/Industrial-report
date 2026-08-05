package domain

// AnomalyBaselineWindowDefaultDays is how many trailing days form a site's
// "normal" baseline for the naive anomaly check.
const AnomalyBaselineWindowDefaultDays = 7

// AnomalyDropThresholdFraction: a day generating less than this fraction of
// its trailing baseline is flagged. Deliberately conservative (a 50% drop,
// not a subtle one) — this check has no weather/irradiance normalization
// (see concept note's "Optional/later" bucket and PR's deferral), so
// day-to-day variance from genuinely normal causes (cloud cover, season)
// isn't accounted for. A tighter threshold without that normalization
// would produce constant false positives; this is a coarse "something is
// clearly wrong" tool, not the season/weather-aware anomaly detection the
// concept note describes as the fuller version.
const AnomalyDropThresholdFraction = 0.5

// DailyEnergyFromReadings computes a day's true generated energy from
// ordered (by ts) cumulative energy_kwh_total readings, correctly handling
// a counter reset within the day: summing only positive deltas captures
// both the pre-reset and post-reset segments' real generation while
// ignoring the reset's own negative delta, without needing to know exactly
// when in the day the reset occurred. Fewer than 2 readings returns 0 —
// not enough data to compute a delta.
func DailyEnergyFromReadings(orderedEnergyReadings []float64) float64 {
	if len(orderedEnergyReadings) < 2 {
		return 0
	}
	var total float64
	for i := 1; i < len(orderedEnergyReadings); i++ {
		delta := orderedEnergyReadings[i] - orderedEnergyReadings[i-1]
		if delta > 0 {
			total += delta
		}
	}
	return total
}
