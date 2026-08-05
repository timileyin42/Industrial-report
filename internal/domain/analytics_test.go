package domain

import "testing"

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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DailyEnergyFromReadings(c.readings)
			if got != c.want {
				t.Errorf("DailyEnergyFromReadings(%v) = %v, want %v", c.readings, got, c.want)
			}
		})
	}
}
