package domain

import (
	"testing"
	"time"
)

func TestDetectEnergyReset(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	cases := []struct {
		name     string
		previous *float64
		current  float64
		want     bool
	}{
		{"no previous reading is never a reset", nil, 5.0, false},
		{"increase is not a reset", f(100.0), 105.0, false},
		{"equal is not a reset", f(100.0), 100.0, false},
		{"decrease is a reset", f(100.0), 40.0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DetectEnergyReset(c.previous, c.current)
			if got != c.want {
				t.Errorf("DetectEnergyReset(%v, %v) = %v, want %v", c.previous, c.current, got, c.want)
			}
		})
	}
}

func TestDetectEnergyReset_LegitimateBackfillDoesNotFalsePositive(t *testing.T) {
	// A backfilled reading must be compared against the baseline immediately
	// preceding it BY TS, not against whatever was inserted most recently by
	// arrival order. Here the "previous by ts" for an older backfilled
	// reading is an even-older, smaller value, so no reset should trip —
	// this documents the semantics the ingestor's query must respect
	// (`WHERE ts < $2 ORDER BY ts DESC LIMIT 1`), even though this pure
	// function doesn't do the query itself.
	previousByTs := 95.0 // the reading immediately before the backfilled one, chronologically
	backfilledCurrent := 96.0
	if DetectEnergyReset(&previousByTs, backfilledCurrent) {
		t.Errorf("legitimate backfilled reading falsely flagged as a reset")
	}
}

func TestClassifyProvenance(t *testing.T) {
	base := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name       string
		ts         time.Time
		receivedAt time.Time
		want       Provenance
	}{
		{"near real-time", base, base.Add(30 * time.Second), ProvenanceMetered},
		{"just under threshold", base, base.Add(BackfillThreshold - time.Second), ProvenanceMetered},
		{"just over threshold", base, base.Add(BackfillThreshold + time.Second), ProvenanceBackfilled},
		{"far in the past", base, base.Add(2 * time.Hour), ProvenanceBackfilled},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ClassifyProvenance(c.ts, c.receivedAt)
			if got != c.want {
				t.Errorf("ClassifyProvenance(%v, %v) = %v, want %v", c.ts, c.receivedAt, got, c.want)
			}
		})
	}
}

func TestPowerCeilingKW(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	cases := []struct {
		name string
		size *float64
		want float64
	}{
		{"no site size falls back to default", nil, DefaultMaxPlausibleKW},
		{"zero site size falls back to default", f(0), DefaultMaxPlausibleKW},
		{"negative site size falls back to default", f(-5), DefaultMaxPlausibleKW},
		{"2kW site gets 1.5x headroom", f(2.0), 3.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PowerCeilingKW(c.size)
			if got != c.want {
				t.Errorf("PowerCeilingKW(%v) = %v, want %v", c.size, got, c.want)
			}
		})
	}
}

func TestTelemetryPayload_Validate_UsesGivenCeiling(t *testing.T) {
	payload := TelemetryPayload{
		DeviceID:       "ZG-0001",
		Timestamp:      "2026-08-05T12:00:00Z",
		PowerKW:        10.0,
		EnergyKWhTotal: 5.0,
		Status:         StatusOK,
	}

	if _, err := payload.Validate(3.0); err == nil {
		t.Error("expected rejection when power_kw exceeds the given ceiling")
	}
	if _, err := payload.Validate(20.0); err != nil {
		t.Errorf("expected acceptance when power_kw is under the given ceiling, got %v", err)
	}
}

func TestIsCoarseNight(t *testing.T) {
	cases := []struct {
		hour int
		want bool
	}{
		{0, true},
		{4, true},
		{5, false},
		{12, false},
		{19, false},
		{20, true},
		{23, true},
	}
	for _, c := range cases {
		localTime := time.Date(2026, 8, 5, c.hour, 0, 0, 0, time.UTC)
		got := IsCoarseNight(localTime)
		if got != c.want {
			t.Errorf("IsCoarseNight(hour=%d) = %v, want %v", c.hour, got, c.want)
		}
	}
}
