// Package syncengine holds the pluggable "smart inverter provider"
// abstraction behind the self-service Connect-Your-Inverter flow — the
// first such abstraction in this codebase (confirmed by investigation:
// cmd/pvpro-sync and cmd/solarman-sync are each concrete, single-account
// binaries, not built against a shared interface). Adding the next
// manufacturer means implementing Provider, not touching this package,
// cmd/vendor-sync, or either existing standalone connector.
package syncengine

import (
	"context"
	"time"
)

// CloudReading mirrors internal/httpapi's cloudReadingRequest — the
// same vendor-neutral shape every existing connector already forwards
// into POST /v1/cloud-import/:device_id/readings. A Provider's
// FetchReading returns this so the rest of the pipeline (validation,
// reset detection, storage, analytics) never needs to know which
// vendor a reading came from.
type CloudReading struct {
	Timestamp       string
	PowerKW         float64
	EnergyKWhTotal  float64
	Status          string
	PVPowerKW       *float64
	BatterySOCPct   *float64
	BatteryVoltageV *float64
	LoadPowerKW     *float64
	GridPowerKW     *float64
}

// Capabilities lets the rest of the app only expose functionality a
// connected provider actually supports — e.g. hide a battery-SOC ring
// for a connection whose vendor never reports one, rather than showing
// a permanent "—".
type Capabilities struct {
	RealtimeData   bool
	HistoricalData bool
	BatteryData    bool
	GridData       bool
	LoadData       bool
}

// DiscoveredDevice is one inverter/plant found on a customer's vendor
// account during Discover — ExternalRef is whatever identifier
// FetchReading needs to ask that same vendor for this device's data
// again (a serial number, a plant+inverter id pair encoded as a
// string, etc. — Provider-specific, opaque to every caller).
type DiscoveredDevice struct {
	ExternalRef string
	Name        string
	Model       string
}

// AuthType values — what internal/httpapi/vendor_connection_handlers.go
// and the frontend's vendor picker use to decide which form to show.
// Only "password" has a concrete implementation today (ELinterCSP);
// "oauth_token" is reserved for a future vendor (SolarEdge, Enphase,
// ...) that requires it — no OAuth adapter exists yet, this is
// deliberately just the seam for one.
const (
	AuthTypePassword = "password"
	AuthTypeOAuth    = "oauth_token"
)

// Provider is one manufacturer-cloud integration. A single
// implementation can legitimately back several branded apps at once —
// ELinterCSP already covers PV Pro, Sunsynk Connect, and Powerview,
// since all three are white-labeled skins over the same backend.
type Provider interface {
	// Name is this provider's stable key — stored in
	// vendor_connections.provider and used to look it up in Registry.
	Name() string
	DisplayName() string
	AuthType() string
	Capabilities() Capabilities
	DefaultPollInterval() time.Duration

	// Discover lists every device on the account these credentials
	// belong to. email/password are the plaintext vendor login,
	// decrypted by the caller (cmd/vendor-sync) immediately before use
	// — never persisted or logged in plaintext anywhere.
	Discover(ctx context.Context, email, password string) ([]DiscoveredDevice, error)

	// FetchReading returns one device's current reading. externalRef is
	// whatever Discover returned for that device.
	FetchReading(ctx context.Context, email, password, externalRef string) (CloudReading, error)
}

// Registry is the static, in-process map of every Provider this build
// knows about — populated by NewRegistry, not global state, so
// cmd/vendor-sync (and, later, any test) controls exactly what's
// registered rather than relying on package-level init() side effects.
type Registry struct {
	providers map[string]Provider
}

func NewRegistry(providers ...Provider) *Registry {
	r := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, p := range providers {
		r.providers[p.Name()] = p
	}
	return r
}

func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// List is what feeds GET /v1/vendor-providers — the picker the
// customer sees on the Connect screen.
func (r *Registry) List() []Provider {
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}
