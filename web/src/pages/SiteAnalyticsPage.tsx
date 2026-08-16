import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { TrendingUp, Sun, Gauge, Leaf, TriangleAlert } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { KpiCard } from "../components/kpi/KpiCard";
import { LineChart } from "../components/charts/LineChart";
import { DataTable, type Column } from "../components/table/DataTable";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { ExportButton } from "../components/export/ExportButton";
import { AsyncExportButton } from "../components/export/AsyncExportButton";
import { ExportMenuButton } from "../components/export/ExportMenuButton";
import { getSiteEnergy, getSiteSpecificYield, getSitePeak, getSiteCapacityFactor, getSitePerformanceRatio } from "../api/analytics";
import { getSiteEmissions } from "../api/emissions";
import { getCompareHistory, getCompareFleet } from "../api/benchmark";
import { getSiteAnomalies } from "../api/anomalies";
import { downloadSiteTelemetryCSV, downloadSiteSummaryCSV, downloadSiteSummaryPDF } from "../api/exports";
import { getSite } from "../api/sites";
import { useAuth } from "../auth/AuthContext";
import { ApiError, type AnomalyResult } from "../api/types";
type Anomaly = AnomalyResult["flags"][number];

// Reference: design/analytics_insights_zgnis_industrial_intelligence/code.html.
// That mockup fabricates region clustering, month-on-month deltas, and an
// ambient site photo, none of which this endpoint set produces — this
// page ships only what the analytics endpoints actually return: energy,
// specific yield, peak power, capacity factor (nameplate-based, NOT PR —
// see the backend's own capacityFactorDefinition string), real
// weather-adjusted Performance Ratio (historical irradiance from
// internal/weather, needs the site to have a saved location), and
// emissions once a grid factor is configured.
export function SiteAnalyticsPage() {
  const { siteId } = useParams<{ siteId: string }>();
  const { session } = useAuth();
  const isOperator = session?.role === "operator";

  const siteQuery = useQuery({ queryKey: ["site", siteId], queryFn: () => getSite(siteId!), enabled: !!siteId });
  const energyQuery = useQuery({ queryKey: ["site-energy", siteId], queryFn: () => getSiteEnergy(siteId!), enabled: !!siteId });
  const yieldQuery = useQuery({ queryKey: ["site-yield", siteId], queryFn: () => getSiteSpecificYield(siteId!), enabled: !!siteId });
  const peakQuery = useQuery({ queryKey: ["site-peak", siteId], queryFn: () => getSitePeak(siteId!), enabled: !!siteId });
  const cfQuery = useQuery({ queryKey: ["site-cf", siteId], queryFn: () => getSiteCapacityFactor(siteId!), enabled: !!siteId });
  const prQuery = useQuery({ queryKey: ["site-pr", siteId], queryFn: () => getSitePerformanceRatio(siteId!), enabled: !!siteId });
  const emissionsQuery = useQuery({
    queryKey: ["site-emissions", siteId],
    queryFn: () => getSiteEmissions(siteId!),
    enabled: !!siteId,
    retry: false,
  });
  const historyQuery = useQuery({
    queryKey: ["site-compare-history", siteId],
    queryFn: () => getCompareHistory(siteId!),
    enabled: !!siteId,
  });
  // Operator-only on the backend (leaks fleet-wide distribution) — skip
  // entirely for restricted users rather than firing a request that's
  // guaranteed a 403.
  const fleetCompareQuery = useQuery({
    queryKey: ["site-compare-fleet", siteId],
    queryFn: () => getCompareFleet(siteId!),
    enabled: !!siteId && isOperator,
  });
  const anomalyQuery = useQuery({
    queryKey: ["site-anomalies", siteId],
    queryFn: () => getSiteAnomalies(siteId!),
    enabled: !!siteId,
  });

  const anyError = siteQuery.error ?? energyQuery.error ?? yieldQuery.error ?? peakQuery.error ?? cfQuery.error;
  if (anyError instanceof ApiError && anyError.status === 403) {
    return <AccessDenied detail="This site isn't part of your account's access scope." />;
  }
  if (siteQuery.isError || energyQuery.isError || peakQuery.isError || cfQuery.isError) {
    return (
      <ErrorState
        onRetry={() => {
          siteQuery.refetch();
          energyQuery.refetch();
          peakQuery.refetch();
          cfQuery.refetch();
        }}
      />
    );
  }

  if (siteQuery.isLoading || !siteQuery.data) {
    return (
      <>
        <TopNav title="Analytics" />
        <div className="flex-1 p-grid-margin">
          <div className="h-40 glass-card rounded-xl animate-pulse" />
        </div>
      </>
    );
  }

  const site = siteQuery.data;
  const energyPoints = (energyQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.energy_kwh }));
  const yieldPoints = (yieldQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.specific_yield_kwh_per_kwp }));
  const prPoints = (prQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.performance_ratio_pct }));
  const latestPeak = peakQuery.data?.points.at(-1);
  const latestCF = cfQuery.data?.points.at(-1);

  const emissionsUnconfigured = emissionsQuery.error instanceof ApiError && emissionsQuery.error.status === 409;

  const anomalyColumns: Column<Anomaly>[] = [
    // Site-local day, not the viewer's — a "day" bucket boundary should
    // align with midnight at the site, not wherever the viewer is.
    { header: "Day", isMono: true, render: (a) => new Date(a.day).toLocaleDateString(undefined, { timeZone: site?.timezone }) },
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
      <TopNav title={`Analytics — ${site.name ?? site.site_id}`} />
      <div className="flex-1 p-grid-margin space-y-8">
        <div className="flex flex-wrap justify-end gap-2">
          <ExportButton label="Export Telemetry CSV" onExport={() => downloadSiteTelemetryCSV(siteId!)} />
          <ExportMenuButton
            label="Export Summary"
            options={[
              { label: "CSV", onExport: () => downloadSiteSummaryCSV(siteId!) },
              { label: "PDF", onExport: () => downloadSiteSummaryPDF(siteId!) },
            ]}
          />
          <AsyncExportButton label="Queue Telemetry Export" input={{ job_type: "site_telemetry_csv", site_id: siteId! }} />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-gutter">
          <KpiCard
            label="Energy (cumulative)"
            value={energyQuery.data ? energyQuery.data.cumulative_kwh.toFixed(1) : "—"}
            unit="kWh"
            icon={<TrendingUp size={16} className="text-primary" />}
            tone="primary"
          />
          <KpiCard
            label="Peak Power"
            value={latestPeak ? latestPeak.peak_power_kw.toFixed(1) : "—"}
            unit="kW"
            icon={<Sun size={16} className="text-primary" />}
          />
          <KpiCard
            label="Capacity Factor"
            value={latestCF ? latestCF.capacity_factor_pct.toFixed(1) : "—"}
            unit="%"
            icon={<Gauge size={16} className="text-primary" />}
            footer={
              cfQuery.data ? (
                <p className="text-[10px] text-on-surface-variant leading-relaxed">{cfQuery.data.definition}</p>
              ) : undefined
            }
          />
        </div>

        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Energy Output</span>
          </div>
          <div className="h-[220px] p-6">
            {energyQuery.isLoading ? (
              <div className="h-full bg-surface-dim rounded-xl animate-pulse" />
            ) : energyPoints.length < 2 ? (
              <EmptyState compact title="Not enough data yet" body="Energy output will chart here once this site has more history." />
            ) : (
              <LineChart points={energyPoints} color="#2f8fe0" />
            )}
          </div>
        </div>

        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Specific Yield (kWh/kWp)</span>
          </div>
          <div className="h-[220px] p-6">
            {yieldQuery.isLoading ? (
              <div className="h-full bg-surface-dim rounded-xl animate-pulse" />
            ) : yieldQuery.error instanceof ApiError ? (
              <EmptyState compact title="Specific yield unavailable" body={yieldQuery.error.message} />
            ) : yieldPoints.length < 2 ? (
              <EmptyState compact title="Not enough data yet" body="Specific yield will chart here once this site has more history." />
            ) : (
              <LineChart points={yieldPoints} color="#f2a93b" />
            )}
          </div>
        </div>

        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Performance Ratio (%)</span>
          </div>
          <div className="h-[220px] p-6">
            {prQuery.isLoading ? (
              <div className="h-full bg-surface-dim rounded-xl animate-pulse" />
            ) : prQuery.error instanceof ApiError ? (
              <EmptyState compact title="Performance ratio unavailable" body={prQuery.error.message} />
            ) : prPoints.length < 2 ? (
              <EmptyState compact title="Not enough data yet" body="Performance ratio will chart here once this site has more history." />
            ) : (
              <LineChart points={prPoints} color="#1a9c6b" />
            )}
          </div>
          {prQuery.data && (
            <p className="text-[10px] text-on-surface-variant leading-relaxed px-6 pb-6">{prQuery.data.definition}</p>
          )}
        </div>

        <div className="glass-card rounded-2xl overflow-hidden p-6">
          <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Emissions Avoided</span>
          <div className="mt-4">
            {emissionsQuery.isLoading ? (
              <div className="h-24 bg-surface-dim rounded-xl animate-pulse" />
            ) : emissionsUnconfigured ? (
              <EmptyState
                icon={<Leaf size={48} />}
                title="No grid emission factor configured"
                body="Emissions-avoided figures need a grid emission factor (kg CO2/kWh) set first."
                action={
                  isOperator ? (
                    <Link
                      to="/app/settings/emissions"
                      className="bg-primary text-on-primary font-bold px-4 py-2 rounded-full shadow-soft"
                    >
                      Configure emission factor
                    </Link>
                  ) : undefined
                }
              />
            ) : emissionsQuery.data ? (
              <div className="flex items-baseline gap-3">
                <span className="font-data-display-lg text-data-display-lg text-primary">
                  {emissionsQuery.data.cumulative_lifetime_co2_tonnes.toFixed(2)}
                </span>
                <span className="text-on-surface-variant font-data-mono-sm text-data-mono-sm">t CO2 (lifetime)</span>
                {emissionsQuery.data.emission_factor && (
                  <span className="text-[10px] text-on-surface-variant ml-4">
                    Factor: {emissionsQuery.data.emission_factor.kg_co2_per_kwh} kg/kWh ({emissionsQuery.data.emission_factor.country})
                  </span>
                )}
              </div>
            ) : (
              <EmptyState compact title="Emissions unavailable" body="Couldn't load emissions data right now." />
            )}
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-gutter">
          <div className="glass-card rounded-2xl p-6">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">This Period vs. Last</span>
            <div className="mt-4">
              {historyQuery.isLoading ? (
                <div className="h-16 bg-surface-container-high animate-pulse" />
              ) : historyQuery.data ? (
                <div className="flex items-baseline gap-3">
                  <span className="font-data-display-lg text-data-display-lg text-primary">
                    {historyQuery.data.current_energy_kwh.toFixed(1)}
                  </span>
                  <span className="text-on-surface-variant font-data-mono-sm text-data-mono-sm">kWh</span>
                  {historyQuery.data.change_pct != null && (
                    <span className={historyQuery.data.change_pct >= 0 ? "text-primary" : "text-error"}>
                      {historyQuery.data.change_pct >= 0 ? "+" : ""}
                      {historyQuery.data.change_pct.toFixed(1)}% vs. previous period
                    </span>
                  )}
                </div>
              ) : (
                <EmptyState compact title="Not enough history yet" body="This site needs at least two periods of data to compare." />
              )}
            </div>
          </div>

          {isOperator && (
            <div className="glass-card rounded-2xl p-6">
              <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">This Site vs. Fleet</span>
              <div className="mt-4">
                {fleetCompareQuery.isLoading ? (
                  <div className="h-16 bg-surface-container-high animate-pulse" />
                ) : fleetCompareQuery.data ? (
                  <div className="flex items-baseline gap-3">
                    <span className="font-data-display-lg text-data-display-lg text-primary">
                      {fleetCompareQuery.data.percentile_rank.toFixed(0)}
                    </span>
                    <span className="text-on-surface-variant font-data-mono-sm text-data-mono-sm">percentile</span>
                    <span className="text-[10px] text-on-surface-variant ml-2">
                      Fleet avg: {fleetCompareQuery.data.fleet_avg_kwh.toFixed(1)} kWh ({fleetCompareQuery.data.site_count} sites)
                    </span>
                  </div>
                ) : (
                  <EmptyState compact title="Not enough fleet data yet" body="Fleet comparison needs at least one other site with data." />
                )}
              </div>
            </div>
          )}
        </div>

        <div className="space-y-3">
          <span className="font-label-caps text-label-caps text-on-surface-variant uppercase flex items-center gap-2">
            <TriangleAlert size={14} className="text-secondary" /> Anomalies
          </span>
          {anomalyQuery.isLoading ? (
            <div className="h-32 glass-card rounded-xl animate-pulse" />
          ) : !anomalyQuery.data || anomalyQuery.data.flags.length === 0 ? (
            <EmptyState compact title="No anomalies flagged" body="No significant drop below this site's trailing baseline right now." />
          ) : (
            <>
              <DataTable columns={anomalyColumns} rows={anomalyQuery.data.flags} rowKey={(a) => a.day} />
              <p className="text-[10px] text-on-surface-variant italic">{anomalyQuery.data.definition}</p>
            </>
          )}
        </div>
      </div>
    </>
  );
}
