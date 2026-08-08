package registry

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/pagination"
)

// IngestionAudit is the read side of ingestion_audit_log — the ingestor's
// own data-quality/verification trail (every message received, before
// validation), previously write-only. Deliberately separate from AuditLog
// (user_action_audit_log) — see CLAUDE.md: "don't conflate the two."
type IngestionAudit struct {
	q *db.Queries
}

func NewIngestionAudit(q *db.Queries) *IngestionAudit {
	return &IngestionAudit{q: q}
}

// LastReceivedAt is the ingestion-pipeline health signal the Dashboard's
// status widget uses — "how long ago did the ingestor last see anything
// at all" (valid or not), never a synthetic uptime percentage this
// platform has no real data source for. nil means the fleet has never
// received a single message.
func (a *IngestionAudit) LastReceivedAt(ctx context.Context) (*time.Time, error) {
	ts, err := a.q.LastIngestionReceivedAt(ctx)
	if err != nil {
		return nil, err
	}
	if !ts.Valid {
		return nil, nil
	}
	return &ts.Time, nil
}

type IngestionAuditEntry struct {
	ID         int64
	DeviceID   string
	SiteID     *string
	RawPayload json.RawMessage
	ReceivedAt time.Time
	Processed  bool
	Error      *string
}

type ListIngestionAuditInput struct {
	// SiteFilter restricts results to one site — set for restricted users
	// (their own site), nil for operators (no filter). Mirrors listSites'
	// siteFilter pattern in site_handlers.go.
	SiteFilter  *string
	DeviceID    *string
	ErrorsOnly  *bool
	From        *time.Time
	To          *time.Time
	CursorToken string
	Limit       int
}

func (a *IngestionAudit) List(ctx context.Context, in ListIngestionAuditInput) ([]IngestionAuditEntry, string, error) {
	limit := in.Limit
	if limit <= 0 || limit > 200 {
		limit = pagination.DefaultPageLimit
	}

	var cursorReceivedAt pgtype.Timestamptz
	var cursorID pgtype.Int8
	if in.CursorToken != "" {
		c, err := pagination.Decode(in.CursorToken)
		if err != nil {
			return nil, "", err
		}
		cursorReceivedAt = pgtype.Timestamptz{Time: c.Time, Valid: true}
		id, err := strconv.ParseInt(c.Tiebreak, 10, 64)
		if err != nil {
			return nil, "", err
		}
		cursorID = pgtype.Int8{Int64: id, Valid: true}
	}

	var errorsOnly pgtype.Bool
	if in.ErrorsOnly != nil {
		errorsOnly = pgtype.Bool{Bool: *in.ErrorsOnly, Valid: true}
	}

	rows, err := a.q.ListIngestionAuditLog(ctx, db.ListIngestionAuditLogParams{
		SiteID:           textOrNull(in.SiteFilter),
		DeviceID:         textOrNull(in.DeviceID),
		ErrorsOnly:       errorsOnly,
		FromTs:           timestamptzOrNull(in.From),
		ToTs:             timestamptzOrNull(in.To),
		CursorReceivedAt: cursorReceivedAt,
		CursorID:         cursorID,
		PageLimit:        int32(limit),
	})
	if err != nil {
		return nil, "", err
	}

	entries := make([]IngestionAuditEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, IngestionAuditEntry{
			ID:         r.ID,
			DeviceID:   r.DeviceID,
			SiteID:     textPtr(r.SiteID),
			RawPayload: json.RawMessage(r.RawPayload),
			ReceivedAt: r.ReceivedAt.Time,
			Processed:  r.Processed,
			Error:      textPtr(r.Error),
		})
	}

	next := ""
	if len(rows) == limit {
		last := rows[len(rows)-1]
		next, err = pagination.Encode(pagination.Cursor{Time: last.ReceivedAt.Time, Tiebreak: strconv.FormatInt(last.ID, 10)})
		if err != nil {
			return nil, "", err
		}
	}
	return entries, next, nil
}

// VerifyChain re-derives the hash chain migrations/0013 wrote at insert
// time and reports whether it still matches.
func (a *IngestionAudit) VerifyChain(ctx context.Context) (ChainVerifyResult, error) {
	row, err := a.q.VerifyIngestionAuditChain(ctx)
	if err != nil {
		return ChainVerifyResult{}, err
	}
	return toChainVerifyResult(row.MismatchCount, row.FirstBadID), nil
}
