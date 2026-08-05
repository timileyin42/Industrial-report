-- name: GetCurrentEmissionFactor :one
-- The most recently effective factor for a country. Returns pgx.ErrNoRows
-- until an operator has explicitly set one — callers must turn that into a
-- 409, never a fabricated default.
SELECT * FROM grid_emission_factor WHERE country = $1 ORDER BY effective_from DESC LIMIT 1;

-- name: CreateEmissionFactor :one
-- Append-only — never UPDATE. Past reports must stay reproducible even if
-- the official factor changes later.
INSERT INTO grid_emission_factor (kg_co2_per_kwh, country, source, effective_from, created_by_user_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListEmissionFactorHistory :many
SELECT * FROM grid_emission_factor WHERE country = $1 ORDER BY effective_from DESC LIMIT $2;
