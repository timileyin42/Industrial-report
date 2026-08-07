import { apiRequest } from "./client";
import {
  HistoryComparisonSchema,
  FleetComparisonSchema,
  SegmentResultSchema,
  FleetTrendsSchema,
  type HistoryComparison,
  type FleetComparison,
  type SegmentResult,
  type FleetTrends,
} from "./types";
import type { Period } from "./analytics";

export async function getCompareHistory(siteId: string, period: Period = "monthly", asOf?: string): Promise<HistoryComparison> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/analytics/compare/history`, {
    query: { period, as_of: asOf },
  });
  return HistoryComparisonSchema.parse(data);
}

// Operator-only on the backend (leaks fleet-wide distribution by
// construction) — router.go's compareFleet.
export async function getCompareFleet(siteId: string, period: Period = "monthly", asOf?: string): Promise<FleetComparison> {
  const data = await apiRequest<unknown>("/v1/fleet/analytics/compare/fleet", {
    query: { site_id: siteId, period, as_of: asOf },
  });
  return FleetComparisonSchema.parse(data);
}

export type SegmentBy = "system_size_band" | "region";

export async function getBenchmarkSegments(
  segmentBy: SegmentBy = "system_size_band",
  period: Period = "monthly",
  asOf?: string
): Promise<SegmentResult> {
  const data = await apiRequest<unknown>("/v1/fleet/analytics/benchmark", {
    query: { segment_by: segmentBy, period, as_of: asOf },
  });
  return SegmentResultSchema.parse(data);
}

export async function getFleetTrends(period: Period = "monthly", from?: string, to?: string): Promise<FleetTrends> {
  const data = await apiRequest<unknown>("/v1/fleet/analytics/trends", { query: { period, from, to } });
  return FleetTrendsSchema.parse(data);
}
