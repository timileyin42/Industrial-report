import { z } from "zod";
import { apiRequest } from "./client";
import { FleetSummarySchema, type FleetSummary } from "./types";

export async function getFleetSummary(): Promise<FleetSummary> {
  const data = await apiRequest<unknown>("/v1/fleet/summary");
  return FleetSummarySchema.parse(data);
}

// Live "how much power right now" — the most recent reading from every
// currently-online device, summed. Distinct from every other energy
// figure on this platform, which is historical/cumulative.
export async function getCurrentGeneration(): Promise<number> {
  const data = await apiRequest<unknown>("/v1/fleet/current-generation");
  return z.object({ current_power_kw: z.number() }).parse(data).current_power_kw;
}

const TopSiteSchema = z.object({
  site_id: z.string(),
  name: z.string().nullable().optional(),
  energy_kwh: z.number(),
  system_size_kw: z.number().nullable().optional(),
  specific_yield_kwh_per_kwp: z.number(),
});
export type TopSite = z.infer<typeof TopSiteSchema>;

export async function getTopSitesToday(limit = 10): Promise<TopSite[]> {
  const data = await apiRequest<unknown>("/v1/fleet/top-sites", { query: { limit } });
  return z.object({ items: z.array(TopSiteSchema) }).parse(data).items;
}

// "How long ago did the ingestor last see anything at all" — the
// pipeline-health signal this platform actually has a real data source
// for, not a synthetic uptime percentage. null means never.
export async function getIngestionStatus(): Promise<{ lastReceivedAt: string | null }> {
  const data = await apiRequest<unknown>("/v1/fleet/ingestion-status");
  const parsed = z.object({ last_received_at: z.string().nullable() }).parse(data);
  return { lastReceivedAt: parsed.last_received_at };
}

// Shape of GET /v1/fleet/health isn't specified in docs/openapi.yaml
// beyond "200 OK" — per the plan, this needs a read of
// internal/registry/fleet.go's FleetHealth type before building the
// table columns for it (Slice 2, not this slice's Fleet Dashboard —
// today's dashboard only calls getFleetSummary above).
