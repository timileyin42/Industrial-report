package syncengine

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"
)

// FelicitySolar is a Provider for Felicity Solar's FSolar/Shine cloud
// platform — confirmed via market research as one of Nigeria's most
// common hybrid-inverter brands (alongside Deye). Built against
// Felicity's own officially documented partner API
// (github.com/smilebob/felicity_solar_hacs's flsOpenApi.md), not the
// FSolar consumer app's internal endpoints a community Home Assistant
// integration reverse-engineers by scraping the login page's JS bundle
// for a rotating RSA key — that path is fragile (breaks on every
// frontend redeploy) and unnecessary here, since the official API uses
// a fixed, versioned public key and needs no platform-level credential
// at all, unlike Deye's App ID/Secret.
type FelicitySolar struct {
	baseURL string
	client  *http.Client
}

// felicityPublicKeyBase64 is fixed and versioned in Felicity's own
// public API docs (last updated 2025-03-03) — unlike PV Pro's, it's
// not fetched per-login; every account encrypts against this same key.
const felicityPublicKeyBase64 = "MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAK0GDivaRzIKeTmQnAxAYh2LChuHWDp0yHZ0zIvm+Eoi7J+rx7phqR7EtkBDO3HWqAXVkNDeeQaU32P5w1Q4FVUCAwEAAQ=="

