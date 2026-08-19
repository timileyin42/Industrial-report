import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { HomeIcon, Zap, Leaf, TrendingUp, Activity, Bell, AlertTriangle, PowerOff, ShieldOff, Info, CalendarDays } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { KpiCard } from "../components/kpi/KpiCard";
import { CircularProgress } from "../components/kpi/CircularProgress";
import { StatusDonut } from "../components/kpi/StatusDonut";
import { WeatherWidget } from "../components/dashboard/WeatherWidget";
import { EnvironmentalImpactPanel } from "../components/dashboard/EnvironmentalImpactPanel";
import { EnergyFlowIllustration } from "../components/dashboard/EnergyFlowIllustration";
import { FleetMiniMap } from "../components/dashboard/FleetMiniMap";
import { LineChart } from "../components/charts/LineChart";
import { BarChart } from "../components/charts/BarChart";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { useAuth } from "../auth/AuthContext";
import { getFleetSummary, getCurrentGeneration, getCurrentFlow, getTopSitesToday } from "../api/fleet";
import { getFleetEnergy, getFleetPowerCurve } from "../api/analytics";
import { getFleetTrends } from "../api/benchmark";
import { getFleetEmissions } from "../api/emissions";
import { getFleetHealth } from "../api/fleetHealth";
import { getPrimarySite, listSites } from "../api/sites";
import { listAllDevices } from "../api/devices";
import { listFleetAlerts } from "../api/alerts";
import { downloadFleetSummaryCSV, downloadFleetSummaryPDF } from "../api/exports";
import { ExportMenuButton } from "../components/export/ExportMenuButton";
import { ApiError } from "../api/types";
import { excludeInProgressPeriod } from "../lib/completedDays";

// Light/glass redesign — replaces the earlier dark-industrial Fleet
// Dashboard entirely. Every number here is real — the previous mockup's
// "Total Revenue" KPI and named "Alex" greeting aren't reproduced: this
// platform has no billing/pricing feature, and there's no user-profile
// endpoint yet to know a real display name, so the greeting uses
// time-of-day + role instead of inventing a name.
function greeting() {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
}

const DAY_MS = 24 * 60 * 60 * 1000;

type EnergyTab = "day" | "week" | "month" | "year";

// No "Live" tab — that would need a websocket/streaming subscription
// this platform doesn't have; Current Generation (below) is the honest
// "right now" figure instead, and stays anchored to the actual present
// moment regardless of referenceDate — there's no such thing as "current
// generation as of a past date." Day/Week/Month all use daily buckets
// over a longer window anchored to referenceDate; Year switches to
// monthly buckets so a 12-month chart isn't 365 individual bars.
function rangeForEnergyTab(tab: EnergyTab, referenceDate: Date): { period: "daily" | "monthly"; from: string; to: string } {
  const to = referenceDate.getTime();
  switch (tab) {
    case "day":
      return { period: "daily", from: new Date(to - 2 * DAY_MS).toISOString(), to: referenceDate.toISOString() };
    case "week":
      return { period: "daily", from: new Date(to - 7 * DAY_MS).toISOString(), to: referenceDate.toISOString() };
    case "month":
      return { period: "daily", from: new Date(to - 30 * DAY_MS).toISOString(), to: referenceDate.toISOString() };
    case "year":
      return { period: "monthly", from: new Date(to - 365 * DAY_MS).toISOString(), to: referenceDate.toISOString() };
  }
}

const ONLINE_THRESHOLD_MINUTES = 10;

