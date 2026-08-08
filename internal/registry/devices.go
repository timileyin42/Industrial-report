package registry

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
	"github.com/timileyin42/zgnis-solar/internal/mqttadmin"
	"github.com/timileyin42/zgnis-solar/internal/pagination"
)

type Devices struct {
	q                *db.Queries
	onlineThreshold  time.Duration
	expectedInterval time.Duration
	// mqttAdmin is nil when MQTT_ADMIN_USERNAME/PASSWORD aren't set —
	// the same "optional, additive infra" pattern as R2/email. A nil
	// client means broker credential sync is skipped (logged loudly),
	// not that Register/RotateSecret/Revoke fail.
	mqttAdmin *mqttadmin.Client
}

func NewDevices(q *db.Queries, onlineThreshold, expectedInterval time.Duration, mqttAdmin *mqttadmin.Client) *Devices {
	return &Devices{q: q, onlineThreshold: onlineThreshold, expectedInterval: expectedInterval, mqttAdmin: mqttAdmin}
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
	// BrokerSyncWarning is set when the device's own database record was
	// created successfully but syncing its credential into Mosquitto's
	// dynamic-security plugin failed (or wasn't configured at all) — the
	// device secret above is real and correctly hashed/stored, but the
	// device will not be able to authenticate to the broker until this
	// is resolved (see internal/mqttadmin). Never blocks the registration
	// itself, same principle as an email-send failure never rolling back
	// the invite it was sent for.
	BrokerSyncWarning *string
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

	result := RegisteredDevice{Device: device, Secret: secret}
	if d.mqttAdmin == nil {
		msg := "MQTT broker sync isn't configured (MQTT_ADMIN_USERNAME/PASSWORD unset) — this device's credential must be added to Mosquitto manually before it can publish telemetry."
		result.BrokerSyncWarning = &msg
	} else if err := d.mqttAdmin.CreateDevice(ctx, device.DeviceID, secret); err != nil {
		log.Printf("mqttadmin: failed to sync device %s to broker: %v", device.DeviceID, err)
		msg := "This device was registered, but syncing its credential into the MQTT broker failed — it will not be able to authenticate until this is retried (see server logs)."
		result.BrokerSyncWarning = &msg
	}
	return result, nil
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

// Revoke blocks a device at the app layer (ingestor's revoked_at check)
// regardless of whether the broker-side deletion below succeeds — that
// check is the primary enforcement, this is defense in depth, so a
// broker sync failure here is logged loudly but never fails the call.
func (d *Devices) Revoke(ctx context.Context, actorUserID int64, deviceID string) (db.Device, error) {
	device, err := d.q.RevokeDevice(ctx, deviceID)
	if err != nil {
		return db.Device{}, err
	}
	recordAction(ctx, d.q, actorUserID, "device.revoke", "device", deviceID, nil)
	if d.mqttAdmin != nil {
		if err := d.mqttAdmin.DeleteDevice(ctx, deviceID); err != nil {
			log.Printf("mqttadmin: failed to delete revoked device %s from broker: %v", deviceID, err)
		}
	}
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

	result := RegisteredDevice{Device: device, Secret: secret}
	if d.mqttAdmin == nil {
		msg := "MQTT broker sync isn't configured (MQTT_ADMIN_USERNAME/PASSWORD unset) — update this device's credential in Mosquitto manually or it will stop authenticating."
		result.BrokerSyncWarning = &msg
	} else if err := d.mqttAdmin.SetDevicePassword(ctx, deviceID, secret); err != nil {
		log.Printf("mqttadmin: failed to sync rotated secret for device %s to broker: %v", deviceID, err)
		msg := "The secret was rotated, but syncing it into the MQTT broker failed — this device will be unable to authenticate until this is retried (see server logs)."
		result.BrokerSyncWarning = &msg
	}
	return result, nil
}
