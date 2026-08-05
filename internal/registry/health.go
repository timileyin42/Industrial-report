package registry

import "time"

// computeOnlineAndDataGap is the single source of truth for the two-signal
// online/data_gap model (Phase 2) — previously this "10 minutes" logic was
// duplicated ad hoc in both Fleet.Summary's SQL and the status handler's Go.
//
//   - online:   is the device reachable at all right now (true-outage
//     signal) — based on last_contact_at, updated on every MQTT message the
//     ingestor receives regardless of validation outcome.
//   - data_gap: the device IS reachable, but its most recent *accepted*
//     reading (last_seen_at) is stale relative to the expected reporting
//     interval — it's behind, not down (e.g. replaying a buffered backlog).
//
// A device that is not online is never also reported as having a data gap
// — "unreachable" already implies stale data; data_gap only adds
// information when the device is otherwise online.
func computeOnlineAndDataGap(lastContactAt, lastSeenAt *time.Time, now time.Time, onlineThreshold, expectedInterval time.Duration) (online, dataGap bool) {
	online = lastContactAt != nil && now.Sub(*lastContactAt) < onlineThreshold
	if !online {
		return false, false
	}
	dataGap = lastSeenAt == nil || now.Sub(*lastSeenAt) > expectedInterval
	return online, dataGap
}
