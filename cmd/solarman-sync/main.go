// Command solarman-sync is the cloud-import connector for Solarman's
// Open API (doc.solarmanpv.com) — a real, officially documented API,
// unlike pvpro-sync's reverse-engineered PV Pro connector. Solarman is a
// white-label monitoring backend used by roughly 200 inverter brands
// (Deye, Growatt, Solis, GoodWe, SMA, Sungrow, Sofar, SolaX, Huawei, and
// more), so this one connector covers any customer whose inverter
// reports through it — the same "one connector, many brands" leverage
// pvpro-sync already gets from PV Pro/Sunsynk/Powerview sharing the
// E-linter CSP backend, just at far larger scale and without needing to
// reverse-engineer anything.
//
// Endpoint shapes and the currentData key mapping (T_AC_OP, PVTP,
// Et_ge0, B_left_cap1, E_Puse_t1, PG_Pt1, etc.) are confirmed against two
// independent real, tested Solarman Open API client implementations —
// see solarman_client.go's own comments for exactly what's confirmed vs.
// best-effort (station GPS coordinates specifically aren't confirmed
// against a live account yet, and are handled as optional rather than
// guessed).
//
// Same architecture as pvpro-sync: auto-discovery every cycle (asks
// Solarman for every station/device on the account, not a fixed list),
// reconciled against what's already registered on our platform
// (our_client.go, copied verbatim — it's fully vendor-agnostic), and
// readings forwarded through the same POST /v1/cloud-import/:device_id/
// readings path any other external source could use.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// knownDevice caches what a full discovery pass already resolved for one
// device, so subsequent cycles don't re-issue a cloud-import token or
// re-check existence with our API every poll — same rationale as
// pvpro-sync's identically-named type.
type knownDevice struct {
	siteID string
	token  string
}

func main() {
	appID := mustEnv("SOLARMAN_APP_ID")
	appSecret := mustEnv("SOLARMAN_APP_SECRET")
	email := mustEnv("SOLARMAN_EMAIL")
	password := mustEnv("SOLARMAN_PASSWORD")
	apiBaseURL := mustEnv("API_BASE_URL")
	operatorEmail := mustEnv("API_OPERATOR_EMAIL")
	operatorPassword := mustEnv("API_OPERATOR_PASSWORD")
	// A separate env var from pvpro-sync's own POLL_INTERVAL_SECONDS —
	// both connectors share one .env file, so reusing that name would
	// mean setting it for one silently changes the other's cadence too.
	//
	// Default is far more conservative than pvpro-sync's 30s: Solarman's
	// free tier caps at 200,000 API calls/year, and each cycle costs
	// roughly (1 + stations + devices) calls — list_stations,
	// list_devices per station, currentData per device. Even a modest
	// 10-device/3-station fleet is ~14 calls/cycle; polling every 30s
	// like pvpro-sync would burn the entire annual free-tier budget in
	// about 5 days. 20 minutes keeps a 10-device fleet under roughly a
	// third of the free-tier budget — tune down (and consider Solarman's
	// paid tier) only once real fleet size is known.
	pollInterval := envSeconds("SOLARMAN_POLL_INTERVAL_SECONDS", 1200)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	sm := newSolarmanClient(appID, appSecret, email, password)
	ours := newOurAPIClient(apiBaseURL, operatorEmail, operatorPassword)
	known := map[string]knownDevice{}

	log.Printf("solarman-sync starting: auto-discovery mode, polling every %s", pollInterval)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	runOnce(ctx, sm, ours, known)
	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		case <-ticker.C:
			runOnce(ctx, sm, ours, known)
		}
	}
}

