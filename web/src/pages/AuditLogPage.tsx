import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { TopNav } from "../components/layout/TopNav";
import { DataTable, type Column } from "../components/table/DataTable";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { listAuditActions, type AuditFilters } from "../api/audit";
import { ApiError, type AuditEntry } from "../api/types";

// user_action_audit_log browsing — the Phase 3 catch-up router.go's
// listAuditActions comment refers to. Not the ingestor's data-quality
// audit trail (no read endpoint exists for that yet); this is strictly
// "who accessed/changed what."
export function AuditLogPage() {
  const [filters, setFilters] = useState<AuditFilters>({});
  const [pendingFilters, setPendingFilters] = useState<AuditFilters>({});
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [rows, setRows] = useState<AuditEntry[]>([]);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["audit-actions", filters, cursor],
    queryFn: async () => {
      const result = await listAuditActions(filters, cursor);
      setRows((prev) => (cursor ? [...prev, ...result.items] : result.items));
      return result;
    },
  });

  if (error instanceof ApiError && error.status === 403) return <AccessDenied />;
  if (isError) return <ErrorState onRetry={() => refetch()} />;

  function handleFilterSubmit(e: FormEvent) {
    e.preventDefault();
    setRows([]);
    setCursor(undefined);
    setFilters(pendingFilters);
  }

  const columns: Column<AuditEntry>[] = [
    { header: "When", isMono: true, render: (a) => new Date(a.created_at).toLocaleString() },
    { header: "Actor", render: (a) => a.actor_email ?? "—" },
    { header: "Action", isMono: true, render: (a) => a.action },
    {
      header: "Target",
      isMono: true,
      render: (a) => (a.target_type ? `${a.target_type}:${a.target_id ?? "—"}` : "—"),
    },
  ];

  return (
    <>
      <TopNav title="Audit Log" />
      <div className="flex-1 p-grid-margin space-y-6">
        <form className="bg-surface-container border border-outline-variant p-4 flex flex-wrap gap-4 items-end" onSubmit={handleFilterSubmit}>
          <div className="space-y-1">
            <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="action">
              Action
            </label>
            <input
              id="action"
              type="text"
              placeholder="e.g. revoke_device"
              value={pendingFilters.action ?? ""}
              onChange={(e) => setPendingFilters((f) => ({ ...f, action: e.target.value || undefined }))}
              className="bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm px-3 py-2 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none"
            />
          </div>
          <div className="space-y-1">
            <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="target_type">
              Target Type
            </label>
            <input
              id="target_type"
              type="text"
              placeholder="e.g. device"
              value={pendingFilters.target_type ?? ""}
              onChange={(e) => setPendingFilters((f) => ({ ...f, target_type: e.target.value || undefined }))}
              className="bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm px-3 py-2 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none"
            />
          </div>
          <div className="space-y-1">
            <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="from">
              From
            </label>
            <input
              id="from"
              type="date"
              value={pendingFilters.from?.slice(0, 10) ?? ""}
              onChange={(e) => setPendingFilters((f) => ({ ...f, from: e.target.value ? new Date(e.target.value).toISOString() : undefined }))}
              className="bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm px-3 py-2 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none"
            />
          </div>
          <div className="space-y-1">
            <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="to">
              To
            </label>
            <input
              id="to"
              type="date"
              value={pendingFilters.to?.slice(0, 10) ?? ""}
              onChange={(e) => setPendingFilters((f) => ({ ...f, to: e.target.value ? new Date(e.target.value).toISOString() : undefined }))}
              className="bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm px-3 py-2 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none"
            />
          </div>
          <button
            type="submit"
            className="bg-primary-container text-on-primary-container font-bold px-5 py-2 rounded transition-colors"
          >
            Filter
          </button>
        </form>

        {isLoading && rows.length === 0 ? (
          <div className="h-64 bg-surface-container border border-outline-variant animate-pulse" />
        ) : rows.length === 0 ? (
          <EmptyState title="No matching activity" body="No audit entries found for the current filters." />
        ) : (
          <>
            <DataTable columns={columns} rows={rows} rowKey={(a) => String(a.id)} />
            {data?.nextCursor && (
              <div className="flex justify-center">
                <button
                  onClick={() => setCursor(data.nextCursor)}
                  disabled={isLoading}
                  className="bg-surface-container-high hover:bg-surface-container-highest border border-outline-variant text-on-surface font-body-base px-6 py-2 rounded transition-colors disabled:opacity-60"
                >
                  {isLoading ? "Loading…" : "Load more"}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </>
  );
}
