package registry

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/syncengine"
)

// OAuthToken is an alias for syncengine's — kept as its own name here
// so callers of this package's exported functions don't need to import
// syncengine just to pass one through.
type OAuthToken = syncengine.OAuthToken

// VendorConnections is a customer's own manufacturer-cloud login
// (PV Pro/E-linter CSP first, more vendors later) — the foundation of
// the self-service "Connect Your Inverter" flow. Distinct from
// cmd/pvpro-sync's/cmd/solarman-sync's own single shared operator
// account: those keep working exactly as before, untouched by this.
type VendorConnections struct {
	q      *db.Queries
	cipher cipher.AEAD
}

// vendorCredential is the plaintext shape encrypted into
// vendor_connections.encrypted_credential. Exactly one of the two pairs
// below is populated, depending on the connection's provider's
// AuthType — a password-form vendor's login, or an OAuth vendor's
// access/refresh token. One struct, one column, one encrypt/decrypt
// path regardless of which shape it holds; callers know which fields
// to expect from the connection's own provider name.
type vendorCredential struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`

	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// NewVendorConnections requires VENDOR_CREDENTIAL_ENCRYPTION_KEY (a
// 32-byte AES-256 key, base64-encoded) — same "env var, never
// hardcoded" rule already followed for every other secret in this
// project. Fails fast at startup rather than silently storing
// credentials insecurely or panicking mid-request.
func NewVendorConnections(q *db.Queries) (*VendorConnections, error) {
	keyB64 := os.Getenv("VENDOR_CREDENTIAL_ENCRYPTION_KEY")
	if keyB64 == "" {
		return nil, errors.New("VENDOR_CREDENTIAL_ENCRYPTION_KEY not set")
	}
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("VENDOR_CREDENTIAL_ENCRYPTION_KEY is not valid base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("VENDOR_CREDENTIAL_ENCRYPTION_KEY must decode to 32 bytes (AES-256), got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &VendorConnections{q: q, cipher: gcm}, nil
}

func (v *VendorConnections) encrypt(cred vendorCredential) ([]byte, error) {
	plaintext, err := json.Marshal(cred)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, v.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	// Nonce prepended to the ciphertext (standard AES-GCM convention) —
	// one column holds everything needed to decrypt, nothing split
	// across two places to keep in sync.
	return v.cipher.Seal(nonce, nonce, plaintext, nil), nil
}

func (v *VendorConnections) decrypt(encrypted []byte) (vendorCredential, error) {
	nonceSize := v.cipher.NonceSize()
	if len(encrypted) < nonceSize {
		return vendorCredential{}, errors.New("encrypted credential too short")
	}
	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	plaintext, err := v.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return vendorCredential{}, fmt.Errorf("decrypt credential: %w", err)
	}
	var cred vendorCredential
	if err := json.Unmarshal(plaintext, &cred); err != nil {
		return vendorCredential{}, err
	}
	return cred, nil
}

type CreateConnectionInput struct {
	SiteID   string
	Provider string
	Email    string
	Password string
}

type VendorConnection struct {
	ID           int64
	SiteID       string
	Provider     string
	ExternalRef  *string
	Status       string
	LastSyncedAt *time.Time
	LastError    *string
	CreatedAt    time.Time
}

// Create stores a new connection encrypted and returns it — the
// plaintext password is never logged, never returned, and never
// touches any other table.
func (v *VendorConnections) Create(ctx context.Context, actorUserID int64, in CreateConnectionInput) (VendorConnection, error) {
	if err := validateRequired("provider", in.Provider); err != nil {
		return VendorConnection{}, err
	}
	if err := validateRequired("email", in.Email); err != nil {
		return VendorConnection{}, err
	}
	if err := validateRequired("password", in.Password); err != nil {
		return VendorConnection{}, err
	}
	encrypted, err := v.encrypt(vendorCredential{Email: in.Email, Password: in.Password})
	if err != nil {
		return VendorConnection{}, fmt.Errorf("encrypt credential: %w", err)
	}
	row, err := v.q.CreateVendorConnection(ctx, db.CreateVendorConnectionParams{
		SiteID:              in.SiteID,
		Provider:            in.Provider,
		EncryptedCredential: encrypted,
		CreatedByUserID:     pgtype.Int8{Int64: actorUserID, Valid: actorUserID > 0},
	})
	if err != nil {
		return VendorConnection{}, err
	}
	recordAction(ctx, v.q, actorUserID, "vendor_connection.create", "site", in.SiteID, map[string]any{"provider": in.Provider})
	return VendorConnection{
		ID: row.ID, SiteID: row.SiteID, Provider: row.Provider,
		ExternalRef: textPtr(row.ExternalRef), Status: row.Status,
		LastSyncedAt: timestamptzPtr(row.LastSyncedAt), LastError: textPtr(row.LastError),
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

// CreateFromOAuth stores a connection whose credential is a vendor's
// OAuth token rather than an email/password — the counterpart to
// Create for an OAuthProvider. Called from the OAuth callback handler,
// which has no authenticated session to attribute the action to (the
// vendor redirected the browser here, not our own SPA) — see
// recordSystemAction's own doc comment.
func (v *VendorConnections) CreateFromOAuth(ctx context.Context, siteID, provider string, token OAuthToken) (VendorConnection, error) {
	if err := validateRequired("provider", provider); err != nil {
		return VendorConnection{}, err
	}
	if err := validateRequired("access_token", token.AccessToken); err != nil {
		return VendorConnection{}, err
	}
	encrypted, err := v.encrypt(vendorCredential{
		AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, ExpiresAt: nonZeroTime(token.ExpiresAt),
	})
	if err != nil {
		return VendorConnection{}, fmt.Errorf("encrypt credential: %w", err)
	}
	row, err := v.q.CreateVendorConnection(ctx, db.CreateVendorConnectionParams{
		SiteID:              siteID,
		Provider:            provider,
		EncryptedCredential: encrypted,
		CreatedByUserID:     pgtype.Int8{Valid: false},
	})
	if err != nil {
		return VendorConnection{}, err
	}
	recordSystemAction(ctx, v.q, "vendor_connection.create_oauth", "site", siteID, map[string]any{"provider": provider})
	return VendorConnection{
		ID: row.ID, SiteID: row.SiteID, Provider: row.Provider,
		ExternalRef: textPtr(row.ExternalRef), Status: row.Status,
		LastSyncedAt: timestamptzPtr(row.LastSyncedAt), LastError: textPtr(row.LastError),
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func nonZeroTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (v *VendorConnections) ListForSite(ctx context.Context, siteID string) ([]VendorConnection, error) {
	rows, err := v.q.ListVendorConnectionsForSite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	out := make([]VendorConnection, 0, len(rows))
	for _, r := range rows {
		out = append(out, VendorConnection{
			ID: r.ID, SiteID: r.SiteID, Provider: r.Provider,
			ExternalRef: textPtr(r.ExternalRef), Status: r.Status,
			LastSyncedAt: timestamptzPtr(r.LastSyncedAt), LastError: textPtr(r.LastError),
			CreatedAt: r.CreatedAt.Time,
		})
	}
	return out, nil
}

func (v *VendorConnections) Revoke(ctx context.Context, actorUserID int64, siteID string, connectionID int64) error {
	if err := v.q.RevokeVendorConnection(ctx, db.RevokeVendorConnectionParams{ID: connectionID, SiteID: siteID}); err != nil {
		return err
	}
	recordAction(ctx, v.q, actorUserID, "vendor_connection.revoke", "site", siteID, map[string]any{"connection_id": connectionID})
	return nil
}

// ActiveConnection is what cmd/vendor-sync's poll loop consumes —
// decrypted credential included, since the sync process is the one
// legitimate internal caller that needs the plaintext to actually log
// into the vendor's API. Email/Password are set for a password-form
// connection, AccessToken/RefreshToken/ExpiresAt for an OAuth one —
// vendor-sync knows which fields to expect from the connection's own
// Provider (via its AuthType), same as vendorCredential's storage shape.
type ActiveConnection struct {
	ID           int64
	SiteID       string
	Provider     string
	Email        string
	Password     string
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
	ExternalRef  *string
	LastSyncedAt *time.Time
}

func (v *VendorConnections) ListActiveDecrypted(ctx context.Context) ([]ActiveConnection, error) {
	rows, err := v.q.ListActiveVendorConnections(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ActiveConnection, 0, len(rows))
	for _, r := range rows {
		cred, err := v.decrypt(r.EncryptedCredential)
		if err != nil {
			// One corrupt/undecryptable row must not break every other
			// customer's sync — log and skip, same isolation rule as a
			// live provider failure.
			continue
		}
		out = append(out, ActiveConnection{
			ID: r.ID, SiteID: r.SiteID, Provider: r.Provider,
			Email: cred.Email, Password: cred.Password,
			AccessToken: cred.AccessToken, RefreshToken: cred.RefreshToken, ExpiresAt: cred.ExpiresAt,
			ExternalRef: textPtr(r.ExternalRef), LastSyncedAt: timestamptzPtr(r.LastSyncedAt),
		})
	}
	return out, nil
}

func (v *VendorConnections) MarkSynced(ctx context.Context, id int64, externalRef *string) error {
	return v.q.MarkVendorConnectionSynced(ctx, db.MarkVendorConnectionSyncedParams{
		ID:           id,
		LastSyncedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
		ExternalRef:  textOrNull(externalRef),
	})
}

// UpdateOAuthToken persists a refreshed access/refresh token pair —
// called by cmd/vendor-sync right after RefreshToken succeeds, before
// the new access token is used, so a rotated refresh token (most
// vendors rotate it on every use) is never lost to a crash between
// refreshing and the next poll cycle.
func (v *VendorConnections) UpdateOAuthToken(ctx context.Context, id int64, token OAuthToken) error {
	encrypted, err := v.encrypt(vendorCredential{
		AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, ExpiresAt: nonZeroTime(token.ExpiresAt),
	})
	if err != nil {
		return fmt.Errorf("encrypt refreshed credential: %w", err)
	}
	return v.q.UpdateVendorConnectionCredential(ctx, db.UpdateVendorConnectionCredentialParams{
		ID: id, EncryptedCredential: encrypted,
	})
}

func (v *VendorConnections) MarkError(ctx context.Context, id int64, errMsg string) error {
	return v.q.MarkVendorConnectionError(ctx, db.MarkVendorConnectionErrorParams{
		ID: id, LastError: pgtype.Text{String: errMsg, Valid: true},
	})
}