export function FleetDashboardPage() {
  const { session } = useAuth();
  const isOperator = session?.role === "operator";
  const [energyTab, setEnergyTab] = useState<EnergyTab>("day");
  const [summaryTab, setSummaryTab] = useState<"energy" | "emissions">("energy");
  // Anchors the Generation Overview tabs and the today-vs-yesterday KPI
  // deltas — defaults to right now, but picking an earlier date lets you
  // review the dashboard as of that day instead of only ever "now."
  // Current Generation and Top Performing Sites (Today) are deliberately
  // NOT affected — "live right now" and "today" are inherently the
  // present moment, not something a past date can stand in for.
  const [referenceDate, setReferenceDate] = useState(() => new Date());
  const referenceDateKey = referenceDate.toDateString();

  const summaryQuery = useQuery({ queryKey: ["fleet-summary"], queryFn: getFleetSummary });
  const currentGenQuery = useQuery({ queryKey: ["fleet-current-gen"], queryFn: getCurrentGeneration, refetchInterval: 30_000 });
  const currentFlowQuery = useQuery({ queryKey: ["fleet-current-flow"], queryFn: getCurrentFlow, refetchInterval: 30_000 });
  // Last 7 daily buckets ending at referenceDate — doubles as (a)
  // today-vs-yesterday for the KPI deltas below and (b) the Energy &
  // Emissions Summary bar chart's data, rather than fetching the same
  // window twice for two different reasons.
  const recentEnergyQuery = useQuery({
    queryKey: ["fleet-energy-7d", referenceDateKey],
    queryFn: () => getFleetEnergy({ period: "daily", from: new Date(referenceDate.getTime() - 7 * DAY_MS).toISOString(), to: referenceDate.toISOString() }),
  });
  const recentEmissionsQuery = useQuery({
    queryKey: ["fleet-emissions-7d", referenceDateKey],
    queryFn: () => getFleetEmissions({ period: "daily", from: new Date(referenceDate.getTime() - 7 * DAY_MS).toISOString(), to: referenceDate.toISOString() }),
    retry: false,
  });
  const energyRange = rangeForEnergyTab(energyTab, referenceDate);
  const energyQuery = useQuery({ queryKey: ["fleet-energy", energyTab, referenceDateKey], queryFn: () => getFleetEnergy(energyRange) });
  // Day view gets a real intraday power curve (sunrise-to-sunset shape)
  // instead of the daily-total-energy trend Week/Month/Year use — see
  // getFleetPowerCurve. Only refetches live when viewing today; a past
  // day's curve is finished history and won't change.
  const isViewingToday = referenceDateKey === new Date().toDateString();
  const dayStart = new Date(referenceDate);
  dayStart.setHours(0, 0, 0, 0);
  const dayEnd = new Date(referenceDate);
  dayEnd.setHours(23, 59, 59, 999);
  const powerCurveQuery = useQuery({
    queryKey: ["fleet-power-curve", referenceDateKey],
    queryFn: () => getFleetPowerCurve({ from: dayStart.toISOString(), to: (isViewingToday ? new Date() : dayEnd).toISOString() }),
    enabled: energyTab === "day",
    refetchInterval: energyTab === "day" && isViewingToday ? 60_000 : false,
  });
  const trendsQuery = useQuery({ queryKey: ["fleet-trends-dash"], queryFn: () => getFleetTrends() });
  const emissionsQuery = useQuery({ queryKey: ["fleet-emissions-dash"], queryFn: () => getFleetEmissions(), retry: false });
  const healthQuery = useQuery({ queryKey: ["fleet-health-dash"], queryFn: () => getFleetHealth(undefined, 200) });
  // 404 (no primary site set yet) is an expected, routine state here —
  // not an error to retry or bubble up. See internal/registry/sites.go
  // SetPrimary / SiteDetailPage.tsx's "Set as Primary" action.
  const primarySiteQuery = useQuery({ queryKey: ["primary-site"], queryFn: getPrimarySite, retry: false });
  const sitesQuery = useQuery({ queryKey: ["dashboard-sites"], queryFn: () => listSites(undefined, 200) });
  const devicesQuery = useQuery({ queryKey: ["dashboard-devices"], queryFn: listAllDevices });
  const alertsQuery = useQuery({ queryKey: ["dashboard-alerts"], queryFn: () => listFleetAlerts(50) });
  const topSitesQuery = useQuery({ queryKey: ["top-sites-today"], queryFn: () => getTopSitesToday(5) });

  if (summaryQuery.isError) {
    if (summaryQuery.error instanceof ApiError && summaryQuery.error.status === 403) return <AccessDenied />;
    return <ErrorState onRetry={() => summaryQuery.refetch()} />;
  }

  const data = summaryQuery.data;
  // Week/Month tabs are daily-bucketed multi-day trends — the current
  // in-progress day is excluded, same reasoning as the site/fleet
  // Analytics pages. Year is monthly-bucketed, same fix at that
  // granularity. Day itself uses the separate intraday power curve
  // below, which correctly wants today's still-accumulating shape.
  const energyPoints =
    energyTab === "day"
      ? []
      : excludeInProgressPeriod(energyQuery.data?.points ?? [], energyTab === "year" ? "monthly" : "daily").map((p, i) => ({
          x: i,
          y: p.energy_kwh,
        }));
  const powerCurvePoints = (powerCurveQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.avg_power_kw }));
  // ~6 evenly spaced time-of-day ticks under the Day chart — hour alone
  // is enough (no date), since "Day" already means today (or whatever
  // date's selected above). Fixed to Africa/Lagos, this connector's only
  // real deployment today, same assumption pvpro-sync itself makes —
  // revisit if/when sites outside Nigeria come online.
  const powerCurveTicks = (() => {
    const pts = powerCurveQuery.data?.points ?? [];
    if (pts.length < 2) return [];
    const tickCount = Math.min(6, pts.length);
    const indices = Array.from({ length: tickCount }, (_, i) => Math.round((i / (tickCount - 1)) * (pts.length - 1)));
    return [...new Set(indices)].map((i) => ({
      frac: i / (pts.length - 1),
      label: new Date(pts[i].bucket).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: "Africa/Lagos" }),
    }));
  })();
  const latestTrend = trendsQuery.data?.points.at(-1) ?? null;
  const emissionsUnconfigured = emissionsQuery.error instanceof ApiError && emissionsQuery.error.status === 409;
  const primarySite = primarySiteQuery.data;
  const primarySiteHasLocation = primarySite?.gps_lat != null && primarySite?.gps_lng != null;

  // Today vs. yesterday — the last two points of the same 7-day daily
  // series, not a separate "trend" endpoint. null (not 0) when a day's
  // data genuinely isn't there yet, so the KPI card shows "—" rather
  // than a fabricated 0% change.
  const recentEnergyPoints = recentEnergyQuery.data?.points ?? [];
  const todayEnergyKWh = recentEnergyPoints.at(-1)?.energy_kwh ?? null;
  const yesterdayEnergyKWh = recentEnergyPoints.at(-2)?.energy_kwh ?? null;
  const energyDeltaPct =
    todayEnergyKWh != null && yesterdayEnergyKWh ? ((todayEnergyKWh - yesterdayEnergyKWh) / yesterdayEnergyKWh) * 100 : null;

  const recentEmissionPoints = recentEmissionsQuery.data?.points ?? [];
  const todayCO2Kg = recentEmissionPoints.at(-1)?.kg_co2 ?? null;
  const yesterdayCO2Kg = recentEmissionPoints.at(-2)?.kg_co2 ?? null;
  const co2DeltaPct = todayCO2Kg != null && yesterdayCO2Kg ? ((todayCO2Kg - yesterdayCO2Kg) / yesterdayCO2Kg) * 100 : null;

  // Fleet Status: Online / Offline / Fault / No Data — a real 4-way
  // classification, not the simpler 2-way Active/Idle this replaces.
  // "Fault" comes from the same real signal the Alerts page uses (a
  // device's latest reading reported status=fault in the last 24h);
  // "No Data" (never reported) is distinguished from "Offline" (has
  // reported before, gone quiet) since they mean different things.
  // Revoked devices are excluded — that's its own state, shown elsewhere.
  const devices = devicesQuery.data ?? [];
  const faultDeviceIds = new Set(
    (alertsQuery.data ?? []).filter((a) => a.type === "device_fault" && a.device_id).map((a) => a.device_id as string)
  );
  let onlineCount = 0;
  let offlineCount = 0;
  let faultCount = 0;
  let noDataCount = 0;
  for (const d of devices) {
    if (d.revoked_at) continue;
    if (faultDeviceIds.has(d.device_id)) {
      faultCount++;
      continue;
    }
    if (!d.last_seen_at) {
      noDataCount++;
      continue;
    }
    const ageMinutes = (Date.now() - new Date(d.last_seen_at).getTime()) / 60_000;
    if (ageMinutes < ONLINE_THRESHOLD_MINUTES) onlineCount++;
    else offlineCount++;
  }
  const classifiedTotal = onlineCount + offlineCount + faultCount + noDataCount;

  const sites = sitesQuery.data?.items ?? [];
  const healthBySite = new Map((healthQuery.data?.sites.items ?? []).map((s) => [s.site_id, s]));

  const summaryBarPoints =
    summaryTab === "energy"
      ? recentEnergyPoints.map((p) => ({
          label: new Date(p.period_start).toLocaleDateString(undefined, { month: "short", day: "numeric" }),
          value: p.energy_kwh,
        }))
      : recentEmissionPoints.map((p) => ({
          label: new Date(p.period_start).toLocaleDateString(undefined, { month: "short", day: "numeric" }),
          value: p.kg_co2,
        }));

  return (
    <>
      <TopNav title="Dashboard" />
      <div className="flex-1 p-grid-margin space-y-6">
      {/* Greeting header + weather widget */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="font-body-base text-body-base text-on-surface-variant">
            {greeting()}, {isOperator ? "Operator" : "there"} 👋
          </p>
          <h1 className="font-headline-lg text-headline-lg text-on-surface mt-1">
            Your solar fleet {data && data.total_devices - data.online_devices === 0 ? "is performing well" : "needs a look"}
          </h1>
        </div>
        <div className="flex flex-wrap items-center justify-end gap-3 ml-auto">
          <ExportMenuButton
            label="Export Report"
            options={[
              { label: "CSV", onExport: downloadFleetSummaryCSV },
              { label: "PDF", onExport: downloadFleetSummaryPDF },
            ]}
          />
          <label className="glass-card rounded-full flex items-center gap-2 px-4 py-2.5 cursor-pointer flex-shrink-0" title="Review the dashboard as of this date">
            <CalendarDays size={16} className="text-on-surface-variant" />
            <input
              type="date"
              max={new Date().toISOString().slice(0, 10)}
              value={referenceDate.toISOString().slice(0, 10)}
              onChange={(e) => setReferenceDate(e.target.value ? new Date(`${e.target.value}T12:00:00`) : new Date())}
              className="bg-transparent text-[13px] text-on-surface outline-none font-data-mono-sm"
            />
          </label>
          <WeatherWidget
            lat={primarySiteHasLocation ? primarySite?.gps_lat : undefined}
            lng={primarySiteHasLocation ? primarySite?.gps_lng : undefined}
            siteName={primarySite?.name ?? primarySite?.site_id}
            timezone={primarySite?.timezone}
          />
        </div>
      </div>

      {summaryQuery.isLoading || !data ? (
        <div className="grid grid-cols-1 md:grid-cols-3 xl:grid-cols-5 gap-gutter">
          {[0, 1, 2, 3, 4].map((i) => (
            <div key={i} className="h-32 glass-card rounded-xl animate-pulse" />
          ))}
        </div>
      ) : data.total_sites === 0 ? (
        <EmptyState
          title="No sites registered yet"
          body="Begin monitoring your renewable assets by initializing your first fleet location."
        />
      ) : (
        <>
          {/* KPI row */}
          <div className="grid grid-cols-1 md:grid-cols-3 xl:grid-cols-5 gap-gutter">
            <KpiCard label="Total Sites" value={data.total_sites} tone="primary" icon={<HomeIcon size={16} />} />
            <KpiCard
              label="Total Capacity"
              value={data.total_capacity_kw != null ? (data.total_capacity_kw / 1000).toFixed(1) : "—"}
              unit={data.total_capacity_kw != null ? "MWp" : undefined}
              icon={<Zap size={16} />}
            />
            <KpiCard
              label="Current Generation"
              value={currentGenQuery.data != null ? currentGenQuery.data.toFixed(1) : "—"}
              unit={currentGenQuery.data != null ? "kW" : undefined}
              icon={<Activity size={16} />}
            />
            <KpiCard
              label="Energy Today"
              value={todayEnergyKWh != null ? todayEnergyKWh.toFixed(0) : "—"}
              unit={todayEnergyKWh != null ? "kWh" : undefined}
              icon={<TrendingUp size={16} />}
              trendPct={energyDeltaPct}
            />
            <KpiCard
              label="CO2 Avoided Today"
              value={todayCO2Kg != null ? todayCO2Kg.toFixed(0) : emissionsUnconfigured ? "—" : "—"}
              unit={todayCO2Kg != null ? "kg" : undefined}
              icon={<Leaf size={16} />}
              trendPct={co2DeltaPct}
            />
          </div>

          {/* Generation Overview + Fleet Status */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-gutter">
            <div className="lg:col-span-2 glass-card rounded-xl p-6">
              <div className="flex flex-wrap justify-between items-center gap-3 mb-4">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Generation Overview</span>
                <div className="flex gap-1 glass-card rounded-full p-1">
                  {(["day", "week", "month", "year"] as EnergyTab[]).map((tab) => (
                    <button
                      key={tab}
                      onClick={() => setEnergyTab(tab)}
                      className={`px-3 py-1 rounded-full text-[12px] capitalize transition-colors ${
                        energyTab === tab ? "bg-primary text-on-primary font-semibold" : "text-on-surface-variant hover:text-on-surface"
                      }`}
                    >
                      {tab}
                    </button>
                  ))}
                </div>
              </div>
              <p className="font-data-display-lg text-[20px] text-on-surface mb-2">
                {energyTab === "day"
                  ? isViewingToday && todayEnergyKWh != null
                    ? `${todayEnergyKWh.toFixed(0)} kWh`
                    : powerCurvePoints.length
                      ? // Trapezoid-free estimate from 5-minute average-power buckets: each
                        // point represents 5 minutes, so kWh = sum(avg_kW) * (5/60).
                        `${(powerCurvePoints.reduce((sum, p) => sum + p.y, 0) * (5 / 60)).toFixed(0)} kWh (est.)`
                      : "—"
                  : energyQuery.data
                    ? `${energyQuery.data.cumulative_kwh.toFixed(0)} kWh`
                    : "—"}
              </p>
              <div className="h-[180px]">
                {energyTab === "day" ? (
                  powerCurveQuery.isLoading ? (
                    <div className="h-full bg-surface-dim rounded-lg animate-pulse" />
                  ) : powerCurvePoints.length < 2 ? (
                    <EmptyState compact title="Not enough data yet" body="Today's generation curve will chart here once there are more readings." />
                  ) : (
                    <LineChart points={powerCurvePoints} color="#2f8fe0" xAxisLabels={powerCurveTicks} />
                  )
                ) : energyQuery.isLoading ? (
                  <div className="h-full bg-surface-dim rounded-lg animate-pulse" />
                ) : energyPoints.length < 2 ? (
                  <EmptyState compact title="Not enough data yet" body="Energy generation will chart here once there's more history." />
                ) : (
                  <LineChart points={energyPoints} color="#2f8fe0" />
                )}
              </div>
            </div>

            <div className="glass-card rounded-xl p-6 flex flex-col">
              <span className="font-label-caps text-label-caps text-on-surface-variant uppercase self-start">Fleet Status</span>
              {devicesQuery.isLoading ? (
                <div className="flex-1 flex items-center justify-center"><div className="h-24 w-24 rounded-full bg-surface-dim animate-pulse" /></div>
              ) : (
                <>
                  <div className="flex items-center justify-center mt-4">
                    <StatusDonut
                      segments={[
                        { value: onlineCount, className: "stroke-success" },
                        { value: offlineCount, className: "stroke-error" },
                        { value: faultCount, className: "stroke-secondary" },
                        { value: noDataCount, className: "stroke-outline" },
                      ]}
                      size={120}
                      strokeWidth={12}
                      centerValue={String(classifiedTotal)}
                      centerLabel="Devices"
                    />
                  </div>
                  <div className="mt-4 space-y-1.5 text-[12px]">
                    <div className="flex items-center gap-2">
                      <span className="w-2 h-2 rounded-full bg-success flex-shrink-0" />
                      <span className="text-on-surface-variant">Online</span>
                      <span className="ml-auto font-semibold text-on-surface font-data-mono-sm">{onlineCount}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="w-2 h-2 rounded-full bg-error flex-shrink-0" />
                      <span className="text-on-surface-variant">Offline</span>
                      <span className="ml-auto font-semibold text-on-surface font-data-mono-sm">{offlineCount}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="w-2 h-2 rounded-full bg-secondary flex-shrink-0" />
                      <span className="text-on-surface-variant">Fault</span>
                      <span className="ml-auto font-semibold text-on-surface font-data-mono-sm">{faultCount}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="w-2 h-2 rounded-full bg-outline flex-shrink-0" />
                      <span className="text-on-surface-variant">No Data</span>
                      <span className="ml-auto font-semibold text-on-surface font-data-mono-sm">{noDataCount}</span>
                    </div>
                  </div>
                  <Link
                    to="/app/devices"
                    className="mt-4 text-center text-[12px] font-semibold text-primary hover:underline"
                  >
                    View All Devices →
                  </Link>
                </>
              )}
            </div>
          </div>

          {/* Site Map preview + Top Performing Sites */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-gutter">
            <div className="glass-card rounded-xl p-6">
              <div className="flex justify-between items-center mb-4">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Site Map</span>
              </div>
              {sitesQuery.isLoading ? (
                <div className="h-[220px] bg-surface-dim rounded-lg animate-pulse" />
              ) : (
                <FleetMiniMap sites={sites} healthBySite={healthBySite} height={220} zoom={5} compact />
              )}
              <Link to="/app/map" className="mt-3 block text-center text-[12px] font-semibold text-primary hover:underline">
                View full map →
              </Link>
            </div>

            <div className="lg:col-span-2 glass-card rounded-xl overflow-hidden">
              <div className="p-6 pb-3">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Top Performing Sites (Today)</span>
              </div>
              {topSitesQuery.isLoading ? (
                <div className="h-32 mx-6 mb-6 bg-surface-dim rounded-lg animate-pulse" />
              ) : !topSitesQuery.data || topSitesQuery.data.every((s) => s.energy_kwh === 0) ? (
                <div className="px-6 pb-6">
                  <EmptyState compact title="No generation yet today" body="Top sites will rank here once today's readings come in." />
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[520px] text-[13px]">
                    <thead>
                      <tr className="text-left text-on-surface-variant border-t border-outline-variant/60">
                        <th className="px-6 py-2 font-label-caps text-label-caps">Site</th>
                        <th className="px-3 py-2 font-label-caps text-label-caps text-right">Generation</th>
                        <th className="px-3 py-2 font-label-caps text-label-caps text-right">Capacity</th>
                        <th className="px-6 py-2 font-label-caps text-label-caps text-right">Specific Yield</th>
                      </tr>
                    </thead>
                    <tbody>
                      {topSitesQuery.data.map((s) => (
                        <tr key={s.site_id} className="border-t border-outline-variant/40 hover:bg-white/50 transition-colors">
                          <td className="px-6 py-2.5">
                            <Link to={`/app/sites/${s.site_id}`} className="text-on-surface hover:text-primary transition-colors">
                              {s.name ?? s.site_id}
                            </Link>
                          </td>
                          <td className="px-3 py-2.5 text-right font-data-mono-sm text-data-mono-sm text-on-surface">
                            {s.energy_kwh.toFixed(1)} kWh
                          </td>
                          <td className="px-3 py-2.5 text-right font-data-mono-sm text-data-mono-sm text-on-surface-variant">
                            {s.system_size_kw != null ? `${s.system_size_kw.toFixed(1)} kWp` : "—"}
                          </td>
                          <td className="px-6 py-2.5 text-right font-data-mono-sm text-data-mono-sm text-on-surface-variant">
                            {s.specific_yield_kwh_per_kwp.toFixed(2)} kWh/kWp
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>

          {/* Energy & Emissions Summary + Recent Alerts */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-gutter">
            <div className="lg:col-span-2 glass-card rounded-xl p-6">
              <div className="flex flex-wrap justify-between items-center gap-3 mb-4">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Energy &amp; Emissions Summary</span>
                <div className="flex gap-1 glass-card rounded-full p-1">
                  <button
                    onClick={() => setSummaryTab("energy")}
                    className={`px-3 py-1 rounded-full text-[12px] transition-colors ${summaryTab === "energy" ? "bg-primary text-on-primary font-semibold" : "text-on-surface-variant hover:text-on-surface"}`}
                  >
                    Energy (kWh)
                  </button>
                  <button
                    onClick={() => setSummaryTab("emissions")}
                    className={`px-3 py-1 rounded-full text-[12px] transition-colors ${summaryTab === "emissions" ? "bg-primary text-on-primary font-semibold" : "text-on-surface-variant hover:text-on-surface"}`}
                  >
                    CO2 Avoided (kg)
                  </button>
                </div>
              </div>
              <div className="h-[200px]">
                {(summaryTab === "energy" ? recentEnergyQuery.isLoading : recentEmissionsQuery.isLoading) ? (
                  <div className="h-full bg-surface-dim rounded-lg animate-pulse" />
                ) : summaryBarPoints.length === 0 ? (
                  <EmptyState compact title="Not enough data yet" body="This chart fills in once there's a few days of history." />
                ) : (
                  <BarChart
                    points={summaryBarPoints}
                    color={summaryTab === "energy" ? "#2f8fe0" : "#1a9c6b"}
                    height={200}
                    valueFormatter={(v) => (summaryTab === "energy" ? `${v.toFixed(0)} kWh` : `${v.toFixed(0)} kg`)}
                  />
                )}
              </div>
              {latestTrend && (
                <p className="text-[12px] text-on-surface-variant mt-3">
                  Total this month: <span className="font-semibold text-on-surface">{latestTrend.total_energy_kwh.toFixed(0)} kWh</span>
                  {latestTrend.mom_change_pct != null && (
                    <span className={latestTrend.mom_change_pct >= 0 ? "text-success ml-2" : "text-error ml-2"}>
                      {latestTrend.mom_change_pct >= 0 ? "▲" : "▼"} {Math.abs(latestTrend.mom_change_pct).toFixed(1)}% vs last month
                    </span>
                  )}
                </p>
              )}
            </div>

            <div className="glass-card rounded-xl overflow-hidden flex flex-col">
              <div className="p-6 pb-3 flex justify-between items-center">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Recent Alerts</span>
                <Link to="/app/alerts" className="text-[12px] font-semibold text-primary hover:underline">
                  View All
                </Link>
              </div>
              {alertsQuery.isLoading ? (
                <div className="h-32 mx-6 mb-6 bg-surface-dim rounded-lg animate-pulse" />
              ) : !alertsQuery.data || alertsQuery.data.length === 0 ? (
                <div className="px-6 pb-6">
                  <EmptyState compact title="All clear" body="No active alerts right now." />
                </div>
              ) : (
                <div className="divide-y divide-outline-variant/40">
                  {alertsQuery.data.slice(0, 4).map((alert, i) => {
                    const Icon = alert.type === "device_fault" ? AlertTriangle : alert.type === "device_revoked" ? ShieldOff : alert.type === "device_offline" ? PowerOff : Info;
                    return (
                      <div key={i} className="px-6 py-3 flex items-start gap-3">
                        <Icon
                          size={16}
                          className={`mt-0.5 flex-shrink-0 ${alert.severity === "critical" ? "text-error" : alert.severity === "warning" ? "text-secondary" : "text-on-surface-variant"}`}
                        />
                        <div>
                          <p className="text-[13px] text-on-surface leading-tight">{alert.message}</p>
                          <p className="text-[11px] text-on-surface-variant mt-0.5">
                            {alert.site_name ?? alert.site_id} · {new Date(alert.occurred_at).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })}
                          </p>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>

          {/* Environmental Impact + Performance & Health + Energy Flow */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-gutter">
            <EnvironmentalImpactPanel cumulativeTonnesCO2={emissionsUnconfigured ? null : emissionsQuery.data?.cumulative_lifetime_co2_tonnes ?? null} />

            <div className="glass-card rounded-xl p-6">
              <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Performance &amp; Health</span>
              {/* Only real fleet-health fields — "Inverter Efficiency" and
                  "Battery Health" from the reference screenshot have no
                  data source in this platform (no battery/inverter
                  telemetry concept exists), so aren't reproduced here. */}
              <div className="mt-4 flex justify-around">
                <CircularProgress
                  percent={healthQuery.data?.fleet.devices_reporting_pct ?? 0}
                  size={88}
                  strokeWidth={9}
                  color="#2f8fe0"
                  label="System Availability"
                />
                <CircularProgress
                  percent={healthQuery.data?.fleet.coverage_pct ?? 0}
                  size={88}
                  strokeWidth={9}
                  color="#1a9c6b"
                  label="Fleet Coverage"
                />
              </div>
              <p className="text-[10px] text-on-surface-variant text-center mt-3">
                {(healthQuery.data?.fleet.devices_reporting_pct ?? 0) >= 90 ? "Good" : (healthQuery.data?.fleet.devices_reporting_pct ?? 0) >= 50 ? "Fair" : "Needs Attention"}
              </p>
            </div>

            <div className="glass-card rounded-xl p-6">
              <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Energy Flow</span>
              <EnergyFlowIllustration
                solar={{ label: "Solar Generation", value: energyQuery.data ? `${(energyQuery.data.cumulative_kwh / 1000).toFixed(2)} MWh` : "—", available: !!energyQuery.data }}
                battery={{
                  label: "Battery Storage",
                  value: currentFlowQuery.data?.avg_battery_soc_pct != null ? `${currentFlowQuery.data.avg_battery_soc_pct.toFixed(0)}%` : "—",
                  available: currentFlowQuery.data?.avg_battery_soc_pct != null,
                }}
                grid={{
                  // Sign convention per PV Pro's gridOrMeterPower: positive
                  // = drawing from the grid, negative = exporting excess
                  // solar back to it — flag back if a real export event
                  // ever shows the opposite of what's expected here.
                  label: "Grid Import/Export",
                  value: currentFlowQuery.data
                    ? `${Math.abs(currentFlowQuery.data.grid_kw).toFixed(1)} kW ${currentFlowQuery.data.grid_kw < 0 ? "Export" : "Import"}`
                    : "—",
                  available: !!currentFlowQuery.data,
                }}
                consumption={{
                  label: "Consumption",
                  value: currentFlowQuery.data ? `${currentFlowQuery.data.load_kw.toFixed(1)} kW` : "—",
                  available: !!currentFlowQuery.data,
                }}
                animated={false}
                height={220}
              />
            </div>
          </div>
        </>
      )}

      {healthQuery.data && healthQuery.data.fleet.coverage_pct < 50 && (
        <div className="glass-card rounded-xl px-5 py-3 flex items-center gap-3 text-on-surface-variant">
          <Bell size={16} className="text-secondary" />
          <span className="font-body-base text-body-base">
            Fleet coverage is at {healthQuery.data.fleet.coverage_pct.toFixed(0)}% — check{" "}
            <Link to="/app/fleet-health" className="text-primary underline">
              Fleet Health
            </Link>{" "}
            for details.
          </span>
        </div>
      )}
      </div>
    </>
  );
}
