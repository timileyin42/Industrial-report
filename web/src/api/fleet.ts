import { apiRequest } from "./client";
import { FleetSummarySchema, type FleetSummary } from "./types";

export async function getFleetSummary(): Promise<FleetSummary> {
  const data = await apiRequest<unknown>("/v1/fleet/summary");
  return FleetSummarySchema.parse(data);
}

// Shape of GET /v1/fleet/health isn't specified in docs/openapi.yaml
// beyond "200 OK" — per the plan, this needs a read of
// internal/registry/fleet.go's FleetHealth type before building the
// table columns for it (Slice 2, not this slice's Fleet Dashboard —
// today's dashboard only calls getFleetSummary above).
