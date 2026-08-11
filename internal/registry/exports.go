package registry

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/storage"
)

// maxExportJobRows mirrors export_handlers.go's maxExportRows — the async
// path is for the same data, not an excuse to remove the safety backstop.
const maxExportJobRows = 50000

var (
	ErrExportStorageNotConfigured = errors.New("export storage isn't configured yet — set R2_ACCOUNT_ID/R2_ACCESS_KEY_ID/R2_SECRET_ACCESS_KEY/R2_BUCKET")
	ErrExportJobNotFound          = errors.New("export job not found")
)

// exportJobParams is only ever held in memory (closed over by the
// goroutine spawned in Create) — never persisted. A process restart while
// a job is pending/running loses those params; Get self-heals by marking
// any job stuck past exportJobStaleAfter as failed, since there's no
// in-memory work left to resume it. This is a deliberate scope cut: a
// durable, restart-surviving queue is real infrastructure this platform
// doesn't need yet at current export volumes.
type exportJobParams struct {
	Period   string
	From, To time.Time
	CohortID *string
}

const exportJobStaleAfter = 10 * time.Minute

// Exports runs CSV export jobs asynchronously — the counterpart to the
// synchronous GET .../export/*.csv endpoints (internal/httpapi/export_handlers.go),
// for exports large enough to risk a client-side request timeout.
type Exports struct {
	q         *db.Queries
	telemetry *Telemetry
	analytics *Analytics
	storage   *storage.Client
	jobs      chan func()
}

// NewExports starts a small in-process worker pool. storageClient may be
// nil (R2 not configured) — jobs then fail fast with
// ErrExportStorageNotConfigured instead of hanging.
func NewExports(q *db.Queries, telemetry *Telemetry, analytics *Analytics, storageClient *storage.Client) *Exports {
	e := &Exports{q: q, telemetry: telemetry, analytics: analytics, storage: storageClient, jobs: make(chan func(), 100)}
	for i := 0; i < 4; i++ {
		go func() {
			for job := range e.jobs {
				job()
			}
		}()
	}
	return e
}

type CreateExportJobInput struct {
	RequestedByUserID int64
	JobType           db.ExportJobType
	SiteID            *string // required for site_* job types, must be nil for fleet_summary_csv
	Period            string
	From, To          time.Time
	CohortID          *string
}

// Create validates access (site-scoped jobs need a real site; fleet jobs
// are operator-only, enforced at the HTTP layer same as the sync
// endpoint), inserts the job row as pending, and enqueues the work —
// returning immediately with the job's id for the caller to poll.
func (e *Exports) Create(ctx context.Context, in CreateExportJobInput) (db.ExportJob, error) {
	switch in.JobType {
	case db.ExportJobTypeSiteTelemetryCsv, db.ExportJobTypeSiteSummaryCsv, db.ExportJobTypeSiteSummaryPdf:
		if in.SiteID == nil || *in.SiteID == "" {
			return db.ExportJob{}, errors.New("site_id is required for this job type")
		}
	case db.ExportJobTypeFleetSummaryCsv, db.ExportJobTypeFleetSummaryPdf:
		if in.SiteID != nil {
			return db.ExportJob{}, errors.New("site_id must not be set for a fleet-wide job")
		}
	default:
		return db.ExportJob{}, fmt.Errorf("unknown job_type %q", in.JobType)
	}

	job, err := e.q.CreateExportJob(ctx, db.CreateExportJobParams{
		RequestedByUserID: in.RequestedByUserID,
		JobType:           in.JobType,
		SiteID:            textOrNull(in.SiteID),
	})
	if err != nil {
		return db.ExportJob{}, err
	}

	params := exportJobParams{Period: in.Period, From: in.From, To: in.To, CohortID: in.CohortID}
	e.jobs <- func() { e.run(job.ID, in.JobType, in.SiteID, params) }

	return job, nil
}

func (e *Exports) run(jobID int64, jobType db.ExportJobType, siteID *string, params exportJobParams) {
	ctx := context.Background()

	if e.storage == nil {
		e.fail(ctx, jobID, ErrExportStorageNotConfigured)
		return
	}
	if err := e.q.MarkExportJobRunning(ctx, jobID); err != nil {
		log.Printf("export job %d: mark running: %v", jobID, err)
		return
	}

	data, filename, contentType, err := e.build(ctx, jobType, siteID, params)
	if err != nil {
		e.fail(ctx, jobID, err)
		return
	}

	key := fmt.Sprintf("exports/%d-%s", jobID, filename)
	if err := e.storage.Upload(ctx, key, data, contentType); err != nil {
		e.fail(ctx, jobID, err)
		return
	}

	if err := e.q.MarkExportJobCompleted(ctx, db.MarkExportJobCompletedParams{ID: jobID, ResultKey: pgtype.Text{String: key, Valid: true}}); err != nil {
		log.Printf("export job %d: mark completed: %v", jobID, err)
	}
}

