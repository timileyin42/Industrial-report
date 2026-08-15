package httpapi

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Response-side conversions from pgx's nullable pgtype wrappers to plain Go
// pointers, so dto.go structs marshal to clean JSON.

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func float8Ptr(f pgtype.Float8) *float64 {
	if !f.Valid {
		return nil
	}
	return &f.Float64
}

func int2Ptr(i pgtype.Int2) *int16 {
	if !i.Valid {
		return nil
	}
	return &i.Int16
}

func numericPtr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return nil
	}
	return &f.Float64
}

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
