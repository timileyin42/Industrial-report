package httpapi

import (
	"time"

	"github.com/timileyin42/zgnis-solar/internal/db"
)

// Response DTOs — db.* structs use pgtype wrappers that don't marshal to
// clean JSON, so every handler converts through one of these.

type siteResponse struct {
	SiteID            string    `json:"site_id"`
	Name              *string   `json:"name,omitempty"`
	CohortID          *string   `json:"cohort_id,omitempty"`
	Address           *string   `json:"address,omitempty"`
	GPSLat            *float64  `json:"gps_lat,omitempty"`
	GPSLng            *float64  `json:"gps_lng,omitempty"`
	InverterMakeModel *string   `json:"inverter_make_model,omitempty"`
	SystemSizeKW      *float64  `json:"system_size_kw,omitempty"`
	Timezone          string    `json:"timezone"`
	Country           string    `json:"country"`
	IsPrimary         bool      `json:"is_primary"`
	CreatedAt         time.Time `json:"created_at"`
}

func toSiteResponse(s db.Site) siteResponse {
	return siteResponse{
		SiteID:            s.SiteID,
		Name:              textPtr(s.Name),
		CohortID:          textPtr(s.CohortID),
		Address:           textPtr(s.Address),
		GPSLat:            float8Ptr(s.GpsLat),
		GPSLng:            float8Ptr(s.GpsLng),
		InverterMakeModel: textPtr(s.InverterMakeModel),
		SystemSizeKW:      numericPtr(s.SystemSizeKw),
		Timezone:          s.Timezone,
		Country:           s.Country,
		IsPrimary:         s.IsPrimary,
		CreatedAt:         s.CreatedAt.Time,
	}
}

type deviceResponse struct {
	DeviceID            string     `json:"device_id"`
	SiteID              *string    `json:"site_id,omitempty"`
	RevokedAt           *time.Time `json:"revoked_at,omitempty"`
	LastSeenAt          *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	SecretLastRotatedAt time.Time  `json:"secret_last_rotated_at"`
	InstallNotes        *string    `json:"install_notes,omitempty"`
}

func toDeviceResponse(d db.Device) deviceResponse {
	return deviceResponse{
		DeviceID:            d.DeviceID,
		SiteID:              textPtr(d.SiteID),
		RevokedAt:           timestamptzPtr(d.RevokedAt),
		LastSeenAt:          timestamptzPtr(d.LastSeenAt),
		CreatedAt:           d.CreatedAt.Time,
		SecretLastRotatedAt: d.SecretLastRotatedAt.Time,
		InstallNotes:        textPtr(d.InstallNotes),
	}
}

// deviceSecretResponse is the ONLY response shape that ever carries a
// plaintext secret — returned once, at registration or rotation.
type deviceSecretResponse struct {
	deviceResponse
	Secret string `json:"secret"`
}
