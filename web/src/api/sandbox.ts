import { z } from "zod";
import { ApiError } from "./types";

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "http://localhost:8080";

// Deliberately not using api/client.ts's apiRequest — that wrapper
// injects an Authorization header and reacts to 401 by redirecting to
// /login, neither of which makes sense here: /v1/sandbox is public, no
// account involved at all, and a logged-in operator poking at this
// feature shouldn't accidentally send their own token to it.
export const SandboxReadingSchema = z.object({
  row_number: z.number(),
  ts: z.string().nullable().optional(),
  power_kw: z.number().nullable().optional(),
  energy_kwh_total: z.number().nullable().optional(),
  voltage_v: z.number().nullable().optional(),
  rssi: z.number().nullable().optional(),
  status: z.string().optional(),
  accepted: z.boolean(),
  rejection_reason: z.string().optional(),
  provenance: z.string().optional(),
  is_reset: z.boolean(),
});
export type SandboxReading = z.infer<typeof SandboxReadingSchema>;

export const SandboxResultSchema = z.object({
  run_id: z.string(),
  row_count: z.number(),
  accepted_count: z.number(),
  rejected_count: z.number(),
  readings: z.array(SandboxReadingSchema),
});
export type SandboxResult = z.infer<typeof SandboxResultSchema>;

export async function uploadSandboxCSV(file: File, systemSizeKW?: number): Promise<SandboxResult> {
  const form = new FormData();
  form.append("file", file);
  if (systemSizeKW && systemSizeKW > 0) {
    form.append("system_size_kw", String(systemSizeKW));
  }
  const res = await fetch(`${API_BASE_URL}/v1/sandbox`, { method: "POST", body: form });
  if (!res.ok) {
    const body = await res.json().catch(() => undefined);
    throw new ApiError(res.status, body?.message ?? res.statusText, body);
  }
  return SandboxResultSchema.parse(await res.json());
}

export async function getSandboxRun(runId: string): Promise<SandboxResult> {
  const res = await fetch(`${API_BASE_URL}/v1/sandbox/${encodeURIComponent(runId)}`);
  if (!res.ok) {
    const body = await res.json().catch(() => undefined);
    throw new ApiError(res.status, body?.message ?? res.statusText, body);
  }
  return SandboxResultSchema.parse(await res.json());
}
