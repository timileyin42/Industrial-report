package syncengine

import (
	"errors"
	"strings"
)

// ErrInvalidCredentials should wrap a Provider's login error whenever
// the vendor's own response confirms the account/password itself is
// wrong — never wrapped for a network hiccup, rate limit, or vendor
// outage, since those are worth retrying. cmd/vendor-sync checks for
// this with errors.Is to stop polling a connection instead of retrying
// forever.
//
// Retrying a genuinely wrong password on every poll cycle isn't just
// wasteful: Deye's own login endpoint has been observed live to lock
// out repeated attempts ("Incorrect password, 2 attempt remaining"),
// meaning naive infinite retry could lock a real customer out of their
// own vendor account over nothing worse than a typo during Connect.
var ErrInvalidCredentials = errors.New("invalid vendor credentials")

// looksLikeCredentialError does a best-effort match against a vendor's
// own free-text login-failure message. Confirmed live against all
// three adapters built so far, each with its own phrasing and no
// shared machine-readable error taxonomy: PV Pro's "Account or
// password error", Felicity's "User does not exist"/"Wrong password",
// Deye's "Incorrect password, N attempt remaining". Extend this list
// only with a phrase confirmed the same way — live, not guessed —
// since a false positive here would stop retrying a connection that
// might have actually recovered on its own.
func looksLikeCredentialError(msg string) bool {
	lower := strings.ToLower(msg)
	for _, phrase := range []string{
		"password",
		"does not exist",
		"account not activated",
		"invalid_grant",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}
