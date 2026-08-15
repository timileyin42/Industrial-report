package registry

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/pagination"
)

type Sites struct {
	q *db.Queries
}

func NewSites(q *db.Queries) *Sites {
	return &Sites{q: q}
}

type CreateSiteInput struct {
	SiteID            string
	Name              string
	Address           *string
	GPSLat            *float64
	GPSLng            *float64
	InverterMakeModel *string
	SystemSizeKW      *float64
	InstallDate       *time.Time
	Timezone          string
	CohortID          *string
	// Country resolves which grid emission factor CO2-offset reporting
	// uses for this site (internal/registry/emissions.go) — required,
	// no default, per migrations/0010_site_country.sql's reasoning:
	// guessing a country is exactly the mistake GRID_COUNTRY's old
	// single global default made.
	Country string
}

func (s *Sites) Create(ctx context.Context, actorUserID int64, in CreateSiteInput) (db.Site, error) {
	if err := validateID("site_id", in.SiteID); err != nil {
		return db.Site{}, err
	}
	if err := validateRequired("name", in.Name); err != nil {
		return db.Site{}, err
	}
	if err := validateRequired("timezone", in.Timezone); err != nil {
		return db.Site{}, err
	}
	if err := validateCountry("country", in.Country); err != nil {
		return db.Site{}, err
	}
	if in.SystemSizeKW != nil && *in.SystemSizeKW < 0 {
		return db.Site{}, errNegative("system_size_kw")
	}

	site, err := s.q.CreateSite(ctx, db.CreateSiteParams{
		SiteID:            in.SiteID,
		Name:              pgtype.Text{String: in.Name, Valid: true},
		Address:           textOrNull(in.Address),
		GpsLat:            float8OrNull(in.GPSLat),
		GpsLng:            float8OrNull(in.GPSLng),
		InverterMakeModel: textOrNull(in.InverterMakeModel),
		SystemSizeKw:      numericOrNull(in.SystemSizeKW),
		InstallDate:       dateOrNull(in.InstallDate),
		Timezone:          in.Timezone,
		CohortID:          textOrNull(in.CohortID),
		Country:           in.Country,
	})
	if err != nil {
		return db.Site{}, err
	}
	recordAction(ctx, s.q, actorUserID, "site.create", "site", site.SiteID, nil)
	return site, nil
}

// UpdateCountry corrects a site's country after creation — needed
// because migrating in this column had to backfill every pre-existing
// site to 'NG' (see migrations/0010_site_country.sql); any site that
// backfill guessed wrong needs a real way to be corrected.
func (s *Sites) UpdateCountry(ctx context.Context, actorUserID int64, siteID, country string) (db.Site, error) {
	if err := validateCountry("country", country); err != nil {
		return db.Site{}, err
	}
	site, err := s.q.UpdateSiteCountry(ctx, db.UpdateSiteCountryParams{SiteID: siteID, Country: country})
	if err != nil {
		return db.Site{}, err
	}
	recordAction(ctx, s.q, actorUserID, "site.update_country", "site", site.SiteID, map[string]any{"country": country})
	return site, nil
}

// UpdateLocation sets/corrects a site's GPS coordinates after creation —
// e.g. a cloud-imported site registered before its precise lat/lng was
// known (see cmd/pvpro-sync, whose plant-summary lookup only carries
// province/country-level coordinates; the precise per-plant lat/lng
// needs a separate, per-plant detail call).
func (s *Sites) UpdateLocation(ctx context.Context, actorUserID int64, siteID string, lat, lng float64) (db.Site, error) {
	site, err := s.q.UpdateSiteLocation(ctx, db.UpdateSiteLocationParams{SiteID: siteID, GpsLat: float8OrNull(&lat), GpsLng: float8OrNull(&lng)})
	if err != nil {
		return db.Site{}, err
	}
	recordAction(ctx, s.q, actorUserID, "site.update_location", "site", site.SiteID, map[string]any{"gps_lat": lat, "gps_lng": lng})
	return site, nil
}

