package registry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
)

// Signup is the public self-service "create my own account" path —
// distinct from the existing operator-only Users.Create, which an
// operator uses to add accounts on their own side.
type Signup struct {
	q *db.Queries
}

func NewSignup(q *db.Queries) *Signup {
	return &Signup{q: q}
}

type SelfSignupInput struct {
	Email    string
	Password string
}

type SelfSignupResult struct {
	User db.User
	Site db.Site
}

// SelfSignup creates a fresh, initially empty site and a brand-new
// restricted user for it, in the only order the schema actually
// allows: the site first, then the user with site_id set from the
// start. The reverse (user first, site_id backfilled after) was tried
// and rejected by the database itself — migrations/0002's
// restricted_requires_site CHECK constraint rejects a restricted user
// row the instant it's inserted with a NULL site_id, so there's no
// legal intermediate state to backfill from; and users.site_id is
// itself an FK to sites, so the site has to exist first regardless.
//
// That leaves site creation without a real user to act as its
// audit-log actor yet (every other site-creation path in this
// codebase has an operator already logged in to attribute to). Rather
// than record that one entry under a fabricated or NULL actor,
// site.create is deferred and recorded immediately after the user
// exists, attributed to the new user's own real id — an honest
// "self-registered" attribution either way, just written a moment
// later than sites.Create normally would.
func (s *Signup) SelfSignup(ctx context.Context, in SelfSignupInput) (SelfSignupResult, error) {
	if err := validateRequired("email", in.Email); err != nil {
		return SelfSignupResult{}, err
	}
	if len(in.Password) < 8 {
		return SelfSignupResult{}, fmt.Errorf("password must be at least 8 characters")
	}
	siteID, err := newSelfSignupSiteID()
	if err != nil {
		return SelfSignupResult{}, err
	}
	if err := validateID("site_id", siteID); err != nil {
		return SelfSignupResult{}, err
	}

	hash, err := auth.HashSecret(in.Password)
	if err != nil {
		return SelfSignupResult{}, err
	}

	// Placeholder name/location — the Connect-Your-Inverter step that
	// follows immediately after signup overwrites these with the
	// vendor's own real plant name/coordinates once discovered, same as
	// cmd/pvpro-sync/cmd/solarman-sync already do for a genuinely new
	// plant. Country/timezone default to GB (this platform's other
	// self-service connector, solarman-sync, makes the same UK-first
	// assumption) until real location data arrives.
	site, err := s.q.CreateSite(ctx, db.CreateSiteParams{
		SiteID:   siteID,
		Name:     pgtype.Text{String: "New Site", Valid: true},
		Timezone: "Europe/London",
		Country:  "GB",
	})
	if err != nil {
		return SelfSignupResult{}, fmt.Errorf("create site for new signup: %w", err)
	}

	user, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Email:        in.Email,
		PasswordHash: hash,
		Role:         db.UserRole(domain.RoleRestricted),
		SiteID:       textOrNull(&siteID),
	})
	if err != nil {
		return SelfSignupResult{}, err
	}

	recordAction(ctx, s.q, user.ID, "site.create", "site", site.SiteID, nil)
	recordAction(ctx, s.q, user.ID, "user.self_signup", "user", user.Email, nil)

	return SelfSignupResult{User: user, Site: site}, nil
}

// newSelfSignupSiteID generates a short random, deterministic-format
// site_id (matching validateID's "letters, digits, -, _" rule) rather
// than deriving one from the email — an email can contain characters
// validateID rejects (@, .), and two signups shouldn't collide even if
// the same email were somehow reused.
func newSelfSignupSiteID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "SELF-" + hex.EncodeToString(b), nil
}
