package domain

import (
	"fmt"
	"time"
)

// TelemetryPayload mirrors the MQTT message shape from concept note section 5.
// This is what the datalogger already normalizes Modbus register reads into —
// the ingestor never needs to know which inverter brand or register map
// produced these values. See AGENTS.md item 3.
type TelemetryPayload struct {
	DeviceID       string   `json:"device_id"`
	Timestamp      string   `json:"ts"` // ISO 8601 UTC
	PowerKW        float64  `json:"power_kw"`
	EnergyKWhTotal float64  `json:"energy_kwh_total"`
	VoltageV       *float64 `json:"voltage_v,omitempty"` // optional per spec
	Status         string   `json:"status"`
	// RSSI is listed in concept-note.md §6's own data model ("Optional
	// signal strength for diagnostics") and the telemetry table has
	// carried this column since migrations/0001_init.sql — but until
	// this field existed here, nothing ever actually read it out of an
	// incoming payload, so it was silently dropped regardless of what a
	// real datalogger sent. No validation rule depends on it; it's pure
	// diagnostic signal for spotting a device with a weak/marginal
	// connection before it goes fully offline.
	RSSI *int `json:"rssi,omitempty"`
	// Hybrid-inverter fields (Chisage/Felicity/Extra Power field
	// deployment). All optional, same
	// "missing is valid, not an error" rule as VoltageV: a grid-tie-only
	// inverter with no battery, or any device predating this field,
	// simply won't send these, and that's not a validation failure.
	PVPowerKW       *float64 `json:"pv_power_kw,omitempty"`     // solar-side output, distinct from AC output above
	BatterySOCPct   *float64 `json:"battery_soc_pct,omitempty"` // 0-100
	BatteryVoltageV *float64 `json:"battery_voltage_v,omitempty"`
	PVVoltageV      *float64 `json:"pv_voltage_v,omitempty"`
	OutputVoltageV  *float64 `json:"output_voltage_v,omitempty"`
}

const (
	StatusOK      = "ok"
	StatusFault   = "fault"
	StatusOffline = "offline"
)

// Validate applies the sanity checks called out in concept note section 11:
// impossible readings (negative energy, power far above plausible bounds)
// get rejected here rather than silently stored. maxPlausibleKW is computed
// by the caller via PowerCeilingKW, from the reporting site's system size.
func (p TelemetryPayload) Validate(maxPlausibleKW float64) (time.Time, error) {
	if p.DeviceID == "" {
		return time.Time{}, fmt.Errorf("device_id is required")
	}
	ts, err := time.Parse(time.RFC3339, p.Timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid ts %q: %w", p.Timestamp, err)
	}
	if p.EnergyKWhTotal < 0 {
		return time.Time{}, fmt.Errorf("energy_kwh_total cannot be negative: %v", p.EnergyKWhTotal)
	}
	if p.PowerKW < 0 {
		return time.Time{}, fmt.Errorf("power_kw cannot be negative: %v", p.PowerKW)
	}
	if p.PowerKW > maxPlausibleKW {
		return time.Time{}, fmt.Errorf("power_kw %v exceeds plausible ceiling %v", p.PowerKW, maxPlausibleKW)
	}
	if p.PVPowerKW != nil {
		if *p.PVPowerKW < 0 {
			return time.Time{}, fmt.Errorf("pv_power_kw cannot be negative: %v", *p.PVPowerKW)
		}
		// PV (solar-side) output is the source the AC output is drawn
		// from — same plausibility ceiling as power_kw, not a separate,
		// looser one, since a PV reading far above rated system size is
		// exactly as physically implausible as an AC one.
		if *p.PVPowerKW > maxPlausibleKW {
			return time.Time{}, fmt.Errorf("pv_power_kw %v exceeds plausible ceiling %v", *p.PVPowerKW, maxPlausibleKW)
		}
	}
	if p.BatterySOCPct != nil && (*p.BatterySOCPct < 0 || *p.BatterySOCPct > 100) {
		return time.Time{}, fmt.Errorf("battery_soc_pct %v out of range 0-100", *p.BatterySOCPct)
	}
	switch p.Status {
	case StatusOK, StatusFault, StatusOffline:
	case "":
		p.Status = StatusOK
	default:
		return time.Time{}, fmt.Errorf("unrecognized status %q", p.Status)
	}
	return ts, nil
}

