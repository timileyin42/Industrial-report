import { apiRequest } from "./client";
import { ExportJobSchema, PageSchema, type ExportJob, type ExportJobType } from "./types";

export interface CreateExportJobInput {
  job_type: ExportJobType;
  site_id?: string;
  period?: string;
  from?: string;
  to?: string;
  cohort_id?: string;
}

// Async counterpart to api/exports.ts's sync downloadX() functions — same
// underlying data, but the request returns a job immediately and the
// caller polls until it's done. Useful once an export's range/row count
// risks a client-side request timeout; sync stays the simpler default.
export async function createExportJob(input: CreateExportJobInput): Promise<ExportJob> {
  const { job_type, site_id, period, from, to, cohort_id } = input;
  const data = await apiRequest<unknown>("/v1/exports", {
    method: "POST",
    body: { job_type, site_id, cohort_id },
    query: { period, from, to },
  });
  return ExportJobSchema.parse(data);
}

export async function getExportJob(id: number): Promise<ExportJob> {
  const data = await apiRequest<unknown>(`/v1/exports/${id}`);
  return ExportJobSchema.parse(data);
}

// The Reports page's job history — every export job the caller has ever
// queued (sync exports via api/exports.ts never appear here, they have
// no job record at all).
export async function listExportJobs(): Promise<ExportJob[]> {
  const data = await apiRequest<unknown>("/v1/exports");
  const parsed = PageSchema(ExportJobSchema).parse(data);
  return parsed.items;
}
