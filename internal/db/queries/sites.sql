-- name: CreateSite :one
INSERT INTO sites (site_id, name, address, gps_lat, gps_lng, inverter_make_model, system_size_kw, install_date, timezone, cohort_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetSite :one
SELECT * FROM sites WHERE site_id = $1;

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
