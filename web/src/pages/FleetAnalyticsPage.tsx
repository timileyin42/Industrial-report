import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Leaf, TriangleAlert } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { KpiCard } from "../components/kpi/KpiCard";
import { LineChart } from "../components/charts/LineChart";
import { DataTable, type Column } from "../components/table/DataTable";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { ExportButton } from "../components/export/ExportButton";
import { AsyncExportButton } from "../components/export/AsyncExportButton";
import { getFleetEnergy } from "../api/analytics";
import { getFleetEmissions } from "../api/emissions";
import { getBenchmarkSegments, getFleetTrends } from "../api/benchmark";
import { getFleetAnomalies } from "../api/anomalies";
import { downloadFleetSummaryCSV } from "../api/exports";
import { ApiError, type SegmentResult, type AnomalyResult } from "../api/types";
type Segment = SegmentResult["items"][number];
type Anomaly = AnomalyResult["flags"][number];

// Fleet-wide analogue of SiteAnalyticsPage — operator-only (both
// /v1/fleet/analytics/* endpoints are operatorOnly in router.go).
export function FleetAnalyticsPage() {
  const energyQuery = useQuery({ queryKey: ["fleet-energy"], queryFn: () => getFleetEnergy() });
  const emissionsQuery = useQuery({ queryKey: ["fleet-emissions"], queryFn: () => getFleetEmissions(), retry: false });
  const segmentQuery = useQuery({ queryKey: ["fleet-benchmark"], queryFn: () => getBenchmarkSegments() });
  const trendsQuery = useQuery({ queryKey: ["fleet-trends"], queryFn: () => getFleetTrends() });
  const anomalyQuery = useQuery({ queryKey: ["fleet-anomalies"], queryFn: () => getFleetAnomalies() });

  if (energyQuery.error instanceof ApiError && energyQuery.error.status === 403) {
    return <AccessDenied />;
  }
  if (energyQuery.isError) {
    return <ErrorState onRetry={() => energyQuery.refetch()} />;
  }

  const energyPoints = (energyQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.energy_kwh }));
  const trendPoints = (trendsQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.total_energy_kwh }));
  const emissionsUnconfigured = emissionsQuery.error instanceof ApiError && emissionsQuery.error.status === 409;

  const segmentColumns: Column<Segment>[] = [
    { header: "Segment", render: (s) => s.segment_key },
    { header: "Sites", isMono: true, align: "right", render: (s) => s.site_count },
    { header: "Total Energy", isMono: true, align: "right", render: (s) => `${(s.total_energy_kwh / 1000).toFixed(2)} MWh` },
    { header: "Avg / Site", isMono: true, align: "right", render: (s) => `${s.avg_energy_kwh.toFixed(1)} kWh` },
  ];

  const anomalyColumns: Column<Anomaly>[] = [
    {
      header: "Site",
      render: (a) => (
        <Link to={`/app/sites/${a.site_id}`} className="text-on-surface hover:text-primary transition-colors font-data-mono-sm text-data-mono-sm">
          {a.site_id}
        </Link>
      ),
    },
    { header: "Day", isMono: true, render: (a) => new Date(a.day).toLocaleDateString() },
    { header: "Energy", isMono: true, align: "right", render: (a) => `${a.energy_kwh.toFixed(1)} kWh` },
    { header: "Baseline", isMono: true, align: "right", render: (a) => `${a.baseline_kwh.toFixed(1)} kWh` },
    {
      header: "Drop",
      isMono: true,
      align: "right",
      render: (a) => <span className="text-error">-{(a.drop_fraction * 100).toFixed(0)}%</span>,
    },
  ];

  return (
    <>
      <TopNav title="Fleet Analytics" />
      <div className="flex-1 p-grid-margin space-y-8">
        <div className="flex flex-wrap justify-end gap-2">
          <ExportButton label="Export Fleet Summary CSV" onExport={downloadFleetSummaryCSV} />
          <AsyncExportButton label="Queue Fleet Export" input={{ job_type: "fleet_summary_csv" }} />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-gutter">
          <KpiCard
            label="Fleet Energy (cumulative)"
            value={energyQuery.data ? (energyQuery.data.cumulative_kwh / 1000).toFixed(2) : "—"}
            unit="MWh"
            tone="primary"
          />
          <KpiCard
            label="Emissions Avoided (lifetime)"
            value={emissionsQuery.data ? emissionsQuery.data.cumulative_lifetime_co2_tonnes.toFixed(2) : emissionsUnconfigured ? "—" : "—"}
            unit={emissionsQuery.data ? "t CO2" : undefined}
          />
        </div>

        {/* Fleet spans more than one grid — no single factor represents
            all of it, so the backend returns a per-country breakdown
            instead of one emission_factor (see registry.CountryEmissions).
            An unconfigured country's sites generate real energy that's
            excluded from the total above until someone sets its factor
            — flagged here rather than left invisible. */}
        {emissionsQuery.data?.country_breakdown && (
          <div className="glass-card rounded-2xl p-6 space-y-2">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">
              Emissions by grid
            </span>
            <div className="flex flex-wrap gap-4 pt-2">
              {emissionsQuery.data.country_breakdown.map((c) => (
                <div key={c.country} className="flex items-center gap-2 text-body-base">
                  <span className="font-semibold text-on-surface">{c.country}</span>
                  {c.unconfigured ? (
                    <span className="text-[11px] text-secondary">
                      No emission factor set —{" "}
                      <Link to="/app/settings/emissions" className="underline">
                        configure
                      </Link>
                    </span>
                  ) : (
                    <span className="text-on-surface-variant font-data-mono-sm text-data-mono-sm">
                      {c.cumulative_lifetime_co2_tonnes.toFixed(2)} t CO2
                    </span>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Fleet Energy Output</span>
          </div>
          <div className="h-[240px] p-6">
            {energyQuery.isLoading ? (
              <div className="h-full bg-surface-dim rounded-xl animate-pulse" />
            ) : energyPoints.length < 2 ? (
              <EmptyState title="Not enough data yet" body="Fleet energy output will chart here once there's more history." />
            ) : (
              <LineChart points={energyPoints} color="#2f8fe0" />
            )}
          </div>
        </div>

        {emissionsUnconfigured && (
          <EmptyState
            icon={<Leaf size={48} />}
            title="No grid emission factor configured"
            body="Emissions-avoided figures need a grid emission factor set first."
            action={
              <Link to="/app/settings/emissions" className="bg-primary text-on-primary font-bold px-4 py-2 rounded-full shadow-soft">
                Configure emission factor
              </Link>
            }
          />
        )}

        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Fleet Growth Trend</span>
          </div>
          <div className="h-[220px] p-6">
            {trendsQuery.isLoading ? (
              <div className="h-full bg-surface-dim rounded-xl animate-pulse" />
            ) : trendPoints.length < 2 ? (
              <EmptyState title="Not enough history yet" body="Fleet growth trend will chart here once there's more month-over-month data." />
            ) : (
              <LineChart points={trendPoints} color="#f2a93b" />
            )}
          </div>
        </div>

        <div className="space-y-3">
          <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Segment Benchmark</span>
          {segmentQuery.isLoading ? (
            <div className="h-32 glass-card rounded-xl animate-pulse" />
          ) : !segmentQuery.data || segmentQuery.data.items.length === 0 ? (
            <EmptyState title="No segments yet" body="Segment benchmarking appears here once sites have system sizes configured." />
          ) : (
            <>
              <DataTable columns={segmentColumns} rows={segmentQuery.data.items} rowKey={(s) => s.segment_key} />
              {segmentQuery.data.note && <p className="text-[10px] text-on-surface-variant italic">{segmentQuery.data.note}</p>}
            </>
          )}
        </div>

        <div className="space-y-3">
          <span className="font-label-caps text-label-caps text-on-surface-variant uppercase flex items-center gap-2">
            <TriangleAlert size={14} className="text-secondary" /> Fleet Anomalies
          </span>
          {anomalyQuery.isLoading ? (
            <div className="h-32 glass-card rounded-xl animate-pulse" />
          ) : !anomalyQuery.data || anomalyQuery.data.flags.length === 0 ? (
            <EmptyState title="No anomalies flagged" body="No sites currently show a significant drop below their trailing baseline." />
          ) : (
            <>
              <DataTable columns={anomalyColumns} rows={anomalyQuery.data.flags} rowKey={(a) => `${a.site_id}-${a.day}`} />
              <p className="text-[10px] text-on-surface-variant italic">{anomalyQuery.data.definition}</p>
            </>
          )}
        </div>
      </div>
    </>
  );
}
