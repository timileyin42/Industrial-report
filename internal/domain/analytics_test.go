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

func TestSmoothIsolatedDips(t *testing.T) {
	cases := []struct {
		name    string
		in      []float64
		want    []float64
		changed []int
	}{
		{"too short", []float64{5, 0}, []float64{5, 0}, nil},
		// Real production incident: 31.18 -> 0 -> 28.85 kW across three
		// consecutive 5-minute polls — the middle 0 is replaced with the
		// average of its neighbors so the power curve stays continuous
		// instead of showing a fake instantaneous crash to zero.
		{"isolated power glitch", []float64{31.18, 0, 28.85}, []float64{31.18, 30.015, 28.85}, []int{1}},
		{"no glitch, steady climb", []float64{10, 12, 14, 16}, []float64{10, 12, 14, 16}, nil},
		// A real sustained drop (two low readings in a row) is left
		// untouched — same "isolated means exactly one point" rule as
		// filterSpuriousDips.
		{"sustained drop untouched", []float64{20, 0, 0, 22}, []float64{20, 0, 0, 22}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, changed := SmoothIsolatedDips(c.in)
			for i := range got {
				if math.Abs(got[i]-c.want[i]) > 1e-6 {
					t.Errorf("SmoothIsolatedDips(%v)[%d] = %v, want %v", c.in, i, got[i], c.want[i])
				}
			}
			if len(changed) != len(c.changed) {
				t.Errorf("SmoothIsolatedDips(%v) changed = %v, want %v", c.in, changed, c.changed)
			}
		})
	}
}
