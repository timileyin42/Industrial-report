// Command pvpro-sync is the concrete cloud-import connector for Chisage's
// "PV Pro" app — reverse-engineered here, not officially documented by
// the vendor. PV Pro (package com.elinter.app.pvpro) is a white-labeled
// skin over Chengdu E-linter's shared "CSP" cloud platform, the same
// backend used under other brand names (Sunsynk Connect, Powerview for
// Sol-Ark). Community client libraries for those other brands (e.g.
// github.com/jamesridgway/sunsynk-api-client) document the login flow;
// pvpro_client.go ports that flow with source="pvpro" against
// pv.inteless.com, confirmed working against a real PV Pro account.
//
// This is deliberately a separate, isolated connector, not a change to
// the vendor-agnostic core (internal/registry/cloud_import.go stays
// vendor-blind) — it just authenticates to one specific vendor's cloud
// and forwards normalized readings to POST /v1/cloud-import/:device_id/
// readings like any other external source could.
//
// Auto-discovery: every cycle, this asks PV Pro for every plant/inverter
// on the account (not a fixed list) and reconciles against what's
// already registered on our platform (our_client.go) — a device that
// already exists keeps its existing site_id (so a plant that was
// manually registered under a hand-chosen site_id, like the first two
// test sites, is never duplicated under a second, auto-generated one); a
// genuinely new plant gets a fresh site (site_id "PVPRO-<plantID>")
// created with its precise lat/lon pulled from PV Pro's own per-plant
// detail endpoint, so a new site's location is correct from the moment
// it's created — no manual PATCH /v1/sites/:id/location needed
// afterward, unlike the first two sites this connector was proven
// against.
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
// device, so subsequent cycles don't re-issue a cloud-import token (safe
// to call again — IssueToken rotates — but pointless churn) or re-check
// existence with our API every 30 seconds.
type knownDevice struct {
	siteID     string
	inverterID int64
	token      string
}

func main() {
	pvproUsername := mustEnv("PVPRO_USERNAME")
	pvproPassword := mustEnv("PVPRO_PASSWORD")
	apiBaseURL := mustEnv("API_BASE_URL")
	operatorEmail := mustEnv("API_OPERATOR_EMAIL")
	operatorPassword := mustEnv("API_OPERATOR_PASSWORD")
	pollInterval := envSeconds("POLL_INTERVAL_SECONDS", 30)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pv := newPVProClient(pvproUsername, pvproPassword)
	ours := newOurAPIClient(apiBaseURL, operatorEmail, operatorPassword)
	known := map[string]knownDevice{} // our device_id -> cached info, rebuilt/extended each cycle

	log.Printf("pvpro-sync starting: auto-discovery mode, polling every %s", pollInterval)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	runOnce(ctx, pv, ours, known)
	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		case <-ticker.C:
			runOnce(ctx, pv, ours, known)
		}
	}
}

func runOnce(ctx context.Context, pv *pvproClient, ours *ourAPIClient, known map[string]knownDevice) {
	plants, err := pv.getPlants(ctx)
	if err != nil {
		log.Printf("pvpro: fetch plants: %v", err)
		return
	}

	type toSync struct {
		deviceID string
		inv      pvproInverter
	}
	var syncList []toSync

	for _, plant := range plants {
		inverters, err := pv.getInverters(ctx, plant.ID)
		if err != nil {
			log.Printf("pvpro: fetch inverters for plant %d (%s): %v", plant.ID, plant.Name, err)
			continue
		}
		if len(inverters) == 0 {
			continue
		}

		siteID, err := reconcileSite(ctx, pv, ours, plant, inverters, known)
		if err != nil {
			log.Printf("pvpro: reconcile site for plant %d (%s): %v", plant.ID, plant.Name, err)
			continue
		}

		for _, inv := range inverters {
			if inv.SN == "" {
				continue
			}
			if _, ok := known[inv.SN]; !ok {
				if err := ensureDeviceRegisteredAndTokenCached(ctx, ours, siteID, inv, known); err != nil {
					log.Printf("pvpro: ensure device %s registered: %v", inv.SN, err)
					continue
				}
			}
			syncList = append(syncList, toSync{deviceID: inv.SN, inv: inv})
		}
	}

	for i, item := range syncList {
		if i > 0 {
			time.Sleep(1500 * time.Millisecond) // spread out our own cloud-import calls, same rate class as any other public write endpoint
		}
		kd := known[item.deviceID]
		flow, err := pv.getFlow(ctx, kd.inverterID)
		if err != nil {
			log.Printf("pvpro: fetch flow for inverter %d (device %s): %v", kd.inverterID, item.deviceID, err)
			continue
		}
		reading := buildReading(item.inv, flow)
		if err := submitReading(ctx, kd.token, ours.baseURL, item.deviceID, reading); err != nil {
			log.Printf("pvpro: submit reading for device %s: %v", item.deviceID, err)
			continue
		}
		log.Printf("pvpro: synced device %s — %.2f kW AC, %.2f kW PV, ts=%s", item.deviceID, reading.PowerKW, floatOrZero(reading.PVPowerKW), reading.Timestamp)
	}
}

