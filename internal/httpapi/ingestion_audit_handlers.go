package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

func (h *handlers) ingestionStatus(c echo.Context) error {
	lastReceivedAt, err := h.deps.IngestionAudit.LastReceivedAt(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"last_received_at": lastReceivedAt})
}

type ingestionAuditEntryResponse struct {
	ID         int64          `json:"id"`
	DeviceID   string         `json:"device_id"`
	SiteID     *string        `json:"site_id,omitempty"`
	RawPayload map[string]any `json:"raw_payload,omitempty"`
	ReceivedAt time.Time      `json:"received_at"`
	Processed  bool           `json:"processed"`
	Error      *string        `json:"error,omitempty"`
}

// listIngestionAudit is the read path CLAUDE.md's ingestion pipeline has
// needed since Phase 0 — every message received, before validation,
// browsable for the first time. Restricted users are scoped to their own
// site (via device_id -> devices.site_id), same siteFilter pattern as
// listSites; operators see the whole fleet.
func (h *handlers) listIngestionAudit(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	var siteFilter *string
	if claims.Role == domain.RoleRestricted {
		siteFilter = claims.SiteID
	}

	var deviceID *string
	if v := c.QueryParam("device_id"); v != "" {
		deviceID = &v
	}
	var errorsOnly *bool
	if v := c.QueryParam("errors_only"); v != "" {
		b := v == "true"
		errorsOnly = &b
	}
	from, err := parseOptionalTime(c, "from")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid from timestamp, expected RFC3339")
	}
	to, err := parseOptionalTime(c, "to")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid to timestamp, expected RFC3339")
	}

	entries, next, err := h.deps.IngestionAudit.List(c.Request().Context(), registry.ListIngestionAuditInput{
		SiteFilter:  siteFilter,
		DeviceID:    deviceID,
		ErrorsOnly:  errorsOnly,
		From:        from,
		To:          to,
		CursorToken: c.QueryParam("cursor"),
		Limit:       parseLimit(c),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	out := make([]ingestionAuditEntryResponse, 0, len(entries))
	for _, e := range entries {
		var payload map[string]any
		if len(e.RawPayload) > 0 {
			_ = json.Unmarshal(e.RawPayload, &payload)
		}
		out = append(out, ingestionAuditEntryResponse{
			ID:         e.ID,
			DeviceID:   e.DeviceID,
			SiteID:     e.SiteID,
			RawPayload: payload,
			ReceivedAt: e.ReceivedAt,
			Processed:  e.Processed,
			Error:      e.Error,
		})
	}
	return c.JSON(http.StatusOK, pageResponse[ingestionAuditEntryResponse]{Items: out, NextCursor: next})
}
