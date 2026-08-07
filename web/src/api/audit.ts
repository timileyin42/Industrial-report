import { apiRequest } from "./client";
import { PageSchema, AuditEntrySchema, type AuditEntry } from "./types";

export interface AuditFilters {
  actor_user_id?: string;
  action?: string;
  target_type?: string;
  target_id?: string;
  from?: string;
  to?: string;
}

// user_action_audit_log — who accessed/changed what. Deliberately not the
// same thing as the ingestor's data-quality audit trail (no read endpoint
// exists for that yet, see CLAUDE.md: "don't conflate the two").
export async function listAuditActions(
  filters: AuditFilters = {},
  cursor?: string,
  limit = 50
): Promise<{ items: AuditEntry[]; nextCursor?: string }> {
  const data = await apiRequest<unknown>("/v1/audit/actions", { query: { ...filters, cursor, limit } });
  const parsed = PageSchema(AuditEntrySchema).parse(data);
  return { items: parsed.items, nextCursor: parsed.next_cursor };
}
