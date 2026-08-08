import { useQuery } from "@tanstack/react-query";
import { Link, NavLink } from "react-router-dom";
import {
  Grid2x2,
  MapPin,
  Radio,
  LogOut,
  Map,
  Bell,
  Gauge,
  Zap,
  Leaf,
  FileBarChart,
  Layers,
  Users,
  HeartPulse,
  ScrollText,
  FileClock,
  Settings,
} from "lucide-react";
import { useAuth } from "../../auth/AuthContext";
import { LogoMark } from "../brand/Logo";
import { getIngestionStatus } from "../../api/fleet";

// Freshness threshold for "operational" — deliberately the same
// ONLINE_THRESHOLD_MINUTES concept the rest of the platform uses for a
// single device, applied here to "has the ingestor seen ANY message
// fleet-wide recently" — not a synthetic uptime percentage, since
// there's no real data source for one.
const STALE_MINUTES = 15;

function IngestionStatusWidget() {
  const { data } = useQuery({
    queryKey: ["ingestion-status"],
    queryFn: () => getIngestionStatus(),
    refetchInterval: 60_000,
  });

  if (!data) return null;

  const lastReceivedAt = data.lastReceivedAt ? new Date(data.lastReceivedAt) : null;
  const minutesAgo = lastReceivedAt ? (Date.now() - lastReceivedAt.getTime()) / 60_000 : null;
  const operational = minutesAgo !== null && minutesAgo < STALE_MINUTES;

  return (
    <div className="mx-3 mb-3 glass-card rounded-xl p-3">
      <div className="flex items-center gap-2">
        <span className={`w-2 h-2 rounded-full ${operational ? "bg-success" : "bg-error"} flex-shrink-0`} />
        <div>
          <p className="text-[12px] font-semibold text-on-surface leading-tight">Data Ingestion Status</p>
          <p className={`text-[11px] leading-tight ${operational ? "text-success" : "text-error"}`}>
            {lastReceivedAt === null ? "No data received yet" : operational ? "All Systems Operational" : "No recent data"}
          </p>
        </div>
      </div>
      {lastReceivedAt && (
        <p className="text-[10px] text-on-surface-variant mt-1.5">
          Last received: {minutesAgo! < 1 ? "just now" : `${Math.round(minutesAgo!)} min ago`}
        </p>
      )}
    </div>
  );
}

// Pill-highlighted nav items per the light/glass redesign — no more
// left/right border accent, a soft rounded background instead.
const navItemClass = ({ isActive }: { isActive: boolean }) =>
  [
    "flex items-center gap-3 px-4 py-2 rounded-full transition-colors font-body-base text-body-base text-[14px]",
    isActive
      ? "text-on-primary-container font-semibold bg-primary-container"
      : "text-on-surface-variant hover:bg-surface-dim hover:text-on-surface",
  ].join(" ");

const sectionLabelClass = "px-4 pt-5 pb-1.5 font-label-caps text-label-caps text-on-surface-variant/70 uppercase tracking-widest";

