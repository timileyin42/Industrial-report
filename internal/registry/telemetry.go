package registry

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/pagination"
)

type Telemetry struct {
	q *db.Queries
}

func NewTelemetry(q *db.Queries) *Telemetry {
	return &Telemetry{q: q}
}

type ListTelemetryInput struct {
	SiteID      string
	From        *time.Time
	To          *time.Time
	CursorToken string
	Limit       int
}

func (t *Telemetry) List(ctx context.Context, in ListTelemetryInput) ([]db.ListTelemetryForSiteRow, string, error) {
	limit := in.Limit
	if limit <= 0 || limit > 500 {
		limit = pagination.DefaultPageLimit
	}

	var cursorTs pgtype.Timestamptz
	var cursorDeviceID pgtype.Text
	if in.CursorToken != "" {
		c, err := pagination.Decode(in.CursorToken)
		if err != nil {
			return nil, "", err
		}
		cursorTs = pgtype.Timestamptz{Time: c.Time, Valid: true}
		cursorDeviceID = pgtype.Text{String: c.Tiebreak, Valid: true}
	}

	rows, err := t.q.ListTelemetryForSite(ctx, db.ListTelemetryForSiteParams{
		SiteID:         in.SiteID,
		FromTs:         timestamptzOrNull(in.From),
		ToTs:           timestamptzOrNull(in.To),
		CursorTs:       cursorTs,
		CursorDeviceID: cursorDeviceID,
		PageLimit:      int32(limit),
	})
	if err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) == limit {
		last := rows[len(rows)-1]
		next, err = pagination.Encode(pagination.Cursor{Time: last.Ts.Time, Tiebreak: last.DeviceID})
		if err != nil {
			return nil, "", err
		}
	}
	return rows, next, nil
}
