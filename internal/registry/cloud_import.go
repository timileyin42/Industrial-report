// Package registry: CloudImport is the genuinely vendor-agnostic
// alternative to the MQTT pipeline, for a device whose inverter only
// reports into a manufacturer's own cloud app rather than talking
// Modbus/MQTT directly. This platform deliberately never integrates
// with any specific vendor's proprietary API — every one has a
// different shape, auth model, and most aren't public at all. Instead,
// an operator issues a per-device bearer token here, and whatever
// external glue actually holds that vendor's own login (a scraper, a
// scheduled script, a Google Apps Script watching an export folder)
// pushes readings to us in one fixed JSON shape. We never store or see
// any third-party vendor credential — only our own token, hashed
// exactly like a device's MQTT secret.
package registry

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
)

type CloudImport struct {
	q *db.Queries
}

func NewCloudImport(q *db.Queries) *CloudImport {
	return &CloudImport{q: q}
}

var (
	ErrInvalidCloudImportToken = errors.New("invalid or revoked cloud import token")
	ErrDeviceRevoked           = errors.New("device is revoked")
)

// IssueToken generates a new bearer token for a device's cloud-import
// path, revoking any previous one first — one active credential at a
// time, same model as RotateDeviceSecret. The plaintext token is
// returned exactly once, like a device secret; only its hash is stored.
func (c *CloudImport) IssueToken(ctx context.Context, actorUserID int64, deviceID string) (string, error) {
	if _, err := c.q.GetDevice(ctx, deviceID); err != nil {
		return "", ErrUnknownDevice
	}

	token, err := auth.GenerateDeviceSecret()
	if err != nil {
		return "", err
	}
	hash, err := auth.HashSecret(token)
	if err != nil {
		return "", err
	}

	if err := c.q.RevokeCloudImportTokensForDevice(ctx, deviceID); err != nil {
		return "", err
	}
	if _, err := c.q.CreateCloudImportToken(ctx, db.CreateCloudImportTokenParams{DeviceID: deviceID, TokenHash: hash}); err != nil {
		return "", err
	}
	recordAction(ctx, c.q, actorUserID, "device.cloud_import_token.issue", "device", deviceID, nil)
	return token, nil
}

func (c *CloudImport) authenticate(ctx context.Context, deviceID, token string) error {
	tokens, err := c.q.ListActiveCloudImportTokensForDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	for _, t := range tokens {
		if auth.VerifySecret(t.TokenHash, token) {
			_ = c.q.MarkCloudImportTokenUsed(ctx, t.ID)
			return nil
		}
	}
	return ErrInvalidCloudImportToken
}

// CloudReading is the one fixed shape every external source must
// normalize its own vendor's data into before pushing it here — the
// same fields domain.TelemetryPayload accepts from a real MQTT
// datalogger, so a cloud-imported reading gets identical validation,
// reset detection, and provenance classification. No vendor-specific
// parsing lives on this platform at all.
type CloudReading struct {
	Timestamp       string
	PowerKW         float64
	EnergyKWhTotal  float64
	VoltageV        *float64
	Status          string
	PVPowerKW       *float64
	BatterySOCPct   *float64
	BatteryVoltageV *float64
	PVVoltageV      *float64
	OutputVoltageV  *float64
}

type CloudReadingResult struct {
	Timestamp       string
	Accepted        bool
	RejectionReason string
	Duplicate       bool
}

