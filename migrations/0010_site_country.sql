-- +goose Up

-- CO2-offset reporting has been resolving every site's emission factor
-- through one global GRID_COUNTRY default (internal/registry/emissions.go)
-- — fine while every site was NG, wrong now that the fleet spans NG and
-- GB sites. This adds a real per-site country so SiteEmissions/
-- FleetEmissions can resolve each site's own grid factor.
--
-- Existing rows are backfilled to 'NG' — accurate, not a guess: every
-- site created before this migration genuinely was operating under the
-- NG-only assumption GRID_COUNTRY encoded, so 'NG' reflects their real
-- history. Going forward, per the same "don't assume, confirm with the
-- client" principle 0009_delocalize_defaults.sql applied to timezone, no
-- DEFAULT is set here — every new site must specify its country
-- explicitly (see internal/registry/sites.go's CreateSiteInput
-- validation and web/src/pages/AddSitePage.tsx's now-required field).
ALTER TABLE sites ADD COLUMN country text;
UPDATE sites SET country = 'NG' WHERE country IS NULL;
ALTER TABLE sites ALTER COLUMN country SET NOT NULL;

-- +goose Down
ALTER TABLE sites DROP COLUMN country;
