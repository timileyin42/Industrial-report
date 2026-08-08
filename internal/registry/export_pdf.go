package registry

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

// buildEnergySummaryPDF renders the same period-energy data the *_csv
// export types already produce, as a one-page-per-N-rows readable
// report (concept-note.md's "PDF or CSV" — CSV existed, PDF didn't).
// Deliberately not a redesign of design/reports_exports_*'s mockup:
// that screen fabricates scheduled-export counters and a revenue report
// this backend has no data for (see CLAUDE.md's "RenewableGrid drift"
// warning about not inventing UI beyond what an endpoint actually
// returns) — this report only contains real numbers this endpoint set
// already computes.
func buildEnergySummaryPDF(title string, subtitle string, from, to time.Time, series EnergySeries, generatedAt time.Time) []byte {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(title, false)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 10, title, "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(0, 6, subtitle, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Date range: %s to %s (UTC)", from.UTC().Format("2006-01-02"), to.UTC().Format("2006-01-02")), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated: %s (UTC)", generatedAt.UTC().Format(time.RFC3339)), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, fmt.Sprintf("Cumulative energy: %.2f kWh", series.CumulativeKWh), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	colWidths := []float64{50, 40, 45, 45}
	headers := []string{"Period start", "Energy (kWh)", "Reading count", "Backfilled count"}

	drawHeader := func() {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetFillColor(230, 230, 230)
		for i, h := range headers {
			pdf.CellFormat(colWidths[i], 8, h, "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFont("Helvetica", "", 10)
	}
	drawHeader()

	rowsPerPage := 32
	for i, p := range series.Points {
		if i > 0 && i%rowsPerPage == 0 {
			pdf.AddPage()
			drawHeader()
		}
		pdf.CellFormat(colWidths[0], 7, p.PeriodStart.UTC().Format("2006-01-02"), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[1], 7, fmt.Sprintf("%.2f", p.EnergyKWh), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[2], 7, fmt.Sprintf("%d", p.ReadingCount), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[3], 7, fmt.Sprintf("%d", p.BackfilledCount), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}

	if len(series.Points) == 0 {
		pdf.SetFont("Helvetica", "I", 10)
		pdf.CellFormat(0, 8, "No data in this range.", "", 1, "L", false, 0, "")
	}

	var buf bytes.Buffer
	_ = pdf.Output(&buf)
	return buf.Bytes()
}

// BuildSiteEnergySummaryPDF is the exported entry point httpapi's sync
// download endpoint and the async export job both use.
func BuildSiteEnergySummaryPDF(siteID, period string, from, to time.Time, series EnergySeries) []byte {
	return buildEnergySummaryPDF(fmt.Sprintf("Site Energy Summary — %s", siteID), fmt.Sprintf("Aggregation: %s", period), from, to, series, time.Now())
}

// BuildFleetEnergySummaryPDF is the fleet-wide counterpart to
// BuildSiteEnergySummaryPDF.
func BuildFleetEnergySummaryPDF(period string, from, to time.Time, series EnergySeries) []byte {
	return buildEnergySummaryPDF("Fleet Energy Summary", fmt.Sprintf("Aggregation: %s", period), from, to, series, time.Now())
}
