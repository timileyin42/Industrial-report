// Sandbox is a public, no-login "upload your own readings and see them
// validated the way a real device's would be" feature — deliberately its
// own pair of tables (migrations/0014_sandbox.sql), never touching
// sites/devices/telemetry, so it structurally cannot affect any real
// fleet dashboard, KPI, or RBAC path. Security model for the no-login
// share link is an unguessable random run ID, the same trust model as an
// unlisted Google Doc link — not a password, just not enumerable.
package registry

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
)

// MaxSandboxRows bounds a single upload — this is a public, unauthenticated
// endpoint, so an unbounded CSV is a real abuse/DoS vector, not just a
// performance nicety.
const MaxSandboxRows = 2000

// sandboxRunRetention: lazily deleted on every new upload (see Upload) —
// this data is public and unauthenticated, so nothing else ever purges it.
const sandboxRunRetention = 30 * 24 * time.Hour

type Sandbox struct {
	q *db.Queries
}

func NewSandbox(q *db.Queries) *Sandbox {
	return &Sandbox{q: q}
}

type SandboxReadingResult struct {
	RowNumber       int
	Timestamp       *time.Time
	PowerKW         *float64
	EnergyKWhTotal  *float64
	VoltageV        *float64
	RSSI            *int
	Status          string
	Accepted        bool
	RejectionReason string
	Provenance      string
	IsReset         bool
}

type SandboxUploadResult struct {
	RunID         string
	RowCount      int
	AcceptedCount int
	RejectedCount int
	Readings      []SandboxReadingResult
}

// newSandboxRunID is the entire security model for this feature's
// no-login share link: long enough that guessing one is infeasible,
// generated from crypto/rand (not math/rand — this is a real access
// token, not a display ID).
func newSandboxRunID() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Upload parses a CSV of readings and runs each one through the exact
// same validation/classification logic the real ingestor uses
// (domain.TelemetryPayload.Validate, ClassifyProvenance,
// DetectEnergyReset) — "simulate it like a real connector" means reusing
// the real rules, not a separate, looser check that would show
// misleadingly clean results.
//
// Expected CSV header (case-insensitive): ts (or timestamp), power_kw,
// energy_kwh_total, and optionally voltage_v, status. Extra columns are
// ignored. Rows are re-sorted by parsed timestamp before validation,
// since reset/backfill detection depends on chronological order, not
// upload order.
func (s *Sandbox) Upload(ctx context.Context, r io.Reader, systemSizeKW *float64) (SandboxUploadResult, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return SandboxUploadResult{}, fmt.Errorf("read CSV header: %w", err)
	}

	col := map[string]int{}
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	tsIdx, ok := col["ts"]
	if !ok {
		tsIdx, ok = col["timestamp"]
	}
	powerIdx, hasPower := col["power_kw"]
	energyIdx, hasEnergy := col["energy_kwh_total"]
	if !ok || !hasPower || !hasEnergy {
		return SandboxUploadResult{}, fmt.Errorf("CSV header must include ts (or timestamp), power_kw, and energy_kwh_total")
	}
	voltageIdx, hasVoltage := col["voltage_v"]
	statusIdx, hasStatus := col["status"]
	rssiIdx, hasRSSI := col["rssi"]

	type parsedRow struct {
		rowNumber int
		raw       []string
		ts        *time.Time
		power     *float64
		energy    *float64
		voltage   *float64
		rssi      *int
		status    string
		parseErr  string
	}

	var rows []parsedRow
	rowNumber := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return SandboxUploadResult{}, fmt.Errorf("read CSV row %d: %w", rowNumber+1, err)
		}
		rowNumber++
		if rowNumber > MaxSandboxRows {
			return SandboxUploadResult{}, fmt.Errorf("CSV has more than %d rows — the sandbox is for a quick validation sample, not a bulk import", MaxSandboxRows)
		}

		pr := parsedRow{rowNumber: rowNumber, raw: record}
		if tsIdx >= len(record) {
			pr.parseErr = "missing ts column"
		} else if t, err := parseFlexibleTime(record[tsIdx]); err != nil {
			pr.parseErr = fmt.Sprintf("unparseable ts %q", record[tsIdx])
		} else {
			pr.ts = &t
		}
		if pr.parseErr == "" {
			if powerIdx >= len(record) {
				pr.parseErr = "missing power_kw column"
			} else if v, err := strconv.ParseFloat(strings.TrimSpace(record[powerIdx]), 64); err != nil {
				pr.parseErr = fmt.Sprintf("unparseable power_kw %q", record[powerIdx])
			} else {
				pr.power = &v
			}
		}
		if pr.parseErr == "" {
			if energyIdx >= len(record) {
				pr.parseErr = "missing energy_kwh_total column"
			} else if v, err := strconv.ParseFloat(strings.TrimSpace(record[energyIdx]), 64); err != nil {
				pr.parseErr = fmt.Sprintf("unparseable energy_kwh_total %q", record[energyIdx])
			} else {
				pr.energy = &v
			}
		}
		if hasVoltage && voltageIdx < len(record) && strings.TrimSpace(record[voltageIdx]) != "" {
			if v, err := strconv.ParseFloat(strings.TrimSpace(record[voltageIdx]), 64); err == nil {
				pr.voltage = &v
			}
		}
		if hasStatus && statusIdx < len(record) {
			pr.status = strings.TrimSpace(record[statusIdx])
		}
		if hasRSSI && rssiIdx < len(record) && strings.TrimSpace(record[rssiIdx]) != "" {
			if v, err := strconv.Atoi(strings.TrimSpace(record[rssiIdx])); err == nil {
				pr.rssi = &v
			}
		}
		rows = append(rows, pr)
	}

	// Chronological order, not upload order — reset/backfill detection
	// only makes sense against the immediately preceding reading in
	// real time, same as the real ingestor's per-device ordering.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].ts == nil || rows[j].ts == nil {
			return false
		}
		return rows[i].ts.Before(*rows[j].ts)
	})

	maxPlausibleKW := domain.PowerCeilingKW(systemSizeKW)
	now := time.Now().UTC()
	var previousEnergy *float64
	results := make([]SandboxReadingResult, 0, len(rows))
	accepted, rejected := 0, 0

	for _, pr := range rows {
		res := SandboxReadingResult{RowNumber: pr.rowNumber, Timestamp: pr.ts, PowerKW: pr.power, EnergyKWhTotal: pr.energy, VoltageV: pr.voltage, RSSI: pr.rssi, Status: pr.status}
		if pr.parseErr != "" {
			res.Accepted = false
			res.RejectionReason = pr.parseErr
			results = append(results, res)
			rejected++
			continue
		}

		payload := domain.TelemetryPayload{
			DeviceID:       "sandbox",
			Timestamp:      pr.ts.Format(time.RFC3339),
			PowerKW:        *pr.power,
			EnergyKWhTotal: *pr.energy,
			VoltageV:       pr.voltage,
			RSSI:           pr.rssi,
			Status:         pr.status,
		}
		if _, err := payload.Validate(maxPlausibleKW); err != nil {
			res.Accepted = false
			res.RejectionReason = err.Error()
			results = append(results, res)
			rejected++
			continue
		}

		isReset := domain.DetectEnergyReset(previousEnergy, *pr.energy)
		provenance := domain.ClassifyProvenance(*pr.ts, now)
		previousEnergy = pr.energy

		res.Accepted = true
		res.Provenance = string(provenance)
		res.IsReset = isReset
		results = append(results, res)
		accepted++
	}

	runID, err := newSandboxRunID()
	if err != nil {
		return SandboxUploadResult{}, fmt.Errorf("generate run id: %w", err)
	}

	if _, err := s.q.CreateSandboxRun(ctx, db.CreateSandboxRunParams{
		ID:            runID,
		SystemSizeKw:  numericOrNull(systemSizeKW),
		RowCount:      int32(len(results)),
		AcceptedCount: int32(accepted),
		RejectedCount: int32(rejected),
	}); err != nil {
		return SandboxUploadResult{}, fmt.Errorf("create sandbox run: %w", err)
	}

	for _, res := range results {
		var tsVal pgtype.Timestamptz
		if res.Timestamp != nil {
			tsVal = pgtype.Timestamptz{Time: *res.Timestamp, Valid: true}
		}
		var rssiVal pgtype.Int4
		if res.RSSI != nil {
			rssiVal = pgtype.Int4{Int32: int32(*res.RSSI), Valid: true}
		}
		if err := s.q.CreateSandboxReading(ctx, db.CreateSandboxReadingParams{
			RunID:           runID,
			RowNumber:       int32(res.RowNumber),
			Ts:              tsVal,
			PowerKw:         float8OrNull(res.PowerKW),
			EnergyKwhTotal:  float8OrNull(res.EnergyKWhTotal),
			VoltageV:        float8OrNull(res.VoltageV),
			Status:          textOrNull(strPtrOrNil(res.Status)),
			Accepted:        res.Accepted,
			RejectionReason: textOrNull(strPtrOrNil(res.RejectionReason)),
			Provenance:      textOrNull(strPtrOrNil(res.Provenance)),
			IsReset:         res.IsReset,
			Rssi:            rssiVal,
		}); err != nil {
			return SandboxUploadResult{}, fmt.Errorf("store sandbox reading row %d: %w", res.RowNumber, err)
		}
	}

	// Lazy self-cleanup — this table has no other retention mechanism.
	_ = s.q.DeleteOldSandboxRuns(ctx, pgtype.Timestamptz{Time: time.Now().Add(-sandboxRunRetention), Valid: true})

	return SandboxUploadResult{RunID: runID, RowCount: len(results), AcceptedCount: accepted, RejectedCount: rejected, Readings: results}, nil
}

