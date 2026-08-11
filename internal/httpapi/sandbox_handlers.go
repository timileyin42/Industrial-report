package httpapi

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/registry"
)

// maxSandboxUploadBytes bounds the raw multipart body — a public,
// unauthenticated endpoint with no size cap is a real abuse vector, not
// just a hygiene nicety. Comfortably covers MaxSandboxRows' worth of CSV
// text with room to spare.
const maxSandboxUploadBytes = 2 << 20 // 2 MiB

type sandboxReadingResponse struct {
	RowNumber       int      `json:"row_number"`
	Timestamp       *string  `json:"ts,omitempty"`
	PowerKW         *float64 `json:"power_kw,omitempty"`
	EnergyKWhTotal  *float64 `json:"energy_kwh_total,omitempty"`
	VoltageV        *float64 `json:"voltage_v,omitempty"`
	RSSI            *int     `json:"rssi,omitempty"`
	Status          string   `json:"status,omitempty"`
	Accepted        bool     `json:"accepted"`
	RejectionReason string   `json:"rejection_reason,omitempty"`
	Provenance      string   `json:"provenance,omitempty"`
	IsReset         bool     `json:"is_reset"`
}

type sandboxUploadResponse struct {
	RunID         string                   `json:"run_id"`
	RowCount      int                      `json:"row_count"`
	AcceptedCount int                      `json:"accepted_count"`
	RejectedCount int                      `json:"rejected_count"`
	Readings      []sandboxReadingResponse `json:"readings"`
}

func toSandboxReadingResponses(in []registry.SandboxReadingResult) []sandboxReadingResponse {
	out := make([]sandboxReadingResponse, 0, len(in))
	for _, r := range in {
		resp := sandboxReadingResponse{
			RowNumber:       r.RowNumber,
			PowerKW:         r.PowerKW,
			EnergyKWhTotal:  r.EnergyKWhTotal,
			VoltageV:        r.VoltageV,
			RSSI:            r.RSSI,
			Status:          r.Status,
			Accepted:        r.Accepted,
			RejectionReason: r.RejectionReason,
			Provenance:      r.Provenance,
			IsReset:         r.IsReset,
		}
		if r.Timestamp != nil {
			s := r.Timestamp.Format("2006-01-02T15:04:05Z07:00")
			resp.Timestamp = &s
		}
		out = append(out, resp)
	}
	return out
}

// uploadSandbox is deliberately public (no auth middleware) — see
// router.go's comment on this route group. Rate-limited and size-capped
// since anyone with the link can call it.
func (h *handlers) uploadSandbox(c echo.Context) error {
	req := c.Request()
	req.Body = http.MaxBytesReader(c.Response(), req.Body, maxSandboxUploadBytes)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "expected a multipart form field named \"file\" containing the CSV")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "could not read uploaded file")
	}
	defer file.Close()

	var systemSizeKW *float64
	if v := c.FormValue("system_size_kw"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			systemSizeKW = &f
		}
	}

	result, err := h.deps.Sandbox.Upload(req.Context(), file, systemSizeKW)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, sandboxUploadResponse{
		RunID:         result.RunID,
		RowCount:      result.RowCount,
		AcceptedCount: result.AcceptedCount,
		RejectedCount: result.RejectedCount,
		Readings:      toSandboxReadingResponses(result.Readings),
	})
}

// getSandbox is also public — the run ID itself (a long random token,
// see registry.newSandboxRunID) is the only thing gating access, the same
// trust model as an unlisted share link. A wrong/unknown ID gets a plain
// 404, same as any not-found resource — there's no real user account
// behind this to protect with a 403 instead.
func (h *handlers) getSandbox(c echo.Context) error {
	runID := c.Param("run_id")
	run, readings, err := h.deps.Sandbox.Get(c.Request().Context(), runID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "sandbox run not found")
	}

	return c.JSON(http.StatusOK, sandboxUploadResponse{
		RunID:         run.ID,
		RowCount:      int(run.RowCount),
		AcceptedCount: int(run.AcceptedCount),
		RejectedCount: int(run.RejectedCount),
		Readings:      toSandboxReadingResponses(readings),
	})
}
