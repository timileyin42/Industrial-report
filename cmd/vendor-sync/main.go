// Command vendor-sync is the background poller for the self-service
// "Connect Your Inverter" flow (internal/registry/vendor_connections.go,
// internal/syncengine) — distinct from cmd/pvpro-sync and
// cmd/solarman-sync, which each drive one shared operator account.
// vendor-sync instead polls every customer's own vendor_connections row
// across every provider the syncengine.Registry knows about, isolating
// one connection's failure from every other's exactly like a live
// provider outage must never take down another customer's data (see
// CLAUDE.md's ingestion isolation requirements — the same discipline
// applied here to a second, vendor-cloud-backed ingestion path).
//
// Same ticker-loop convention as pvpro-sync/solarman-sync (this repo's
// only background-job pattern), but multi-account and multi-provider:
// each tick lists every connection that still needs attention (pending,
// active, or previously erroring — never revoked) and, respecting each
// connection's own provider's DefaultPollInterval, discovers and syncs
// it. Talks to Postgres directly (like cmd/api) to read
// vendor_connections, and to this platform's own HTTP API (like
// pvpro-sync/solarman-sync) to register devices and submit readings —
// so device creation goes through the same validation every other
// caller of POST /v1/devices does, rather than a second, divergent path
// straight into the database.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/registry"
	"github.com/timileyin42/zgnis-solar/internal/syncengine"
)

func main() {
	dbURL := withSSLMode(mustEnv("DATABASE_URL"))
	apiBaseURL := mustEnv("API_BASE_URL")
	operatorEmail := mustEnv("API_OPERATOR_EMAIL")
	operatorPassword := mustEnv("API_OPERATOR_PASSWORD")
	tickInterval := envSeconds("POLL_INTERVAL_SECONDS", 30)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	vendorConns, err := registry.NewVendorConnections(db.New(pool))
	if err != nil {
		log.Fatalf("vendor connections: %v", err)
	}
	// Registering the next provider is the only change needed here —
	// same single line cmd/api/main.go wires it with. See
	// internal/syncengine/elinter_csp.go's own doc comment.
	providers := syncengine.NewRegistry(syncengine.NewELinterCSP())
	ours := newOurAPIClient(apiBaseURL, operatorEmail, operatorPassword)
	httpClient := &http.Client{Timeout: 20 * time.Second}

	log.Printf("vendor-sync starting: polling every %s across %d provider(s)", tickInterval, len(providers.List()))
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	runOnce(ctx, vendorConns, providers, ours, httpClient)
	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return
		case <-ticker.C:
			runOnce(ctx, vendorConns, providers, ours, httpClient)
		}
	}
}

