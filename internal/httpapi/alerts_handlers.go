package httpapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type alertResponse struct {
	Type       string    `json:"type"`
	Severity   string    `json:"severity"`
	SiteID     string    `json:"site_id"`
	SiteName   *string   `json:"site_name,omitempty"`
	DeviceID   *string   `json:"device_id,omitempty"`
	Message    string    `json:"message"`
	OccurredAt time.Time `json:"occurred_at"`
}

func toAlertResponse(a registry.Alert) alertResponse {
	return alertResponse{
		Type: a.Type, Severity: string(a.Severity), SiteID: a.SiteID,
		SiteName: a.SiteName, DeviceID: a.DeviceID, Message: a.Message, OccurredAt: a.OccurredAt,
	}
}

func (h *handlers) fleetAlerts(c echo.Context) error {
	alerts, err := h.deps.Alerts.Fleet(c.Request().Context(), parseLimit(c))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	out := make([]alertResponse, 0, len(alerts))
	for _, a := range alerts {
		out = append(out, toAlertResponse(a))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": out})
}
