import { apiRequest } from "./client";
import { PageSchema, SiteSchema, type Site } from "./types";

export interface CreateSiteInput {
  site_id: string;
  name: string;
  address?: string;
  gps_lat?: number;
  gps_lng?: number;
  inverter_make_model?: string;
  system_size_kw?: number;
  timezone: string;
  cohort_id?: string;
}

export async function listSites(cursor?: string, limit = 50): Promise<{ items: Site[]; nextCursor?: string }> {
  const data = await apiRequest<unknown>("/v1/sites", { query: { cursor, limit } });
  const parsed = PageSchema(SiteSchema).parse(data);
  return { items: parsed.items, nextCursor: parsed.next_cursor };
}

export async function getSite(siteId: string): Promise<Site> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}`);
  return SiteSchema.parse(data);
}

export async function createSite(input: CreateSiteInput): Promise<Site> {
  const data = await apiRequest<unknown>("/v1/sites", { method: "POST", body: input });
  return SiteSchema.parse(data);
}
