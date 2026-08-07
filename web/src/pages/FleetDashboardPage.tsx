import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { HomeIcon, Zap, WifiOff, TrendingUp, HeartPulse, ScrollText, BarChart3, UserPlus } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { KpiCard } from "../components/kpi/KpiCard";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { getFleetSummary } from "../api/fleet";
import { getFleetEnergy } from "../api/analytics";
import { ApiError } from "../api/types";

// Reference: design/fleet_dashboard_zgnis_industrial_intelligence/code.html.
// The mockup's "Today's Generation" card and trend deltas ("+2 new this
// month", "4.2% vs avg") aren't computed by any endpoint and stay
// omitted, but the 4th KPI slot itself is now filled with a real number:
// trailing-30-day fleet energy from GET /v1/fleet/analytics/energy
// (Slice 2's addition — Slice 1 shipped only the 3 fields fleet/summary
// returns directly).
export function FleetDashboardPage() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["fleet-summary"],
    queryFn: getFleetSummary,
  });
  const energyQuery = useQuery({ queryKey: ["fleet-energy-30d"], queryFn: () => getFleetEnergy() });

  if (isError) {
    if (error instanceof ApiError && error.status === 403) return <AccessDenied />;
    return <ErrorState onRetry={() => refetch()} />;
  }

  return (
    <>
      <TopNav title="Fleet Overview" />
      <div className="flex-1 p-grid-margin space-y-8">
        {isLoading || !data ? (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-gutter">
            {[0, 1, 2].map((i) => (
              <div key={i} className="h-32 bg-surface-container border border-outline-variant animate-pulse" />
            ))}
          </div>
        ) : data.total_sites === 0 ? (
          <EmptyState
            title="No sites registered yet"
            body="Begin monitoring your renewable assets by initializing your first fleet location."
          />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-gutter">
            <KpiCard
              label="Total Sites"
              value={data.total_sites}
              tone="primary"
              icon={<HomeIcon size={16} className="text-primary" />}
            />
            <KpiCard
              label="Total Capacity"
              value={data.total_capacity_kw != null ? (data.total_capacity_kw / 1000).toFixed(1) : "—"}
              unit={data.total_capacity_kw != null ? "MWp" : undefined}
              icon={<Zap size={16} className="text-primary" />}
            />
            <KpiCard
              label="Devices Offline"
              value={Math.max(data.total_devices - data.online_devices, 0)}
              tone={data.total_devices - data.online_devices > 0 ? "error" : "primary"}
              icon={<WifiOff size={16} className="text-error" />}
            />
            <KpiCard
              label="Energy (30d)"
              value={
                energyQuery.isLoading
                  ? "…"
                  : energyQuery.data
                    ? (energyQuery.data.cumulative_kwh / 1000).toFixed(2)
                    : "—"
              }
              unit={energyQuery.data ? "MWh" : undefined}
              icon={<TrendingUp size={16} className="text-primary" />}
            />
          </div>
        )}

        {/* Fleet Health and Audit Log live only in the desktop sidebar
            (see Sidebar.tsx) to keep the mobile bottom nav to 4 items —
            these links are the mobile-reachable path to them. */}
        <div className="flex flex-wrap gap-3">
          <Link
            to="/app/analytics"
            className="flex items-center gap-2 bg-surface-container border border-outline-variant hover:border-primary-container text-on-surface-variant hover:text-on-surface font-body-base text-body-base px-4 py-2 rounded transition-colors"
          >
            <BarChart3 size={16} />
            <span>Fleet Analytics</span>
          </Link>
          <Link
            to="/app/fleet-health"
            className="flex items-center gap-2 bg-surface-container border border-outline-variant hover:border-primary-container text-on-surface-variant hover:text-on-surface font-body-base text-body-base px-4 py-2 rounded transition-colors"
          >
            <HeartPulse size={16} />
            <span>Fleet Health</span>
          </Link>
          <Link
            to="/app/audit"
            className="flex items-center gap-2 bg-surface-container border border-outline-variant hover:border-primary-container text-on-surface-variant hover:text-on-surface font-body-base text-body-base px-4 py-2 rounded transition-colors"
          >
            <ScrollText size={16} />
            <span>Audit Log</span>
          </Link>
          <Link
            to="/app/users/invite"
            className="flex items-center gap-2 bg-surface-container border border-outline-variant hover:border-primary-container text-on-surface-variant hover:text-on-surface font-body-base text-body-base px-4 py-2 rounded transition-colors"
          >
            <UserPlus size={16} />
            <span>Invite User</span>
          </Link>
        </div>
      </div>
    </>
  );
}
