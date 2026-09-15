package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ourAPIClient talks to this platform's own operator-only endpoints
// (device creation, cloud-import token issuance) — same shape as
// cmd/pvpro-sync's and cmd/solarman-sync's own copy of this file (this
// codebase's existing convention for a standalone sync binary: each
// one carries its own small copy rather than sharing a package, since
// none of them import each other). Unlike those two, vendor-sync never
// creates a site itself — every vendor_connections row already belongs
// to a site created at self-signup time — so this trims out
// siteExists/createSite entirely.
type ourAPIClient struct {
	baseURL    string
	email      string
	password   string
	httpClient *http.Client
	jwt        string
	jwtAt      time.Time
}

func newOurAPIClient(baseURL, email, password string) *ourAPIClient {
	return &ourAPIClient{
		baseURL:    baseURL,
		email:      email,
		password:   password,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *ourAPIClient) ensureAuth(ctx context.Context) error {
	if c.jwt != "" && time.Since(c.jwtAt) < 22*time.Hour {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"email": c.email, "password": c.password})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login to our API failed: %d %s", resp.StatusCode, string(b))
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	c.jwt = out.Token
	c.jwtAt = time.Now()
	return nil
}

func (c *ourAPIClient) doJSON(ctx context.Context, method, path string, body any, out any) (int, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return 0, err
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.jwt)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if out != nil && resp.StatusCode < 300 {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

// findDeviceSite mirrors cmd/pvpro-sync's own helper — used here only
// to check whether a discovered device is already registered, never to
// reconcile which site it belongs to (that's already fixed: the
// connection's own site_id).
func (c *ourAPIClient) findDeviceSite(ctx context.Context, deviceID string) (siteID string, exists bool, err error) {
	var out struct {
		SiteID string `json:"site_id"`
	}
	status, err := c.doJSON(ctx, http.MethodGet, "/v1/devices/"+deviceID, nil, &out)
	if err != nil {
		return "", false, err
	}
	if status == http.StatusNotFound {
		return "", false, nil
	}
	if status >= 300 {
		return "", false, fmt.Errorf("GET /v1/devices/%s: unexpected status %d", deviceID, status)
	}
	return out.SiteID, true, nil
}

type newDeviceInput struct {
	DeviceID      string `json:"device_id"`
	SiteID        string `json:"site_id"`
	InverterBrand string `json:"inverter_brand,omitempty"`
	InverterModel string `json:"inverter_model,omitempty"`
	InstallNotes  string `json:"install_notes,omitempty"`
}

func (c *ourAPIClient) createDevice(ctx context.Context, in newDeviceInput) error {
	status, err := c.doJSON(ctx, http.MethodPost, "/v1/devices", in, nil)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("create device %s: unexpected status %d", in.DeviceID, status)
	}
	return nil
}

func (c *ourAPIClient) issueCloudImportToken(ctx context.Context, deviceID string) (string, error) {
	var out struct {
		Token string `json:"token"`
	}
	status, err := c.doJSON(ctx, http.MethodPost, "/v1/devices/"+deviceID+"/cloud-import-token", nil, &out)
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("issue cloud-import token for %s: unexpected status %d", deviceID, status)
	}
	return out.Token, nil
}

// cloudReading mirrors internal/httpapi's cloudReadingRequest — the
// wire shape POST /v1/cloud-import/:device_id/readings expects.
// syncengine.CloudReading (what a Provider returns) has no JSON tags of
// its own since it's an internal Go contract, not something ever
// marshaled directly — this is the one place that maps it onto the
// wire.
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

func submitReading(ctx context.Context, httpClient *http.Client, token, apiBaseURL, deviceID string, reading cloudReading) error {
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

	resp, err := httpClient.Do(req)
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
