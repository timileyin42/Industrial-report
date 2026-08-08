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
	Sites          *registry.Sites
	Devices        *registry.Devices
	Users          *registry.Users
	Fleet          *registry.Fleet
	Telemetry      *registry.Telemetry
	Analytics      *registry.Analytics
	Emissions      *registry.Emissions
	Benchmark      *registry.Benchmark
	Anomaly        *registry.Anomaly
	AuditLog       *registry.AuditLog
	IngestionAudit *registry.IngestionAudit
	Invites        *registry.Invites
	PasswordReset  *registry.PasswordReset
	Exports        *registry.Exports
	Alerts         *registry.Alerts
	Issuer         auth.TokenIssuer
}

func NewRouter(deps Deps) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(corsMiddleware())

	h := &handlers{deps: deps}

	loginLimiter := middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(1))
	registerLimiter := middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(2))
	// Same rate as login — both are public, credential-adjacent endpoints
	// (accept-invite sets a password; password-reset issues a token).
	authLimiter := middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(1))

	v1 := e.Group("/v1")

	// Public
	v1.POST("/auth/login", h.login, loginLimiter)
	v1.POST("/invites/accept", h.acceptInvite, authLimiter)
	v1.POST("/auth/password-reset/request", h.requestPasswordReset, authLimiter)
	v1.POST("/auth/password-reset/confirm", h.confirmPasswordReset, authLimiter)

	// Authenticated
	authed := v1.Group("", auth.RequireAuth(deps.Issuer))

	operatorOnly := auth.RequireRole(domain.RoleOperator)

	authed.POST("/users", h.createUser, operatorOnly)
	authed.GET("/users", h.listUsers, operatorOnly)
	authed.PATCH("/users/:user_id/disabled", h.setUserDisabled, operatorOnly)
	authed.POST("/users/invite", h.createInvite, operatorOnly)

	authed.POST("/sites", h.createSite, operatorOnly)
	authed.GET("/sites", h.listSites)
	authed.GET("/sites/primary", h.getPrimarySite)
	authed.GET("/cohorts", h.listCohorts, operatorOnly)
	authed.GET("/fleet/alerts", h.fleetAlerts, operatorOnly)
	authed.GET("/sites/:site_id", h.getSite, auth.RequireSiteAccess(h.resolveSiteFromParam))
	authed.PATCH("/sites/:site_id/country", h.updateSiteCountry, operatorOnly)
	authed.PATCH("/sites/:site_id/primary", h.setSitePrimary, operatorOnly)
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
	authed.GET("/sites/:site_id/analytics/performance-ratio", h.sitePerformanceRatio, siteAccess)
	authed.GET("/sites/:site_id/analytics/peak", h.sitePeak, siteAccess)
	authed.GET("/sites/:site_id/analytics/capacity-factor", h.siteCapacityFactor, siteAccess)
	authed.GET("/sites/:site_id/analytics/emissions", h.siteEmissions, siteAccess)
	authed.GET("/sites/:site_id/analytics/compare/history", h.compareHistory, siteAccess)
	authed.GET("/sites/:site_id/analytics/anomalies", h.siteAnomalies, siteAccess)
	authed.GET("/sites/:site_id/export/telemetry.csv", h.siteTelemetryCSV, siteAccess)
	authed.GET("/sites/:site_id/export/summary.csv", h.siteSummaryCSV, siteAccess)

	// Phase 3 — analytics/KPIs (fleet-wide, operator-only: cross-site
	// comparisons leak fleet-wide distribution by construction)
	authed.GET("/fleet/current-generation", h.currentGeneration, operatorOnly)
	authed.GET("/fleet/top-sites", h.topSitesToday, operatorOnly)
	authed.GET("/fleet/ingestion-status", h.ingestionStatus, operatorOnly)
	authed.GET("/fleet/analytics/energy", h.fleetEnergy, operatorOnly)
	authed.GET("/fleet/analytics/specific-yield", h.fleetSpecificYield, operatorOnly)
	authed.GET("/fleet/analytics/performance-ratio", h.fleetPerformanceRatio, operatorOnly)
	authed.GET("/fleet/analytics/emissions", h.fleetEmissions, operatorOnly)
	authed.GET("/fleet/analytics/compare/fleet", h.compareFleet, operatorOnly)
	authed.GET("/fleet/analytics/benchmark", h.benchmarkSegments, operatorOnly)
	authed.GET("/fleet/analytics/trends", h.fleetTrends, operatorOnly)
	authed.GET("/fleet/analytics/cohorts/:cohort_id", h.fleetCohort, operatorOnly)
	authed.GET("/fleet/analytics/anomalies", h.fleetAnomalies, operatorOnly)
	authed.GET("/fleet/export/summary.csv", h.fleetSummaryCSV, operatorOnly)

	// Slice 3 — async export jobs, the counterpart to the sync CSV
	// endpoints above. Access rules are enforced inside the handler
	// itself (job_type/site_id come from the body, not a :site_id path
	// param), not route middleware — see createExportJob's comment.
	authed.POST("/exports", h.createExportJob)
	authed.GET("/exports", h.listExportJobs)
	authed.GET("/exports/:id", h.getExportJob)

	// Phase 3 — grid emission factor config (versioned, never invented —
	// see internal/registry/emissions.go)
	authed.POST("/config/emission-factor", h.setEmissionFactor, operatorOnly)
	authed.GET("/config/emission-factor", h.getEmissionFactor)

	// Phase 3 — admin audit-log browsing (migration 0002's deferred TODO)
	authed.GET("/audit/actions", h.listAuditActions, operatorOnly)

	// Slice 3 — ingestion (data-quality) audit read path, finally exposed.
	// No RequireSiteAccess middleware here (no :site_id path param) —
	// scoping for restricted users happens inside the handler, same
	// siteFilter pattern as listSites/listDevices.
	authed.GET("/audit/ingestion", h.listIngestionAudit)

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
		AllowMethods: []string{echo.GET, echo.POST, echo.PATCH},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType},
	})
}
