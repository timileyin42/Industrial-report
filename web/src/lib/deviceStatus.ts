import type { Status } from "../components/status/StatusBadge";

// Mirrors the backend's ONLINE_THRESHOLD_MINUTES default (see README /
// cmd/api/main.go's envMinutes("ONLINE_THRESHOLD_MINUTES", 10)) — derived
// client-side from last_seen_at rather than an N+1 per-row call to
// GET /devices/:id/status on a paginated list.
export const ONLINE_THRESHOLD_MINUTES = 10;

export function deriveDeviceStatus(lastSeenAt: string | null | undefined, revokedAt: string | null | undefined): Status {
  if (revokedAt) return "maintenance"; // revoked is its own distinct state, not folded into "offline"
  if (!lastSeenAt) return "offline";
  const ageMinutes = (Date.now() - new Date(lastSeenAt).getTime()) / 60_000;
  return ageMinutes < ONLINE_THRESHOLD_MINUTES ? "online" : "offline";
}
