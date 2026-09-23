package syncengine

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
	"net/http"
	"time"
)

// ELinterCSP is the reference Provider implementation — Chengdu
// E-linter's shared "CSP" cloud backend, white-labeled as PV Pro,
// Sunsynk Connect, and Powerview. Login/discovery/flow logic here is
// adapted from cmd/pvpro-sync/pvpro_client.go, which has been verified
// working against a real account throughout this project — that
// standalone connector (Luther's own single shared account) is
// untouched; this is a second, independent, multi-tenant-capable
// adapter built on the same proven request shapes for customers who
// self-connect their own separate accounts.
//
// Chosen as the reference implementation specifically because it's
// already proven and password-form (matching the agreed low-friction
// UX), unlike an OAuth-only vendor that would need business approval
// from the vendor first, or a token-generation flow that's real
// friction for a non-technical customer — see the plan file for the
// full reasoning.
type ELinterCSP struct {
	baseURL string
	source  string
	client  *http.Client
}

// NewELinterCSP defaults source="pvpro" — the brand this project has
// verified end-to-end. Sunsynk/Powerview accounts share the same
// backend under a different source value; adding them (if a customer
// turns out to use one of those apps instead) is a follow-up, not a
// rewrite of this type.
func NewELinterCSP() *ELinterCSP {
	return &ELinterCSP{
		baseURL: "https://pv.inteless.com",
		source:  "pvpro",
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *ELinterCSP) Name() string        { return "elinter_csp" }
func (p *ELinterCSP) DisplayName() string { return "PV Pro / Chisage" }
func (p *ELinterCSP) AuthType() string    { return AuthTypePassword }
func (p *ELinterCSP) Capabilities() Capabilities {
	return Capabilities{RealtimeData: true, BatteryData: true, GridData: true, LoadData: true}
}

// DefaultPollInterval matches cmd/pvpro-sync's own proven cadence for
// this backend.
func (p *ELinterCSP) DefaultPollInterval() time.Duration { return 30 * time.Second }

// session holds one login's token — created fresh per Discover/
// FetchReading call rather than cached on the Provider itself, since a
// single ELinterCSP instance is shared across every customer's
// connection in cmd/vendor-sync; caching a token on the struct would
// leak one customer's session into another's request.
type elinterSession struct {
	baseURL     string
	client      *http.Client
	accessToken string
}

func (p *ELinterCSP) login(ctx context.Context, email, password string) (*elinterSession, error) {
	publicKey, err := p.fetchPublicKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch public key: %w", err)
	}
	encryptedPassword, err := encryptElinterPassword(publicKey, password)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}

	nonce := time.Now().UnixMilli()
	first10 := publicKey
	if len(first10) > 10 {
		first10 = first10[:10]
	}
	sign := md5Hex(fmt.Sprintf("nonce=%d&source=%s%s", nonce, p.source, first10))

	reqBody, _ := json.Marshal(map[string]any{
		"sign":       sign,
		"nonce":      nonce,
		"username":   email,
		"password":   encryptedPassword,
		"grant_type": "password",
		"client_id":  "csp-web",
		"source":     p.source,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/oauth/token/new", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Data    struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.Success {
		if looksLikeCredentialError(out.Msg) {
			return nil, fmt.Errorf("%w: login failed: %s", ErrInvalidCredentials, out.Msg)
		}
		return nil, fmt.Errorf("login failed: %s", out.Msg)
	}
	return &elinterSession{baseURL: p.baseURL, client: p.client, accessToken: out.Data.AccessToken}, nil
}

func (p *ELinterCSP) fetchPublicKey(ctx context.Context) (string, error) {
	nonce := time.Now().UnixMilli()
	sign := md5Hex(fmt.Sprintf("nonce=%d&source=%sPOWER_VIEW", nonce, p.source))
	url := fmt.Sprintf("%s/anonymous/publicKey?source=%s&nonce=%d&sign=%s", p.baseURL, p.source, nonce, sign)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Data == "" {
		return "", fmt.Errorf("empty public key in response")
	}
	return out.Data, nil
}

