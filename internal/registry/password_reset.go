package registry

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/email"
)

const passwordResetTTL = 1 * time.Hour

var ErrInvalidOrExpiredResetToken = errors.New("reset token is invalid or has expired")

type PasswordReset struct {
	q       *db.Queries
	sender  email.Sender
	baseURL string
}

func NewPasswordReset(q *db.Queries, sender email.Sender, baseURL string) *PasswordReset {
	return &PasswordReset{q: q, sender: sender, baseURL: baseURL}
}

// Request never reveals whether the email exists — same rationale as
// Users.Authenticate returning one error for both "no such user" and
// "wrong password". A non-existent email just silently sends nothing.
func (p *PasswordReset) Request(ctx context.Context, emailAddr string) error {
	user, err := p.q.GetUserByEmail(ctx, emailAddr)
	if err != nil {
		return nil
	}
	if user.DisabledAt.Valid {
		return nil
	}

	token, err := auth.GenerateDeviceSecret()
	if err != nil {
		return err
	}
	tokenHash, err := auth.HashSecret(token)
	if err != nil {
		return err
	}
	if _, err := p.q.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().UTC().Add(passwordResetTTL), Valid: true},
	}); err != nil {
		return err
	}

	subject, html := email.PasswordResetEmail(p.baseURL + "/reset-password?token=" + token)
	if err := p.sender.Send(ctx, user.Email, subject, html); err != nil {
		// A failed send here must not leak account-existence via a
		// different HTTP outcome than the "email doesn't exist" path
		// above — log loudly, still return nil to the caller.
		log.Printf("password reset: failed to send email to %s: %v", user.Email, err)
	}
	return nil
}

func (p *PasswordReset) Confirm(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	tokens, err := p.q.ListActivePasswordResetTokens(ctx)
	if err != nil {
		return err
	}

	var matched *db.PasswordResetToken
	for idx := range tokens {
		if auth.VerifySecret(tokens[idx].TokenHash, token) {
			matched = &tokens[idx]
			break
		}
	}
	if matched == nil {
		return ErrInvalidOrExpiredResetToken
	}

	hash, err := auth.HashSecret(newPassword)
	if err != nil {
		return err
	}
	if err := p.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{ID: matched.UserID, PasswordHash: hash}); err != nil {
		return err
	}
	return p.q.MarkPasswordResetTokenUsed(ctx, matched.ID)
}
