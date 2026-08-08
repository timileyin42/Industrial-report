package registry

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/pagination"
)

// recordAction writes to user_action_audit_log — the admin-action trail,
// kept separate from ingestion_audit_log per CLAUDE.md ("don't conflate the
// two"). A failure to audit-log is logged loudly but never blocks the
// underlying action that already succeeded.
func recordAction(ctx context.Context, q *db.Queries, actorUserID int64, action, targetType, targetID string, metadata map[string]any) {
	var metaJSON []byte
	if metadata != nil {
		var err error
		metaJSON, err = json.Marshal(metadata)
		if err != nil {
			log.Printf("audit: marshal metadata for action %s: %v", action, err)
		}
	}
	err := q.CreateUserActionAuditLog(ctx, db.CreateUserActionAuditLogParams{
		ActorUserID: pgtype.Int8{Int64: actorUserID, Valid: true},
		Action:      action,
		TargetType:  pgtype.Text{String: targetType, Valid: targetType != ""},
		TargetID:    pgtype.Text{String: targetID, Valid: targetID != ""},
		Metadata:    metaJSON,
	})
	if err != nil {
		log.Printf("audit: failed to record action %s on %s/%s: %v", action, targetType, targetID, err)
	}
}

// AuditLog is the read side of user_action_audit_log — the Phase 3
// catch-up on migration 0002's own deferred TODO ("a browsing/reporting
// UI on this table is Phase 3").
type AuditLog struct {
	q *db.Queries
}

func NewAuditLog(q *db.Queries) *AuditLog {
	return &AuditLog{q: q}
}

type AuditEntry struct {
	ID         int64
	ActorEmail *string
	Action     string
	TargetType *string
	TargetID   *string
	Metadata   []byte
	CreatedAt  time.Time
}

type ListAuditInput struct {
	ActorUserID *int64
	Action      *string
	TargetType  *string
	TargetID    *string
	From        *time.Time
	To          *time.Time
	CursorToken string
	Limit       int
}

func (a *AuditLog) List(ctx context.Context, in ListAuditInput) ([]AuditEntry, string, error) {
	limit := in.Limit
	if limit <= 0 || limit > 200 {
		limit = pagination.DefaultPageLimit
	}

	var actorUserID pgtype.Int8
	if in.ActorUserID != nil {
		actorUserID = pgtype.Int8{Int64: *in.ActorUserID, Valid: true}
	}

	var cursorCreatedAt pgtype.Timestamptz
	var cursorID pgtype.Int8
	if in.CursorToken != "" {
		c, err := pagination.Decode(in.CursorToken)
		if err != nil {
			return nil, "", err
		}
		cursorCreatedAt = pgtype.Timestamptz{Time: c.Time, Valid: true}
		id, err := strconv.ParseInt(c.Tiebreak, 10, 64)
		if err != nil {
			return nil, "", err
		}
		cursorID = pgtype.Int8{Int64: id, Valid: true}
	}

	rows, err := a.q.ListUserActionAuditLog(ctx, db.ListUserActionAuditLogParams{
		ActorUserID:     actorUserID,
		Action:          textOrNull(in.Action),
		TargetType:      textOrNull(in.TargetType),
		TargetID:        textOrNull(in.TargetID),
		FromTs:          timestamptzOrNull(in.From),
		ToTs:            timestamptzOrNull(in.To),
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
		PageLimit:       int32(limit),
	})
	if err != nil {
		return nil, "", err
	}

	entries := make([]AuditEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, AuditEntry{
			ID:         r.ID,
			ActorEmail: textPtr(r.ActorEmail),
			Action:     r.Action,
			TargetType: textPtr(r.TargetType),
			TargetID:   textPtr(r.TargetID),
			Metadata:   r.Metadata,
			CreatedAt:  r.CreatedAt.Time,
		})
	}

	next := ""
	if len(rows) == limit {
		last := rows[len(rows)-1]
		next, err = pagination.Encode(pagination.Cursor{Time: last.CreatedAt.Time, Tiebreak: strconv.FormatInt(last.ID, 10)})
		if err != nil {
			return nil, "", err
		}
	}
	return entries, next, nil
}

// VerifyChain re-derives the hash chain migrations/0013 wrote at insert
// time and reports whether it still matches — see
// VerifyUserActionAuditChain's own doc comment for why the recompute
// happens in SQL rather than Go.
func (a *AuditLog) VerifyChain(ctx context.Context) (ChainVerifyResult, error) {
	row, err := a.q.VerifyUserActionAuditChain(ctx)
	if err != nil {
		return ChainVerifyResult{}, err
	}
	return toChainVerifyResult(row.MismatchCount, row.FirstBadID), nil
}
