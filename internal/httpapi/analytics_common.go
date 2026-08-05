package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// parseDateParam accepts either a bare date (2026-08-05) or a full RFC3339
// timestamp, since analytics query params are naturally date-grained while
// the rest of the API uses RFC3339 (see parseOptionalTime in
// site_handlers.go, used for telemetry's finer-grained from/to).
func parseDateParam(c echo.Context, param string) (*time.Time, error) {
	v := c.QueryParam(param)
	if v == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return &t, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// parsePeriod validates the period query param, defaulting to "monthly" —
// the most common granularity for the analytics endpoints (energy, yield,
// capacity-factor, comparisons, trends).
func parsePeriod(c echo.Context) (string, error) {
	v := c.QueryParam("period")
	if v == "" {
		return "monthly", nil
	}
	switch v {
	case "daily", "weekly", "monthly":
		return v, nil
	default:
		return "", echo.NewHTTPError(http.StatusBadRequest, "period must be daily, weekly, or monthly")
	}
}

// parseAnalyticsRange resolves the [from, to] window for an analytics
// query, defaulting to the trailing 30 days when not supplied. A bare date
// for "to" (e.g. 2026-08-05, no time-of-day) is treated as the END of that
// day — otherwise it would parse to midnight and exclude every reading
// from later the same day, which is correct for day-bucketed queries
// (their comparison is at day granularity) but silently wrong for anything
// comparing against raw timestamps (e.g. the telemetry CSV export).
func parseAnalyticsRange(c echo.Context) (from, to time.Time, err error) {
	toRaw := c.QueryParam("to")
	toPtr, err := parseDateParam(c, "to")
	if err != nil {
		return time.Time{}, time.Time{}, echo.NewHTTPError(http.StatusBadRequest, "invalid to date")
	}
	to = time.Now().UTC()
	if toPtr != nil {
		to = *toPtr
		if _, rfcErr := time.Parse(time.RFC3339, toRaw); rfcErr != nil {
			// toRaw was a bare date, not a full timestamp — extend to the
			// end of that day so same-day readings aren't excluded.
			to = to.Add(24*time.Hour - time.Nanosecond)
		}
	}

	fromPtr, err := parseDateParam(c, "from")
	if err != nil {
		return time.Time{}, time.Time{}, echo.NewHTTPError(http.StatusBadRequest, "invalid from date")
	}
	from = to.AddDate(0, 0, -30)
	if fromPtr != nil {
		from = *fromPtr
	}
	return from, to, nil
}

func parseAsOf(c echo.Context) (time.Time, error) {
	t, err := parseDateParam(c, "as_of")
	if err != nil {
		return time.Time{}, echo.NewHTTPError(http.StatusBadRequest, "invalid as_of date")
	}
	if t == nil {
		now := time.Now().UTC()
		return now, nil
	}
	return *t, nil
}

func parseWindowDays(c echo.Context, def int) int {
	v := c.QueryParam("window_days")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
