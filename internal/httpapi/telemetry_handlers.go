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
			Timestamp: r.Ts.Time,
			DeviceID:  r.DeviceID,
			PowerKW:   r.PowerKw,
			EnergyKWh: r.EnergyKwhTotal,
			VoltageV:  float8Ptr(r.VoltageV),
			Status:    string(r.Status),
			RSSI:      rssi,
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
