import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { HomeIcon, Zap, Leaf, TrendingUp, HeartPulse, ScrollText, BarChart3, UserPlus, Bell } from "lucide-react";
import { KpiCard } from "../components/kpi/KpiCard";
import { CircularProgress } from "../components/kpi/CircularProgress";
import { WeatherWidget } from "../components/dashboard/WeatherWidget";
import { EnvironmentalImpactPanel } from "../components/dashboard/EnvironmentalImpactPanel";
import { EnergyFlowIllustration } from "../components/dashboard/EnergyFlowIllustration";
import { LineChart } from "../components/charts/LineChart";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { useAuth } from "../auth/AuthContext";
import { getFleetSummary } from "../api/fleet";
import { getFleetEnergy } from "../api/analytics";
import { getFleetTrends } from "../api/benchmark";
import { getFleetEmissions } from "../api/emissions";
import { getFleetHealth } from "../api/fleetHealth";
import { listSites } from "../api/sites";
import { ApiError } from "../api/types";

// Light/glass redesign — replaces the earlier dark-industrial Fleet
// Dashboard entirely. Every number here is real (fleet summary, fleet
// health, fleet energy/trends, fleet emissions, real weather for the
// first site with coordinates) — the previous mockup's "Total Revenue"
// KPI and named "Alex" greeting aren't reproduced: this platform has no
// billing/pricing feature, and there's no user-profile endpoint yet to
// know a real display name, so the greeting uses time-of-day + role
// instead of inventing a name.
function greeting() {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
}

