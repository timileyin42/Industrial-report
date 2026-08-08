import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { TriangleAlert } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { LineChart } from "../components/charts/LineChart";
import { DataTable, type Column } from "../components/table/DataTable";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { getFleetSpecificYield, getFleetPerformanceRatio } from "../api/analytics";
import { getBenchmarkSegments } from "../api/benchmark";
import { getFleetAnomalies } from "../api/anomalies";
import { ApiError, type SegmentResult, type AnomalyResult } from "../api/types";
type Segment = SegmentResult["items"][number];
type Anomaly = AnomalyResult["flags"][number];

// Fleet-wide "how well is it performing" metrics — split out of what used
// to be one combined FleetAnalyticsPage, matching the Performance/Energy/
// Emissions/Reports section split. Site-level performance still lives on
// SiteAnalyticsPage (reached from Site Detail), not duplicated here.
export function PerformancePage() {
  const yieldQuery = useQuery({ queryKey: ["fleet-yield"], queryFn: () => getFleetSpecificYield(), retry: false });
  const prQuery = useQuery({ queryKey: ["fleet-pr"], queryFn: () => getFleetPerformanceRatio(), retry: false });
  const segmentQuery = useQuery({ queryKey: ["fleet-benchmark"], queryFn: () => getBenchmarkSegments() });
  const anomalyQuery = useQuery({ queryKey: ["fleet-anomalies"], queryFn: () => getFleetAnomalies() });

  if (segmentQuery.error instanceof ApiError && segmentQuery.error.status === 403) {
    return <AccessDenied />;
  }
  if (segmentQuery.isError) {
    return <ErrorState onRetry={() => segmentQuery.refetch()} />;
  }

  const yieldPoints = (yieldQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.specific_yield_kwh_per_kwp }));
  const prPoints = (prQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.performance_ratio_pct }));

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
      <TopNav title="Performance" />
      <div className="flex-1 p-grid-margin space-y-8">
        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Fleet Specific Yield (kWh/kWp)</span>
          </div>
          <div className="h-[240px] p-6">
            {yieldQuery.isLoading ? (
              <div className="h-full bg-surface-dim rounded-xl animate-pulse" />
            ) : yieldQuery.error instanceof ApiError ? (
              <EmptyState compact title="Specific yield unavailable" body={yieldQuery.error.message} />
            ) : yieldPoints.length < 2 ? (
              <EmptyState compact title="Not enough data yet" body="Fleet specific yield will chart here once there's more history." />
            ) : (
              <LineChart points={yieldPoints} color="#f2a93b" />
            )}
          </div>
        </div>

        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Fleet Performance Ratio (%)</span>
          </div>
          <div className="h-[240px] p-6">
            {prQuery.isLoading ? (
              <div className="h-full bg-surface-dim rounded-xl animate-pulse" />
            ) : prQuery.error instanceof ApiError ? (
              <EmptyState compact title="Performance ratio unavailable" body={prQuery.error.message} />
            ) : prPoints.length < 2 ? (
              <EmptyState compact title="Not enough data yet" body="Fleet performance ratio will chart here once there's more history." />
            ) : (
              <LineChart points={prPoints} color="#1a9c6b" />
            )}
          </div>
          {prQuery.data && (
            <p className="text-[10px] text-on-surface-variant leading-relaxed px-6 pb-6">{prQuery.data.definition}</p>
          )}
        </div>

        <div className="space-y-3">
          <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Segment Benchmark</span>
          {segmentQuery.isLoading ? (
            <div className="h-32 glass-card rounded-xl animate-pulse" />
          ) : !segmentQuery.data || segmentQuery.data.items.length === 0 ? (
            <EmptyState compact title="No segments yet" body="Segment benchmarking appears here once sites have system sizes configured." />
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
            <EmptyState compact title="No anomalies flagged" body="No sites currently show a significant drop below their trailing baseline." />
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
