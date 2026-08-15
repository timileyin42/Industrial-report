package httpapi

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type cloudImportTokenResponse struct {
	Token string `json:"token"`
}

// issueCloudImportToken is operator-only (JWT auth, same as device
// registration) — generating a new bearer token for a device's
// cloud-import path is an admin action, not something the eventual
// external data source itself does.
func (h *handlers) issueCloudImportToken(c echo.Context) error {
	claims, _ := auth.GetClaims(c)
	deviceID := c.Param("device_id")

	token, err := h.deps.CloudImport.IssueToken(c.Request().Context(), claims.UserID, deviceID)
	if err != nil {
		if err == registry.ErrUnknownDevice {
			return echo.NewHTTPError(http.StatusNotFound, "device not found")
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, cloudImportTokenResponse{Token: token})
}

type cloudReadingRequest struct {
	Timestamp       string   `json:"ts"`
	PowerKW         float64  `json:"power_kw"`
	EnergyKWhTotal  float64  `json:"energy_kwh_total"`
	VoltageV        *float64 `json:"voltage_v,omitempty"`
	Status          string   `json:"status,omitempty"`
	PVPowerKW       *float64 `json:"pv_power_kw,omitempty"`
	BatterySOCPct   *float64 `json:"battery_soc_pct,omitempty"`
	BatteryVoltageV *float64 `json:"battery_voltage_v,omitempty"`
	PVVoltageV      *float64 `json:"pv_voltage_v,omitempty"`
	OutputVoltageV  *float64 `json:"output_voltage_v,omitempty"`
}

type submitCloudReadingsRequest struct {
	Readings []cloudReadingRequest `json:"readings"`
}

type cloudReadingResultResponse struct {
	Timestamp       string `json:"ts"`
	Accepted        bool   `json:"accepted"`
	Duplicate       bool   `json:"duplicate,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

type submitCloudReadingsResponse struct {
	AcceptedCount int                           `json:"accepted_count"`
	RejectedCount int                           `json:"rejected_count"`
	Readings      []cloudReadingResultResponse `json:"readings"`
}

// submitCloudReadings is deliberately NOT behind auth.RequireAuth (this
// isn't a logged-in dashboard user) and NOT MQTT — it's the genuinely
// vendor-agnostic cloud-import path: whatever external source holds a
// vendor's own credentials (a scraper, a scheduled script, a Google Apps
// Script watching an export folder) authenticates with this device's own
// bearer token instead, verified inside registry.CloudImport.
// SubmitReadings. See migrations/0017_cloud_import.sql.
func (h *handlers) submitCloudReadings(c echo.Context) error {
	deviceID := c.Param("device_id")
	token := bearerToken(c.Request().Header.Get("Authorization"))
	if token == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
	}

	var req submitCloudReadingsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if len(req.Readings) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "readings must not be empty")
	}

	readings := make([]registry.CloudReading, 0, len(req.Readings))
	for _, r := range req.Readings {
		readings = append(readings, registry.CloudReading{
			Timestamp:       r.Timestamp,
			PowerKW:         r.PowerKW,
			EnergyKWhTotal:  r.EnergyKWhTotal,
			VoltageV:        r.VoltageV,
			Status:          r.Status,
			PVPowerKW:       r.PVPowerKW,
			BatterySOCPct:   r.BatterySOCPct,
			BatteryVoltageV: r.BatteryVoltageV,
			PVVoltageV:      r.PVVoltageV,
			OutputVoltageV:  r.OutputVoltageV,
		})
	}

	results, err := h.deps.CloudImport.SubmitReadings(c.Request().Context(), deviceID, token, readings)
	if err != nil {
		switch err {
		case registry.ErrInvalidCloudImportToken:
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid or revoked token")
		case registry.ErrUnknownDevice:
			return echo.NewHTTPError(http.StatusNotFound, "device not found")
		case registry.ErrDeviceRevoked:
			return echo.NewHTTPError(http.StatusForbidden, "device is revoked")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to process readings")
		}
	}

	resp := submitCloudReadingsResponse{Readings: make([]cloudReadingResultResponse, 0, len(results))}
	for _, r := range results {
		if r.Accepted {
			resp.AcceptedCount++
		} else {
			resp.RejectedCount++
		}
		resp.Readings = append(resp.Readings, cloudReadingResultResponse{
			Timestamp: r.Timestamp, Accepted: r.Accepted, Duplicate: r.Duplicate, RejectionReason: r.RejectionReason,
		})
	}
	return c.JSON(http.StatusCreated, resp)
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
