import { apiRequest } from "./client";
import { EmissionFactorSchema, EmissionsSeriesSchema, type EmissionFactor, type EmissionsSeries } from "./types";
import type { AnalyticsRange } from "./analytics";

export async function getSiteEmissions(siteId: string, range: AnalyticsRange = {}): Promise<EmissionsSeries> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/analytics/emissions`, {
    query: range,
  });
  return EmissionsSeriesSchema.parse(data);
}

export async function getFleetEmissions(range: AnalyticsRange & { cohort_id?: string } = {}): Promise<EmissionsSeries> {
  const data = await apiRequest<unknown>("/v1/fleet/analytics/emissions", { query: range });
  return EmissionsSeriesSchema.parse(data);
}

export async function getEmissionFactor(): Promise<EmissionFactor> {
  const data = await apiRequest<unknown>("/v1/config/emission-factor");
  return EmissionFactorSchema.parse(data);
}

export interface SetEmissionFactorInput {
  kg_co2_per_kwh: number;
  country: string;
  source: string;
  effective_from: string;
}

export async function setEmissionFactor(input: SetEmissionFactorInput): Promise<EmissionFactor> {
  const data = await apiRequest<unknown>("/v1/config/emission-factor", { method: "POST", body: input });
  return EmissionFactorSchema.parse(data);
}
