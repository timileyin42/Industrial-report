package registry

import (
	"context"
	"testing"
	"time"

	"github.com/timileyin42/zgnis-solar/internal/auth"
)

// TestRegisterDeviceStoresOnlyHash confirms Register never persists the
// plaintext secret it hands back — only a bcrypt hash the plaintext
// verifies against. This is the property RegisterDevicePage.tsx's "shown
// exactly once" UI promise depends on: if the plaintext were recoverable
// from the database, that promise would be a lie.
func TestRegisterDeviceStoresOnlyHash(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	sites := NewSites(q)
	devices := NewDevices(q, 10*time.Minute, 5*time.Minute)
	ctx := context.Background()

	siteID := uniqueID("site-secret-")
	createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: siteID, Name: "Secret Test Site", Timezone: "UTC", Country: "NG"})

	deviceID := uniqueID("dev-secret-")
	registered, err := devices.Register(ctx, 1, RegisterDeviceInput{DeviceID: deviceID, SiteID: siteID})
	if err != nil {
		t.Fatalf("register device: %v", err)
	}
	if registered.Secret == "" {
		t.Fatal("expected a non-empty plaintext secret from Register")
	}

	stored, err := devices.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("get device: %v", err)
	}
	if stored.SecretHash == registered.Secret {
		t.Fatal("secret_hash column contains the plaintext secret verbatim — must be hashed")
	}
	if !auth.VerifySecret(stored.SecretHash, registered.Secret) {
		t.Fatal("stored hash does not verify against the plaintext secret that was just issued")
	}
}

// TestRotateSecretInvalidatesOldOne confirms the old secret genuinely
// stops verifying once rotated — RegisterDevicePage/DevicesListPage both
// tell the operator "the old one no longer works" as a fact, not a hope.
func TestRotateSecretInvalidatesOldOne(t *testing.T) {
	q := testQueries(t)
	pool := testRawPool(t)
	sites := NewSites(q)
	devices := NewDevices(q, 10*time.Minute, 5*time.Minute)
	ctx := context.Background()

	siteID := uniqueID("site-rotate-")
	createTestSite(t, ctx, sites, pool, CreateSiteInput{SiteID: siteID, Name: "Rotate Test Site", Timezone: "UTC", Country: "NG"})

	deviceID := uniqueID("dev-rotate-")
	original, err := devices.Register(ctx, 1, RegisterDeviceInput{DeviceID: deviceID, SiteID: siteID})
	if err != nil {
		t.Fatalf("register device: %v", err)
	}

	rotated, err := devices.RotateSecret(ctx, 1, deviceID)
	if err != nil {
		t.Fatalf("rotate secret: %v", err)
	}
	if rotated.Secret == original.Secret {
		t.Fatal("rotation returned the same secret as before — expected a genuinely new one")
	}

	stored, err := devices.Get(ctx, deviceID)
	if err != nil {
		t.Fatalf("get device: %v", err)
	}
	if auth.VerifySecret(stored.SecretHash, original.Secret) {
		t.Fatal("the OLD secret still verifies against the stored hash after rotation — it must be invalidated")
	}
	if !auth.VerifySecret(stored.SecretHash, rotated.Secret) {
		t.Fatal("the NEW secret does not verify against the stored hash after rotation")
	}
}
