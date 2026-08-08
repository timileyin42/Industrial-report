import { z } from "zod";
import { apiRequest } from "./client";
import { CohortSchema, type Cohort } from "./types";

// Cohorts are derived from sites' own free-text cohort_id field — there's
// no separate cohorts table, so this list only ever contains cohorts that
// currently have at least one site assigned (see internal/registry/sites.go
// ListCohorts).
export async function listCohorts(): Promise<Cohort[]> {
  const data = await apiRequest<unknown>("/v1/cohorts");
  return z.object({ items: z.array(CohortSchema) }).parse(data).items;
}
