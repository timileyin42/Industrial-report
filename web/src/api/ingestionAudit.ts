import { apiRequest } from "./client";
import { PageSchema, IngestionAuditEntrySchema, type IngestionAuditEntry } from "./types";

export interface IngestionAuditFilters {
  device_id?: string;
  errors_only?: boolean;
  from?: string;
  to?: string;
}

// ingestion_audit_log — the ingestor's data-quality trail (every message
// received, before validation). Deliberately distinct from
// user_action_audit_log (api/audit.ts) — see CLAUDE.md.
export async function listIngestionAudit(
  filters: IngestionAuditFilters = {},
  cursor?: string,
  limit = 50
): Promise<{ items: IngestionAuditEntry[]; nextCursor?: string }> {
  const data = await apiRequest<unknown>("/v1/audit/ingestion", {
    query: { ...filters, errors_only: filters.errors_only ? "true" : undefined, cursor, limit },
  });
  const parsed = PageSchema(IngestionAuditEntrySchema).parse(data);
  return { items: parsed.items, nextCursor: parsed.next_cursor };
}
