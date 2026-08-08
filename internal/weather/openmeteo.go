// Package weather fetches historical solar irradiance from Open-Meteo's
// archive API — free, keyless, same provider the dashboard's live
// weather widget already uses client-side (web/src/lib/weather.ts), but
// this is the backend's own call for actual historical data, needed to
// compute Performance Ratio: "how much of the sunlight this site
// actually received did it convert into power," as opposed to Capacity
// Factor's weather-blind "output vs. flat theoretical max."
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type HourlyIrradiance struct {
	Time             time.Time
	ShortwaveWattsM2 float64
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// FetchHistoricalIrradiance returns hourly global horizontal irradiance
// (shortwave radiation, W/m²) for one location over [from, to]. Open-Meteo
// returns hourly timestamps in GMT by default (no &timezone= param sent)
// — deliberately left as GMT/UTC to match this platform's "everything
// stored and compared in UTC" rule, rather than requesting the site's
// local time and having to convert back.
func FetchHistoricalIrradiance(ctx context.Context, lat, lng float64, from, to time.Time) ([]HourlyIrradiance, error) {
	url := fmt.Sprintf(
		"https://archive-api.open-meteo.com/v1/archive?latitude=%s&longitude=%s&start_date=%s&end_date=%s&hourly=shortwave_radiation",
		strconv.FormatFloat(lat, 'f', 6, 64),
		strconv.FormatFloat(lng, 'f', 6, 64),
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("open-meteo archive: unexpected status %d: %s", res.StatusCode, string(body))
	}

	var parsed struct {
		Hourly struct {
			Time               []string  `json:"time"`
			ShortwaveRadiation []float64 `json:"shortwave_radiation"`
		} `json:"hourly"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	out := make([]HourlyIrradiance, 0, len(parsed.Hourly.Time))
	for i, t := range parsed.Hourly.Time {
		if i >= len(parsed.Hourly.ShortwaveRadiation) {
			break
		}
		ts, err := time.Parse("2006-01-02T15:04", t)
		if err != nil {
			continue
		}
		out = append(out, HourlyIrradiance{Time: ts, ShortwaveWattsM2: parsed.Hourly.ShortwaveRadiation[i]})
	}
	return out, nil
}

// DailyTotalsKWhPerM2 integrates hourly W/m² readings into a per-day
// kWh/m² total (each hourly reading × 1 hour = Wh/m², summed then
// converted to kWh) — this is the "how much sunlight actually landed on
// one square meter this day" figure Performance Ratio's expected-output
// calculation needs. Keyed by "YYYY-MM-DD" (UTC, per the package-level
// comment on FetchHistoricalIrradiance).
func DailyTotalsKWhPerM2(hours []HourlyIrradiance) map[string]float64 {
	byDay := make(map[string]float64)
	for _, h := range hours {
		day := h.Time.Format("2006-01-02")
		byDay[day] += h.ShortwaveWattsM2 / 1000.0
	}
	return byDay
}