// SetPrimary marks siteID as the fleet's one primary/home site, clearing
// the flag from whichever site (if any) held it before. This is what the
// Fleet Dashboard's weather widget resolves its location from — an
// explicit choice, never an implicit "whatever site was created most
// recently" guess (see FleetDashboardPage.tsx / WeatherWidget wiring).
// Two sequential statements rather than one transaction: the unique
// partial index on sites.is_primary (migrations/0011_primary_site.sql)
// is what actually guarantees at most one primary site, even under a
// race, and this action is infrequent/admin-only, not a hot path worth
// adding transaction plumbing for.
func (s *Sites) SetPrimary(ctx context.Context, actorUserID int64, siteID string) (db.Site, error) {
	if err := s.q.UnsetAllPrimarySites(ctx); err != nil {
		return db.Site{}, err
	}
	site, err := s.q.SetSitePrimary(ctx, siteID)
	if err != nil {
		return db.Site{}, err
	}
	recordAction(ctx, s.q, actorUserID, "site.set_primary", "site", site.SiteID, nil)
	return site, nil
}

// PrimarySite returns the fleet's current primary site, or
// pgx.ErrNoRows if none has ever been set — callers (the Fleet Dashboard
// API) must treat that as "no default configured yet," never silently
// fall back to picking any site.
func (s *Sites) PrimarySite(ctx context.Context) (db.Site, error) {
	return s.q.GetPrimarySite(ctx)
}

func (s *Sites) Get(ctx context.Context, siteID string) (db.Site, error) {
	return s.q.GetSite(ctx, siteID)
}

type Cohort struct {
	CohortID        string
	SiteCount       int64
	TotalCapacityKW float64
}

// ListCohorts derives the fleet's cohorts from what's actually assigned
// on sites — there's no dedicated cohorts table, so a cohort with zero
// sites simply doesn't exist as a listable entity, which is the correct
// behavior for a free-text grouping field rather than a managed one.
func (s *Sites) ListCohorts(ctx context.Context) ([]Cohort, error) {
	rows, err := s.q.ListCohorts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Cohort, 0, len(rows))
	for _, r := range rows {
		capacity := numericToFloat(r.TotalCapacityKw)
		var capacityVal float64
		if capacity != nil {
			capacityVal = *capacity
		}
		out = append(out, Cohort{CohortID: r.CohortID.String, SiteCount: r.SiteCount, TotalCapacityKW: capacityVal})
	}
	return out, nil
}

// List returns a page of sites plus the cursor for the next page (empty
// string = no more pages). siteFilter restricts to a single site (used to
// scope a restricted-role caller); nil means "all sites."
func (s *Sites) List(ctx context.Context, siteFilter *string, cursorToken string, limit int) ([]db.Site, string, error) {
	if limit <= 0 || limit > 200 {
		limit = pagination.DefaultPageLimit
	}

	var cursorCreatedAt pgtype.Timestamptz
	var cursorSiteID pgtype.Text
	if cursorToken != "" {
		c, err := pagination.Decode(cursorToken)
		if err != nil {
			return nil, "", err
		}
		cursorCreatedAt = pgtype.Timestamptz{Time: c.Time, Valid: true}
		cursorSiteID = pgtype.Text{String: c.Tiebreak, Valid: true}
	}

	sites, err := s.q.ListSites(ctx, db.ListSitesParams{
		SiteFilter:      textOrNull(siteFilter),
		CursorCreatedAt: cursorCreatedAt,
		CursorSiteID:    cursorSiteID,
		PageLimit:       int32(limit),
	})
	if err != nil {
		return nil, "", err
	}

	next := ""
	if len(sites) == limit {
		last := sites[len(sites)-1]
		next, err = pagination.Encode(pagination.Cursor{Time: last.CreatedAt.Time, Tiebreak: last.SiteID})
		if err != nil {
			return nil, "", err
		}
	}
	return sites, next, nil
}
