import { apiRequest } from "./client";
import { PageSchema, TelemetryPointSchema, type TelemetryPoint } from "./types";

export async function listSiteTelemetry(
  siteId: string,
  opts: { from?: string; to?: string; cursor?: string; limit?: number } = {}
): Promise<{ items: TelemetryPoint[]; nextCursor?: string }> {
  const data = await apiRequest<unknown>(`/v1/sites/${encodeURIComponent(siteId)}/telemetry`, {
    query: { from: opts.from, to: opts.to, cursor: opts.cursor, limit: opts.limit ?? 100 },
  });
  const parsed = PageSchema(TelemetryPointSchema).parse(data);
  return { items: parsed.items, nextCursor: parsed.next_cursor };
}
