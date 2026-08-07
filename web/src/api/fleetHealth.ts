import { apiRequest } from "./client";
import { FleetHealthSchema, type FleetHealth } from "./types";

export async function getFleetHealth(cursor?: string, limit = 50): Promise<FleetHealth> {
  const data = await apiRequest<unknown>("/v1/fleet/health", { query: { cursor, limit } });
  return FleetHealthSchema.parse(data);
}
