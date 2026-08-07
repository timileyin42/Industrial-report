-- name: CreateSite :one
INSERT INTO sites (site_id, name, address, gps_lat, gps_lng, inverter_make_model, system_size_kw, install_date, timezone, cohort_id, country)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetSite :one
SELECT * FROM sites WHERE site_id = $1;

-- name: UpdateSiteCountry :one
-- Corrects a site's country after creation — needed because the
-- migration backfilling this column had to guess 'NG' for every
-- pre-existing row (see migrations/0010_site_country.sql).
UPDATE sites SET country = $2 WHERE site_id = $1
RETURNING *;

-- name: ListSiteCountries :many
-- Site->country lookup for fleet-wide emissions, which must resolve each
-- site's own grid factor rather than one global default (see
-- internal/registry/emissions.go FleetEmissions). Unpaginated: this is an
-- internal aggregation input, not a user-facing list, and is bounded by
-- fleet size, not telemetry volume.
SELECT site_id, country FROM sites
WHERE sqlc.narg('cohort_id')::text IS NULL OR cohort_id = sqlc.narg('cohort_id');

-- name: ListSites :many
-- Keyset pagination: pass cursor_created_at/cursor_site_id as NULL for the
-- first page. When restricting to a single site (restricted-role callers),
-- pass site_filter; NULL means "all sites" (operator).
SELECT * FROM sites
WHERE (sqlc.narg('site_filter')::text IS NULL OR site_id = sqlc.narg('site_filter'))
  AND (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL
    OR (created_at, site_id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_site_id')::text)
  )
ORDER BY created_at DESC, site_id DESC
LIMIT sqlc.arg('page_limit');

-- name: FleetTotals :one
SELECT
    count(*)::bigint AS total_sites,
    coalesce(sum(system_size_kw), 0)::numeric AS total_capacity_kw
FROM sites;
