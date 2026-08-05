-- +goose Up

-- Versioned, append-only (never UPDATE) so a past report stays reproducible
-- even if the official factor changes later — concept note §10 explicitly
-- requires "stating the factor and period used alongside every figure."
-- No seed row: the concept note says "confirm and set the current official
-- value" — this is a client-confirmed figure an operator must explicitly
-- set (POST /v1/config/emission-factor), never an invented/assumed default.
CREATE TABLE grid_emission_factor (
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kg_co2_per_kwh     numeric NOT NULL CHECK (kg_co2_per_kwh > 0),
    country            text NOT NULL DEFAULT 'NG',
    source             text NOT NULL,
    effective_from     timestamptz NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    created_by_user_id bigint REFERENCES users(id)
);

CREATE INDEX idx_emission_factor_effective ON grid_emission_factor (country, effective_from DESC);

-- +goose Down
DROP TABLE IF EXISTS grid_emission_factor;
