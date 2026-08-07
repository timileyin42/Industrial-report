-- +goose Up

-- Phase 0-3 defaulted every site to Africa/Lagos and every grid emission
-- factor to Nigeria (NG) — reasonable for the platform's first client,
-- wrong as a general default now that the product isn't Nigeria-specific.
-- UTC is the only timezone default that's never a wrong guess; existing
-- rows are untouched (this only changes what NEW rows get when the
-- caller doesn't specify one — see internal/httpapi/site_handlers.go and
-- GRID_COUNTRY in cmd/api/main.go for the matching application-level
-- defaults).
ALTER TABLE sites ALTER COLUMN timezone SET DEFAULT 'UTC';

-- +goose Down
ALTER TABLE sites ALTER COLUMN timezone SET DEFAULT 'Africa/Lagos';
