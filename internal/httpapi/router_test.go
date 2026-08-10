package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

// This suite exists for exactly the requirement CLAUDE.md calls
// non-negotiable: "a restricted user hitting the API directly with
// another site's ID must get a 403, not data." Route-level middleware
// (auth.RequireRole / auth.RequireSiteAccess) is checked here end to end
// through the real router — not by re-deriving the same logic in a unit
// test that could drift from what's actually wired in router.go.

// testAPI bundles everything an httpapi test needs: a router backed by
// the real registry (same DATABASE_URL-skip convention as the registry
// package's own tests) and a token issuer to mint test JWTs directly,
// bypassing the login endpoint/bcrypt entirely — RequireAuth/RequireRole/
// RequireSiteAccess only ever look at JWT claims, never at a DB user row,
// so this is a faithful shortcut, not a shortcut that skips what's being
// tested.
type testAPI struct {
	router *echo.Echo
	issuer auth.TokenIssuer
	pool   *pgxpool.Pool
	q      *db.Queries
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping integration test that needs a real database")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)
	q := db.New(pool)

	sites := registry.NewSites(q)
	devices := registry.NewDevices(q, 10*time.Minute, 5*time.Minute, nil)
	fleet := registry.NewFleet(sites, devices, 10*time.Minute, 5*time.Minute, 24*time.Hour)
	telemetry := registry.NewTelemetry(q)
	analytics := registry.NewAnalytics(q)

	issuer := auth.NewTokenIssuer("test-secret-not-for-real-use")
	router := NewRouter(Deps{
		Sites:     sites,
		Devices:   devices,
		Fleet:     fleet,
		Telemetry: telemetry,
		Analytics: analytics,
		Issuer:    issuer,
	})
	return &testAPI{router: router, issuer: issuer, pool: pool, q: q}
}

func (a *testAPI) token(t *testing.T, role domain.Role, siteID *string) string {
	t.Helper()
	tok, _, err := a.issuer.Issue(1, role, siteID)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return tok
}

func (a *testAPI) do(t *testing.T, method, path, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	a.router.ServeHTTP(rec, req)
	return rec
}

func (a *testAPI) createSite(t *testing.T, ctx context.Context, siteID string) {
	t.Helper()
	sites := registry.NewSites(a.q)
	if _, err := sites.Create(ctx, 1, registry.CreateSiteInput{
		SiteID: siteID, Name: "RBAC Test Site", Timezone: "UTC", Country: "NG",
	}); err != nil {
		t.Fatalf("create test site: %v", err)
	}
	t.Cleanup(func() {
		cctx := context.Background()
		_, _ = a.pool.Exec(cctx, `DELETE FROM devices WHERE site_id = $1`, siteID)
		_, _ = a.pool.Exec(cctx, `DELETE FROM sites WHERE site_id = $1`, siteID)
	})
}

func uniqueSiteID(prefix string) string {
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func TestRequireAuth_MissingTokenIs401NotForbidden(t *testing.T) {
	api := newTestAPI(t)
	rec := api.do(t, http.MethodGet, "/v1/fleet/summary", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a request with no bearer token, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequireRole_RestrictedUserGetsForbiddenOnOperatorOnlyEndpoint(t *testing.T) {
	api := newTestAPI(t)
	siteID := uniqueSiteID("site-rbac-")
	token := api.token(t, domain.RoleRestricted, &siteID)

	rec := api.do(t, http.MethodGet, "/v1/fleet/summary", token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a restricted user on an operator-only endpoint, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequireRole_OperatorCanReachOperatorOnlyEndpoint(t *testing.T) {
	api := newTestAPI(t)
	token := api.token(t, domain.RoleOperator, nil)

	rec := api.do(t, http.MethodGet, "/v1/fleet/summary", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for an operator on an operator-only endpoint, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestRequireSiteAccess_RestrictedUserForbiddenOnAnotherSite is the exact
// scenario CLAUDE.md calls non-negotiable — a restricted user editing the
// URL to another site's ID must get a 403, not that site's real data and
// not a 404 that would leak whether the ID exists.
func TestRequireSiteAccess_RestrictedUserForbiddenOnAnotherSite(t *testing.T) {
	api := newTestAPI(t)
	ctx := context.Background()
	ownSiteID := uniqueSiteID("site-own-")
	otherSiteID := uniqueSiteID("site-other-")
	api.createSite(t, ctx, ownSiteID)
	api.createSite(t, ctx, otherSiteID)

	token := api.token(t, domain.RoleRestricted, &ownSiteID)

	rec := api.do(t, http.MethodGet, "/v1/sites/"+otherSiteID, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a restricted user requesting another site, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestRequireSiteAccess_RestrictedUserAllowedOnOwnSite is the control
// case — confirms the 403 above is actually about site mismatch, not a
// broken route/handler that would 403 or error on any request.
func TestRequireSiteAccess_RestrictedUserAllowedOnOwnSite(t *testing.T) {
	api := newTestAPI(t)
	ctx := context.Background()
	ownSiteID := uniqueSiteID("site-own-")
	api.createSite(t, ctx, ownSiteID)

	token := api.token(t, domain.RoleRestricted, &ownSiteID)

	rec := api.do(t, http.MethodGet, "/v1/sites/"+ownSiteID, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a restricted user requesting their own site, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestRequireSiteAccess_DeviceScopedRouteForbidsAnotherSitesDevice checks
// the device-scoped variant (resolveSiteFromDeviceParam), which resolves
// site access via a DB lookup on the device rather than the URL directly
// — a different code path from the site-scoped test above, and one that's
// easy to get wrong (e.g. by forgetting to join through to site_id).
func TestRequireSiteAccess_DeviceScopedRouteForbidsAnotherSitesDevice(t *testing.T) {
	api := newTestAPI(t)
	ctx := context.Background()
	ownSiteID := uniqueSiteID("site-own-")
	otherSiteID := uniqueSiteID("site-other-")
	api.createSite(t, ctx, ownSiteID)
	api.createSite(t, ctx, otherSiteID)

	deviceID := uniqueSiteID("device-other-")
	if _, err := api.pool.Exec(ctx,
		`INSERT INTO devices (device_id, site_id, secret_hash) VALUES ($1, $2, 'test-hash')`,
		deviceID, otherSiteID,
	); err != nil {
		t.Fatalf("seed device: %v", err)
	}
	t.Cleanup(func() { _, _ = api.pool.Exec(context.Background(), `DELETE FROM devices WHERE device_id = $1`, deviceID) })

	token := api.token(t, domain.RoleRestricted, &ownSiteID)

	rec := api.do(t, http.MethodGet, "/v1/devices/"+deviceID, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a restricted user requesting a device belonging to another site, got %d: %s", rec.Code, rec.Body.String())
	}
}
