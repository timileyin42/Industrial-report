package httpapi

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/registry"
)

func servePDF(c echo.Context, filename string, data []byte) error {
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.Blob(http.StatusOK, "application/pdf", data)
}

// maxExportRangeDays bounds every CSV export — an unbounded date range on
// an export endpoint is exactly the kind of "fine for now" gap CLAUDE.md
// warns becomes a production problem at scale.
const maxExportRangeDays = 90

// maxExportRows caps a single telemetry.csv export's total row count as a
// safety backstop, independent of the date-range bound above (a
// high-frequency device over 90 days could still be a lot of rows).
const maxExportRows = 50000

func validateExportRange(from, to time.Time) error {
	if to.Before(from) {
		return echo.NewHTTPError(http.StatusBadRequest, "to must be after from")
	}
	if to.Sub(from) > maxExportRangeDays*24*time.Hour {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("export range cannot exceed %d days", maxExportRangeDays))
	}
	return nil
}

func csvWriter(c echo.Context, filename string) (*csv.Writer, error) {
	c.Response().Header().Set(echo.HeaderContentType, "text/csv")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Response().WriteHeader(http.StatusOK)
	return csv.NewWriter(c.Response()), nil
}

// siteTelemetryCSV streams a site's raw telemetry — site-scoped.
func (h *handlers) siteTelemetryCSV(c echo.Context) error {
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}
	if err := validateExportRange(from, to); err != nil {
		return err
	}
	siteID := c.Param("site_id")

	w, err := csvWriter(c, fmt.Sprintf("%s-telemetry.csv", siteID))
	if err != nil {
		return err
	}
	_ = w.Write([]string{"ts", "device_id", "power_kw", "energy_kwh_total", "voltage_v", "status", "rssi"})

	ctx := c.Request().Context()
	cursor := ""
	rowCount := 0
	for {
		rows, next, err := h.deps.Telemetry.List(ctx, registry.ListTelemetryInput{
			SiteID: siteID, From: &from, To: &to, CursorToken: cursor, Limit: 500,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		for _, r := range rows {
			voltage := ""
			if r.VoltageV.Valid {
				voltage = fmt.Sprintf("%v", r.VoltageV.Float64)
			}
			rssi := ""
			if r.Rssi.Valid {
				rssi = fmt.Sprintf("%d", r.Rssi.Int32)
			}
			_ = w.Write([]string{
				r.Ts.Time.UTC().Format(time.RFC3339),
				r.DeviceID,
				fmt.Sprintf("%v", r.PowerKw),
				fmt.Sprintf("%v", r.EnergyKwhTotal),
				voltage,
				string(r.Status),
				rssi,
			})
			rowCount++
		}
		if next == "" || rowCount >= maxExportRows {
			break
		}
		cursor = next
	}
	w.Flush()
	return nil
}

// siteSummaryCSV streams a site's period energy summary — site-scoped.
func (h *handlers) siteSummaryCSV(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}
	if err := validateExportRange(from, to); err != nil {
		return err
	}
	siteID := c.Param("site_id")

	series, err := h.deps.Analytics.SiteEnergy(c.Request().Context(), siteID, period, from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	w, err := csvWriter(c, fmt.Sprintf("%s-summary.csv", siteID))
	if err != nil {
		return err
	}
	_ = w.Write([]string{"period_start", "energy_kwh", "reading_count", "backfilled_count"})
	for _, p := range series.Points {
		_ = w.Write([]string{
			p.PeriodStart.UTC().Format(time.RFC3339),
			fmt.Sprintf("%v", p.EnergyKWh),
			fmt.Sprintf("%d", p.ReadingCount),
			fmt.Sprintf("%d", p.BackfilledCount),
		})
	}
	w.Flush()
	return nil
}

// siteSummaryPDF is siteSummaryCSV's PDF counterpart — same data, a
// human-readable report shape instead of a data-shape file (see
// registry/export_pdf.go).
func (h *handlers) siteSummaryPDF(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}
	if err := validateExportRange(from, to); err != nil {
		return err
	}
	siteID := c.Param("site_id")

	series, err := h.deps.Analytics.SiteEnergy(c.Request().Context(), siteID, period, from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	data := registry.BuildSiteEnergySummaryPDF(siteID, period, from, to, series)
	return servePDF(c, fmt.Sprintf("%s-summary.pdf", siteID), data)
}

// fleetSummaryPDF is fleetSummaryCSV's PDF counterpart — operator-only.
func (h *handlers) fleetSummaryPDF(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}
	if err := validateExportRange(from, to); err != nil {
		return err
	}
	var cohortID *string
	if v := c.QueryParam("cohort_id"); v != "" {
		cohortID = &v
	}

	series, err := h.deps.Analytics.FleetEnergy(c.Request().Context(), cohortID, period, from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	data := registry.BuildFleetEnergySummaryPDF(period, from, to, series)
	return servePDF(c, "fleet-summary.pdf", data)
}

// fleetSummaryCSV streams a fleet-wide (optionally cohort-scoped) period
// energy summary — operator-only.
func (h *handlers) fleetSummaryCSV(c echo.Context) error {
	period, err := parsePeriod(c)
	if err != nil {
		return err
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}
	if err := validateExportRange(from, to); err != nil {
		return err
	}
	var cohortID *string
	if v := c.QueryParam("cohort_id"); v != "" {
		cohortID = &v
	}

	series, err := h.deps.Analytics.FleetEnergy(c.Request().Context(), cohortID, period, from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	w, err := csvWriter(c, "fleet-summary.csv")
	if err != nil {
		return err
	}
	_ = w.Write([]string{"period_start", "energy_kwh", "reading_count", "backfilled_count"})
	for _, p := range series.Points {
		_ = w.Write([]string{
			p.PeriodStart.UTC().Format(time.RFC3339),
			fmt.Sprintf("%v", p.EnergyKWh),
			fmt.Sprintf("%d", p.ReadingCount),
			fmt.Sprintf("%d", p.BackfilledCount),
		})
	}
	w.Flush()
	return nil
}
