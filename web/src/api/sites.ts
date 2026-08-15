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
  // Resolves which grid emission factor this site's CO2-offset reporting
  // uses (backend migrations/0010_site_country.sql) — required, no
  // server-side default.
  country: string;
}

export async function updateSiteCountry(siteId: string, country: string): Promise<Site> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/country`, {
    method: "PATCH",
    body: { country },
  });
  return SiteSchema.parse(data);
}

// Sets/corrects a site's GPS coordinates after creation — e.g. a
// cloud-imported site registered before its precise lat/lng was known.
export async function updateSiteLocation(siteId: string, gpsLat: number, gpsLng: number): Promise<Site> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/location`, {
    method: "PATCH",
    body: { gps_lat: gpsLat, gps_lng: gpsLng },
  });
  return SiteSchema.parse(data);
}

// Marks this site as the fleet's one primary/home site — what the Fleet
// Dashboard's weather widget resolves its location from. Setting a new
// one clears the flag from whichever site held it before (server-side,
// atomic — see internal/registry/sites.go SetPrimary).
export async function setSitePrimary(siteId: string): Promise<Site> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/primary`, { method: "PATCH" });
  return SiteSchema.parse(data);
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

// Backs the Fleet Dashboard's weather widget. Throws (404, surfaced as
// ApiError) when no site has been marked primary yet — callers must show
// an explicit "no primary site set" state, never silently fall back to
// picking any site (see FleetDashboardPage.tsx's WeatherWidget wiring).
export async function getPrimarySite(): Promise<Site> {
  const data = await apiRequest<unknown>("/v1/sites/primary");
  return SiteSchema.parse(data);
}

export async function createSite(input: CreateSiteInput): Promise<Site> {
  const data = await apiRequest<unknown>("/v1/sites", { method: "POST", body: input });
  return SiteSchema.parse(data);
}
