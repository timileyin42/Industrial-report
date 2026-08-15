// Command pvpro-sync is the concrete cloud-import connector for Chisage's
// "PV Pro" app — reverse-engineered here, not officially documented by
// the vendor. PV Pro (package com.elinter.app.pvpro) is a white-labeled
// skin over Chengdu E-linter's shared "CSP" cloud platform, the same
// backend used under other brand names (Sunsynk Connect, Powerview for
// Sol-Ark). Community client libraries for those other brands (e.g.
// github.com/jamesridgway/sunsynk-api-client) document the login flow;
// this file ports that flow with source="pvpro" against pv.inteless.com,
// confirmed working against a real PV Pro account.
//
// This is deliberately a separate, isolated connector, not a change to
// the vendor-agnostic core (internal/registry/cloud_import.go stays
// vendor-blind) — it just authenticates to one specific vendor's cloud
// and forwards normalized readings to POST /v1/cloud-import/:device_id/
// readings like any other external source could. Config (which PV Pro
// plant/inverter maps to which of our device IDs, and that device's
// cloud-import token) is a small JSON blob via PVPRO_SYNC_CONFIG, kept
// out of the core schema on purpose — see internal/registry/cloud_import.go's
// package comment for why.
package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
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

const pvproSource = "pvpro"

// deviceConfig maps one of our own device IDs to the PV Pro plant/inverter
// it should be synced from, plus the cloud-import bearer token issued for
// that device (see POST /v1/devices/:device_id/cloud-import-token).
type deviceConfig struct {
	DeviceID         string `json:"device_id"`
	PlantID          int64  `json:"plant_id"`
	InverterID       int64  `json:"inverter_id"`
	CloudImportToken string `json:"cloud_import_token"`
}

func main() {
	username := mustEnv("PVPRO_USERNAME")
	password := mustEnv("PVPRO_PASSWORD")
	apiBaseURL := mustEnv("API_BASE_URL")
	pollInterval := envSeconds("POLL_INTERVAL_SECONDS", 30)

	var configs []deviceConfig
	if err := json.Unmarshal([]byte(mustEnv("PVPRO_SYNC_CONFIG")), &configs); err != nil {
		log.Fatalf("invalid PVPRO_SYNC_CONFIG: %v", err)
	}
	if len(configs) == 0 {
		log.Fatal("PVPRO_SYNC_CONFIG has no devices configured")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pv := newPVProClient(username, password)

	log.Printf("pvpro-sync starting: %d device(s), polling every %s", len(configs), pollInterval)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	runOnce(ctx, pv, apiBaseURL, configs)
	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		case <-ticker.C:
			runOnce(ctx, pv, apiBaseURL, configs)
		}
	}
}