// Provenance mirrors the telemetry.provenance enum — how a reading was
// obtained. Never blended silently: a backfilled reading must always be
// tagged as such, never mixed in as if it were metered. See CLAUDE.md.
type Provenance string

const (
	ProvenanceMetered    Provenance = "metered"
	ProvenanceEstimated  Provenance = "estimated"
	ProvenanceBackfilled Provenance = "backfilled"
)

// BackfillThreshold is the gap between a reading's own timestamp and the
// time it was actually received, beyond which it's classified as a device
// catching up after an outage rather than a merely-slow live message.
// 15 minutes = 3x the concept note's upper-bound target reporting interval
// (5 min) — generous enough to absorb normal network jitter/retry delay.
const BackfillThreshold = 15 * time.Minute

// ClassifyProvenance decides metered vs. backfilled from how stale a
// reading was by the time it arrived. Never returns Estimated — nothing in
// this pipeline synthesizes readings; that provenance value exists for a
// future gap-filling feature, not for anything the ingestor does today.
func ClassifyProvenance(ts, receivedAt time.Time) Provenance {
	if receivedAt.Sub(ts) > BackfillThreshold {
		return ProvenanceBackfilled
	}
	return ProvenanceMetered
}

// PowerCeilingHeadroom is the multiplier over a site's nameplate
// system_size_kw used as the plausibility ceiling. 1.5x: inverter AC output
// is commonly provisioned above DC array nameplate, and momentary readings
// can legitimately spike — 1.5x still firmly catches "far above system
// size" miswiring/garbage-data cases without false-positiving on normal
// short-term overproduction.
const PowerCeilingHeadroom = 1.5

// DefaultMaxPlausibleKW is the fallback ceiling for a device whose site has
// no system_size_kw set yet — same fixed value Phase 0 used, now explicitly
// a fallback rather than the norm.
const DefaultMaxPlausibleKW = 100.0

// PowerCeilingKW computes the plausibility ceiling for a reading, given the
// reporting site's system size (nil if not set).
func PowerCeilingKW(systemSizeKW *float64) float64 {
	if systemSizeKW == nil || *systemSizeKW <= 0 {
		return DefaultMaxPlausibleKW
	}
	return *systemSizeKW * PowerCeilingHeadroom
}

// Quality flags: review-worthy conditions on an otherwise-accepted, real
// reading. Distinct from Provenance (how a reading was obtained) — these
// describe something noteworthy about the reading's *content*.
const (
	QualityFlagEnergyReset        = "energy_reset"
	QualityFlagNightNonzeroOutput = "night_nonzero_output"
)

// DetectEnergyReset flags a reading whose cumulative energy counter is
// lower than the immediately-preceding reading (by ts, not arrival order) —
// a real hardware event (reboot, inverter replacement), not corrupt data.
// The caller must supply the previous reading by ts, never by arrival
// order, or a legitimate backfilled reading would false-positive here (see
// internal/registry — the query is `ts < $2 ORDER BY ts DESC LIMIT 1`).
// A nil previous (no earlier reading exists at all) is never a reset.
func DetectEnergyReset(previous *float64, current float64) bool {
	return previous != nil && current < *previous
}

// IsCoarseNight is a deliberately coarse, dependency-free day/night
// heuristic: local civil time between 20:00 and 05:00 counts as night. It
// will misjudge the ~30-60 minute window around actual dawn/dusk depending
// on season and latitude — acceptable because this only ever produces an
// advisory quality flag, never a rejection. A real sunrise/sunset or
// solar-position calculation (keyed on the site's gps_lat/gps_lng) would be
// more accurate but needs a new dependency; that tradeoff is deliberately
// not made here — this heuristic exists to satisfy concept note §11's
// day/night check using only what's already in the stack (site timezone).
func IsCoarseNight(localTime time.Time) bool {
	hour := localTime.Hour()
	return hour >= 20 || hour < 5
}
