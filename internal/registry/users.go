package registry

import (
	"context"
	"errors"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/domain"
)

type Users struct {
	q *db.Queries
}

func NewUsers(q *db.Queries) *Users {
	return &Users{q: q}
}

var ErrInvalidCredentials = errors.New("invalid email or password")

type CreateUserInput struct {
	Email    string
	Password string
	Role     domain.Role
	SiteID   *string
}

func (u *Users) Create(ctx context.Context, actorUserID int64, in CreateUserInput) (db.User, error) {
	if err := validateRequired("email", in.Email); err != nil {
		return db.User{}, err
	}
	if len(in.Password) < 8 {
		return db.User{}, errors.New("password must be at least 8 characters")
	}
	if in.Role != domain.RoleOperator && in.Role != domain.RoleRestricted {
		return db.User{}, errors.New("role must be 'operator' or 'restricted'")
	}
	if in.Role == domain.RoleRestricted {
		if in.SiteID == nil || *in.SiteID == "" {
			return db.User{}, errors.New("site_id is required for a restricted user")
		}
		if _, err := u.q.GetSite(ctx, *in.SiteID); err != nil {
			return db.User{}, ErrUnknownSite
		}
	}

	hash, err := auth.HashSecret(in.Password)
	if err != nil {
		return db.User{}, err
	}

	user, err := u.q.CreateUser(ctx, db.CreateUserParams{
		Email:        in.Email,
		PasswordHash: hash,
		Role:         db.UserRole(in.Role),
		SiteID:       textOrNull(in.SiteID),
	})
	if err != nil {
		return db.User{}, err
	}
	recordAction(ctx, u.q, actorUserID, "user.create", "user", user.Email, map[string]any{"role": string(in.Role)})
	return user, nil
}

// Authenticate verifies email+password and returns the matching user.
// Returns ErrInvalidCredentials for both "no such user" and "wrong
// password" — the caller must not be able to distinguish the two.
func (u *Users) Authenticate(ctx context.Context, email, password string) (db.User, error) {
	user, err := u.q.GetUserByEmail(ctx, email)
	if err != nil {
		return db.User{}, ErrInvalidCredentials
	}
	if user.DisabledAt.Valid {
		return db.User{}, ErrInvalidCredentials
	}
	if !auth.VerifySecret(user.PasswordHash, password) {
		return db.User{}, ErrInvalidCredentials
	}
	return user, nil
}

// RecordLogin writes the auth.login entry to the admin audit trail.
func (u *Users) RecordLogin(ctx context.Context, userID int64) {
	recordAction(ctx, u.q, userID, "auth.login", "user", "", nil)
}