func NewFelicitySolar() *FelicitySolar {
	return &FelicitySolar{
		baseURL: "https://shine-api.felicitysolar.com",
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *FelicitySolar) Name() string        { return "felicity_solar" }
func (p *FelicitySolar) DisplayName() string { return "Felicity Solar" }
func (p *FelicitySolar) AuthType() string    { return AuthTypePassword }
func (p *FelicitySolar) Capabilities() Capabilities {
	return Capabilities{RealtimeData: true, BatteryData: true, GridData: true, LoadData: true}
}
func (p *FelicitySolar) DefaultPollInterval() time.Duration { return 60 * time.Second }

// session holds one login's token — created fresh per Discover/
// FetchReading call rather than cached on the Provider itself, same
// multi-tenant-safety reasoning as ELinterCSP's own session type: a
// single FelicitySolar instance is shared across every customer's
// connection in cmd/vendor-sync.
type felicitySession struct {
	baseURL string
	client  *http.Client
	// token already includes Felicity's own "Bearer_" prefix (confirmed
	// in their login response example) — sent as-is in Authorization,
	// never "Bearer " + token.
	token string
}

func (p *FelicitySolar) login(ctx context.Context, email, password string) (*felicitySession, error) {
	encryptedPassword, err := encryptFelicityPassword(password)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}
	reqBody, _ := json.Marshal(map[string]string{
		"userName": email,
		"password": encryptedPassword,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/openApi/sec/login", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// `data` changes shape with outcome — confirmed live, not from the
	// docs (whose own failure example shows this too, easy to miss):
	// on success it's {token, tokenExpireTime, ...}; on failure
	// (wrong password, etc.) it's a plain string ("Wrong password"),
	// not an object. Decoding straight into a fixed struct throws a
	// JSON type error on every failure instead of surfacing Felicity's
	// real message — caught by testing this against the real API with
	// a wrong password.
	var out struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Code != 200 {
		return nil, fmt.Errorf("login failed: %s", out.Message)
	}
	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(out.Data, &data); err != nil || data.Token == "" {
		return nil, fmt.Errorf("login response missing token: %s", out.Message)
	}
	return &felicitySession{baseURL: p.baseURL, client: p.client, token: data.Token}, nil
}

// encryptFelicityPassword mirrors the Java example in Felicity's own
// API docs: RSA/PKCS1v15 with their fixed public key, base64-encoded.
// The key ships as bare base64 DER (no PEM headers), unlike PV Pro's.
//
// Uses a hand-rolled PKCS1v15 encrypt (rsaEncryptPKCS1v15 below)
// instead of the standard library's rsa.EncryptPKCS1v15 — confirmed
// live against Felicity's real API that this published key really is
// only 512 bits, and Go 1.24+ refuses that call for any RSA key under
// 1024 bits. GODEBUG=rsa1024min=0 would suppress it, but that disables
// the same guard for every RSA operation in this whole binary, and
// Go's own docs recommend against setting it outside tests. The
// weakness that guard protects against is a *private* key being
// factorizable — that risk is Felicity's to own, not something our use
// of their already-public key for one outbound encrypt call increases.
func encryptFelicityPassword(password string) (string, error) {
	der, err := base64.StdEncoding.DecodeString(felicityPublicKeyBase64)
	if err != nil {
		return "", fmt.Errorf("decode felicity public key: %w", err)
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return "", err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("felicity public key is not RSA")
	}
	encrypted, err := rsaEncryptPKCS1v15(rsaPub, []byte(password))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// rsaEncryptPKCS1v15 implements RFC 8017 §7.2.1 directly — see
// encryptFelicityPassword's comment for why this doesn't just call
// crypto/rsa.EncryptPKCS1v15.
func rsaEncryptPKCS1v15(pub *rsa.PublicKey, msg []byte) ([]byte, error) {
	k := (pub.N.BitLen() + 7) / 8
	if len(msg) > k-11 {
		return nil, fmt.Errorf("message too long for %d-bit RSA key", pub.N.BitLen())
	}
	em := make([]byte, k)
	em[0] = 0x00
	em[1] = 0x02
	psEnd := k - len(msg) - 1
	if err := fillNonZeroRandom(em[2:psEnd]); err != nil {
		return nil, err
	}
	em[psEnd] = 0x00
	copy(em[psEnd+1:], msg)

	m := new(big.Int).SetBytes(em)
	c := new(big.Int).Exp(m, big.NewInt(int64(pub.E)), pub.N)
	out := make([]byte, k)
	c.FillBytes(out)
	return out, nil
}

// fillNonZeroRandom fills b with random bytes, none of them zero — the
// padding-string requirement for PKCS1v15 type-2 (encryption) padding.
func fillNonZeroRandom(b []byte) error {
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return err
	}
	for i, v := range b {
		for v == 0 {
			var single [1]byte
			if _, err := io.ReadFull(rand.Reader, single[:]); err != nil {
				return err
			}
			v = single[0]
		}
		b[i] = v
	}
	return nil
}

func (s *felicitySession) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", s.token)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// Discover lists every device on the account via the official
// devices/list endpoint. That endpoint's own response (confirmed via
// Felicity's docs) carries only deviceSn plus firmware-version fields —
// no plant/friendly name — so Name is left for the caller to fall back
// on (cmd/vendor-sync already falls back to the device's own serial
// when Name is empty, same as every other provider).
func (p *FelicitySolar) Discover(ctx context.Context, email, password string) ([]DiscoveredDevice, error) {
	sess, err := p.login(ctx, email, password)
	if err != nil {
		return nil, err
	}
	// data is a RawMessage first, then only unmarshaled into the real
	// list shape once code==200 — same "data's shape depends on
	// outcome" issue confirmed live in login(), applied defensively
	// here too rather than waiting to hit it again.
	var out struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := sess.getJSON(ctx, "/openApi/devices/list?pageNum=1&pageSize=100", &out); err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	if out.Code != 200 {
		return nil, fmt.Errorf("list devices: %s (code %d)", out.Message, out.Code)
	}
	var data struct {
		DataList []struct {
			DeviceSn string `json:"deviceSn"`
		} `json:"dataList"`
	}
	if err := json.Unmarshal(out.Data, &data); err != nil {
		return nil, fmt.Errorf("list devices: unexpected response shape: %w", err)
	}
	devices := make([]DiscoveredDevice, 0, len(data.DataList))
	for _, d := range data.DataList {
		if d.DeviceSn == "" {
			continue
		}
		devices = append(devices, DiscoveredDevice{ExternalRef: d.DeviceSn})
	}
	return devices, nil
}

// felicityDataPoint mirrors one record from deviceDataHistory — fields
// confirmed against Felicity's own official API docs (flsOpenApi.md).
// Pointers throughout: Felicity's own example responses show explicit
// JSON null for a metric a given device/model doesn't report (e.g. a
// grid-tie-only unit has no battery fields) — decoding into *float64
// preserves that "genuinely absent" signal instead of silently
// collapsing null to a fabricated 0, same discipline CLAUDE.md requires
// everywhere else in this pipeline.
type felicityDataPoint struct {
	DeviceDataTime string   `json:"deviceDataTime"`
	PvTotalPower   *float64 `json:"pvTotalPower"` // W — "Total PV Power (pv1+pv2+pv3+pv4)"
	TotalEnergy    *float64 `json:"totalEnergy"`  // kWh — "Total power generation" (lifetime)
	EmsSoc         *float64 `json:"emsSoc"`       // % — battery SOC
	EmsVoltage     *float64 `json:"emsVoltage"`   // V — battery voltage
	MeterPower     *float64 `json:"meterPower"`   // W — home load power
	AcTtlInpower   *float64 `json:"acTtlInpower"` // W — total grid power; negative = feeding/export
}

// FetchReading pulls today's most recent history record for one
// device — Felicity's API has no dedicated "latest" endpoint (see the
// docs' own endpoint list), so this queries deviceDataHistory for
// today's date at page size 1, which every reference implementation
// checked returns records most-recent-first.
func (p *FelicitySolar) FetchReading(ctx context.Context, email, password, externalRef string) (CloudReading, error) {
	sess, err := p.login(ctx, email, password)
	if err != nil {
		return CloudReading{}, err
	}
	today := time.Now().UTC().Format("2006-01-02")
	var out struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	path := fmt.Sprintf("/openApi/data/deviceDataHistory/%s?dateStr=%s&pageNum=1&pageSize=1", externalRef, today)
	if err := sess.getJSON(ctx, path, &out); err != nil {
		return CloudReading{}, fmt.Errorf("device history for %s: %w", externalRef, err)
	}
	if out.Code != 200 {
		return CloudReading{}, fmt.Errorf("device history for %s: %s (code %d)", externalRef, out.Message, out.Code)
	}
	var data struct {
		DataList []felicityDataPoint `json:"dataList"`
	}
	if err := json.Unmarshal(out.Data, &data); err != nil {
		return CloudReading{}, fmt.Errorf("device history for %s: unexpected response shape: %w", externalRef, err)
	}
	if len(data.DataList) == 0 {
		return CloudReading{}, fmt.Errorf("no readings returned for device %s today", externalRef)
	}
	d := data.DataList[0]
	// PowerKW/EnergyKWhTotal are required, non-pointer CloudReading
	// fields — a null here means Felicity has no current data for this
	// device at all (offline/never reported), which is worth failing
	// loudly on (MarkError) rather than silently submitting a
	// fabricated 0 that would look like a real reading.
	if d.PvTotalPower == nil || d.TotalEnergy == nil {
		return CloudReading{}, fmt.Errorf("device %s: no generation data reported today", externalRef)
	}

	reading := CloudReading{
		Timestamp:      d.DeviceDataTime,
		PowerKW:        *d.PvTotalPower / 1000.0,
		EnergyKWhTotal: *d.TotalEnergy,
		Status:         "ok",
	}
	if d.EmsSoc != nil {
		reading.BatterySOCPct = d.EmsSoc
	}
	if d.EmsVoltage != nil {
		reading.BatteryVoltageV = d.EmsVoltage
	}
	if d.MeterPower != nil {
		reading.LoadPowerKW = floatPtr(*d.MeterPower / 1000.0)
	}
	// Felicity's own docs: "negative value is the feeding power" for
	// acTtlInpower — same raw positive-import convention PV Pro's
	// gridOrMeterPower uses, so the same negation applies here to match
	// this app's canonical GridPowerKW sign (see elinter_csp.go).
	if d.AcTtlInpower != nil {
		reading.GridPowerKW = floatPtr(-*d.AcTtlInpower / 1000.0)
	}
	return reading, nil
}
