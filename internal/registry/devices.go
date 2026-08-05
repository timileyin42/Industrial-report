package registry

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/pagination"
)

type Devices struct {
	q                *db.Queries
	onlineThreshold  time.Duration
	expectedInterval time.Duration
}

func NewDevices(q *db.Queries, onlineThreshold, expectedInterval time.Duration) *Devices {
	return &Devices{q: q, onlineThreshold: onlineThreshold, expectedInterval: expectedInterval}
}

var ErrUnknownSite = errors.New("site does not exist")

type RegisterDeviceInput struct {
	DeviceID     string
	SiteID       string
	InstallNotes *string
}

// RegisteredDevice carries the plaintext secret — the only time it ever
// exists outside the caller's registration request. Never logged, never
// persisted, never returned again after this call.
type RegisteredDevice struct {
	Device db.Device
	Secret string
}

func (d *Devices) Register(ctx context.Context, actorUserID int64, in RegisterDeviceInput) (RegisteredDevice, error) {
	if err := validateID("device_id", in.DeviceID); err != nil {
		return RegisteredDevice{}, err
	}
	if err := validateRequired("site_id", in.SiteID); err != nil {
		return RegisteredDevice{}, err
	}
	if _, err := d.q.GetSite(ctx, in.SiteID); err != nil {
		return RegisteredDevice{}, ErrUnknownSite
	}

	secret, err := auth.GenerateDeviceSecret()
	if err != nil {
		return RegisteredDevice{}, err
	}
	hash, err := auth.HashSecret(secret)
	if err != nil {
		return RegisteredDevice{}, err
	}

	device, err := d.q.CreateDevice(ctx, db.CreateDeviceParams{
		DeviceID:     in.DeviceID,
		SiteID:       pgtype.Text{String: in.SiteID, Valid: true},
		SecretHash:   hash,
		InstallNotes: textOrNull(in.InstallNotes),
	})
	if err != nil {
		return RegisteredDevice{}, err
	}
	recordAction(ctx, d.q, actorUserID, "device.register", "device", device.DeviceID, map[string]any{"site_id": in.SiteID})
	return RegisteredDevice{Device: device, Secret: secret}, nil
}

func (d *Devices) Get(ctx context.Context, deviceID string) (db.Device, error) {
	return d.q.GetDevice(ctx, deviceID)
}

type DeviceStatus struct {
	DeviceID      string
	LastSeenAt    *time.Time
	LastContactAt *time.Time
	Online        bool
	DataGap       bool
	Revoked       bool
}

// Status is the consolidated online/data-gap computation (see health.go) —
// replaces the ad hoc Go math that used to live in the HTTP handler.
func (d *Devices) Status(ctx context.Context, deviceID string) (DeviceStatus, error) {
	device, err := d.q.GetDevice(ctx, deviceID)
	if err != nil {
		return DeviceStatus{}, err
	}
	lastSeenAt := timestamptzPtr(device.LastSeenAt)
	lastContactAt := timestamptzPtr(device.LastContactAt)
	online, dataGap := computeOnlineAndDataGap(lastContactAt, lastSeenAt, time.Now().UTC(), d.onlineThreshold, d.expectedInterval)
	return DeviceStatus{
		DeviceID:      device.DeviceID,
		LastSeenAt:    lastSeenAt,
		LastContactAt: lastContactAt,
		Online:        online,
		DataGap:       dataGap,
		Revoked:       device.RevokedAt.Valid,
	}, nil
}

// SiteIDForDevice resolves a device to its site — used by
// auth.RequireSiteAccess to scope restricted callers on device-facing
// routes (GET /v1/devices/:device_id, GET /v1/devices/:device_id/status).
func (d *Devices) SiteIDForDevice(ctx context.Context, deviceID string) (string, error) {
	device, err := d.q.GetDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}
	if !device.SiteID.Valid {
		return "", errors.New("device has no site mapping")
	}
	return device.SiteID.String, nil
}

func (d *Devices) List(ctx context.Context, siteFilter *string, cursorToken string, limit int) ([]db.Device, string, error) {
	if limit <= 0 || limit > 200 {
		limit = pagination.DefaultPageLimit
	}

	var cursorCreatedAt pgtype.Timestamptz
	var cursorDeviceID pgtype.Text
	if cursorToken != "" {
		c, err := pagination.Decode(cursorToken)
		if err != nil {
			return nil, "", err
		}
		cursorCreatedAt = pgtype.Timestamptz{Time: c.Time, Valid: true}
		cursorDeviceID = pgtype.Text{String: c.Tiebreak, Valid: true}
	}

	devices, err := d.q.ListDevices(ctx, db.ListDevicesParams{
		SiteFilter:      textOrNull(siteFilter),
		CursorCreatedAt: cursorCreatedAt,
		CursorDeviceID:  cursorDeviceID,
		PageLimit:       int32(limit),
	})
	if err != nil {
		return nil, "", err
	}

	next := ""
	if len(devices) == limit {
		last := devices[len(devices)-1]
		next, err = pagination.Encode(pagination.Cursor{Time: last.CreatedAt.Time, Tiebreak: last.DeviceID})
		if err != nil {
			return nil, "", err
		}
	}
	return devices, next, nil
}

func (d *Devices) Revoke(ctx context.Context, actorUserID int64, deviceID string) (db.Device, error) {
	device, err := d.q.RevokeDevice(ctx, deviceID)
	if err != nil {
		return db.Device{}, err
	}
	recordAction(ctx, d.q, actorUserID, "device.revoke", "device", deviceID, nil)
	return device, nil
}

func (d *Devices) RotateSecret(ctx context.Context, actorUserID int64, deviceID string) (RegisteredDevice, error) {
	secret, err := auth.GenerateDeviceSecret()
	if err != nil {
		return RegisteredDevice{}, err
	}
	hash, err := auth.HashSecret(secret)
	if err != nil {
		return RegisteredDevice{}, err
	}
	device, err := d.q.RotateDeviceSecret(ctx, db.RotateDeviceSecretParams{DeviceID: deviceID, SecretHash: hash})
	if err != nil {
		return RegisteredDevice{}, err
	}
	recordAction(ctx, d.q, actorUserID, "device.rotate_secret", "device", deviceID, nil)
	return RegisteredDevice{Device: device, Secret: secret}, nil
}