// runOnce fetches every configured plant's inverter list once (not once
// per device — several devices can share a plant, like the two at
// Promise Lodge), then per device merges in the flow endpoint's
// battery/PV detail and forwards a single reading to our own
// cloud-import endpoint.
func runOnce(ctx context.Context, pv *pvproClient, apiBaseURL string, configs []deviceConfig) {
	plantIDs := map[int64]bool{}
	for _, c := range configs {
		plantIDs[c.PlantID] = true
	}

	inverters := map[int64]pvproInverter{} // inverter id -> its record
	for plantID := range plantIDs {
		list, err := pv.getInverters(ctx, plantID)
		if err != nil {
			log.Printf("pvpro: fetch inverters for plant %d: %v", plantID, err)
			continue
		}
		for _, inv := range list {
			inverters[inv.ID] = inv
		}
	}

	for i, c := range configs {
		if i > 0 {
			time.Sleep(1500 * time.Millisecond) // spread out our own cloud-import calls, same rate class as any other public write endpoint
		}
		inv, ok := inverters[c.InverterID]
		if !ok {
			log.Printf("pvpro: inverter %d (device %s) not found in plant %d's inverter list", c.InverterID, c.DeviceID, c.PlantID)
			continue
		}
		flow, err := pv.getFlow(ctx, c.InverterID)
		if err != nil {
			log.Printf("pvpro: fetch flow for inverter %d (device %s): %v", c.InverterID, c.DeviceID, err)
			continue
		}

		reading := buildReading(inv, flow)
		if err := submitReading(ctx, apiBaseURL, c.DeviceID, c.CloudImportToken, reading); err != nil {
			log.Printf("pvpro: submit reading for device %s: %v", c.DeviceID, err)
			continue
		}
		log.Printf("pvpro: synced device %s — %.2f kW AC, %.2f kW PV, ts=%s", c.DeviceID, reading.PowerKW, floatOrZero(reading.PVPowerKW), reading.Timestamp)
	}
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
func buildReading(inv pvproInverter, flow pvproFlow) cloudReading {
	status := "ok"
	if inv.Status != 1 {
		status = "fault"
	}
	return cloudReading{
		Timestamp:       inv.UpdateAt,
		PowerKW:         inv.Pac / 1000.0,
		EnergyKWhTotal:  inv.Etotal,
		Status:          status,
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

func submitReading(ctx context.Context, apiBaseURL, deviceID, token string, reading cloudReading) error {
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

// --- PV Pro / E-linter CSP client ---

type pvproClient struct {
	baseURL        string
	username       string
	password       string
	httpClient     *http.Client
	accessToken    string
	tokenExpiresAt time.Time
}

func newPVProClient(username, password string) *pvproClient {
	return &pvproClient{
		baseURL:    "https://pv.inteless.com",
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type pvproInverter struct {
	ID       int64   `json:"id"`
	SN       string  `json:"sn"`
	Status   int     `json:"status"`
	Pac      float64 `json:"pac"`
	Etotal   float64 `json:"etotal"`
	UpdateAt string  `json:"updateAt"`
}

type pvproFlow struct {
	PVPower float64 `json:"pvPower"`
	SOC     float64 `json:"soc"`
	BattV   float64 `json:"battV"`
}

func (c *pvproClient) ensureToken(ctx context.Context) error {
	if c.accessToken != "" && time.Now().Before(c.tokenExpiresAt) {
		return nil
	}
	return c.login(ctx)
}

func (c *pvproClient) login(ctx context.Context) error {
	publicKey, err := c.fetchPublicKey(ctx)
	if err != nil {
		return fmt.Errorf("fetch public key: %w", err)
	}
	encryptedPassword, err := encryptPassword(publicKey, c.password)
	if err != nil {
		return fmt.Errorf("encrypt password: %w", err)
	}

	nonce := time.Now().UnixMilli()
	first10 := publicKey
	if len(first10) > 10 {
		first10 = first10[:10]
	}
	sign := md5Hex(fmt.Sprintf("nonce=%d&source=%s%s", nonce, pvproSource, first10))

	reqBody, _ := json.Marshal(map[string]any{
		"sign":       sign,
		"nonce":      nonce,
		"username":   c.username,
		"password":   encryptedPassword,
		"grant_type": "password",
		"client_id":  "csp-web",
		"source":     pvproSource,
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/oauth/token/new", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}
	if !tokenResp.Success {
		return fmt.Errorf("login failed: %s", tokenResp.Msg)
	}
	c.accessToken = tokenResp.Data.AccessToken
	// 60s safety buffer, same margin the community client for this same backend uses.
	c.tokenExpiresAt = time.Now().Add(time.Duration(tokenResp.Data.ExpiresIn)*time.Second - 60*time.Second)
	return nil
}

func (c *pvproClient) fetchPublicKey(ctx context.Context) (string, error) {
	nonce := time.Now().UnixMilli()
	sign := md5Hex(fmt.Sprintf("nonce=%d&source=%sPOWER_VIEW", nonce, pvproSource))

	url := fmt.Sprintf("%s/anonymous/publicKey?source=%s&nonce=%d&sign=%s", c.baseURL, pvproSource, nonce, sign)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var pkResp struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pkResp); err != nil {
		return "", err
	}
	if pkResp.Data == "" {
		return "", fmt.Errorf("empty public key in response")
	}
	return pkResp.Data, nil
}

func encryptPassword(publicKeyBase64, password string) (string, error) {
	pemStr := "-----BEGIN PUBLIC KEY-----\n" + publicKeyBase64 + "\n-----END PUBLIC KEY-----"
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("public key is not RSA")
	}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, []byte(password))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func (c *pvproClient) getInverters(ctx context.Context, plantID int64) ([]pvproInverter, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/v1/plant/%d/inverters?page=1&limit=50&status=-1&sn=&id=%d&type=-2", c.baseURL, plantID, plantID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Data struct {
			Infos []pvproInverter `json:"infos"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Data.Infos, nil
}

func (c *pvproClient) getFlow(ctx context.Context, inverterID int64) (pvproFlow, error) {
	if err := c.ensureToken(ctx); err != nil {
		return pvproFlow{}, err
	}
	url := fmt.Sprintf("%s/api/v1/inverter/%d/flow", c.baseURL, inverterID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return pvproFlow{}, err
	}
	defer resp.Body.Close()

	var out struct {
		Data pvproFlow `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return pvproFlow{}, err
	}
	return out.Data, nil
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
