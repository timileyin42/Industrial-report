package syncengine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// DeyeCloud is a Provider for Deye's official Cloud OpenAPI — confirmed
// via market research as Nigeria's single most popular hybrid-inverter
// brand. Unlike ELinterCSP/FelicitySolar, it needs a platform-level
// credential of our own (DEYE_APP_ID/DEYE_APP_SECRET, from
// developer.deyecloud.com/app) alongside each customer's own account
// email/password — every customer still just types their normal Deye
// account login through the Connect Your Inverter form; the app
// credential is what lets *us* talk to Deye's API at all, never
// something a customer sees or enters.
//
// Reads per-device measure-point data via /device/latest rather than
// the coarser /station/latest aggregate: station/latest's fields
// (confirmed across several community clients) cover instantaneous
// power but expose no genuine lifetime energy total, only
// today/period figures. /device/latest's PVCumulativePowerGenerationActive
// measure point does — confirmed against a real Deye hybrid inverter's
// own field dump (a community Prometheus exporter's cloud_parameters.py
// documents exactly that verification), not inferred from
// documentation alone.
type DeyeCloud struct {
	baseURL   string
	appID     string
	appSecret string
	client    *http.Client
}

// NewDeyeCloud requires the app-level credential from Deye's developer
// portal — appID/appSecret are ours, not any customer's. baseURL is
// fixed to the EU/Africa/APAC data center (Deye's own regional
// routing places Nigeria's traffic here, not the US or India
// endpoints — see developer.deyecloud.com's region table).
func NewDeyeCloud(appID, appSecret string) *DeyeCloud {
	return &DeyeCloud{
		baseURL:   "https://eu1-developer.deyecloud.com/v1.0",
		appID:     appID,
		appSecret: appSecret,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *DeyeCloud) Name() string        { return "deye_cloud" }
func (p *DeyeCloud) DisplayName() string { return "Deye Inverter" }
func (p *DeyeCloud) AuthType() string    { return AuthTypePassword }
func (p *DeyeCloud) Capabilities() Capabilities {
	return Capabilities{RealtimeData: true, BatteryData: true, GridData: true, LoadData: true}
}
func (p *DeyeCloud) DefaultPollInterval() time.Duration { return 60 * time.Second }

// session holds one login's token — created fresh per Discover/
// FetchReading call rather than cached on the Provider itself, same
// multi-tenant-safety reasoning as every other adapter in this
// package: a single DeyeCloud instance is shared across every
// customer's connection in cmd/vendor-sync.
type deyeSession struct {
	baseURL     string
	client      *http.Client
	accessToken string
}

// login exchanges the app-level credential plus one customer's own
// account email/password for a bearer token. Deye expects the password
// as a SHA-256 hex digest, not plaintext or RSA-encrypted like PV Pro/
// Felicity — confirmed against Deye's own official sample code
// (DeyeCloudDevelopers/deye-openapi-client-sample-code).
func (p *DeyeCloud) login(ctx context.Context, email, password string) (*deyeSession, error) {
	sum := sha256.Sum256([]byte(password))
	reqBody, _ := json.Marshal(map[string]string{
		"appSecret": p.appSecret,
		"email":     email,
		"password":  hex.EncodeToString(sum[:]),
		"companyId": "0", // "0" = personal account, per Deye's own docs
	})
	url := fmt.Sprintf("%s/account/token?appId=%s", p.baseURL, p.appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Deye's token endpoint uses two different envelopes depending on
	// outcome — confirmed live, not from the docs (which only show the
	// success shape): success is {success, msg, accessToken, ...}, but
	// a rejected login (e.g. wrong password) comes back as a standard
	// OAuth2-style error body, {error, error_description, code, param},
	// with `success`/`msg` simply absent. Decoding only the success
	// shape left a real login failure's actual reason silently dropped
	// (out.Msg stayed "" since that key was never present) — caught by
	// testing this against the real API with a wrong password, not
	// found by reading Deye's docs alone.
	var out struct {
		Success          bool   `json:"success"`
		Msg              string `json:"msg"`
		AccessToken      string `json:"accessToken"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.AccessToken == "" {
		msg := out.Msg
		if msg == "" {
			msg = out.Error
		}
		if msg == "" {
			msg = out.ErrorDescription
		}
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		if looksLikeCredentialError(msg) {
			return nil, fmt.Errorf("%w: login failed: %s", ErrInvalidCredentials, msg)
		}
		return nil, fmt.Errorf("login failed: %s", msg)
	}
	return &deyeSession{baseURL: p.baseURL, client: p.client, accessToken: out.AccessToken}, nil
}

func (s *deyeSession) postJSON(ctx context.Context, path string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// Discover lists every inverter across every station (plant) on this
// account via station/listWithDevice — one call, and one that
// sidesteps a documented real-world gotcha: Deye's own /device/list
// has been observed to reject an otherwise-valid token with code
// 2101019 on personal (non-organization) accounts, while
// station/listWithDevice does not.
func (p *DeyeCloud) Discover(ctx context.Context, email, password string) ([]DiscoveredDevice, error) {
	sess, err := p.login(ctx, email, password)
	if err != nil {
		return nil, err
	}
	var out struct {
		Success     bool   `json:"success"`
		Msg         string `json:"msg"`
		StationList []struct {
			Name            string `json:"name"`
			DeviceListItems []struct {
				DeviceSn   string `json:"deviceSn"`
				DeviceType string `json:"deviceType"`
			} `json:"deviceListItems"`
		} `json:"stationList"`
	}
	body := map[string]any{"page": 1, "size": 100, "deviceType": "INVERTER"}
	if err := sess.postJSON(ctx, "/station/listWithDevice", body, &out); err != nil {
		return nil, fmt.Errorf("list stations: %w", err)
	}
	if !out.Success {
		return nil, fmt.Errorf("list stations: %s", out.Msg)
	}
	var devices []DiscoveredDevice
	for _, station := range out.StationList {
		for _, dev := range station.DeviceListItems {
			if dev.DeviceSn == "" {
				continue
			}
			devices = append(devices, DiscoveredDevice{ExternalRef: dev.DeviceSn, Name: station.Name})
		}
	}
	return devices, nil
}

// deyeMeasurePoint is one {key, value} pair from /device/latest's
// dataList — Deye's API is measure-point-keyed rather than fixed
// named fields; a device only reports the points its own model/
// firmware actually supports.
type deyeMeasurePoint struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// FetchReading maps a small, verified subset of Deye's ~60 measure
// points onto CloudReading. Every key name below was confirmed against
// a real Deye hybrid inverter's own /device/latest dump (see this
// type's own doc comment) — not guessed from documentation alone, the
// same standard this project holds every other connector to. alt keys
// are tried when the primary key is absent, covering firmware/model
// variation the reference project also had to account for.
func (p *DeyeCloud) FetchReading(ctx context.Context, email, password, externalRef string) (CloudReading, error) {
	sess, err := p.login(ctx, email, password)
	if err != nil {
		return CloudReading{}, err
	}
	var out struct {
		Success        bool   `json:"success"`
		Msg            string `json:"msg"`
		DeviceDataList []struct {
			DeviceSn       string             `json:"deviceSn"`
			CollectionTime int64              `json:"collectionTime"`
			DataList       []deyeMeasurePoint `json:"dataList"`
		} `json:"deviceDataList"`
	}
	if err := sess.postJSON(ctx, "/device/latest", map[string]any{"deviceList": []string{externalRef}}, &out); err != nil {
		return CloudReading{}, fmt.Errorf("device latest for %s: %w", externalRef, err)
	}
	if !out.Success {
		return CloudReading{}, fmt.Errorf("device latest for %s: %s", externalRef, out.Msg)
	}
	if len(out.DeviceDataList) == 0 {
		return CloudReading{}, fmt.Errorf("no data returned for device %s", externalRef)
	}

	device := out.DeviceDataList[0]
	values := make(map[string]float64, len(device.DataList))
	for _, point := range device.DataList {
		if f, err := strconv.ParseFloat(point.Value, 64); err == nil {
			values[point.Key] = f
		}
	}
	lookup := func(keys ...string) (float64, bool) {
		for _, k := range keys {
			if v, ok := values[k]; ok {
				return v, true
			}
		}
		return 0, false
	}

	// PowerKW/EnergyKWhTotal are required, non-pointer CloudReading
	// fields — a missing measure point here means Deye has nothing
	// current for this device (likely offline), worth failing loudly
	// on (MarkError) rather than submitting a fabricated 0.
	powerW, ok := lookup("InverterOutputPowerL1L2")
	if !ok {
		return CloudReading{}, fmt.Errorf("device %s: no InverterOutputPowerL1L2 in response — inverter may be offline", externalRef)
	}
	energyTotal, ok := lookup("PVCumulativePowerGenerationActive", "TotalActiveProduction")
	if !ok {
		return CloudReading{}, fmt.Errorf("device %s: no lifetime generation total in response", externalRef)
	}

	reading := CloudReading{
		Timestamp:      time.Unix(device.CollectionTime, 0).UTC().Format(time.RFC3339),
		PowerKW:        powerW / 1000.0,
		EnergyKWhTotal: energyTotal,
		Status:         "ok",
	}
	if v, ok := lookup("SOC"); ok {
		reading.BatterySOCPct = floatPtr(v)
	}
	if v, ok := lookup("BatteryVoltage"); ok {
		reading.BatteryVoltageV = floatPtr(v)
	}
	if v, ok := lookup("TotalConsumptionPower", "UPSLoadPower"); ok {
		reading.LoadPowerKW = floatPtr(v / 1000.0)
	}
	// Deye's own docs: "positive = importing" for TotalGridPower — same
	// raw convention PV Pro's gridOrMeterPower uses, so the same
	// negation applies here to match this app's canonical GridPowerKW
	// sign (see elinter_csp.go).
	if v, ok := lookup("TotalGridPower"); ok {
		reading.GridPowerKW = floatPtr(-v / 1000.0)
	}
	return reading, nil
}
