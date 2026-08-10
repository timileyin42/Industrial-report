package registry

import (
	"bytes"
	"testing"
	"time"
)

// TestBuildSiteEnergySummaryPDF_ProducesValidPDF is a pure unit test (no
// DB) — confirms the generator actually emits a well-formed PDF (starts
// with the %PDF magic bytes, non-trivially sized) rather than silently
// producing empty or corrupt output. Doesn't assert on rendered pixel
// content — fpdf's own output correctness isn't this codebase's concern,
// only that this function feeds it real data and gets real bytes back.
func TestBuildSiteEnergySummaryPDF_ProducesValidPDF(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC)
	series := EnergySeries{
		Points: []EnergyPoint{
			{PeriodStart: from, EnergyKWh: 12.5, ReadingCount: 288, BackfilledCount: 0},
			{PeriodStart: from.AddDate(0, 0, 1), EnergyKWh: 14.2, ReadingCount: 288, BackfilledCount: 3},
		},
		CumulativeKWh: 26.7,
	}

	data := BuildSiteEnergySummaryPDF("TEST-SITE-001", "daily", from, to, series)
	if len(data) < 500 {
		t.Fatalf("expected a real PDF of nontrivial size, got %d bytes", len(data))
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatalf("expected output to start with the %%PDF magic bytes, got %q", data[:min(20, len(data))])
	}
}

func TestBuildFleetEnergySummaryPDF_ProducesValidPDF(t *testing.T) {
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC)

	// Fleet report with zero points — the "no data in this range" branch
	// must still produce a valid PDF, not an empty/broken file.
	data := BuildFleetEnergySummaryPDF("monthly", from, to, EnergySeries{})
	if len(data) < 500 {
		t.Fatalf("expected a real PDF of nontrivial size even with zero data points, got %d bytes", len(data))
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatalf("expected output to start with the %%PDF magic bytes, got %q", data[:min(20, len(data))])
	}
}