func runOnce(ctx context.Context, sm *solarmanClient, ours *ourAPIClient, known map[string]knownDevice) {
	stations, err := sm.listStations(ctx)
	if err != nil {
		log.Printf("solarman: fetch stations: %v", err)
		return
	}

	var syncList []string // device SNs to sync this cycle

	for _, station := range stations {
		devices, err := sm.listDevices(ctx, station.ID)
		if err != nil {
			log.Printf("solarman: fetch devices for station %d (%s): %v", station.ID, station.Name, err)
			continue
		}
		// Only INVERTER-type devices are registered as platform devices —
		// a station's own "BATTERY"/"COLLECTOR" entries are finer-grained
		// diagnostics for the same physical system, not a second thing to
		// monitor separately; the inverter's own currentData already
		// includes battery SOC/power when one is attached (see
		// solarman_client.go's key-mapping comment).
		var inverters []solarmanDevice
		for _, d := range devices {
			if d.DeviceType == "INVERTER" && d.SN != "" {
				inverters = append(inverters, d)
			}
		}
		if len(inverters) == 0 {
			continue
		}

		siteID, err := reconcileSite(ctx, ours, station, inverters, known)
		if err != nil {
			log.Printf("solarman: reconcile site for station %d (%s): %v", station.ID, station.Name, err)
			continue
		}

		for _, inv := range inverters {
			if _, ok := known[inv.SN]; !ok {
				if err := ensureDeviceRegisteredAndTokenCached(ctx, ours, siteID, inv, known); err != nil {
					log.Printf("solarman: ensure device %s registered: %v", inv.SN, err)
					continue
				}
			}
			syncList = append(syncList, inv.SN)
		}
	}

	for i, sn := range syncList {
		if i > 0 {
			time.Sleep(1500 * time.Millisecond) // spread out our own cloud-import calls
		}
		kd := known[sn]
		data, err := sm.currentData(ctx, sn)
		if err != nil {
			log.Printf("solarman: fetch current data for %s: %v", sn, err)
			continue
		}
		reading := buildReading(data)
		if err := submitReading(ctx, kd.token, ours.baseURL, sn, reading); err != nil {
			log.Printf("solarman: submit reading for device %s: %v", sn, err)
			continue
		}
		log.Printf("solarman: synced device %s — %.2f kW AC, %.2f kW PV, ts=%s", sn, reading.PowerKW, floatOrZero(reading.PVPowerKW), reading.Timestamp)
	}
}

// reconcileSite mirrors pvpro-sync's own — reuses an existing site_id if
// ANY of the station's inverters are already registered, otherwise
// creates a genuinely new one.
func reconcileSite(ctx context.Context, ours *ourAPIClient, station solarmanStation, inverters []solarmanDevice, known map[string]knownDevice) (string, error) {
	for _, inv := range inverters {
		if kd, ok := known[inv.SN]; ok {
			return kd.siteID, nil
		}
		siteID, exists, err := ours.findDeviceSite(ctx, inv.SN)
		if err != nil {
			return "", err
		}
		if exists {
			return siteID, nil
		}
	}

	siteID := fmt.Sprintf("SOLARMAN-%d", station.ID)
	alreadyExists, err := ours.siteExists(ctx, siteID)
	if err != nil {
		return "", err
	}
	if alreadyExists {
		return siteID, nil
	}

	lat, lng, hasCoords := stationCoords(station.Raw)
	if !hasCoords {
		log.Printf("solarman: station %d (%q) has no recognized GPS field — registering without location, set it manually via PATCH /v1/sites/%s/location. Raw fields seen: %v",
			station.ID, station.Name, siteID, rawKeys(station.Raw))
	}
	if err := ours.createSite(ctx, newSiteInput{
		SiteID: siteID,
		Name:   station.Name,
		GPSLat: lat,
		GPSLng: lng,
		// Timezone/country aren't confirmed fields on the station object
		// yet either — defaulting to the operator account's own known
		// deployment (UK) rather than guessing a per-station value.
		// Correct manually if a station turns out to be elsewhere.
		Timezone: "Europe/London",
		Country:  "GB",
	}); err != nil {
		return "", fmt.Errorf("create site: %w", err)
	}
	log.Printf("solarman: auto-registered new site %s (%q)", siteID, station.Name)
	return siteID, nil
}

