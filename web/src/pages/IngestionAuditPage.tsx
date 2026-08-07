import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { TopNav } from "../components/layout/TopNav";
import { DataTable, type Column } from "../components/table/DataTable";
import { StatusBadge } from "../components/status/StatusBadge";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { listIngestionAudit, type IngestionAuditFilters } from "../api/ingestionAudit";
import type { IngestionAuditEntry } from "../api/types";

// ingestion_audit_log's first-ever read path — every message received,
// before validation, browsable now. Kept as its own page/name ("Ingestion
// Log") distinct from Audit Log (user_action_audit_log) so the two data
// sets never get conflated in the UI either, per CLAUDE.md. Visible to
// every role — the backend scopes restricted users to their own site's
// devices server-side (see listIngestionAudit in router.go), so there's
// no operator gate here.
export function IngestionAuditPage() {
  const [filters, setFilters] = useState<IngestionAuditFilters>({});
  const [pendingFilters, setPendingFilters] = useState<IngestionAuditFilters>({});
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [rows, setRows] = useState<IngestionAuditEntry[]>([]);
  const [expanded, setExpanded] = useState<number | null>(null);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ["ingestion-audit", filters, cursor],
    queryFn: async () => {
      const result = await listIngestionAudit(filters, cursor);
      setRows((prev) => (cursor ? [...prev, ...result.items] : result.items));
      return result;
    },
  });

  if (isError) return <ErrorState onRetry={() => refetch()} />;

  function handleFilterSubmit(e: FormEvent) {
    e.preventDefault();
    setRows([]);
    setCursor(undefined);
    setFilters(pendingFilters);
  }

  const columns: Column<IngestionAuditEntry>[] = [
    { header: "Received", isMono: true, render: (e) => new Date(e.received_at).toLocaleString() },
    { header: "Device", isMono: true, render: (e) => e.device_id },
    { header: "Site", isMono: true, render: (e) => e.site_id ?? "—" },
    {
      header: "Status",
      render: (e) =>
        e.error ? (
          <StatusBadge status="offline" label="ERROR" />
        ) : e.processed ? (
          <StatusBadge status="online" label="PROCESSED" />
        ) : (
          <StatusBadge status="degraded" label="PENDING" />
        ),
    },
    {
      header: "Payload",
      align: "right",
      render: (e) => (
        <button
          onClick={() => setExpanded(expanded === e.id ? null : e.id)}
          className="text-on-surface-variant hover:text-primary transition-colors font-body-base"
        >
          {expanded === e.id ? "Hide" : "View"}
        </button>
      ),
    },
  ];

  return (
    <>
      <TopNav title="Ingestion Log" />
      <div className="flex-1 p-grid-margin space-y-6">
        <form className="glass-card rounded-2xl p-4 flex flex-wrap gap-4 items-end" onSubmit={handleFilterSubmit}>
          <div className="space-y-1">
            <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="device_id">
              Device ID
            </label>
            <input
              id="device_id"
              type="text"
              placeholder="e.g. ZG-LOAD-00188"
              value={pendingFilters.device_id ?? ""}
              onChange={(e) => setPendingFilters((f) => ({ ...f, device_id: e.target.value || undefined }))}
              className="bg-white/70 border border-outline-variant text-on-surface font-body-base text-body-base px-3 py-2 rounded-xl focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none"
            />
          </div>
          <label className="flex items-center gap-2 font-body-base text-body-base text-on-surface-variant pb-2">
            <input
              type="checkbox"
              checked={pendingFilters.errors_only ?? false}
              onChange={(e) => setPendingFilters((f) => ({ ...f, errors_only: e.target.checked || undefined }))}
              className="accent-primary"
            />
            Errors only
          </label>
          <button type="submit" className="bg-primary hover:opacity-90 text-on-primary font-bold px-5 py-2 rounded-full transition-colors shadow-soft">
            Filter
          </button>
        </form>

        {isLoading && rows.length === 0 ? (
          <div className="h-64 glass-card rounded-xl animate-pulse" />
        ) : rows.length === 0 ? (
          <EmptyState title="No ingestion activity" body="No ingestion audit entries found for the current filters." />
        ) : (
          <>
            <DataTable columns={columns} rows={rows} rowKey={(e) => String(e.id)} />
            {rows
              .filter((e) => e.id === expanded)
              .map((e) => (
                <div key={e.id} className="glass-card rounded-xl p-4 overflow-x-auto">
                  {e.error && <p className="text-error font-body-base mb-2">{e.error}</p>}
                  <pre className="font-data-mono-sm text-data-mono-sm text-on-surface-variant whitespace-pre-wrap">
                    {JSON.stringify(e.raw_payload, null, 2)}
                  </pre>
                </div>
              ))}
            {data?.nextCursor && (
              <div className="flex justify-center">
                <button
                  onClick={() => setCursor(data.nextCursor)}
                  disabled={isLoading}
                  className="glass-card rounded-full text-on-surface hover:text-primary font-body-base px-6 py-2 transition-colors disabled:opacity-60"
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
