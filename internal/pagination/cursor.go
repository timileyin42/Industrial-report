// Package pagination implements opaque keyset cursors shared by every list
// endpoint. Keyset (not offset) per CLAUDE.md: offset pagination degrades at
// fleet scale and is unstable under concurrent inserts.
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

const DefaultPageLimit = 50

// Cursor is the sort key of the last row on the previous page: a timestamp
// plus a string tiebreaker (site_id/device_id/etc.) for rows sharing that
// timestamp.
type Cursor struct {
	Time     time.Time `json:"t"`
	Tiebreak string    `json:"b"`
}

func Encode(c Cursor) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func Decode(token string) (Cursor, error) {
	var c Cursor
	if token == "" {
		return c, errors.New("empty cursor")
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, err
	}
	return c, nil
}