func ensureDeviceRegisteredAndTokenCached(ctx context.Context, ours *ourAPIClient, siteID string, inv solarmanDevice, known map[string]knownDevice) error {
	_, exists, err := ours.findDeviceSite(ctx, inv.SN)
	if err != nil {
		return err
	}
	if !exists {
		if err := ours.createDevice(ctx, newDeviceInput{
			DeviceID:     inv.SN,
			SiteID:       siteID,
			InstallNotes: "Auto-registered via solarman-sync (Solarman Open API)",
		}); err != nil {
			return fmt.Errorf("create device: %w", err)
		}
		log.Printf("solarman: auto-registered new device %s under site %s", inv.SN, siteID)
	}

	token, err := ours.issueCloudImportToken(ctx, inv.SN)
	if err != nil {
		return fmt.Errorf("issue cloud-import token: %w", err)
	}
	known[inv.SN] = knownDevice{siteID: siteID, token: token}
	return nil
}

func rawKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// cloudReading mirrors internal/httpapi's cloudReadingRequest — same
// shape pvpro-sync already submits, since both connectors feed the same
// vendor-agnostic endpoint.
type cloudReading struct {
	Timestamp       string   `json:"ts"`
	PowerKW         float64  `json:"power_kw"`
	EnergyKWhTotal  float64  `json:"energy_kwh_total"`
	Status          string   `json:"status"`
	PVPowerKW       *float64 `json:"pv_power_kw,omitempty"`
	BatterySOCPct   *float64 `json:"battery_soc_pct,omitempty"`
	BatteryVoltageV *float64 `json:"battery_voltage_v,omitempty"`
	LoadPowerKW     *float64 `json:"load_power_kw,omitempty"`
	GridPowerKW     *float64 `json:"grid_power_kw,omitempty"`
}

// buildReading maps Solarman's dataList key/value bag onto our schema.
// Key names (T_AC_OP, PVTP, Et_ge0, B_left_cap1, E_Puse_t1, PG_Pt1) are
// confirmed against a real, tested community client's sensor-key
// mapping — not guessed. Power keys are watts (divided by 1000 for our
// kW convention); Et_ge0 (cumulative solar production) is already kWh.
//
// status is always "ok", never derived from INV_ST1 — that field is
// diagnostic text with no documented fault-code mapping, same reasoning
// as pvpro-sync's inv.Status: guessing wrong feeds false alerts, which
// is worse than reporting nothing. A real Solarman fault/event-log
// endpoint, if one exists, is future work.
//
// PG_Pt1 (grid power) arrives positive-while-exporting per Solarman's
// own convention — inverted here to positive-while-importing, matching
// every other grid_power_kw producer on this platform (see pvpro-sync's
// buildReading and the Energy Flow widget's own sign-convention note).
//
// battery_voltage_v is deliberately left unset: pack voltage
// (V_BAP1) only appears on Solarman's separate "BATTERY" device-type
// response, which this connector doesn't fetch (see runOnce's comment on
// why only INVERTER devices are synced) — a real gap, not an oversight,
// flagged here rather than silently sending nothing and looking finished.
func buildReading(data solarmanCurrentData) cloudReading {
	ac, _ := data.value("T_AC_OP")
	energy, _ := data.value("Et_ge0")

	reading := cloudReading{
		Timestamp:      time.Unix(data.CollectionTime, 0).UTC().Format(time.RFC3339),
		PowerKW:        ac / 1000.0,
		EnergyKWhTotal: energy,
		Status:         "ok",
	}
	if pv, ok := data.value("PVTP"); ok {
		reading.PVPowerKW = floatPtr(pv / 1000.0)
	}
	if soc, ok := data.value("B_left_cap1"); ok {
		reading.BatterySOCPct = floatPtr(soc)
	}
	if load, ok := data.value("E_Puse_t1"); ok {
		reading.LoadPowerKW = floatPtr(load / 1000.0)
	}
	if grid, ok := data.value("PG_Pt1"); ok {
		reading.GridPowerKW = floatPtr(-grid / 1000.0)
	}
	return reading
}

func floatPtr(f float64) *float64 { return &f }
func floatOrZero(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func submitReading(ctx context.Context, token, apiBaseURL, deviceID string, reading cloudReading) error {
	body, err := json.Marshal(struct {
		Readings []cloudReading `json:"readings"`
	}{Readings: []cloudReading{reading}})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v1/cloud-import/%s/readings", apiBaseURL, deviceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("cloud-import returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}

func envSeconds(key string, def int) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return time.Duration(def) * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("invalid %s %q: %v", key, v, err)
	}
	return time.Duration(n) * time.Second
}
