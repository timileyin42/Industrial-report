package registry

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
	"github.com/timileyin42/zgnis-solar/internal/email"
)

const inviteTTL = 7 * 24 * time.Hour

var ErrInvalidOrExpiredInvite = errors.New("invite token is invalid or has expired")

// Invites replaces "operator sets a plaintext password directly" (still
// available via POST /v1/users, unchanged) with "operator invites, user
// sets their own password" — the user row is created immediately with an
// unusable generated password_hash, so the account exists (and can be
// site-scoped/role-assigned right away) but can't log in until accepted.
type Invites struct {
	q       *db.Queries
	sender  email.Sender
	baseURL string
}

func NewInvites(q *db.Queries, sender email.Sender, baseURL string) *Invites {
	return &Invites{q: q, sender: sender, baseURL: baseURL}
}

type CreateInviteInput struct {
	Email  string
	Role   domain.Role
	SiteID *string
}

func (i *Invites) Create(ctx context.Context, actorUserID int64, in CreateInviteInput) (db.User, error) {
	if err := validateRequired("email", in.Email); err != nil {
		return db.User{}, err
	}
	if in.Role != domain.RoleOperator && in.Role != domain.RoleRestricted {
		return db.User{}, errors.New("role must be 'operator' or 'restricted'")
	}
	if in.Role == domain.RoleRestricted {
		if in.SiteID == nil || *in.SiteID == "" {
			return db.User{}, errors.New("site_id is required for a restricted user")
		}
		if _, err := i.q.GetSite(ctx, *in.SiteID); err != nil {
			return db.User{}, ErrUnknownSite
		}
	}

	// Unusable password: a random secret nobody knows, hashed the same
	// way a real password would be — see migrations/0007's comment.
	unusable, err := auth.GenerateDeviceSecret()
	if err != nil {
		return db.User{}, err
	}
	passwordHash, err := auth.HashSecret(unusable)
	if err != nil {
		return db.User{}, err
	}

	user, err := i.q.CreateUser(ctx, db.CreateUserParams{
		Email:        in.Email,
		PasswordHash: passwordHash,
		Role:         db.UserRole(in.Role),
		SiteID:       textOrNull(in.SiteID),
	})
	if err != nil {
		return db.User{}, err
	}

	token, err := auth.GenerateDeviceSecret()
	if err != nil {
		return db.User{}, err
	}
	tokenHash, err := auth.HashSecret(token)
	if err != nil {
		return db.User{}, err
	}
	if _, err := i.q.CreateInvite(ctx, db.CreateInviteParams{
		UserID:          user.ID,
		TokenHash:       tokenHash,
		InvitedByUserID: pgtype.Int8{Int64: actorUserID, Valid: true},
		ExpiresAt:       pgtype.Timestamptz{Time: time.Now().UTC().Add(inviteTTL), Valid: true},
	}); err != nil {
		return db.User{}, err
	}

	subject, html := email.InviteEmail(i.baseURL + "/accept-invite?token=" + token)
	if err := i.sender.Send(ctx, user.Email, subject, html); err != nil {
		return db.User{}, err
	}

	recordAction(ctx, i.q, actorUserID, "user.invite", "user", user.Email, map[string]any{"role": string(in.Role)})
	return user, nil
}

// Accept verifies the token against every active (unexpired, unaccepted)
// invite — see migrations/0007's comment on why this can't be a direct
// lookup — sets the user's real password, and marks the invite accepted.
func (i *Invites) Accept(ctx context.Context, token, password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	invites, err := i.q.ListActiveInvites(ctx)
	if err != nil {
		return err
	}

	var matched *db.Invite
	for idx := range invites {
		if auth.VerifySecret(invites[idx].TokenHash, token) {
			matched = &invites[idx]
			break
		}
	}
	if matched == nil {
		return ErrInvalidOrExpiredInvite
	}

	hash, err := auth.HashSecret(password)
	if err != nil {
		return err
	}
	if err := i.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{ID: matched.UserID, PasswordHash: hash}); err != nil {
		return err
	}
	return i.q.MarkInviteAccepted(ctx, matched.ID)
}