func runOnce(ctx context.Context, vendorConns *registry.VendorConnections, providers *syncengine.Registry, ours *ourAPIClient, httpClient *http.Client) {
	conns, err := vendorConns.ListActiveDecrypted(ctx)
	if err != nil {
		log.Printf("vendor-sync: list connections: %v", err)
		return
	}

	// Connections are independent customer accounts, often on
	// different vendor clouds — sync them concurrently (bounded) so one
	// slow provider doesn't delay every other customer's poll, same
	// isolation goal as the per-connection error handling below.
	const maxConcurrent = 8
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	for _, conn := range conns {
		provider, ok := providers.Get(conn.Provider)
		if !ok {
			log.Printf("vendor-sync: connection %d: unknown provider %q, skipping", conn.ID, conn.Provider)
			continue
		}
		if conn.LastSyncedAt != nil && time.Since(*conn.LastSyncedAt) < provider.DefaultPollInterval() {
			continue // not due yet — respects this provider's own pace
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(conn registry.ActiveConnection, provider syncengine.Provider) {
			defer wg.Done()
			defer func() { <-sem }()
			syncConnection(ctx, vendorConns, provider, ours, httpClient, conn)
		}(conn, provider)
	}
	wg.Wait()
}

// bindProviderCalls returns discover/fetchReading closures already
// bound to the right credential shape for this connection's provider —
// email/password for a PasswordAuthProvider, a (possibly just-
// refreshed) bearer token for an OAuthProvider. Isolating this
// type-switch here means the rest of syncConnection below doesn't need
// to know or care which auth flow a given connection uses.
//
// An OAuth token is refreshed a couple of minutes before it actually
// expires, not reactively on a failed call (same "don't wait for
// something to break" discipline as cmd/pvpro-sync's own JWT refresh),
// and the refreshed pair is persisted immediately — before it's ever
// used — since most vendors rotate the refresh token on every use; a
// crash between refreshing and the next poll must never strand the
// connection on a refresh token that's already been consumed.
func bindProviderCalls(ctx context.Context, vendorConns *registry.VendorConnections, provider syncengine.Provider, conn registry.ActiveConnection) (
	discover func(context.Context) ([]syncengine.DiscoveredDevice, error),
	fetchReading func(context.Context, string) (syncengine.CloudReading, error),
	err error,
) {
	switch p := provider.(type) {
	case syncengine.PasswordAuthProvider:
		return func(ctx context.Context) ([]syncengine.DiscoveredDevice, error) {
				return p.Discover(ctx, conn.Email, conn.Password)
			}, func(ctx context.Context, ref string) (syncengine.CloudReading, error) {
				return p.FetchReading(ctx, conn.Email, conn.Password, ref)
			}, nil

	case syncengine.OAuthProvider:
		accessToken := conn.AccessToken
		if conn.ExpiresAt == nil || time.Now().Add(2*time.Minute).After(*conn.ExpiresAt) {
			newToken, err := p.RefreshToken(ctx, conn.RefreshToken)
			if err != nil {
				return nil, nil, fmt.Errorf("refresh token: %w", err)
			}
			if err := vendorConns.UpdateOAuthToken(ctx, conn.ID, newToken); err != nil {
				log.Printf("vendor-sync: connection %d: persist refreshed token: %v", conn.ID, err)
			}
			accessToken = newToken.AccessToken
		}
		return func(ctx context.Context) ([]syncengine.DiscoveredDevice, error) {
				return p.Discover(ctx, accessToken)
			}, func(ctx context.Context, ref string) (syncengine.CloudReading, error) {
				return p.FetchReading(ctx, accessToken, ref)
			}, nil

	default:
		// Reachable only if a Provider is ever registered whose AuthType
		// doesn't match either concrete interface it implements — a wiring
		// bug in cmd/api's/cmd/vendor-sync's own Registry construction,
		// not a per-customer failure, but still isolated to this one
		// connection rather than panicking the whole poll cycle.
		return nil, nil, fmt.Errorf("provider %q implements neither PasswordAuthProvider nor OAuthProvider", provider.Name())
	}
}

// syncConnection discovers and syncs one customer's vendor account.
// Any failure here — bad/expired credentials, the vendor's cloud being
// unavailable, a single device's reading fetch failing — marks only
// this one connection as errored and returns; it never touches another
// connection or another customer's site/device rows.
func syncConnection(ctx context.Context, vendorConns *registry.VendorConnections, provider syncengine.Provider, ours *ourAPIClient, httpClient *http.Client, conn registry.ActiveConnection) {
	discover, fetchReading, err := bindProviderCalls(ctx, vendorConns, provider, conn)
	if err != nil {
		log.Printf("vendor-sync: connection %d (%s): %v", conn.ID, conn.Provider, err)
		_ = vendorConns.MarkError(ctx, conn.ID, err.Error())
		return
	}

	devices, err := discover(ctx)
	if err != nil {
		log.Printf("vendor-sync: connection %d (%s): discover: %v", conn.ID, conn.Provider, err)
		_ = vendorConns.MarkError(ctx, conn.ID, "discover: "+err.Error())
		return
	}
	if len(devices) == 0 {
		log.Printf("vendor-sync: connection %d (%s): no devices found on vendor account", conn.ID, conn.Provider)
		_ = vendorConns.MarkError(ctx, conn.ID, "no inverters found on this vendor account")
		return
	}
	// A vendor account with more than one plant/inverter is an explicitly
	// deferred edge case (see the plan) — every discovered device is
	// still synced below, but the connection's own external_ref (a
	// single text column) only ever records the first, since today's
	// self-signup flow assumes one device per connection.
	firstRef := devices[0].ExternalRef

	var syncedAny bool
	var lastErr error
	for _, dev := range devices {
		if err := ensureDeviceRegistered(ctx, ours, conn.SiteID, dev); err != nil {
			log.Printf("vendor-sync: connection %d: register device %s: %v", conn.ID, dev.ExternalRef, err)
			lastErr = err
			continue
		}
		token, err := ours.issueCloudImportToken(ctx, dev.ExternalRef)
		if err != nil {
			log.Printf("vendor-sync: connection %d: issue token for %s: %v", conn.ID, dev.ExternalRef, err)
			lastErr = err
			continue
		}
		reading, err := fetchReading(ctx, dev.ExternalRef)
		if err != nil {
			log.Printf("vendor-sync: connection %d: fetch reading for %s: %v", conn.ID, dev.ExternalRef, err)
			lastErr = err
			continue
		}
		wire := cloudReading{
			Timestamp: reading.Timestamp, PowerKW: reading.PowerKW, EnergyKWhTotal: reading.EnergyKWhTotal,
			Status: reading.Status, PVPowerKW: reading.PVPowerKW, BatterySOCPct: reading.BatterySOCPct,
			BatteryVoltageV: reading.BatteryVoltageV, LoadPowerKW: reading.LoadPowerKW, GridPowerKW: reading.GridPowerKW,
		}
		if err := submitReading(ctx, httpClient, token, ours.baseURL, dev.ExternalRef, wire); err != nil {
			log.Printf("vendor-sync: connection %d: submit reading for %s: %v", conn.ID, dev.ExternalRef, err)
			lastErr = err
			continue
		}
		syncedAny = true
		log.Printf("vendor-sync: connection %d (%s): synced device %s — %.2f kW, ts=%s", conn.ID, conn.Provider, dev.ExternalRef, reading.PowerKW, reading.Timestamp)
	}

	if syncedAny {
		_ = vendorConns.MarkSynced(ctx, conn.ID, &firstRef)
		return
	}
	msg := "sync failed for every discovered device"
	if lastErr != nil {
		msg = lastErr.Error()
	}
	_ = vendorConns.MarkError(ctx, conn.ID, msg)
}

func ensureDeviceRegistered(ctx context.Context, ours *ourAPIClient, siteID string, dev syncengine.DiscoveredDevice) error {
	_, exists, err := ours.findDeviceSite(ctx, dev.ExternalRef)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	name := dev.Name
	if name == "" {
		name = dev.ExternalRef
	}
	return ours.createDevice(ctx, newDeviceInput{
		DeviceID:      dev.ExternalRef,
		SiteID:        siteID,
		InverterModel: dev.Model,
		InstallNotes:  "Auto-registered via vendor-sync (self-service " + name + " connection)",
	})
}

func withSSLMode(dbURL string) string {
	if strings.Contains(dbURL, "sslmode=") {
		return dbURL
	}
	mode := os.Getenv("DATABASE_SSLMODE")
	if mode == "" {
		mode = "disable"
	}
	sep := "?"
	if strings.Contains(dbURL, "?") {
		sep = "&"
	}
	return dbURL + sep + "sslmode=" + mode
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
