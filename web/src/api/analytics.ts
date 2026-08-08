import { apiRequest } from "./client";
import {
  EnergySeriesSchema,
  YieldSeriesSchema,
  PeakSeriesSchema,
  CapacityFactorSeriesSchema,
  PerformanceRatioSeriesSchema,
  type EnergySeries,
  type YieldSeries,
  type PeakSeries,
  type CapacityFactorSeries,
  type PerformanceRatioSeries,
} from "./types";

export type Period = "daily" | "weekly" | "monthly";

export interface AnalyticsRange {
  period?: Period;
  from?: string;
  to?: string;
  [key: string]: string | number | undefined;
}

export async function getSiteEnergy(siteId: string, range: AnalyticsRange = {}): Promise<EnergySeries> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/analytics/energy`, {
    query: range,
  });
  return EnergySeriesSchema.parse(data);
}

export async function getFleetEnergy(range: AnalyticsRange & { cohort_id?: string } = {}): Promise<EnergySeries> {
  const data = await apiRequest<unknown>("/v1/fleet/analytics/energy", { query: range });
  return EnergySeriesSchema.parse(data);
}

export async function getSiteSpecificYield(siteId: string, range: AnalyticsRange = {}): Promise<YieldSeries> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/analytics/specific-yield`, {
    query: range,
  });
  return YieldSeriesSchema.parse(data);
}

export async function getFleetSpecificYield(range: AnalyticsRange & { cohort_id?: string } = {}): Promise<YieldSeries> {
  const data = await apiRequest<unknown>("/v1/fleet/analytics/specific-yield", { query: range });
  return YieldSeriesSchema.parse(data);
}

export async function getSitePeak(siteId: string, range: Omit<AnalyticsRange, "period"> = {}): Promise<PeakSeries> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/analytics/peak`, {
    query: range,
  });
  return PeakSeriesSchema.parse(data);
}

export async function getSiteCapacityFactor(siteId: string, range: AnalyticsRange = {}): Promise<CapacityFactorSeries> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/analytics/capacity-factor`, {
    query: range,
  });
  return CapacityFactorSeriesSchema.parse(data);
}

export async function getSitePerformanceRatio(siteId: string, range: AnalyticsRange = {}): Promise<PerformanceRatioSeries> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/analytics/performance-ratio`, {
    query: range,
  });
  return PerformanceRatioSeriesSchema.parse(data);
}

export async function getFleetPerformanceRatio(range: AnalyticsRange & { cohort_id?: string } = {}): Promise<PerformanceRatioSeries> {
  const data = await apiRequest<unknown>("/v1/fleet/analytics/performance-ratio", { query: range });
  return PerformanceRatioSeriesSchema.parse(data);
}
