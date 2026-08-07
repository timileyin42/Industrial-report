import { apiRequest } from "./client";
import { AnomalyResultSchema, type AnomalyResult } from "./types";

export async function getSiteAnomalies(siteId: string, windowDays?: number, asOf?: string): Promise<AnomalyResult> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/analytics/anomalies`, {
    query: { window_days: windowDays, as_of: asOf },
  });
  return AnomalyResultSchema.parse(data);
}

// Operator-only on the backend — router.go's fleetAnomalies.
export async function getFleetAnomalies(windowDays?: number, asOf?: string): Promise<AnomalyResult> {
  const data = await apiRequest<unknown>("/v1/fleet/analytics/anomalies", { query: { window_days: windowDays, as_of: asOf } });
  return AnomalyResultSchema.parse(data);
}
