import { useQuery } from "@tanstack/react-query";
import { TopNav } from "../components/layout/TopNav";
import { KpiCard } from "../components/kpi/KpiCard";
import { LineChart } from "../components/charts/LineChart";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { ExportButton } from "../components/export/ExportButton";
import { AsyncExportButton } from "../components/export/AsyncExportButton";
import { getFleetEnergy } from "../api/analytics";
import { getFleetTrends } from "../api/benchmark";
import { downloadFleetSummaryCSV, downloadFleetSummaryPDF } from "../api/exports";
import { ApiError } from "../api/types";

// Fleet-wide generation view — split out of the former combined
// FleetAnalyticsPage. Site-level energy still lives on SiteAnalyticsPage.
export function EnergyPage() {
  const energyQuery = useQuery({ queryKey: ["fleet-energy"], queryFn: () => getFleetEnergy() });
  const trendsQuery = useQuery({ queryKey: ["fleet-trends"], queryFn: () => getFleetTrends() });

  if (energyQuery.error instanceof ApiError && energyQuery.error.status === 403) {
    return <AccessDenied />;
  }
  if (energyQuery.isError) {
    return <ErrorState onRetry={() => energyQuery.refetch()} />;
  }

  const energyPoints = (energyQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.energy_kwh }));
  const trendPoints = (trendsQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.total_energy_kwh }));

  return (
    <>
      <TopNav title="Energy" />
      <div className="flex-1 p-grid-margin space-y-8">
        <div className="flex flex-wrap justify-end gap-2">
          <ExportButton label="Export Fleet Summary CSV" onExport={downloadFleetSummaryCSV} />
          <ExportButton label="Export Fleet Summary PDF" onExport={downloadFleetSummaryPDF} />
          <AsyncExportButton label="Queue Fleet Export" input={{ job_type: "fleet_summary_csv" }} />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-gutter">
          <KpiCard
            label="Fleet Energy (cumulative)"
            value={energyQuery.data ? (energyQuery.data.cumulative_kwh / 1000).toFixed(2) : "—"}
            unit="MWh"
            tone="primary"
          />
        </div>

        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Fleet Energy Output</span>
          </div>
          <div className="h-[240px] p-6">
            {energyQuery.isLoading ? (
              <div className="h-full bg-surface-dim rounded-xl animate-pulse" />
            ) : energyPoints.length < 2 ? (
              <EmptyState compact title="Not enough data yet" body="Fleet energy output will chart here once there's more history." />
            ) : (
              <LineChart points={energyPoints} color="#2f8fe0" />
            )}
          </div>
        </div>

        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Fleet Growth Trend</span>
          </div>
          <div className="h-[220px] p-6">
            {trendsQuery.isLoading ? (
              <div className="h-full bg-surface-dim rounded-xl animate-pulse" />
            ) : trendPoints.length < 2 ? (
              <EmptyState compact title="Not enough history yet" body="Fleet growth trend will chart here once there's more month-over-month data." />
            ) : (
              <LineChart points={trendPoints} color="#f2a93b" />
            )}
          </div>
        </div>
      </div>
    </>
  );
}
