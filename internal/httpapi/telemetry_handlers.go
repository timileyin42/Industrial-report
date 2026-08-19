package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type telemetryPointResponse struct {
	Timestamp time.Time `json:"ts"`
	DeviceID  string    `json:"device_id"`
	PowerKW   float64   `json:"power_kw"`
	EnergyKWh float64   `json:"energy_kwh_total"`
	VoltageV  *float64  `json:"voltage_v,omitempty"`
	Status    string    `json:"status"`
	RSSI      *int32    `json:"rssi,omitempty"`
	// Hybrid-inverter fields — see migrations/0016. Omitted for any
	// reading that predates them or came from a non-hybrid device.
	PVPowerKW       *float64 `json:"pv_power_kw,omitempty"`
	BatterySOCPct   *int16   `json:"battery_soc_pct,omitempty"`
	BatteryVoltageV *float64 `json:"battery_voltage_v,omitempty"`
	PVVoltageV      *float64 `json:"pv_voltage_v,omitempty"`
	OutputVoltageV  *float64 `json:"output_voltage_v,omitempty"`
	LoadPowerKW     *float64 `json:"load_power_kw,omitempty"`
	GridPowerKW     *float64 `json:"grid_power_kw,omitempty"`
}

// listTelemetry replaces Phase 0's unbounded 24h scan with an auth-gated,
// paginated read (retrofit — see plan). Access control (auth.RequireAuth +
// auth.RequireSiteAccess) is applied at the route level in router.go.
func (h *handlers) listTelemetry(c echo.Context) error {
	from, err := parseOptionalTime(c, "from")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid from timestamp, expected RFC3339")
	}
	to, err := parseOptionalTime(c, "to")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid to timestamp, expected RFC3339")
	}

	rows, next, err := h.deps.Telemetry.List(c.Request().Context(), registry.ListTelemetryInput{
		SiteID:      c.Param("site_id"),
		From:        from,
		To:          to,
		CursorToken: c.QueryParam("cursor"),
		Limit:       parseLimit(c),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	items := make([]telemetryPointResponse, 0, len(rows))
	for _, r := range rows {
		var rssi *int32
		if r.Rssi.Valid {
			rssi = &r.Rssi.Int32
		}
		items = append(items, telemetryPointResponse{
			Timestamp:       r.Ts.Time,
			DeviceID:        r.DeviceID,
			PowerKW:         r.PowerKw,
			EnergyKWh:       r.EnergyKwhTotal,
			VoltageV:        float8Ptr(r.VoltageV),
			Status:          string(r.Status),
			RSSI:            rssi,
			PVPowerKW:       float8Ptr(r.PvPowerKw),
			BatterySOCPct:   int2Ptr(r.BatterySocPct),
			BatteryVoltageV: float8Ptr(r.BatteryVoltageV),
			PVVoltageV:      float8Ptr(r.PvVoltageV),
			OutputVoltageV:  float8Ptr(r.OutputVoltageV),
			LoadPowerKW:     float8Ptr(r.LoadPowerKw),
			GridPowerKW:     float8Ptr(r.GridPowerKw),
		})
	}
	return c.JSON(http.StatusOK, pageResponse[telemetryPointResponse]{Items: items, NextCursor: next})
}

type deviceStatusResponse struct {
	DeviceID      string     `json:"device_id"`
	LastSeenAt    *time.Time `json:"last_seen_at"`
	LastContactAt *time.Time `json:"last_contact_at"`
	Online        bool       `json:"online"`
	DataGap       bool       `json:"data_gap"`
	Revoked       bool       `json:"revoked"`
}

// deviceStatus is Phase 0's endpoint, retrofitted with auth + site-scope
// (resolveSiteFromDeviceParam in router.go). Phase 2 replaces the inline
// "10 minutes" Go math with the consolidated two-signal online/data_gap
// model (registry.Devices.Status) and adds last_contact_at/data_gap
// additively — existing fields are unchanged.
func (h *handlers) deviceStatus(c echo.Context) error {
	status, err := h.deps.Devices.Status(c.Request().Context(), c.Param("device_id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "device not found")
	}

	return c.JSON(http.StatusOK, deviceStatusResponse{
		DeviceID:      status.DeviceID,
		LastSeenAt:    status.LastSeenAt,
		LastContactAt: status.LastContactAt,
		Online:        status.Online,
		DataGap:       status.DataGap,
		Revoked:       status.Revoked,
	})
}
