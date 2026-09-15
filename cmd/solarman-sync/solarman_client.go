package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// solarmanBaseURL is Solarman's real, documented Open API (doc.solarmanpv.com)
// — unlike pvpro-sync's reverse-engineered PV Pro connector, this one is
// built against an officially supported API (App ID/Secret issued by
// Solarman after a manual request to customerservice@solarmanpv.com).
// Solarman's own datalogger/cloud platform is used as a white-label
// backend by roughly 200 inverter brands (Deye, Growatt, Solis, GoodWe,
// SMA, Sungrow, Sofar, SolaX, Huawei, and more) — one connector here
// covers all of them the same way one pvpro-sync connector already
// covers PV Pro/Sunsynk/Powerview (which share the E-linter CSP backend).
const solarmanBaseURL = "https://globalapi.solarmanpv.com"

type solarmanClient struct {
	appID      string
	appSecret  string
	email      string
	password   string
	httpClient *http.Client

	accessToken string
	tokenAt     time.Time
	expiresIn   time.Duration
}

func newSolarmanClient(appID, appSecret, email, password string) *solarmanClient {
	return &solarmanClient{
		appID:      appID,
		appSecret:  appSecret,
		email:      email,
		password:   password,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *solarmanClient) ensureToken(ctx context.Context) error {
	if c.accessToken != "" && time.Since(c.tokenAt) < c.expiresIn {
		return nil
	}
	return c.authenticate(ctx)
}

// authenticate mirrors the flow verified against two independent working
// Solarman Open API clients (a Home Assistant integration's api.py, and
// the vendor's own documented "account/v1.0/token" endpoint): password is
// sent as its SHA-256 hex digest, never plaintext, alongside the app
// secret and account email.
func (c *solarmanClient) authenticate(ctx context.Context) error {
	passHash := sha256.Sum256([]byte(c.password))
	body, _ := json.Marshal(map[string]string{
		"appSecret": c.appSecret,
		"email":     c.email,
		"password":  hex.EncodeToString(passHash[:]),
	})
	url := fmt.Sprintf("%s/account/v1.0/token?appId=%s&language=en", solarmanBaseURL, c.appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out struct {
		Success     bool   `json:"success"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		Code        string `json:"code"`
		Msg         string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if !out.Success || out.AccessToken == "" {
		return fmt.Errorf("solarman auth failed: code=%s msg=%s", out.Code, out.Msg)
	}
	c.accessToken = out.AccessToken
	c.tokenAt = time.Now()
	// Refreshed a minute early, same margin as the reference
	// implementations, so a request never races an expiring token.
	c.expiresIn = time.Duration(out.ExpiresIn)*time.Second - time.Minute
	return nil
}

func (c *solarmanClient) post(ctx context.Context, path string, body, out any) error {
	if err := c.ensureToken(ctx); err != nil {
		return err
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, solarmanBaseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode %s response: %w (body: %s)", path, err, truncate(string(respBody), 300))
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// solarmanStation is deliberately loose (raw map alongside the two fields
// we're confident about) rather than a fully-typed struct: id/name are
// confirmed from two independent real Solarman Open API clients, but the
// exact key for GPS coordinates isn't confirmed against a live account
// yet. stationCoords() tries the plausible candidates and falls back to
// "no coordinates" rather than guessing wrong and silently registering a
// site at (0,0) — same discipline as every other "don't fabricate a
// field we haven't verified" case in this codebase.
type solarmanStation struct {
	ID   int64          `json:"id"`
	Name string         `json:"name"`
	Raw  map[string]any `json:"-"`
}

func (c *solarmanClient) listStations(ctx context.Context) ([]solarmanStation, error) {
	var out struct {
		Success     bool             `json:"success"`
		Msg         string           `json:"msg"`
		StationList []map[string]any `json:"stationList"`
	}
	if err := c.post(ctx, "/station/v1.0/list", map[string]any{"page": 1, "size": 100}, &out); err != nil {
		return nil, err
	}
	if !out.Success {
		return nil, fmt.Errorf("list stations: %s", out.Msg)
	}
	stations := make([]solarmanStation, 0, len(out.StationList))
	for _, raw := range out.StationList {
		id, _ := raw["id"].(float64)
		name, _ := raw["name"].(string)
		stations = append(stations, solarmanStation{ID: int64(id), Name: name, Raw: raw})
	}
	return stations, nil
}

// stationCoords tries the field-name candidates seen across similar
// vendor APIs. Returns ok=false (never a fabricated 0,0) if none match —
// the site is then created without GPS and needs one manual
// PATCH /v1/sites/:id/location, same as any site registered without
// known coordinates.
func stationCoords(raw map[string]any) (lat, lng float64, ok bool) {
	latCandidates := []string{"locationLat", "lat", "latitude"}
	lngCandidates := []string{"locationLng", "lng", "longitude", "locationLon", "lon"}
	var latVal, lngVal float64
	var latOK, lngOK bool
	for _, k := range latCandidates {
		if v, present := raw[k].(float64); present && v != 0 {
			latVal, latOK = v, true
			break
		}
	}
	for _, k := range lngCandidates {
		if v, present := raw[k].(float64); present && v != 0 {
			lngVal, lngOK = v, true
			break
		}
	}
	return latVal, lngVal, latOK && lngOK
}

type solarmanDevice struct {
	SN         string `json:"deviceSn"`
	DeviceType string `json:"deviceType"` // "INVERTER", "BATTERY", "COLLECTOR" — confirmed values
	Raw        map[string]any
}

func (c *solarmanClient) listDevices(ctx context.Context, stationID int64) ([]solarmanDevice, error) {
	var out struct {
		Success         bool             `json:"success"`
		Msg             string           `json:"msg"`
		DeviceListItems []map[string]any `json:"deviceListItems"`
	}
	if err := c.post(ctx, "/station/v1.0/device", map[string]any{"stationId": stationID}, &out); err != nil {
		return nil, err
	}
	if !out.Success {
		return nil, fmt.Errorf("list devices for station %d: %s", stationID, out.Msg)
	}
	devices := make([]solarmanDevice, 0, len(out.DeviceListItems))
	for _, raw := range out.DeviceListItems {
		sn, _ := raw["deviceSn"].(string)
		dt, _ := raw["deviceType"].(string)
		devices = append(devices, solarmanDevice{SN: sn, DeviceType: dt, Raw: raw})
	}
	return devices, nil
}

// currentDataPoint is one entry in Solarman's dataList — a flat
// key/value bag rather than a fixed schema, since ~200 brands' differing
// register maps all normalize into the same response shape. Key names
// below (T_AC_OP, PVTP, Et_ge0, etc.) are confirmed against a real,
// tested community client's sensor-key mapping, not guessed.
type currentDataPoint struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

type solarmanCurrentData struct {
	Success        bool               `json:"success"`
	Msg            string             `json:"msg"`
	CollectionTime int64              `json:"collectionTime"`
	DataList       []currentDataPoint `json:"dataList"`
}

func (c *solarmanClient) currentData(ctx context.Context, deviceSN string) (solarmanCurrentData, error) {
	var out solarmanCurrentData
	if err := c.post(ctx, "/device/v1.0/currentData", map[string]any{"deviceSn": deviceSN}, &out); err != nil {
		return solarmanCurrentData{}, err
	}
	if !out.Success {
		return solarmanCurrentData{}, fmt.Errorf("current data for %s: %s", deviceSN, out.Msg)
	}
	return out, nil
}

// value looks up one key in dataList and parses it as a float64. ok=false
// (never a fabricated 0) when the key is genuinely absent from this
// device's response — different inverter models/brands behind Solarman
// report different subsets of keys.
func (d solarmanCurrentData) value(key string) (float64, bool) {
	for _, p := range d.DataList {
		if p.Key == key {
			var f float64
			if _, err := fmt.Sscanf(p.Value, "%g", &f); err != nil {
				return 0, false
			}
			return f, true
		}
	}
	return 0, false
}
