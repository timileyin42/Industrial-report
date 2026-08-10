package registry

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
)

func cleanupExportJob(t *testing.T, pool *pgxpool.Pool, jobID int64) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM export_jobs WHERE id = $1`, jobID)
	})
}

// TestExportsCreate_ValidatesJobTypeSiteRequirements is the same rule
// internal/httpapi/export_job_handlers.go enforces at the HTTP layer,
// checked again here at the registry layer directly: site-scoped job
// types require a real site_id, fleet-wide ones must not have one, and an
// unrecognized job_type is rejected outright — before anything ever
// touches the database or the async worker pool.
func TestExportsCreate_ValidatesJobTypeSiteRequirements(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()
	exports := NewExports(q, NewTelemetry(q), NewAnalytics(q), nil)

	site := createTestSite(t, ctx, NewSites(q), pool, CreateSiteInput{
		SiteID: uniqueID("site-export-"), Name: "Export Test Site", Timezone: "UTC", Country: "NG",
	})
	siteID := site.SiteID

	cases := []struct {
		name    string
		jobType db.ExportJobType
		siteID  *string
		wantErr bool
	}{
		{"site_telemetry_csv without site_id is rejected", db.ExportJobTypeSiteTelemetryCsv, nil, true},
		{"site_summary_csv without site_id is rejected", db.ExportJobTypeSiteSummaryCsv, nil, true},
		{"site_summary_pdf without site_id is rejected", db.ExportJobTypeSiteSummaryPdf, nil, true},
		{"fleet_summary_csv with a site_id is rejected", db.ExportJobTypeFleetSummaryCsv, &siteID, true},
		{"fleet_summary_pdf with a site_id is rejected", db.ExportJobTypeFleetSummaryPdf, &siteID, true},
		{"unknown job_type is rejected", db.ExportJobType("not_a_real_type"), nil, true},
		{"site_summary_pdf with a site_id is accepted", db.ExportJobTypeSiteSummaryPdf, &siteID, false},
		{"fleet_summary_pdf without a site_id is accepted", db.ExportJobTypeFleetSummaryPdf, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			job, err := exports.Create(ctx, CreateExportJobInput{
				RequestedByUserID: 1,
				JobType:           c.jobType,
				SiteID:            c.siteID,
				Period:            "daily",
				From:              time.Now().Add(-24 * time.Hour),
				To:                time.Now(),
			})
			if c.wantErr {
				if err == nil {
					cleanupExportJob(t, pool, job.ID)
					t.Fatalf("expected an error for %s, got none", c.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected acceptance for %s, got %v", c.name, err)
			}
			cleanupExportJob(t, pool, job.ID)
		})
	}
}

// TestExportsGet_FailsFastWithoutStorageConfigured confirms the async
// worker actually runs (not just that Create() returns a row) and fails
// clearly rather than hanging when R2 isn't configured — the same
// nil-storage-client path production would hit if someone forgot to set
// R2_ACCOUNT_ID/R2_ACCESS_KEY_ID/R2_SECRET_ACCESS_KEY/R2_BUCKET.
func TestExportsGet_FailsFastWithoutStorageConfigured(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	ctx := context.Background()
	exports := NewExports(q, NewTelemetry(q), NewAnalytics(q), nil)

	job, err := exports.Create(ctx, CreateExportJobInput{
		RequestedByUserID: 1,
		JobType:           db.ExportJobTypeFleetSummaryCsv,
		Period:            "daily",
		From:              time.Now().Add(-24 * time.Hour),
		To:                time.Now(),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	cleanupExportJob(t, pool, job.ID)

	deadline := time.Now().Add(5 * time.Second)
	var got db.ExportJob
	for time.Now().Before(deadline) {
		got, err = exports.Get(ctx, job.ID, domain.RoleOperator, nil)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.Status == db.ExportJobStatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got.Status != db.ExportJobStatusFailed {
		t.Fatalf("expected the job to reach status=failed within 5s, last saw %q", got.Status)
	}
	if !got.Error.Valid || got.Error.String != ErrExportStorageNotConfigured.Error() {
		t.Fatalf("expected error %q, got %+v", ErrExportStorageNotConfigured, got.Error)
	}
}
