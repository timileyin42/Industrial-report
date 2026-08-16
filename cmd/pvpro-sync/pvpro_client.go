package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"context"
)

const pvproSource = "pvpro"

// --- PV Pro / E-linter CSP client ---
//
// Reverse-engineered — see main.go's package comment. Login flow ported
// from community client libraries for other brands on this same shared
// backend (e.g. github.com/jamesridgway/sunsynk-api-client), with
// source="pvpro" swapped in; confirmed working against a real PV Pro
// account.

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

type pvproPlant struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type pvproPlantDetail struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Timezone struct {
		Code string `json:"code"`
	} `json:"timezone"`
	Realtime struct {
		TotalPower float64 `json:"totalPower"`
	} `json:"realtime"`
}

type pvproInverter struct {
	ID       int64   `json:"id"`
	SN       string  `json:"sn"`
	Model    string  `json:"model"`
	Status   int     `json:"status"`
	Pac      float64 `json:"pac"`
	Etotal   float64 `json:"etotal"`
	UpdateAt string  `json:"updateAt"`
}

type pvproFlow struct {
	PVPower float64 `json:"pvPower"`
	SOC     float64 `json:"soc"`
	BattV   float64 `json:"battV"`
	// ExistsBattery: some inverters (e.g. grid-tie-only models) genuinely
	// have no battery — PV Pro still returns soc/battV as 0 in that case,
	// not null, so this flag is the only way to tell "no battery" apart
	// from "battery reads 0 right now." Must be checked before treating
	// SOC/BattV as real readings (see buildReading in main.go).
	ExistsBattery bool `json:"existsBattery"`
	// BatteryFlowDatas: when a site has more than one physical battery
	// pack (batteryNum > 1), the top-level BattV above is 0 — the real
	// per-pack voltage only shows up here. A single-pack site has BattV
	// populated directly and doesn't need this fallback.
	BatteryFlowDatas []struct {
		Voltage float64 `json:"voltage"`
	} `json:"batteryFlowDatas"`
}

// BatteryVoltage returns the best available battery voltage reading:
// the top-level field when populated (single-pack sites), else the
// average of whatever per-pack voltages are present (multi-pack
// sites, where the top-level field is always 0).
func (f pvproFlow) BatteryVoltage() (float64, bool) {
	if f.BattV != 0 {
		return f.BattV, true
	}
	var sum float64
	var n int
	for _, pack := range f.BatteryFlowDatas {
		if pack.Voltage == 0 {
			continue
		}
		sum += pack.Voltage
		n++
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
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

// getPlants lists every plant visible to this account — the basis for
// auto-discovery. PV Pro's summary fields here only carry a coarse
// province/country-level location (not used); getPlantDetail below is
// what has the precise per-plant lat/lon.
func (c *pvproClient) getPlants(ctx context.Context) ([]pvproPlant, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/plants?page=1&limit=100", nil)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Data struct {
			Infos []pvproPlant `json:"infos"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Data.Infos, nil
}

// getPlantDetail is the only endpoint observed to carry a plant's
// precise GPS coordinates (the summary/list endpoints only have
// province/country-level location) — used once, when auto-registering a
// genuinely new plant, so its site record gets a correct location at
// creation time rather than needing a manual fix afterward.
func (c *pvproClient) getPlantDetail(ctx context.Context, plantID int64) (pvproPlantDetail, error) {
	if err := c.ensureToken(ctx); err != nil {
		return pvproPlantDetail{}, err
	}
	url := fmt.Sprintf("%s/api/v1/plant/%d?lan=en&id=%d", c.baseURL, plantID, plantID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return pvproPlantDetail{}, err
	}
	defer resp.Body.Close()

	var out struct {
		Data pvproPlantDetail `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return pvproPlantDetail{}, err
	}
	return out.Data, nil
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

// getFlow is keyed by the inverter's serial number, not its internal
// numeric id — /api/v1/inverter/{id}/flow (id-keyed) always reports
// existsBattery=false and 0 for every power field regardless of the
// inverter's real state; only the SN-keyed path returns real data. This
// endpoint also returns HTTP 200 with a body-level {"code":N,"msg":...}
// on failure (e.g. code 2 "No Permissions"), so success must be
// checked from the body, not the status code.
func (c *pvproClient) getFlow(ctx context.Context, sn string) (pvproFlow, error) {
	if err := c.ensureToken(ctx); err != nil {
		return pvproFlow{}, err
	}
	url := fmt.Sprintf("%s/api/v1/inverter/%s/flow", c.baseURL, sn)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return pvproFlow{}, err
	}
	defer resp.Body.Close()

	var out struct {
		Code    int       `json:"code"`
		Msg     string    `json:"msg"`
		Success bool      `json:"success"`
		Data    pvproFlow `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return pvproFlow{}, err
	}
	if out.Code != 0 || !out.Success {
		return pvproFlow{}, fmt.Errorf("flow for %s: %s (code %d)", sn, out.Msg, out.Code)
	}
	return out.Data, nil
}

// getPVVoltage reads the panel-string DC voltage(s) — a separate
// endpoint from getFlow, which doesn't carry this field. A hybrid
// inverter can have multiple independent MPPT strings; this averages
// whatever strings are reporting rather than picking just the first,
// since the UI shows one "Solar (PV), V" figure per device. Returns
// ok=false (no error) when the inverter has no PV strings to report,
// same "genuinely absent, not a fetch failure" distinction as
// pvproFlow.ExistsBattery.
func (c *pvproClient) getPVVoltage(ctx context.Context, sn string) (voltage float64, ok bool, err error) {
	if err := c.ensureToken(ctx); err != nil {
		return 0, false, err
	}
	url := fmt.Sprintf("%s/api/v1/inverter/%s/realtime/input", c.baseURL, sn)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			PVIV []struct {
				Vpv string `json:"vpv"`
			} `json:"pvIV"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, false, err
	}
	if out.Code != 0 {
		return 0, false, fmt.Errorf("realtime input for %s: %s (code %d)", sn, out.Msg, out.Code)
	}
	var sum float64
	var n int
	for _, s := range out.Data.PVIV {
		v, err := strconv.ParseFloat(s.Vpv, 64)
		if err != nil {
			continue
		}
		sum += v
		n++
	}
	if n == 0 {
		return 0, false, nil
	}
	return sum / float64(n), true, nil
}

// getOutputVoltage reads the inverter's AC output voltage(s) — a
// hybrid inverter can be single or multi-phase; this averages
// whatever phases are reporting, same rationale as getPVVoltage.
func (c *pvproClient) getOutputVoltage(ctx context.Context, sn string) (voltage float64, ok bool, err error) {
	if err := c.ensureToken(ctx); err != nil {
		return 0, false, err
	}
	url := fmt.Sprintf("%s/api/v1/inverter/%s/realtime/output", c.baseURL, sn)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()

	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			VIP []struct {
				Volt string `json:"volt"`
			} `json:"vip"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, false, err
	}
	if out.Code != 0 {
		return 0, false, fmt.Errorf("realtime output for %s: %s (code %d)", sn, out.Msg, out.Code)
	}
	var sum float64
	var n int
	for _, s := range out.Data.VIP {
		v, err := strconv.ParseFloat(s.Volt, 64)
		if err != nil {
			continue
		}
		sum += v
		n++
	}
	if n == 0 {
		return 0, false, nil
	}
	return sum / float64(n), true, nil
}