// reconcileSite decides which of our site_ids a plant's inverters belong
// to — reusing an existing one if ANY of the plant's inverters are
// already registered (never creating a second site for a plant we
// already know), or creating a genuinely new one (with precise lat/lon
// pulled fresh from PV Pro) only when none of them are.
func reconcileSite(ctx context.Context, pv *pvproClient, ours *ourAPIClient, plant pvproPlant, inverters []pvproInverter, known map[string]knownDevice) (string, error) {
	for _, inv := range inverters {
		if inv.SN == "" {
			continue
		}
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

	// None of this plant's inverters are registered anywhere yet — this
	// is a genuinely new plant. site_id is deterministic from PV Pro's
	// own plant ID, so re-running discovery never computes a different
	// id for the same plant.
	siteID := fmt.Sprintf("PVPRO-%d", plant.ID)
	alreadyExists, err := ours.siteExists(ctx, siteID)
	if err != nil {
		return "", err
	}
	if alreadyExists {
		return siteID, nil
	}

	detail, err := pv.getPlantDetail(ctx, plant.ID)
	if err != nil {
		return "", fmt.Errorf("fetch plant detail: %w", err)
	}
	timezone := detail.Timezone.Code
	if timezone == "" {
		timezone = "Africa/Lagos" // every known Chisage deployment for this connector is Nigeria
	}
	if err := ours.createSite(ctx, newSiteInput{
		SiteID:       siteID,
		Name:         plant.Name,
		Address:      plant.Address,
		GPSLat:       detail.Lat,
		GPSLng:       detail.Lon,
		SystemSizeKW: detail.Realtime.TotalPower,
		Timezone:     timezone,
		Country:      "NG", // this connector's only known deployment — see package comment
	}); err != nil {
		return "", fmt.Errorf("create site: %w", err)
	}
	log.Printf("pvpro: auto-registered new site %s (%q) at %.6f,%.6f", siteID, plant.Name, detail.Lat, detail.Lon)
	return siteID, nil
}

func ensureDeviceRegisteredAndTokenCached(ctx context.Context, ours *ourAPIClient, siteID string, inv pvproInverter, known map[string]knownDevice) error {
	_, exists, err := ours.findDeviceSite(ctx, inv.SN)
	if err != nil {
		return err
	}
	if !exists {
		if err := ours.createDevice(ctx, newDeviceInput{
			DeviceID:      inv.SN,
			SiteID:        siteID,
			InverterBrand: "chisage",
			InverterModel: inv.Model,
			InstallNotes:  "Auto-registered via pvpro-sync (PV Pro cloud import)",
		}); err != nil {
			return fmt.Errorf("create device: %w", err)
		}
		log.Printf("pvpro: auto-registered new device %s (model %s) under site %s", inv.SN, inv.Model, siteID)
	}

	token, err := ours.issueCloudImportToken(ctx, inv.SN)
	if err != nil {
		return fmt.Errorf("issue cloud-import token: %w", err)
	}
	known[inv.SN] = knownDevice{siteID: siteID, inverterID: inv.ID, token: token}
	return nil
}

// cloudReading mirrors internal/httpapi's cloudReadingRequest — this
// connector is just one more caller of the same public endpoint every
// external source uses.
type cloudReading struct {
	Timestamp       string   `json:"ts"`
	PowerKW         float64  `json:"power_kw"`
	EnergyKWhTotal  float64  `json:"energy_kwh_total"`
	Status          string   `json:"status"`
	PVPowerKW       *float64 `json:"pv_power_kw,omitempty"`
	BatterySOCPct   *float64 `json:"battery_soc_pct,omitempty"`
	BatteryVoltageV *float64 `json:"battery_voltage_v,omitempty"`
}

// buildReading maps PV Pro's field names onto ours. Power fields
// (pac/pvPower) are in watts per every community client for this same
// backend family — divided by 1000 for our kW convention. etotal/soc/
// battV need no conversion (already kWh / percent / volts). power_kw
// comes from the inverter list's own pac field (confirmed per-inverter,
// not plant-aggregated) rather than the flow endpoint, which only
// exposes pvPower/soc/battV — no direct AC-output figure.
//
// status is always "ok" — deliberately NOT derived from inv.Status.
// That field is a coarse numeric code whose meanings aren't documented
// and don't match the naive guess "1 = fine, anything else = fault":
// status 2 was observed on a perfectly healthy, "Normal"-badged inverter
// simply idle at night. Since our own status feeds the Alerts page's
// fault detection, guessing wrong here would raise false fault alerts —
// worse than reporting nothing. PV Pro's own Event log (F02/W27/F03
// style codes, not this status field) is the real fault signal; wiring
// that up is future work, not something to fake with an unverified
// guess.
func buildReading(inv pvproInverter, flow pvproFlow) cloudReading {
	return cloudReading{
		Timestamp:       inv.UpdateAt,
		PowerKW:         inv.Pac / 1000.0,
		EnergyKWhTotal:  inv.Etotal,
		Status:          "ok",
		PVPowerKW:       floatPtr(flow.PVPower / 1000.0),
		BatterySOCPct:   floatPtr(flow.SOC),
		BatteryVoltageV: floatPtr(flow.BattV),
	}
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