var ErrSandboxRunNotFound = fmt.Errorf("sandbox run not found")

func (s *Sandbox) Get(ctx context.Context, runID string) (db.SandboxRun, []SandboxReadingResult, error) {
	run, err := s.q.GetSandboxRun(ctx, runID)
	if err != nil {
		return db.SandboxRun{}, nil, ErrSandboxRunNotFound
	}
	rows, err := s.q.ListSandboxReadings(ctx, runID)
	if err != nil {
		return db.SandboxRun{}, nil, err
	}
	out := make([]SandboxReadingResult, 0, len(rows))
	for _, r := range rows {
		res := SandboxReadingResult{
			RowNumber: int(r.RowNumber),
			Status:    textPtrValue(r.Status),
			Accepted:  r.Accepted,
			IsReset:   r.IsReset,
		}
		if r.Ts.Valid {
			t := r.Ts.Time.UTC()
			res.Timestamp = &t
		}
		if r.PowerKw.Valid {
			v := r.PowerKw.Float64
			res.PowerKW = &v
		}
		if r.EnergyKwhTotal.Valid {
			v := r.EnergyKwhTotal.Float64
			res.EnergyKWhTotal = &v
		}
		if r.VoltageV.Valid {
			v := r.VoltageV.Float64
			res.VoltageV = &v
		}
		if r.Rssi.Valid {
			v := int(r.Rssi.Int32)
			res.RSSI = &v
		}
		if r.RejectionReason.Valid {
			res.RejectionReason = r.RejectionReason.String
		}
		if r.Provenance.Valid {
			res.Provenance = r.Provenance.String
		}
		out = append(out, res)
	}
	return run, out, nil
}

func parseFlexibleTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized time format")
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func textPtrValue(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}