func encryptElinterPassword(publicKeyBase64, password string) (string, error) {
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

type elinterInverter struct {
	SN       string  `json:"sn"`
	Model    string  `json:"model"`
	Pac      float64 `json:"pac"`
	Etotal   float64 `json:"etotal"`
	UpdateAt string  `json:"updateAt"`
}

// Discover flattens every inverter across every plant on this account.
// Unlike cmd/pvpro-sync's own auto-discovery (which reconciles many
// plants against many possible sites for Luther's single account), the
// self-service flow already has exactly one site per connection — every
// device found here registers under that one site. A customer with
// multiple plants on one vendor account is a real but deliberately
// deferred case (see the plan's "explicitly out of scope").
func (p *ELinterCSP) Discover(ctx context.Context, email, password string) ([]DiscoveredDevice, error) {
	sess, err := p.login(ctx, email, password)
	if err != nil {
		return nil, err
	}

	var plants []struct {
		ID int64 `json:"id"`
	}
	if err := sess.getJSON(ctx, "/api/v1/plants?page=1&limit=100", &struct {
		Data *struct {
			Infos *[]struct {
				ID int64 `json:"id"`
			} `json:"infos"`
		} `json:"data"`
	}{Data: &struct {
		Infos *[]struct {
			ID int64 `json:"id"`
		} `json:"infos"`
	}{Infos: &plants}}); err != nil {
		return nil, fmt.Errorf("list plants: %w", err)
	}

	var devices []DiscoveredDevice
	for _, plant := range plants {
		var out struct {
			Data struct {
				Infos []elinterInverter `json:"infos"`
			} `json:"data"`
		}
		url := fmt.Sprintf("/api/v1/plant/%d/inverters?page=1&limit=50&status=-1&sn=&id=%d&type=-2", plant.ID, plant.ID)
		if err := sess.getJSON(ctx, url, &out); err != nil {
			return nil, fmt.Errorf("list inverters for plant %d: %w", plant.ID, err)
		}
		for _, inv := range out.Data.Infos {
			if inv.SN == "" {
				continue
			}
			devices = append(devices, DiscoveredDevice{ExternalRef: inv.SN, Name: inv.SN, Model: inv.Model})
		}
	}
	return devices, nil
}

type elinterFlow struct {
	PVPower          float64 `json:"pvPower"`
	SOC              float64 `json:"soc"`
	BattV            float64 `json:"battV"`
	LoadPower        float64 `json:"loadOrEpsPower"`
	GridPower        float64 `json:"gridOrMeterPower"`
	ExistsBattery    bool    `json:"existsBattery"`
	ExistsLoad       bool    `json:"existsLoad"`
	ExistsGrid       bool    `json:"existsGrid"`
	BatteryFlowDatas []struct {
		Voltage float64 `json:"voltage"`
	} `json:"batteryFlowDatas"`
}

func (f elinterFlow) batteryVoltage() (float64, bool) {
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

// FetchReading mirrors cmd/pvpro-sync's own buildReading — same
// confirmed field mapping, same "status always ok, never guessed from
// the undocumented numeric status code" reasoning (see the original's
// comment; reproduced in this package's provider.go doc rather than
// duplicated verbatim here). PV/output voltage (the two extra realtime
// endpoints pvpro-sync also calls) are deliberately not fetched here —
// a reasonable trim for the reference implementation's first pass, not
// an oversight; can be added the same way if a real connected account
// shows they're needed.
func (p *ELinterCSP) FetchReading(ctx context.Context, email, password, externalRef string) (CloudReading, error) {
	sess, err := p.login(ctx, email, password)
	if err != nil {
		return CloudReading{}, err
	}

	var invOut struct {
		Data struct {
			Infos []elinterInverter `json:"infos"`
		} `json:"data"`
	}
	// The inverter-list endpoint is plant-scoped, but pac/etotal/
	// updateAt are per-inverter regardless of which plant we ask
	// through — sn is the actual lookup key downstream. Since we don't
	// carry the plant id in externalRef (see Discover's comment), reuse
	// getFlow's own sn-keyed endpoint for the pieces that come from it,
	// and pull pac/etotal from a direct per-device summary instead of
	// a full plant re-list.
	var summary struct {
		Data elinterInverter `json:"data"`
	}
	if err := sess.getJSON(ctx, "/api/v1/inverter/"+externalRef, &summary); err != nil {
		return CloudReading{}, fmt.Errorf("inverter summary for %s: %w", externalRef, err)
	}
	invOut.Data.Infos = []elinterInverter{summary.Data}

	var flowOut struct {
		Code    int         `json:"code"`
		Msg     string      `json:"msg"`
		Success bool        `json:"success"`
		Data    elinterFlow `json:"data"`
	}
	if err := sess.getJSON(ctx, "/api/v1/inverter/"+externalRef+"/flow", &flowOut); err != nil {
		return CloudReading{}, fmt.Errorf("flow for %s: %w", externalRef, err)
	}
	if flowOut.Code != 0 || !flowOut.Success {
		return CloudReading{}, fmt.Errorf("flow for %s: %s (code %d)", externalRef, flowOut.Msg, flowOut.Code)
	}

	inv := summary.Data
	flow := flowOut.Data
	reading := CloudReading{
		Timestamp:      inv.UpdateAt,
		PowerKW:        inv.Pac / 1000.0,
		EnergyKWhTotal: inv.Etotal,
		Status:         "ok",
		PVPowerKW:      floatPtr(flow.PVPower / 1000.0),
	}
	if flow.ExistsBattery {
		reading.BatterySOCPct = floatPtr(flow.SOC)
		if v, ok := flow.batteryVoltage(); ok {
			reading.BatteryVoltageV = floatPtr(v)
		}
	}
	if flow.ExistsLoad {
		reading.LoadPowerKW = floatPtr(flow.LoadPower / 1000.0)
	}
	if flow.ExistsGrid {
		reading.GridPowerKW = floatPtr(-flow.GridPower / 1000.0)
	}
	return reading, nil
}

func (s *elinterSession) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.accessToken)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func floatPtr(f float64) *float64 { return &f }
