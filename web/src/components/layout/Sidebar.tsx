import { useQuery } from "@tanstack/react-query";
import { Link, NavLink } from "react-router-dom";
import type { ComponentType } from "react";
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
  PanelLeftClose,
  PanelLeftOpen,
} from "lucide-react";
import { useAuth } from "../../auth/AuthContext";
import { LogoMark } from "../brand/Logo";
import { getIngestionStatus } from "../../api/fleet";
import { useSidebar } from "./SidebarContext";

// Freshness threshold for "operational" — deliberately the same
// ONLINE_THRESHOLD_MINUTES concept the rest of the platform uses for a
// single device, applied here to "has the ingestor seen ANY message
// fleet-wide recently" — not a synthetic uptime percentage, since
// there's no real data source for one.
const STALE_MINUTES = 15;

function IngestionStatusWidget({ collapsed }: { collapsed: boolean }) {
  const { data } = useQuery({
    queryKey: ["ingestion-status"],
    queryFn: () => getIngestionStatus(),
    refetchInterval: 60_000,
  });

  if (!data) return null;

  const lastReceivedAt = data.lastReceivedAt ? new Date(data.lastReceivedAt) : null;
  const minutesAgo = lastReceivedAt ? (Date.now() - lastReceivedAt.getTime()) / 60_000 : null;
  const operational = minutesAgo !== null && minutesAgo < STALE_MINUTES;

  if (collapsed) {
    return (
      <div className="mx-auto mb-3" title={operational ? "All Systems Operational" : "No recent data"}>
        <span className={`block w-2.5 h-2.5 rounded-full ${operational ? "bg-success" : "bg-error"}`} />
      </div>
    );
  }

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
// left/right border accent, a soft rounded background instead. Centered,
// icon-only (no gap needed) when the sidebar is collapsed.
const navItemClass = (collapsed: boolean) =>
  ({ isActive }: { isActive: boolean }) =>
    [
      "flex items-center rounded-full transition-colors font-body-base text-body-base text-[14px]",
      collapsed ? "justify-center px-0 py-2.5" : "gap-3 px-4 py-2",
      isActive
        ? "text-on-primary-container font-semibold bg-primary-container"
        : "text-on-surface-variant hover:bg-surface-dim hover:text-on-surface",
    ].join(" ");

function NavItem({
  to,
  icon: Icon,
  label,
  collapsed,
  end,
}: {
  to: string;
  icon: ComponentType<{ size?: number }>;
  label: string;
  collapsed: boolean;
  end?: boolean;
}) {
  return (
    <NavLink to={to} end={end} className={navItemClass(collapsed)} title={collapsed ? label : undefined}>
      <Icon size={18} />
      {!collapsed && <span>{label}</span>}
    </NavLink>
  );
}

const sectionLabelClass = "px-4 pt-5 pb-1.5 font-label-caps text-label-caps text-on-surface-variant/70 uppercase tracking-widest";

// Grouped into Main/Analytics/Management/System sections — structure
// only, not a copied visual system: still the light/glass tokens/
// components used everywhere else in this app, just reorganized
// information architecture (site-level analytics/health/audit are
// reached from Site Detail or their own admin flows, not duplicated
// here). Everything below operator-only stays operator-only; nothing
// here loosens the existing role checks in routes.tsx/router.go.
//
// Collapsible to an icon-only rail (see SidebarContext) — AppLayout
// reads the same collapsed state to shrink the main content's left
// margin to match, so the two never drift out of sync.
//
// Responsive per DESIGN.md's documented breakpoint (mobile < 768px: "side
// nav collapses to a bottom bar"). Below md, this component renders
// nothing; MobileNav (same file) renders the bottom bar instead, used
// together in AppLayout.
export function Sidebar() {
  const { session, logout } = useAuth();
  const { collapsed, toggle } = useSidebar();
  const isOperator = session?.role === "operator";

  return (
    <aside
      className={`hidden md:flex fixed left-4 top-4 bottom-4 glass-card rounded-2xl flex-col py-grid-margin z-50 transition-[width] duration-200 ${
        collapsed ? "w-[72px]" : "w-[240px]"
      }`}
    >
      <div className={`flex items-center mb-6 ${collapsed ? "flex-col gap-3 px-0" : "justify-between px-6"}`}>
        <Link to="/" className="flex items-center gap-2 hover:opacity-90 transition-opacity" aria-label="Back to home">
          <LogoMark size={20} />
          {!collapsed && (
            <h1 className="font-headline-md text-headline-md font-bold text-on-surface leading-tight">
              Clean Energy Analytics
            </h1>
          )}
        </Link>
        <button
          onClick={toggle}
          className="text-on-surface-variant hover:text-primary transition-colors flex-shrink-0"
          title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          {collapsed ? <PanelLeftOpen size={18} /> : <PanelLeftClose size={18} />}
        </button>
      </div>
      <nav className={`flex-1 space-y-0.5 overflow-y-auto ${collapsed ? "px-2" : "px-3"}`}>
        {!collapsed && <p className={sectionLabelClass}>Main</p>}
        {isOperator && <NavItem to="/app" icon={Grid2x2} label="Dashboard" collapsed={collapsed} end />}
        <NavItem to="/app/sites" icon={MapPin} label="Sites" collapsed={collapsed} />
        <NavItem to="/app/devices" icon={Radio} label="Devices" collapsed={collapsed} />
        {isOperator && <NavItem to="/app/map" icon={Map} label="Map View" collapsed={collapsed} />}
        {isOperator && <NavItem to="/app/alerts" icon={Bell} label="Alerts" collapsed={collapsed} />}

        {isOperator && (
          <>
            {!collapsed && <p className={sectionLabelClass}>Analytics</p>}
            <NavItem to="/app/analytics/performance" icon={Gauge} label="Performance" collapsed={collapsed} />
            <NavItem to="/app/analytics/energy" icon={Zap} label="Energy" collapsed={collapsed} />
            <NavItem to="/app/analytics/emissions" icon={Leaf} label="Emissions" collapsed={collapsed} />
            <NavItem to="/app/reports" icon={FileBarChart} label="Reports" collapsed={collapsed} />

            {!collapsed && <p className={sectionLabelClass}>Management</p>}
            <NavItem to="/app/cohorts" icon={Layers} label="Cohorts / Projects" collapsed={collapsed} />
            <NavItem to="/app/users" icon={Users} label="Users & Roles" collapsed={collapsed} />
            <NavItem to="/app/devices/new" icon={Radio} label="Device Registry" collapsed={collapsed} />

            {!collapsed && <p className={sectionLabelClass}>System</p>}
            <NavItem to="/app/fleet-health" icon={HeartPulse} label="Fleet Health" collapsed={collapsed} />
            <NavItem to="/app/settings" icon={Settings} label="Settings" collapsed={collapsed} />
            <NavItem to="/app/audit" icon={ScrollText} label="Audit Log" collapsed={collapsed} />
          </>
        )}
        <NavItem to="/app/ingestion-log" icon={FileClock} label="Ingestion Log" collapsed={collapsed} />
      </nav>
      {isOperator && <IngestionStatusWidget collapsed={collapsed} />}
      <div className={`pt-4 border-t border-outline-variant ${collapsed ? "px-2" : "px-3"}`}>
        <button
          onClick={logout}
          title={collapsed ? "Logout" : undefined}
          className={`w-full flex items-center text-on-surface-variant hover:bg-surface-dim hover:text-on-surface transition-colors rounded-full font-body-base ${
            collapsed ? "justify-center py-2.5" : "gap-3 px-4 py-2.5"
          }`}
        >
          <LogOut size={20} />
          {!collapsed && <span>Logout</span>}
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