// SubmitReadings authenticates the bearer token, then runs every
// reading through the exact same validation/classification pipeline the
// MQTT ingestor uses (domain.TelemetryPayload.Validate, ClassifyProvenance,
// DetectEnergyReset) before inserting — a cloud-imported reading is held
// to the same "never silently store impossible data" standard as a real
// device's, not a looser one.
func (c *CloudImport) SubmitReadings(ctx context.Context, deviceID, token string, readings []CloudReading) ([]CloudReadingResult, error) {
	if err := c.authenticate(ctx, deviceID, token); err != nil {
		return nil, err
	}

	ctxInfo, err := c.q.GetDeviceWithSiteContext(ctx, deviceID)
	if err != nil {
		return nil, ErrUnknownDevice
	}
	if ctxInfo.RevokedAt.Valid {
		return nil, ErrDeviceRevoked
	}
	siteID := ctxInfo.SiteID.String
	var systemSizeKW *float64
	if ctxInfo.SystemSizeKw.Valid {
		v, err := ctxInfo.SystemSizeKw.Float64Value()
		if err == nil && v.Valid {
			systemSizeKW = &v.Float64
		}
	}
	ceiling := domain.PowerCeilingKW(systemSizeKW)
	now := time.Now().UTC()

	results := make([]CloudReadingResult, 0, len(readings))
	var maxAcceptedTS *time.Time
	for _, r := range readings {
		res := CloudReadingResult{Timestamp: r.Timestamp}

		// Audit first, unconditionally, before validation — same
		// discipline as the MQTT ingestor: a reading that fails
		// validation below is still recorded, never silently dropped.
		rawPayload, _ := json.Marshal(r)
		auditID, auditErr := c.q.CreateIngestionAuditRow(ctx, db.CreateIngestionAuditRowParams{DeviceID: deviceID, RawPayload: rawPayload})
		if auditErr != nil {
			log.Printf("cloud import: failed to write audit row for %s: %v", deviceID, auditErr)
		}
		markAudit := func(readingErr error) {
			if auditErr != nil {
				return // no audit row exists to update
			}
			if readingErr != nil {
				if err := c.q.MarkIngestionAuditError(ctx, db.MarkIngestionAuditErrorParams{ID: auditID, Error: pgtype.Text{String: readingErr.Error(), Valid: true}}); err != nil {
					log.Printf("cloud import: mark audit error for %s: %v", deviceID, err)
				}
				return
			}
			if err := c.q.MarkIngestionAuditProcessed(ctx, auditID); err != nil {
				log.Printf("cloud import: mark audit processed for %s: %v", deviceID, err)
			}
		}

		payload := domain.TelemetryPayload{
			DeviceID:        deviceID,
			Timestamp:       r.Timestamp,
			PowerKW:         r.PowerKW,
			EnergyKWhTotal:  r.EnergyKWhTotal,
			VoltageV:        r.VoltageV,
			Status:          r.Status,
			PVPowerKW:       r.PVPowerKW,
			BatterySOCPct:   r.BatterySOCPct,
			BatteryVoltageV: r.BatteryVoltageV,
			PVVoltageV:      r.PVVoltageV,
			OutputVoltageV:  r.OutputVoltageV,
		}

		ts, err := payload.Validate(ceiling)
		if err != nil {
			res.RejectionReason = err.Error()
			results = append(results, res)
			markAudit(err)
			continue
		}

		provenance := domain.ClassifyProvenance(ts, now)
		qualityFlags := []string{}
		previousEnergy, err := c.previousEnergy(ctx, deviceID, ts)
		if err != nil {
			log.Printf("cloud import: lookup previous energy for %s: %v", deviceID, err)
		} else if domain.DetectEnergyReset(previousEnergy, r.EnergyKWhTotal) {
			qualityFlags = append(qualityFlags, domain.QualityFlagEnergyReset)
		}

		rows, err := c.q.InsertTelemetryReading(ctx, db.InsertTelemetryReadingParams{
			DeviceID:        deviceID,
			SiteID:          siteID,
			Ts:              pgtype.Timestamptz{Time: ts, Valid: true},
			PowerKw:         r.PowerKW,
			EnergyKwhTotal:  r.EnergyKWhTotal,
			VoltageV:        float8OrNull(r.VoltageV),
			Status:          db.ReadingStatus(coalesceStatus(r.Status)),
			Provenance:      db.ProvenanceType(provenance),
			QualityFlags:    qualityFlags,
			PvPowerKw:       float8OrNull(r.PVPowerKW),
			BatterySocPct:   int2OrNull(r.BatterySOCPct),
			BatteryVoltageV: float8OrNull(r.BatteryVoltageV),
			PvVoltageV:      float8OrNull(r.PVVoltageV),
			OutputVoltageV:  float8OrNull(r.OutputVoltageV),
		})
		if err != nil {
			res.RejectionReason = "insert failed: " + err.Error()
			results = append(results, res)
			markAudit(err)
			continue
		}

		res.Accepted = true
		res.Duplicate = rows == 0 // ON CONFLICT DO NOTHING — already had this exact (device_id, ts)
		results = append(results, res)
		markAudit(nil)
		if maxAcceptedTS == nil || ts.After(*maxAcceptedTS) {
			maxAcceptedTS = &ts
		}
	}

	if _, err := c.q.UpdateDeviceLastContact(ctx, db.UpdateDeviceLastContactParams{DeviceID: deviceID, LastContactAt: pgtype.Timestamptz{Time: now, Valid: true}}); err != nil {
		log.Printf("cloud import: update last_contact_at for %s: %v", deviceID, err)
	}
	if maxAcceptedTS != nil {
		if err := c.q.AdvanceDeviceLastSeen(ctx, db.AdvanceDeviceLastSeenParams{DeviceID: deviceID, LastSeenAt: pgtype.Timestamptz{Time: *maxAcceptedTS, Valid: true}}); err != nil {
			log.Printf("cloud import: advance last_seen_at for %s: %v", deviceID, err)
		}
	}
	return results, nil
}

func (c *CloudImport) previousEnergy(ctx context.Context, deviceID string, ts time.Time) (*float64, error) {
	v, err := c.q.PreviousEnergyBeforeTS(ctx, db.PreviousEnergyBeforeTSParams{DeviceID: deviceID, Ts: pgtype.Timestamptz{Time: ts, Valid: true}})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

func coalesceStatus(s string) string {
	if s == "" {
		return domain.StatusOK
	}
	return s
}
