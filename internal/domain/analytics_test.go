package domain

import (
	"math"
	"testing"
)

func TestDailyEnergyFromReadings(t *testing.T) {
	cases := []struct {
		name     string
		readings []float64
		want     float64
	}{
		{"no readings", nil, 0},
		{"one reading", []float64{100}, 0},
		{"normal increasing day", []float64{100, 102, 105.5}, 5.5},
		{"reset mid-day", []float64{100, 105, 40, 42}, 7}, // (105-100) + (42-40), reset delta ignored
		{"flat day", []float64{50, 50, 50}, 0},
		// Real production incident: a single glitched poll reported
		// energy_kwh_total=0 while power_kw was genuinely nonzero — the
		// counter never actually reset, it recovered on the very next
		// reading. Naively summing positive deltas would count the
		// 0->1928.4 jump as ~1928 kWh of fake generation.
		{"isolated glitch reading", []float64{1928.3, 1928.3, 0, 1928.4, 1931.1}, 2.8},
		// Two consecutive low readings IS treated as a genuine reset
		// (the isolated-glitch filter requires an immediate next-reading
		// recovery, which two-in-a-row doesn't have) — must not be
		// filtered away.
		{"genuine reset stays low for two readings", []float64{100, 105, 0, 1, 8}, 13}, // (105-100)+(1-0)+(8-1), reset delta ignored
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DailyEnergyFromReadings(c.readings)
			if math.Abs(got-c.want) > 1e-6 {
				t.Errorf("DailyEnergyFromReadings(%v) = %v, want %v", c.readings, got, c.want)
			}
		})
	}
}
