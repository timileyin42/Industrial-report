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
// (site/device creation, cloud-import token issuance) — needed so
// auto-discovery can reconcile what PV Pro reports against what's
// already registered, rather than blindly recreating everything on
// every restart. Logs in with the same operator credential a human
// would use (API_OPERATOR_EMAIL/PASSWORD), not a special service
// account — this platform has no separate "machine credential" concept
// yet, and adding one is more machinery than three devices justifies.
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
	// JWTs from this platform are issued for 24h (see auth.NewTokenIssuer)
	// — refreshed a couple hours early rather than reactively on 401, so
	// a mid-cycle expiry never costs a full poll's worth of failed calls.
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

// findDeviceSite returns the site_id an already-registered device
// belongs to, and whether it exists at all — the reconciliation check
// that keeps auto-discovery from ever creating a duplicate site for a
// plant that already has a manually or previously auto-registered
// device in it.
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

func (c *ourAPIClient) siteExists(ctx context.Context, siteID string) (bool, error) {
	status, err := c.doJSON(ctx, http.MethodGet, "/v1/sites/"+siteID, nil, nil)
	if err != nil {
		return false, err
	}
	if status == http.StatusNotFound {
		return false, nil
	}
	if status >= 300 {
		return false, fmt.Errorf("GET /v1/sites/%s: unexpected status %d", siteID, status)
	}
	return true, nil
}

type newSiteInput struct {
	SiteID       string  `json:"site_id"`
	Name         string  `json:"name"`
	Address      string  `json:"address,omitempty"`
	GPSLat       float64 `json:"gps_lat"`
	GPSLng       float64 `json:"gps_lng"`
	SystemSizeKW float64 `json:"system_size_kw"`
	Timezone     string  `json:"timezone"`
	Country      string  `json:"country"`
}

func (c *ourAPIClient) createSite(ctx context.Context, in newSiteInput) error {
	status, err := c.doJSON(ctx, http.MethodPost, "/v1/sites", in, nil)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("create site %s: unexpected status %d", in.SiteID, status)
	}
	return nil
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