export function FleetDashboardPage() {
  const { session } = useAuth();
  const isOperator = session?.role === "operator";

  const summaryQuery = useQuery({ queryKey: ["fleet-summary"], queryFn: getFleetSummary });
  const energyQuery = useQuery({ queryKey: ["fleet-energy-30d"], queryFn: () => getFleetEnergy() });
  const trendsQuery = useQuery({ queryKey: ["fleet-trends-dash"], queryFn: () => getFleetTrends() });
  const emissionsQuery = useQuery({ queryKey: ["fleet-emissions-dash"], queryFn: () => getFleetEmissions(), retry: false });
  const healthQuery = useQuery({ queryKey: ["fleet-health-dash"], queryFn: () => getFleetHealth() });
  const sitesQuery = useQuery({ queryKey: ["sites-for-weather"], queryFn: () => listSites(undefined, 50) });

  if (summaryQuery.isError) {
    if (summaryQuery.error instanceof ApiError && summaryQuery.error.status === 403) return <AccessDenied />;
    return <ErrorState onRetry={() => summaryQuery.refetch()} />;
  }

  const data = summaryQuery.data;
  const energyPoints = (energyQuery.data?.points ?? []).map((p, i) => ({ x: i, y: p.energy_kwh }));
  const latestTrend = trendsQuery.data?.points.at(-1)?.mom_change_pct ?? null;
  const emissionsUnconfigured = emissionsQuery.error instanceof ApiError && emissionsQuery.error.status === 409;
  const firstSiteWithLocation = (sitesQuery.data?.items ?? []).find((s) => s.gps_lat != null && s.gps_lng != null);

  const healthSites = healthQuery.data?.sites.items ?? [];
  const activeSites = healthSites.filter((s) => s.online_devices > 0).length;
  const idleSites = healthSites.filter((s) => s.online_devices === 0).length;
  const capacityPct = healthSites.length > 0 ? (activeSites / healthSites.length) * 100 : 0;

  return (
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
        <WeatherWidget
          lat={firstSiteWithLocation?.gps_lat}
          lng={firstSiteWithLocation?.gps_lng}
          siteName={firstSiteWithLocation?.name ?? firstSiteWithLocation?.site_id}
        />
      </div>

      {summaryQuery.isLoading || !data ? (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-gutter">
          {[0, 1, 2, 3].map((i) => (
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
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-gutter">
            <KpiCard
              label="Total Sites"
              value={data.total_sites}
              tone="primary"
              icon={<HomeIcon size={16} />}
            />
            <KpiCard
              label="Total Capacity"
              value={data.total_capacity_kw != null ? (data.total_capacity_kw / 1000).toFixed(1) : "—"}
              unit={data.total_capacity_kw != null ? "MWp" : undefined}
              icon={<Zap size={16} />}
            />
            <KpiCard
              label="Energy (30d)"
              value={energyQuery.data ? (energyQuery.data.cumulative_kwh / 1000).toFixed(2) : "—"}
              unit={energyQuery.data ? "MWh" : undefined}
              icon={<TrendingUp size={16} />}
              trendPct={latestTrend}
            />
            <KpiCard
              label="CO2 Offset"
              value={emissionsQuery.data ? emissionsQuery.data.cumulative_lifetime_co2_tonnes.toFixed(2) : "—"}
              unit={emissionsQuery.data ? "t" : undefined}
              icon={<Leaf size={16} />}
            />
          </div>

          {/* Capacity ring + Energy Generation chart */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-gutter">
            <div className="glass-card rounded-xl p-6 flex flex-col items-center justify-center gap-4">
              <span className="font-label-caps text-label-caps text-on-surface-variant uppercase self-start">Total Capacity</span>
              <CircularProgress
                percent={capacityPct}
                size={140}
                strokeWidth={14}
                color="#2f8fe0"
                value={`${activeSites}/${healthSites.length || data.total_sites}`}
              />
              <div className="flex gap-6 text-[12px] text-on-surface-variant">
                <span><span className="text-primary font-semibold">{activeSites}</span> Active</span>
                <span><span className="font-semibold">{idleSites}</span> Idle</span>
              </div>
            </div>

            <div className="lg:col-span-2 glass-card rounded-xl p-6">
              <div className="flex justify-between items-center mb-4">
                <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Energy Generation</span>
                <span className="font-data-display-lg text-[20px] text-on-surface">
                  {energyQuery.data ? `${energyQuery.data.cumulative_kwh.toFixed(0)} kWh` : "—"}
                </span>
              </div>
              <div className="h-[180px]">
                {energyQuery.isLoading ? (
                  <div className="h-full bg-surface-dim rounded-lg animate-pulse" />
                ) : energyPoints.length < 2 ? (
                  <EmptyState title="Not enough data yet" body="Energy generation will chart here once there's more history." />
                ) : (
                  <LineChart points={energyPoints} color="#2f8fe0" />
                )}
              </div>
            </div>
          </div>

          {/* Environmental Impact + Performance & Health */}
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
                battery={{ label: "Battery Storage", value: "—", available: false }}
                grid={{ label: "Grid Import/Export", value: "—", available: false }}
                consumption={{ label: "Consumption", value: "—", available: false }}
                animated={false}
                height={220}
              />
            </div>
          </div>
        </>
      )}

      {/* Quick links to sections without dedicated nav/sidebar slots on
          mobile — see Sidebar.tsx's comment on why. */}
      <div className="flex flex-wrap gap-3">
        <Link to="/app/analytics" className="flex items-center gap-2 glass-card rounded-full px-4 py-2 text-on-surface-variant hover:text-primary transition-colors">
          <BarChart3 size={16} /> <span>Fleet Analytics</span>
        </Link>
        <Link to="/app/fleet-health" className="flex items-center gap-2 glass-card rounded-full px-4 py-2 text-on-surface-variant hover:text-primary transition-colors">
          <HeartPulse size={16} /> <span>Fleet Health</span>
        </Link>
        <Link to="/app/audit" className="flex items-center gap-2 glass-card rounded-full px-4 py-2 text-on-surface-variant hover:text-primary transition-colors">
          <ScrollText size={16} /> <span>Audit Log</span>
        </Link>
        <Link to="/app/users/invite" className="flex items-center gap-2 glass-card rounded-full px-4 py-2 text-on-surface-variant hover:text-primary transition-colors">
          <UserPlus size={16} /> <span>Invite User</span>
        </Link>
      </div>

      {/* "Maintenance"/"Billing"/"Reports" nav items from the reference
          screenshot aren't real features here yet — no backend for
          scheduled maintenance, invoicing, or a reports-job system
          exists, so they're deliberately left out rather than added as
          dead links. Alerts similarly aren't a persisted, actionable
          feature yet (anomalies are queryable, not pushed) — the bell
          below is a placeholder for that gap, not a working inbox. */}
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
  );
}
