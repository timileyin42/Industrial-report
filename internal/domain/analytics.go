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

// filterSpuriousDips drops a single reading that plunges far below both
// its immediate neighbors and is immediately followed by a return to the
// earlier trajectory — a lone glitched sample (observed in production: a
// vendor API hiccup reported energy_kwh_total=0 for exactly one 5-minute
// poll while power_kw was genuinely nonzero at that instant), not a
// genuine counter reset. Naively summing positive deltas around such a
// point counts the recovery jump back up to the real value as if it were
// new generation — e.g. a glitched drop to 0 between two ~1928 kWh
// readings falsely added ~1928 kWh to a single day's total. A genuine
// reset stays low across multiple consecutive readings as the counter
// climbs back up from near-zero, which this deliberately leaves
// untouched — only an isolated single-point dip is dropped.
func filterSpuriousDips(readings []float64) []float64 {
	if len(readings) < 3 {
		return readings
	}
	out := make([]float64, 0, len(readings))
	out = append(out, readings[0])
	for i := 1; i < len(readings)-1; i++ {
		prev, cur, next := readings[i-1], readings[i], readings[i+1]
		if cur < prev*0.5 && cur < next*0.5 && next >= prev*0.9 {
			continue // isolated glitch — drop it, don't diff against it at all
		}
		out = append(out, cur)
	}
	out = append(out, readings[len(readings)-1])
	return out
}

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
	readings := filterSpuriousDips(orderedEnergyReadings)
	var total float64
	for i := 1; i < len(readings); i++ {
		delta := readings[i] - readings[i-1]
		if delta > 0 {
			total += delta
		}
	}
	return total
}
