package registry

import (
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Small conversion helpers between plain Go values (as used in HTTP
// request/response structs) and pgx's nullable pgtype wrappers. Kept in one
// place so every registry file uses the same nil/zero conventions.

func textOrNull(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func float8OrNull(f *float64) pgtype.Float8 {
	if f == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *f, Valid: true}
}

func float8Ptr(f pgtype.Float8) *float64 {
	if !f.Valid {
		return nil
	}
	return &f.Float64
}

func int2OrNull(f *float64) pgtype.Int2 {
	if f == nil {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: int16(*f), Valid: true}
}

func numericOrNull(f *float64) pgtype.Numeric {
	var n pgtype.Numeric
	if f == nil {
		return n
	}
	_ = n.Scan(strconv.FormatFloat(*f, 'f', -1, 64))
	return n
}

func numericToFloat(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return nil
	}
	return &f.Float64
}

func dateOrNull(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func timestamptzOrNull(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
