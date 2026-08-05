// Package httpapi holds Echo handlers and route wiring only — all
// validation and persistence logic lives in internal/registry, per the
// repo convention of no business logic in cmd/*/main.go or its transport
// layer.
package httpapi

import (
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

type Deps struct {
	Sites     *registry.Sites
	Devices   *registry.Devices
	Users     *registry.Users
	Fleet     *registry.Fleet
	Telemetry *registry.Telemetry
	Analytics *registry.Analytics
	Emissions *registry.Emissions
	Benchmark *registry.Benchmark
	Anomaly   *registry.Anomaly
	AuditLog  *registry.AuditLog
	Issuer    auth.TokenIssuer
}

func NewRouter(deps Deps) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(corsMiddleware())

	h := &handlers{deps: deps}

	loginLimiter := middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(1))
	registerLimiter := middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(2))

	v1 := e.Group("/v1")

	// Public
	v1.POST("/auth/login", h.login, loginLimiter)

	// Authenticated
	authed := v1.Group("", auth.RequireAuth(deps.Issuer))

	operatorOnly := auth.RequireRole(domain.RoleOperator)

	authed.POST("/users", h.createUser, operatorOnly)

	authed.POST("/sites", h.createSite, operatorOnly)
	authed.GET("/sites", h.listSites)
	authed.GET("/sites/:site_id", h.getSite, auth.RequireSiteAccess(h.resolveSiteFromParam))
	authed.GET("/sites/:site_id/telemetry", h.listTelemetry, auth.RequireSiteAccess(h.resolveSiteFromParam))

	authed.POST("/devices", h.registerDevice, operatorOnly, registerLimiter)
	authed.GET("/devices", h.listDevices)
	authed.GET("/devices/:device_id", h.getDevice, auth.RequireSiteAccess(h.resolveSiteFromDeviceParam))
	authed.GET("/devices/:device_id/status", h.deviceStatus, auth.RequireSiteAccess(h.resolveSiteFromDeviceParam))
	authed.POST("/devices/:device_id/revoke", h.revokeDevice, operatorOnly)
	authed.POST("/devices/:device_id/rotate-secret", h.rotateDeviceSecret, operatorOnly)

	authed.GET("/fleet/summary", h.fleetSummary, operatorOnly)
	authed.GET("/fleet/health", h.fleetHealth, operatorOnly)

	// Phase 3 — analytics/KPIs (site-scoped)
	siteAccess := auth.RequireSiteAccess(h.resolveSiteFromParam)
	authed.GET("/sites/:site_id/analytics/energy", h.siteEnergy, siteAccess)
	authed.GET("/sites/:site_id/analytics/specific-yield", h.siteSpecificYield, siteAccess)
	authed.GET("/sites/:site_id/analytics/peak", h.sitePeak, siteAccess)
	authed.GET("/sites/:site_id/analytics/capacity-factor", h.siteCapacityFactor, siteAccess)
	authed.GET("/sites/:site_id/analytics/emissions", h.siteEmissions, siteAccess)
	authed.GET("/sites/:site_id/analytics/compare/history", h.compareHistory, siteAccess)
	authed.GET("/sites/:site_id/analytics/anomalies", h.siteAnomalies, siteAccess)
	authed.GET("/sites/:site_id/export/telemetry.csv", h.siteTelemetryCSV, siteAccess)
	authed.GET("/sites/:site_id/export/summary.csv", h.siteSummaryCSV, siteAccess)

	// Phase 3 — analytics/KPIs (fleet-wide, operator-only: cross-site
	// comparisons leak fleet-wide distribution by construction)
	authed.GET("/fleet/analytics/energy", h.fleetEnergy, operatorOnly)
	authed.GET("/fleet/analytics/emissions", h.fleetEmissions, operatorOnly)
	authed.GET("/fleet/analytics/compare/fleet", h.compareFleet, operatorOnly)
	authed.GET("/fleet/analytics/benchmark", h.benchmarkSegments, operatorOnly)
	authed.GET("/fleet/analytics/trends", h.fleetTrends, operatorOnly)
	authed.GET("/fleet/analytics/cohorts/:cohort_id", h.fleetCohort, operatorOnly)
	authed.GET("/fleet/analytics/anomalies", h.fleetAnomalies, operatorOnly)
	authed.GET("/fleet/export/summary.csv", h.fleetSummaryCSV, operatorOnly)

	// Phase 3 — grid emission factor config (versioned, never invented —
	// see internal/registry/emissions.go)
	authed.POST("/config/emission-factor", h.setEmissionFactor, operatorOnly)
	authed.GET("/config/emission-factor", h.getEmissionFactor)

	// Phase 3 — admin audit-log browsing (migration 0002's deferred TODO)
	authed.GET("/audit/actions", h.listAuditActions, operatorOnly)

	return e
}

type handlers struct {
	deps Deps
}

func (h *handlers) resolveSiteFromParam(c echo.Context) (string, error) {
	return c.Param("site_id"), nil
}

func (h *handlers) resolveSiteFromDeviceParam(c echo.Context) (string, error) {
	return h.deps.Devices.SiteIDForDevice(c.Request().Context(), c.Param("device_id"))
}

// corsMiddleware reads an explicit allow-list from DASHBOARD_ORIGINS
// (comma-separated) — never a wildcard, per CLAUDE.md. No real dashboard
// origin exists yet, so this defaults to a localhost placeholder that must
// be revisited before any real frontend deploys.
func corsMiddleware() echo.MiddlewareFunc {
	origins := []string{"http://localhost:3000"}
	if v := os.Getenv("DASHBOARD_ORIGINS"); v != "" {
		origins = strings.Split(v, ",")
	}
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: origins,
		AllowMethods: []string{echo.GET, echo.POST},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType},
	})
}
