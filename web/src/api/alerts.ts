import { z } from "zod";
import { apiRequest } from "./client";
import { AlertSchema, type Alert } from "./types";

export async function listFleetAlerts(limit = 50): Promise<Alert[]> {
  const data = await apiRequest<unknown>("/v1/fleet/alerts", { query: { limit } });
  return z.object({ items: z.array(AlertSchema) }).parse(data).items;
}
