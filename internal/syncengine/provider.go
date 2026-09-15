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

// Provider is the metadata every manufacturer-cloud integration
// exposes regardless of how it authenticates — enough to drive the
// vendor picker (GET /v1/vendor-providers) and Registry's bookkeeping.
// The actual sync capability lives on one of the two more specific
// interfaces below; AuthType() is what a caller switches on to know
// which one a given Provider also implements.
//
// A single implementation can legitimately back several branded apps
// at once — ELinterCSP already covers PV Pro, Sunsynk Connect, and
// Powerview, since all three are white-labeled skins over the same
// backend.
type Provider interface {
	// Name is this provider's stable key — stored in
	// vendor_connections.provider and used to look it up in Registry.
	Name() string
	DisplayName() string
	AuthType() string
	Capabilities() Capabilities
	DefaultPollInterval() time.Duration
}

// PasswordAuthProvider is a Provider whose AuthType is
// AuthTypePassword — the customer types the same email/password they
// already use in the vendor's own app (see ELinterCSP, the one
// concrete implementation today).
type PasswordAuthProvider interface {
	Provider

	// Discover lists every device on the account these credentials
	// belong to. email/password are the plaintext vendor login,
	// decrypted by the caller (cmd/vendor-sync) immediately before use
	// — never persisted or logged in plaintext anywhere.
	Discover(ctx context.Context, email, password string) ([]DiscoveredDevice, error)

	// FetchReading returns one device's current reading. externalRef is
	// whatever Discover returned for that device.
	FetchReading(ctx context.Context, email, password, externalRef string) (CloudReading, error)
}

// OAuthToken is what a vendor's token endpoint returns — stored
// encrypted in vendor_connections.encrypted_credential exactly like a
// password-form vendor's email/password is, just token-shaped instead.
// ExpiresAt drives cmd/vendor-sync's decision to refresh before a poll
// rather than reactively on a failed call.
type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// OAuthProvider is a Provider whose AuthType is AuthTypeOAuth — used
// by a vendor that has no password-login API at all (SolarEdge,
// Enphase, ...), so the customer's password never touches this
// platform. No concrete implementation exists yet (see this package's
// doc comment on ELinterCSP being the only one so far) — this is the
// seam a future one plugs into.
//
// The redirect crosses out of this app and back, so unlike
// PasswordAuthProvider's Discover/FetchReading, nothing here is called
// directly from an HTTP handler's request/response cycle in one shot:
// AuthorizationURL and ExchangeCode happen a request apart (the
// vendor's own consent screen sits in between), and RefreshToken is
// called later still, from cmd/vendor-sync's poll loop.
type OAuthProvider interface {
	Provider

	// AuthorizationURL builds the vendor's consent-screen URL. state is
	// opaque to the Provider — the caller (internal/httpapi) signs and
	// later verifies it, this just has to echo it back in the URL
	// exactly as vendors' own OAuth2 flows require. redirectURI must be
	// byte-for-byte what's registered with the vendor's own OAuth app.
	AuthorizationURL(state, redirectURI string) string

	// ExchangeCode trades a callback's authorization code for a real
	// token — called once, immediately after the vendor redirects back
	// to internal/httpapi's callback handler. redirectURI must match
	// the one AuthorizationURL was built with (most vendors require the
	// exact same value on both calls).
	ExchangeCode(ctx context.Context, code, redirectURI string) (OAuthToken, error)

	// RefreshToken exchanges a still-valid refresh token for a new
	// access token — called by cmd/vendor-sync before a stored token
	// expires, never reactively on a 401 (see this package's reliability
	// requirements).
	RefreshToken(ctx context.Context, refreshToken string) (OAuthToken, error)

	// Discover/FetchReading mirror PasswordAuthProvider's, just against
	// a bearer access token instead of a vendor login.
	Discover(ctx context.Context, accessToken string) ([]DiscoveredDevice, error)
	FetchReading(ctx context.Context, accessToken, externalRef string) (CloudReading, error)
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
