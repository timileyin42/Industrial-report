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