// Grouped into Main/Analytics/Management/System sections — structure
// only, not a copied visual system: still the light/glass tokens/
// components used everywhere else in this app, just reorganized
// information architecture (site-level analytics/health/audit are
// reached from Site Detail or their own admin flows, not duplicated
// here). Everything below operator-only stays operator-only; nothing
// here loosens the existing role checks in routes.tsx/router.go.
//
// Responsive per DESIGN.md's documented breakpoint (mobile < 768px: "side
// nav collapses to a bottom bar"). Below md, this component renders
// nothing; MobileNav (same file) renders the bottom bar instead, used
// together in AppLayout.
export function Sidebar() {
  const { session, logout } = useAuth();
  const isOperator = session?.role === "operator";

  return (
    <aside className="hidden md:flex fixed left-4 top-4 bottom-4 w-[240px] glass-card rounded-2xl flex-col py-grid-margin z-50">
      <Link to="/" className="px-6 mb-6 flex items-start gap-2 hover:opacity-90 transition-opacity" aria-label="Back to home">
        <LogoMark size={20} />
        <h1 className="font-headline-md text-headline-md font-bold text-on-surface leading-tight">
          Clean Energy Analytics
        </h1>
      </Link>
      <nav className="flex-1 px-3 space-y-0.5 overflow-y-auto">
        <p className={sectionLabelClass}>Main</p>
        {isOperator && (
          <NavLink to="/app" className={navItemClass} end>
            <Grid2x2 size={18} />
            <span>Dashboard</span>
          </NavLink>
        )}
        <NavLink to="/app/sites" className={navItemClass}>
          <MapPin size={18} />
          <span>Sites</span>
        </NavLink>
        <NavLink to="/app/devices" className={navItemClass}>
          <Radio size={18} />
          <span>Devices</span>
        </NavLink>
        {isOperator && (
          <NavLink to="/app/map" className={navItemClass}>
            <Map size={18} />
            <span>Map View</span>
          </NavLink>
        )}
        {isOperator && (
          <NavLink to="/app/alerts" className={navItemClass}>
            <Bell size={18} />
            <span>Alerts</span>
          </NavLink>
        )}

        {isOperator && (
          <>
            <p className={sectionLabelClass}>Analytics</p>
            <NavLink to="/app/analytics/performance" className={navItemClass}>
              <Gauge size={18} />
              <span>Performance</span>
            </NavLink>
            <NavLink to="/app/analytics/energy" className={navItemClass}>
              <Zap size={18} />
              <span>Energy</span>
            </NavLink>
            <NavLink to="/app/analytics/emissions" className={navItemClass}>
              <Leaf size={18} />
              <span>Emissions</span>
            </NavLink>
            <NavLink to="/app/reports" className={navItemClass}>
              <FileBarChart size={18} />
              <span>Reports</span>
            </NavLink>

            <p className={sectionLabelClass}>Management</p>
            <NavLink to="/app/cohorts" className={navItemClass}>
              <Layers size={18} />
              <span>Cohorts / Projects</span>
            </NavLink>
            <NavLink to="/app/users" className={navItemClass}>
              <Users size={18} />
              <span>Users &amp; Roles</span>
            </NavLink>
            <NavLink to="/app/devices/new" className={navItemClass}>
              <Radio size={18} />
              <span>Device Registry</span>
            </NavLink>

            <p className={sectionLabelClass}>System</p>
            <NavLink to="/app/fleet-health" className={navItemClass}>
              <HeartPulse size={18} />
              <span>Fleet Health</span>
            </NavLink>
            <NavLink to="/app/settings" className={navItemClass}>
              <Settings size={18} />
              <span>Settings</span>
            </NavLink>
            <NavLink to="/app/audit" className={navItemClass}>
              <ScrollText size={18} />
              <span>Audit Log</span>
            </NavLink>
          </>
        )}
        <NavLink to="/app/ingestion-log" className={navItemClass}>
          <FileClock size={18} />
          <span>Ingestion Log</span>
        </NavLink>
      </nav>
      {isOperator && <IngestionStatusWidget />}
      <div className="px-3 pt-4 border-t border-outline-variant">
        <button
          onClick={logout}
          className="w-full flex items-center gap-3 px-4 py-2.5 text-on-surface-variant hover:bg-surface-dim hover:text-on-surface transition-colors rounded-full font-body-base"
        >
          <LogOut size={20} />
          <span>Logout</span>
        </button>
      </div>
    </aside>
  );
}

const mobileNavItemClass = ({ isActive }: { isActive: boolean }) =>
  [
    "flex flex-col items-center justify-center gap-0.5 px-4 py-1 rounded-full transition-transform active:scale-95 font-label-caps text-label-caps",
    isActive ? "bg-primary-container text-on-primary-container" : "text-on-surface-variant",
  ].join(" ");

// Bottom tab bar — reference: design/fleet_overview_zgnis_mobile/code.html's
// bottom nav pattern, adapted to this app's actual 4 sections (Fleet/
// Sites/Devices/Alerts) rather than that mockup's own Home/Alerts/
// Profile — there's still no profile page, but Alerts is real now.
export function MobileNav() {
  const { session } = useAuth();
  const isOperator = session?.role === "operator";

  return (
    <nav className="md:hidden fixed bottom-3 left-3 right-3 z-50 flex justify-around items-center h-16 px-density-base glass-card rounded-2xl">
      {isOperator && (
        <NavLink to="/app" className={mobileNavItemClass} end>
          <Grid2x2 size={20} />
          <span>Fleet</span>
        </NavLink>
      )}
      <NavLink to="/app/sites" className={mobileNavItemClass}>
        <MapPin size={20} />
        <span>Sites</span>
      </NavLink>
      <NavLink to="/app/devices" className={mobileNavItemClass}>
        <Radio size={20} />
        <span>Devices</span>
      </NavLink>
      {isOperator && (
        <NavLink to="/app/alerts" className={mobileNavItemClass}>
          <Bell size={20} />
          <span>Alerts</span>
        </NavLink>
      )}
    </nav>
  );
}
