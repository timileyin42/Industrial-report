// Command api serves the dashboard/admin API backed by TimescaleDB.
// Per concept note section 8: the dashboard only ever reads from the
// platform's own database — never from a device or manufacturer cloud.
//
// This file is wiring only — handlers live in internal/httpapi, business
// logic in internal/registry, queries in internal/db (sqlc-generated).
package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/httpapi"
	"github.com/timileyin42/zgnis-solar/internal/registry"
)

func main() {
	dbURL := mustEnv("DATABASE_URL")
	jwtSecret := mustEnv("JWT_SECRET")

	onlineThreshold := envMinutes("ONLINE_THRESHOLD_MINUTES", 10)
	expectedInterval := envMinutes("EXPECTED_READING_INTERVAL_MINUTES", 5)
	coverageWindow := envHours("COVERAGE_WINDOW_HOURS", 24)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	sites := registry.NewSites(queries)
	devices := registry.NewDevices(queries, onlineThreshold, expectedInterval)
	users := registry.NewUsers(queries)
	fleet := registry.NewFleet(sites, devices, onlineThreshold, expectedInterval, coverageWindow)
	telemetry := registry.NewTelemetry(queries)
	analytics := registry.NewAnalytics(queries)
	emissions := registry.NewEmissions(analytics, queries)
	benchmark := registry.NewBenchmark(analytics)
	anomaly := registry.NewAnomaly(analytics)
	auditLog := registry.NewAuditLog(queries)

	e := httpapi.NewRouter(httpapi.Deps{
		Sites:     sites,
		Devices:   devices,
		Users:     users,
		Fleet:     fleet,
		Telemetry: telemetry,
		Analytics: analytics,
		Emissions: emissions,
		Benchmark: benchmark,
		Anomaly:   anomaly,
		AuditLog:  auditLog,
		Issuer:    auth.NewTokenIssuer(jwtSecret),
	})

	log.Println("api listening on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}

func envMinutes(key string, def int) time.Duration {
	return time.Duration(envInt(key, def)) * time.Minute
}

func envHours(key string, def int) time.Duration {
	return time.Duration(envInt(key, def)) * time.Hour
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("invalid %s %q: %v", key, v, err)
	}
	return n
}
