import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useParams, Link } from "react-router-dom";
import {
  Copy,
  FlaskConical,
  ArrowLeft,
  ListChecks,
  Zap,
  TrendingUp,
  AlertTriangle,
  RotateCcw,
  XCircle,
} from "lucide-react";
import { LogoMark } from "../components/brand/Logo";
import { SandboxBadge } from "../components/sandbox/SandboxBadge";
import { KpiCard } from "../components/kpi/KpiCard";
import { StatusDonut } from "../components/kpi/StatusDonut";
import { CircularProgress } from "../components/kpi/CircularProgress";
import { EnvironmentalImpactPanel } from "../components/dashboard/EnvironmentalImpactPanel";
import { EnergyFlowIllustration } from "../components/dashboard/EnergyFlowIllustration";
import { LineChart } from "../components/charts/LineChart";
import { BarChart } from "../components/charts/BarChart";
import { DataTable, type Column } from "../components/table/DataTable";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { getSandboxRun, type SandboxReading } from "../api/sandbox";
import { ApiError } from "../api/types";

// The same section layout as FleetDashboardPage — KPI row, generation
// chart + status donut, energy bar chart + issues list, environmental
// impact + health gauges + energy flow — built from whatever a sandbox
// upload actually produces, not a scaled-down summary. A "Site Map"
// section mirrors the real dashboard's structurally, but honestly shows
// why it's empty (a CSV upload has no GPS data) rather than fabricating
// a location.
export function SandboxResultsPage() {
  const { runId } = useParams<{ runId: string }>();
  const [copied, setCopied] = useState(false);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["sandbox", runId],
    queryFn: () => getSandboxRun(runId!),
    enabled: !!runId,
    retry: false,
  });

  if (isError) {
    if (error instanceof ApiError && error.status === 404) {
      return (
        <div className="min-h-screen flex items-center justify-center px-6">
          <SandboxBadge />
          <EmptyState title="Sandbox run not found" body="This link is invalid, or the run has expired (sandbox data is kept for 30 days)." />
        </div>
      );
    }
    return (
      <div className="min-h-screen flex items-center justify-center px-6">
        <SandboxBadge />
        <ErrorState onRetry={() => refetch()} />
      </div>
    );
  }

  function copyLink() {
    navigator.clipboard.writeText(window.location.href).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  const readings = data?.readings ?? [];
  const accepted = readings.filter((r) => r.accepted);
  const rejected = readings.filter((r) => !r.accepted);
  const resets = accepted.filter((r) => r.is_reset);

  // Chronological — the API already returns rows sorted by row_number,
  // which registry.Sandbox.Upload assigns after re-sorting by parsed ts.
  const cumulativeEnergyKWh = accepted.length ? Math.max(...accepted.map((r) => r.energy_kwh_total ?? 0)) - Math.min(...accepted.map((r) => r.energy_kwh_total ?? 0)) : 0;
  const peakPowerKW = accepted.length ? Math.max(...accepted.map((r) => r.power_kw ?? 0)) : 0;
  const acceptanceRate = readings.length ? (accepted.length / readings.length) * 100 : 0;
  const resetFreeRate = accepted.length ? ((accepted.length - resets.length) / accepted.length) * 100 : 100;

  const generationPoints = accepted
    .filter((r) => r.power_kw != null)
    .map((r, i) => ({ x: i, y: r.power_kw! }));

  // Per-calendar-day energy delta, for the bar chart — same "bucket by
  // day" idea as the real dashboard's Energy & Emissions Summary, just
  // computed client-side from this run's rows instead of a rollup query.
  const byDay = new Map<string, { first: number; last: number }>();
  for (const r of accepted) {
    if (!r.ts || r.energy_kwh_total == null) continue;
    const day = r.ts.slice(0, 10);
    const entry = byDay.get(day);
    if (!entry) byDay.set(day, { first: r.energy_kwh_total, last: r.energy_kwh_total });
    else entry.last = r.energy_kwh_total;
  }
  const dailyBarPoints = Array.from(byDay.entries()).map(([day, { first, last }]) => ({
    label: new Date(day).toLocaleDateString(undefined, { month: "short", day: "numeric" }),
    value: Math.max(0, last - first),
  }));

  // Rough, clearly-labeled estimate only — there's no real grid emission
  // factor for a sandbox upload with no country, unlike the real
  // dashboard's actual configured factor. 0.5 kg CO2/kWh is a generic
  // placeholder order-of-magnitude, not a citable figure.
  const estimatedTonnesCO2 = accepted.length ? (cumulativeEnergyKWh * 0.5) / 1000 : null;

  const columns: Column<SandboxReading>[] = [
    { header: "Row", isMono: true, render: (r) => String(r.row_number) },
    { header: "Timestamp", isMono: true, render: (r) => (r.ts ? new Date(r.ts).toISOString() : "—") },
    { header: "Power (kW)", isMono: true, align: "right", render: (r) => (r.power_kw != null ? r.power_kw.toFixed(2) : "—") },
    { header: "Energy (kWh)", isMono: true, align: "right", render: (r) => (r.energy_kwh_total != null ? r.energy_kwh_total.toFixed(2) : "—") },
    { header: "RSSI (dBm)", isMono: true, align: "right", render: (r) => (r.rssi != null ? String(r.rssi) : "—") },
    {
      header: "Result",
      render: (r) =>
        r.accepted ? (
          <span className="flex items-center gap-1.5 text-success text-[12px] font-semibold">
            {r.is_reset ? <RotateCcw size={14} /> : <ListChecks size={14} />}
            Accepted{r.is_reset ? " (reset detected)" : ""}
          </span>
        ) : (
          <span className="flex items-center gap-1.5 text-error text-[12px]">
            <XCircle size={14} /> {r.rejection_reason ?? "Rejected"}
          </span>
        ),
    },
  ];

  return (
    <div className="min-h-screen text-on-surface px-6 py-10">
      <SandboxBadge />
      <main className="max-w-7xl mx-auto space-y-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <Link to="/" className="flex items-center gap-2 hover:opacity-90 transition-opacity">
            <LogoMark size={24} />
            <span className="font-headline-md text-headline-md font-bold text-on-surface">Clean Energy Analytics</span>
          </Link>
          <div className="flex items-center gap-3">
            <button
              onClick={copyLink}
              className="flex items-center gap-2 glass-card rounded-full text-on-surface hover:text-primary font-body-base text-body-base px-4 py-2 transition-colors"
            >
              <Copy size={15} /> {copied ? "Link copied!" : "Copy shareable link"}
            </button>
            <Link to="/sandbox" className="flex items-center gap-1.5 text-[13px] text-on-surface-variant hover:text-primary transition-colors">
              <ArrowLeft size={14} /> New upload
            </Link>
          </div>
        </div>

        <div className="flex items-center gap-2 text-on-surface-variant">
          <FlaskConical size={18} />
          <h1 className="font-headline-lg text-headline-lg font-bold text-on-surface">Sandbox Validation Run</h1>
        </div>

        {isLoading || !data ? (
          <div className="grid grid-cols-1 md:grid-cols-3 xl:grid-cols-5 gap-gutter">
            {[0, 1, 2, 3, 4].map((i) => (
              <div key={i} className="h-32 glass-card rounded-xl animate-pulse" />
            ))}
          </div>
        ) : (
          <>
            {/* KPI row */}
            <div className="grid grid-cols-1 md:grid-cols-3 xl:grid-cols-5 gap-gutter">
              <KpiCard label="Rows Uploaded" value={data.row_count} tone="primary" icon={<ListChecks size={16} />} />
              <KpiCard label="Accepted" value={data.accepted_count} icon={<ListChecks size={16} />} />
              <KpiCard label="Rejected" value={data.rejected_count} icon={<AlertTriangle size={16} />} />
              <KpiCard
                label="Net Energy"
                value={cumulativeEnergyKWh.toFixed(1)}
                unit="kWh"
                icon={<Zap size={16} />}
              />
              <KpiCard label="Peak Power" value={peakPowerKW.toFixed(1)} unit="kW" icon={<TrendingUp size={16} />} />
            </div>

            {/* Generation Overview + Validation Status */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-gutter">
              <div className="lg:col-span-2 glass-card rounded-xl p-6">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Generation Overview</span>
                <p className="font-data-display-lg text-[20px] text-on-surface mt-2 mb-2">
                  {accepted.length} accepted reading{accepted.length === 1 ? "" : "s"}
                </p>
                <div className="h-[180px]">
                  {generationPoints.length < 2 ? (
                    <EmptyState compact title="Not enough accepted data yet" body="Upload more accepted rows to see a power trend." />
                  ) : (
                    <LineChart points={generationPoints} color="#2f8fe0" />
                  )}
                </div>
              </div>

              <div className="glass-card rounded-xl p-6 flex flex-col">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase self-start">Validation Status</span>
                <div className="flex items-center justify-center mt-4">
                  <StatusDonut
                    segments={[
                      { value: accepted.length - resets.length, className: "stroke-success" },
                      { value: rejected.length, className: "stroke-error" },
                      { value: resets.length, className: "stroke-secondary" },
                    ]}
                    size={120}
                    strokeWidth={12}
                    centerValue={String(data.row_count)}
                    centerLabel="Rows"
                  />
                </div>
                <div className="mt-4 space-y-1.5 text-[12px]">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-success flex-shrink-0" />
                    <span className="text-on-surface-variant">Accepted, clean</span>
                    <span className="ml-auto font-semibold text-on-surface font-data-mono-sm">{accepted.length - resets.length}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-error flex-shrink-0" />
                    <span className="text-on-surface-variant">Rejected</span>
                    <span className="ml-auto font-semibold text-on-surface font-data-mono-sm">{rejected.length}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-secondary flex-shrink-0" />
                    <span className="text-on-surface-variant">Reset detected</span>
                    <span className="ml-auto font-semibold text-on-surface font-data-mono-sm">{resets.length}</span>
                  </div>
                </div>
              </div>
            </div>

            {/* Site Map (honest empty state — no GPS in a CSV upload) + Issues list */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-gutter">
              <div className="glass-card rounded-xl p-6">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Site Map</span>
                <div className="h-[220px] mt-4">
                  <EmptyState
                    compact
                    title="No location in this upload"
                    body="CSV readings don't carry GPS coordinates — the real app's Site Map uses each site's saved location instead."
                  />
                </div>
              </div>

              <div className="lg:col-span-2 glass-card rounded-xl overflow-hidden">
                <div className="p-6 pb-3">
                  <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Rejected Rows</span>
                </div>
                {rejected.length === 0 ? (
                  <div className="px-6 pb-6">
                    <EmptyState compact title="Nothing rejected" body="Every row in this upload passed validation." />
                  </div>
                ) : (
                  <table className="w-full text-[13px]">
                    <thead>
                      <tr className="text-left text-on-surface-variant border-t border-outline-variant/60">
                        <th className="px-6 py-2 font-label-caps text-label-caps">Row</th>
                        <th className="px-3 py-2 font-label-caps text-label-caps">Reason</th>
                      </tr>
                    </thead>
                    <tbody>
                      {rejected.slice(0, 6).map((r) => (
                        <tr key={r.row_number} className="border-t border-outline-variant/40">
                          <td className="px-6 py-2.5 font-data-mono-sm text-data-mono-sm">{r.row_number}</td>
                          <td className="px-3 py-2.5 text-error">{r.rejection_reason}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </div>

            {/* Energy Summary bar chart + Validation Issues */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-gutter">
              <div className="lg:col-span-2 glass-card rounded-xl p-6">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Energy By Day</span>
                <div className="h-[200px] mt-4">
                  {dailyBarPoints.length === 0 ? (
                    <EmptyState compact title="Not enough data yet" body="This chart fills in once accepted rows span more than one day." />
                  ) : (
                    <BarChart points={dailyBarPoints} color="#2f8fe0" height={200} valueFormatter={(v) => `${v.toFixed(0)} kWh`} />
                  )}
                </div>
              </div>

              <div className="glass-card rounded-xl overflow-hidden flex flex-col">
                <div className="p-6 pb-3">
                  <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Validation Issues</span>
                </div>
                {resets.length === 0 && rejected.length === 0 ? (
                  <div className="px-6 pb-6">
                    <EmptyState compact title="All clear" body="No resets or rejections in this upload." />
                  </div>
                ) : (
                  <div className="divide-y divide-outline-variant/40">
                    {[...resets.map((r) => ({ r, kind: "reset" as const })), ...rejected.map((r) => ({ r, kind: "rejected" as const }))]
                      .slice(0, 4)
                      .map(({ r, kind }, i) => (
                        <div key={i} className="px-6 py-3 flex items-start gap-3">
                          {kind === "reset" ? (
                            <RotateCcw size={16} className="mt-0.5 flex-shrink-0 text-secondary" />
                          ) : (
                            <AlertTriangle size={16} className="mt-0.5 flex-shrink-0 text-error" />
                          )}
                          <div>
                            <p className="text-[13px] text-on-surface leading-tight">
                              {kind === "reset" ? "Energy counter reset detected" : r.rejection_reason}
                            </p>
                            <p className="text-[11px] text-on-surface-variant mt-0.5">Row {r.row_number}</p>
                          </div>
                        </div>
                      ))}
                  </div>
                )}
              </div>
            </div>

            {/* Environmental Impact + Data Quality + Energy Flow */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-gutter">
              <EnvironmentalImpactPanel cumulativeTonnesCO2={estimatedTonnesCO2} />

              <div className="glass-card rounded-xl p-6">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Data Quality</span>
                <div className="mt-4 flex justify-around">
                  <CircularProgress percent={acceptanceRate} size={88} strokeWidth={9} color="#2f8fe0" label="Acceptance Rate" />
                  <CircularProgress percent={resetFreeRate} size={88} strokeWidth={9} color="#1a9c6b" label="Reset-Free Rate" />
                </div>
                <p className="text-[10px] text-on-surface-variant text-center mt-3">
                  {acceptanceRate >= 90 ? "Good" : acceptanceRate >= 50 ? "Fair" : "Needs Attention"}
                </p>
              </div>

              <div className="glass-card rounded-xl p-6">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Energy Flow</span>
                <EnergyFlowIllustration
                  solar={{ label: "Solar Generation", value: accepted.length ? `${(cumulativeEnergyKWh / 1000).toFixed(2)} MWh` : "—", available: accepted.length > 0 }}
                  battery={{ label: "Battery Storage", value: "—", available: false }}
                  grid={{ label: "Grid Import/Export", value: "—", available: false }}
                  consumption={{ label: "Consumption", value: "—", available: false }}
                  animated={false}
                  height={220}
                />
              </div>
            </div>

            {/* Full row-by-row detail */}
            <div className="glass-card rounded-2xl overflow-hidden">
              <div className="px-6 py-3 border-b border-outline-variant/60">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Row-by-Row Detail</span>
              </div>
              <DataTable columns={columns} rows={readings} rowKey={(r) => String(r.row_number)} />
            </div>
          </>
        )}
      </main>
    </div>
  );
}
