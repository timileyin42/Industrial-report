package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type auditEntryResponse struct {
	ID         int64          `json:"id"`
	ActorEmail *string        `json:"actor_email,omitempty"`
	Action     string         `json:"action"`
	TargetType *string        `json:"target_type,omitempty"`
	TargetID   *string        `json:"target_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// listAuditActions is the Phase 3 catch-up on migration 0002's own
// deferred TODO ("a browsing/reporting UI on this table is Phase 3") —
// operator-only.
func (h *handlers) listAuditActions(c echo.Context) error {
	var actorUserID *int64
	if v := c.QueryParam("actor_user_id"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid actor_user_id")
		}
		actorUserID = &n
	}
	var action, targetType, targetID *string
	if v := c.QueryParam("action"); v != "" {
		action = &v
	}
	if v := c.QueryParam("target_type"); v != "" {
		targetType = &v
	}
	if v := c.QueryParam("target_id"); v != "" {
		targetID = &v
	}
	from, err := parseOptionalTime(c, "from")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid from timestamp, expected RFC3339")
	}
	to, err := parseOptionalTime(c, "to")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid to timestamp, expected RFC3339")
	}

	entries, next, err := h.deps.AuditLog.List(c.Request().Context(), registry.ListAuditInput{
		ActorUserID: actorUserID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		From:        from,
		To:          to,
		CursorToken: c.QueryParam("cursor"),
		Limit:       parseLimit(c),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	out := make([]auditEntryResponse, 0, len(entries))
	for _, e := range entries {
		var meta map[string]any
		if len(e.Metadata) > 0 {
			_ = json.Unmarshal(e.Metadata, &meta)
		}
		out = append(out, auditEntryResponse{
			ID:         e.ID,
			ActorEmail: e.ActorEmail,
			Action:     e.Action,
			TargetType: e.TargetType,
			TargetID:   e.TargetID,
			Metadata:   meta,
			CreatedAt:  e.CreatedAt,
		})
	}
	return c.JSON(http.StatusOK, pageResponse[auditEntryResponse]{Items: out, NextCursor: next})
}

type chainVerifyResponse struct {
	Valid         bool   `json:"valid"`
	MismatchCount int64  `json:"mismatch_count"`
	FirstBadID    *int64 `json:"first_bad_id,omitempty"`
}

// verifyAuditActions proves (or disproves) that user_action_audit_log
// hasn't been tampered with since it was written — concept-note.md §11's
// "verification-readiness" requirement, not just an append-only
// convention. Operator-only, same as browsing the log itself.
func (h *handlers) verifyAuditActions(c echo.Context) error {
	result, err := h.deps.AuditLog.VerifyChain(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, chainVerifyResponse{Valid: result.Valid, MismatchCount: result.MismatchCount, FirstBadID: result.FirstBadID})
}

// verifyIngestionAudit is the same check for ingestion_audit_log — the
// data-quality/verification trail, kept separate per CLAUDE.md.
func (h *handlers) verifyIngestionAudit(c echo.Context) error {
	result, err := h.deps.IngestionAudit.VerifyChain(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, chainVerifyResponse{Valid: result.Valid, MismatchCount: result.MismatchCount, FirstBadID: result.FirstBadID})
}
