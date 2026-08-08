-- +goose Up

-- Nothing in the schema previously identified a fleet's "home"/default
-- site — the dashboard's weather widget picked whatever site was most
-- recently created and happened to have a saved location, which meant a
-- test site created for map-picker testing (see LocationPicker work)
-- could silently become "the" location shown on the Fleet Dashboard.
-- This adds a real, explicit primary-site flag instead of an implicit
-- sort-order guess. The partial unique index enforces "at most one
-- primary site" at the database level, not just in application code.
ALTER TABLE sites ADD COLUMN is_primary boolean NOT NULL DEFAULT false;
CREATE UNIQUE INDEX idx_sites_one_primary ON sites (is_primary) WHERE is_primary = true;

-- +goose Down
DROP INDEX idx_sites_one_primary;
ALTER TABLE sites DROP COLUMN is_primary;