func (e *Exports) fail(ctx context.Context, jobID int64, err error) {
	log.Printf("export job %d failed: %v", jobID, err)
	if dbErr := e.q.MarkExportJobFailed(ctx, db.MarkExportJobFailedParams{ID: jobID, Error: pgtype.Text{String: err.Error(), Valid: true}}); dbErr != nil {
		log.Printf("export job %d: mark failed: %v", jobID, dbErr)
	}
}

func (e *Exports) build(ctx context.Context, jobType db.ExportJobType, siteID *string, params exportJobParams) ([]byte, string, string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	switch jobType {
	case db.ExportJobTypeSiteTelemetryCsv:
		_ = w.Write([]string{"ts", "device_id", "power_kw", "energy_kwh_total", "voltage_v", "status", "rssi"})
		cursor := ""
		rowCount := 0
		for {
			rows, next, err := e.telemetry.List(ctx, ListTelemetryInput{SiteID: *siteID, From: &params.From, To: &params.To, CursorToken: cursor, Limit: 500})
			if err != nil {
				return nil, "", "", err
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
					r.Ts.Time.UTC().Format(time.RFC3339), r.DeviceID,
					fmt.Sprintf("%v", r.PowerKw), fmt.Sprintf("%v", r.EnergyKwhTotal), voltage, string(r.Status), rssi,
				})
				rowCount++
			}
			if next == "" || rowCount >= maxExportJobRows {
				break
			}
			cursor = next
		}
		w.Flush()
		return buf.Bytes(), fmt.Sprintf("%s-telemetry.csv", *siteID), "text/csv", nil

	case db.ExportJobTypeSiteSummaryCsv:
		series, err := e.analytics.SiteEnergy(ctx, *siteID, params.Period, params.From, params.To)
		if err != nil {
			return nil, "", "", err
		}
		writeEnergySeriesCSV(w, series)
		return buf.Bytes(), fmt.Sprintf("%s-summary.csv", *siteID), "text/csv", nil

	case db.ExportJobTypeFleetSummaryCsv:
		series, err := e.analytics.FleetEnergy(ctx, params.CohortID, params.Period, params.From, params.To)
		if err != nil {
			return nil, "", "", err
		}
		writeEnergySeriesCSV(w, series)
		return buf.Bytes(), "fleet-summary.csv", "text/csv", nil

	case db.ExportJobTypeSiteSummaryPdf:
		series, err := e.analytics.SiteEnergy(ctx, *siteID, params.Period, params.From, params.To)
		if err != nil {
			return nil, "", "", err
		}
		data := BuildSiteEnergySummaryPDF(*siteID, params.Period, params.From, params.To, series)
		return data, fmt.Sprintf("%s-summary.pdf", *siteID), "application/pdf", nil

	case db.ExportJobTypeFleetSummaryPdf:
		series, err := e.analytics.FleetEnergy(ctx, params.CohortID, params.Period, params.From, params.To)
		if err != nil {
			return nil, "", "", err
		}
		data := BuildFleetEnergySummaryPDF(params.Period, params.From, params.To, series)
		return data, "fleet-summary.pdf", "application/pdf", nil

	default:
		return nil, "", "", fmt.Errorf("unknown job_type %q", jobType)
	}
}

func writeEnergySeriesCSV(w *csv.Writer, series EnergySeries) {
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
}

// Get self-heals a job stuck past exportJobStaleAfter (see
// exportJobParams' comment) and enforces the same access rule as the
// sync export endpoints: a restricted user may only see their own site's
// jobs, never a fleet-wide job or another site's.
func (e *Exports) Get(ctx context.Context, id int64, role domain.Role, callerSiteID *string) (db.ExportJob, error) {
	job, err := e.q.GetExportJob(ctx, id)
	if err != nil {
		return db.ExportJob{}, ErrExportJobNotFound
	}

	if role == domain.RoleRestricted {
		if !job.SiteID.Valid || callerSiteID == nil || job.SiteID.String != *callerSiteID {
			return db.ExportJob{}, ErrExportJobNotFound
		}
	}

	if (job.Status == db.ExportJobStatusPending || job.Status == db.ExportJobStatusRunning) &&
		time.Since(job.CreatedAt.Time) > exportJobStaleAfter {
		_ = e.q.MarkExportJobFailed(ctx, db.MarkExportJobFailedParams{
			ID:    job.ID,
			Error: pgtype.Text{String: "job timed out (worker may have restarted)", Valid: true},
		})
		job.Status = db.ExportJobStatusFailed
	}

	return job, nil
}

func (e *Exports) ListForUser(ctx context.Context, userID int64, limit int) ([]db.ExportJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return e.q.ListExportJobsForUser(ctx, db.ListExportJobsForUserParams{RequestedByUserID: userID, Limit: int32(limit)})
}

// PresignResult returns a short-lived download URL for a completed job.
func (e *Exports) PresignResult(ctx context.Context, job db.ExportJob) (string, error) {
	if job.Status != db.ExportJobStatusCompleted || !job.ResultKey.Valid {
		return "", errors.New("job isn't completed yet")
	}
	if e.storage == nil {
		return "", ErrExportStorageNotConfigured
	}
	return e.storage.PresignGet(ctx, job.ResultKey.String, 15*time.Minute)
}
