import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { TopNav } from "../components/layout/TopNav";
import { KpiCard } from "../components/kpi/KpiCard";
import { DataTable, type Column } from "../components/table/DataTable";
import { StatusBadge } from "../components/status/StatusBadge";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { getFleetHealth } from "../api/fleetHealth";
import { ApiError, type SiteHealth } from "../api/types";

// Phase 2's data-quality dashboard, browsable from the frontend for the
// first time in Slice 3. Deliberately a separate page from Fleet
// Overview/Analytics — /v1/fleet/health has its own stable totals
// contract distinct from /v1/fleet/summary (see fleet_health_handlers.go).
export function FleetHealthPage() {
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [rows, setRows] = useState<SiteHealth[]>([]);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["fleet-health", cursor],
    queryFn: async () => {
      const result = await getFleetHealth(cursor);
      setRows((prev) => (cursor ? [...prev, ...result.sites.items] : result.sites.items));
      return result;
    },
  });

  if (error instanceof ApiError && error.status === 403) return <AccessDenied />;
  if (isError) return <ErrorState onRetry={() => refetch()} />;

  const columns: Column<SiteHealth>[] = [
    {
      header: "Site",
      render: (s) => (
        <Link to={`/app/sites/${s.site_id}`} className="text-on-surface hover:text-primary transition-colors">
          <p className="font-body-base font-bold">{s.site_name ?? s.site_id}</p>
          <p className="text-xs font-data-mono-sm text-on-surface-variant">{s.site_id}</p>
        </Link>
      ),
    },
    {
      header: "Devices",
      isMono: true,
      align: "right",
      render: (s) => `${s.online_devices} / ${s.total_devices}`,
    },
    {
      header: "Coverage",
      align: "right",
      render: (s) => (
        <StatusBadge
          status={s.coverage_pct >= 90 ? "online" : s.coverage_pct >= 50 ? "degraded" : "offline"}
          label={`${s.coverage_pct.toFixed(0)}%`}
        />
      ),
    },
    {
      header: "Worst Last Seen",
      isMono: true,
      align: "right",
      render: (s) => (s.worst_last_seen_at ? new Date(s.worst_last_seen_at).toLocaleString() : "—"),
    },
  ];

  return (
    <>
      <TopNav title="Fleet Health" />
      <div className="flex-1 p-grid-margin space-y-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-gutter">
          <KpiCard
            label="Devices Reporting"
            value={data ? data.fleet.devices_reporting_pct.toFixed(1) : isLoading ? "…" : "—"}
            unit="%"
            tone="primary"
          />
          <KpiCard label="Fleet Coverage" value={data ? data.fleet.coverage_pct.toFixed(1) : isLoading ? "…" : "—"} unit="%" />
          <KpiCard
            label="Total Devices"
            value={data ? data.fleet.total_devices : isLoading ? "…" : "—"}
            footer={
              data ? (
                <p className="text-[10px] text-on-surface-variant">
                  Online threshold: {data.online_threshold_minutes}m · Coverage window: {data.coverage_window_hours}h
                </p>
              ) : undefined
            }
          />
        </div>

        {isLoading && rows.length === 0 ? (
          <div className="h-64 glass-card rounded-xl animate-pulse" />
        ) : rows.length === 0 ? (
          <EmptyState title="No sites yet" body="Site health will appear here once sites and devices are registered." />
        ) : (
          <>
            <DataTable columns={columns} rows={rows} rowKey={(s) => s.site_id} />
            {data?.sites.next_cursor && (
              <div className="flex justify-center">
                <button
                  onClick={() => setCursor(data.sites.next_cursor)}
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
