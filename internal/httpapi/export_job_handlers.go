package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

func parseInt64Param(c echo.Context, name string) (int64, error) {
	return strconv.ParseInt(c.Param(name), 10, 64)
}

type createExportJobRequest struct {
	JobType  string  `json:"job_type"`
	SiteID   *string `json:"site_id,omitempty"`
	Period   string  `json:"period,omitempty"`
	CohortID *string `json:"cohort_id,omitempty"`
}

type exportJobResponse struct {
	ID          int64      `json:"id"`
	JobType     string     `json:"job_type"`
	SiteID      *string    `json:"site_id,omitempty"`
	Status      string     `json:"status"`
	Error       *string    `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DownloadURL *string    `json:"download_url,omitempty"`
}

// createExportJob is the async counterpart to the sync CSV endpoints —
// access rules mirror them exactly (site jobs need the caller's own site,
// fleet jobs are operator-only), just enforced here in the handler
// instead of route middleware, since job_type/site_id come from the body
// rather than a :site_id path param.
func (h *handlers) createExportJob(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	var req createExportJobRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	jobType := db.ExportJobType(req.JobType)
	switch jobType {
	case db.ExportJobTypeSiteTelemetryCsv, db.ExportJobTypeSiteSummaryCsv, db.ExportJobTypeSiteSummaryPdf:
		if req.SiteID == nil || *req.SiteID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "site_id is required for this job type")
		}
		if claims.Role == domain.RoleRestricted && (claims.SiteID == nil || *claims.SiteID != *req.SiteID) {
			return echo.NewHTTPError(http.StatusForbidden, "not authorized for this site")
		}
	case db.ExportJobTypeFleetSummaryCsv, db.ExportJobTypeFleetSummaryPdf:
		if claims.Role != domain.RoleOperator {
			return echo.NewHTTPError(http.StatusForbidden, "fleet-wide exports are operator-only")
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "job_type must be site_telemetry_csv, site_summary_csv, fleet_summary_csv, site_summary_pdf, or fleet_summary_pdf")
	}

	period, err := parsePeriod(c)
	if err != nil {
		period = "monthly"
	}
	if req.Period != "" {
		period = req.Period
	}
	from, to, err := parseAnalyticsRange(c)
	if err != nil {
		return err
	}
	if err := validateExportRange(from, to); err != nil {
		return err
	}

	job, err := h.deps.Exports.Create(c.Request().Context(), registry.CreateExportJobInput{
		RequestedByUserID: claims.UserID,
		JobType:           jobType,
		SiteID:            req.SiteID,
		Period:            period,
		From:              from,
		To:                to,
		CohortID:          req.CohortID,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusAccepted, toExportJobResponse(job, ""))
}

func (h *handlers) getExportJob(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	id, err := parseInt64Param(c, "id")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job id")
	}

	job, err := h.deps.Exports.Get(c.Request().Context(), id, claims.Role, claims.SiteID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "export job not found")
	}

	downloadURL := ""
	if job.Status == db.ExportJobStatusCompleted {
		url, err := h.deps.Exports.PresignResult(c.Request().Context(), job)
		if err == nil {
			downloadURL = url
		}
	}
	return c.JSON(http.StatusOK, toExportJobResponse(job, downloadURL))
}

func (h *handlers) listExportJobs(c echo.Context) error {
	claims, _ := auth.GetClaims(c)

	jobs, err := h.deps.Exports.ListForUser(c.Request().Context(), claims.UserID, parseLimit(c))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	out := make([]exportJobResponse, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, toExportJobResponse(j, ""))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": out})
}

func toExportJobResponse(job db.ExportJob, downloadURL string) exportJobResponse {
	resp := exportJobResponse{
		ID:        job.ID,
		JobType:   string(job.JobType),
		SiteID:    textPtr(job.SiteID),
		Status:    string(job.Status),
		Error:     textPtr(job.Error),
		CreatedAt: job.CreatedAt.Time,
	}
	if job.CompletedAt.Valid {
		t := job.CompletedAt.Time
		resp.CompletedAt = &t
	}
	if downloadURL != "" {
		resp.DownloadURL = &downloadURL
	}
	return resp
}
