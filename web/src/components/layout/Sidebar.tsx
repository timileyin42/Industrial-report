import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
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
  FlaskConical,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
  Menu,
  X,
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

interface NavEntry {
  to: string;
  icon: ComponentType<{ size?: number }>;
  label: string;
  end?: boolean;
  operatorOnly?: boolean;
}

interface NavSection {
  label: string | null; // null = ungrouped (Main section, no header)
  items: NavEntry[];
  operatorOnly?: boolean; // whole section gated, not just individual items
}

// Single source of truth for every nav destination — both the desktop
// Sidebar and the mobile "More" drawer render from this, so the two can
// never drift apart the way two hand-duplicated lists would.
const NAV_SECTIONS: NavSection[] = [
  {
    label: "Main",
    items: [
      { to: "/app", icon: Grid2x2, label: "Dashboard", end: true, operatorOnly: true },
      { to: "/app/sites", icon: MapPin, label: "Sites" },
      { to: "/app/devices", icon: Radio, label: "Devices" },
      { to: "/app/map", icon: Map, label: "Map View", operatorOnly: true },
      { to: "/app/alerts", icon: Bell, label: "Alerts", operatorOnly: true },
    ],
  },
  {
    label: "Analytics",
    operatorOnly: true,
    items: [
      { to: "/app/analytics/performance", icon: Gauge, label: "Performance" },
      { to: "/app/analytics/energy", icon: Zap, label: "Energy" },
      { to: "/app/analytics/emissions", icon: Leaf, label: "Emissions" },
      { to: "/app/reports", icon: FileBarChart, label: "Reports" },
    ],
  },
  {
    label: "Management",
    operatorOnly: true,
    items: [
      { to: "/app/cohorts", icon: Layers, label: "Cohorts / Projects" },
      { to: "/app/users", icon: Users, label: "Users & Roles" },
      { to: "/app/devices/new", icon: Radio, label: "Device Registry" },
    ],
  },
  {
    label: "System",
    operatorOnly: true,
    items: [
      { to: "/app/fleet-health", icon: HeartPulse, label: "Fleet Health" },
      { to: "/app/settings", icon: Settings, label: "Settings" },
      { to: "/app/audit", icon: ScrollText, label: "Audit Log" },
    ],
  },
  {
    label: null,
    items: [
      { to: "/app/ingestion-log", icon: FileClock, label: "Ingestion Log" },
      // /sandbox is a public page outside the authenticated /app tree
      // entirely (see routes.tsx) — this link just gives an already
      // logged-in operator a quick way to reach it too, same page
      // anyone with the share link would land on.
      { to: "/sandbox", icon: FlaskConical, label: "Sandbox" },
    ],
  },
];

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
  onClick,
}: {
  to: string;
  icon: ComponentType<{ size?: number }>;
  label: string;
  collapsed: boolean;
  end?: boolean;
  onClick?: () => void;
}) {
  return (
    <NavLink to={to} end={end} onClick={onClick} className={navItemClass(collapsed)} title={collapsed ? label : undefined}>
      <Icon size={18} />
      {!collapsed && <span>{label}</span>}
    </NavLink>
  );
}

const sectionLabelClass = "px-4 pt-5 pb-1.5 font-label-caps text-label-caps text-on-surface-variant/70 uppercase tracking-widest";

function renderSections(isOperator: boolean, collapsed: boolean, onNavigate?: () => void) {
  return NAV_SECTIONS.map((section) => {
    if (section.operatorOnly && !isOperator) return null;
    const items = section.items.filter((item) => isOperator || !item.operatorOnly);
    if (items.length === 0) return null;
    return (
      <div key={section.label ?? "ungrouped"}>
        {section.label && !collapsed && <p className={sectionLabelClass}>{section.label}</p>}
        {items.map((item) => (
          <NavItem key={item.to} {...item} collapsed={collapsed} onClick={onNavigate} />
        ))}
      </div>
    );
  });
}

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
// nothing; MobileNav (same file) renders the bottom bar + "More" drawer
// instead, used together in AppLayout. The drawer covers every item this
// component does (via the shared NAV_SECTIONS above) — the bottom bar
// alone only fits 4 destinations, which used to mean everything else
// (Reports, Users, Settings, Audit Log, ...) was simply unreachable on a
// phone.
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
        {renderSections(isOperator, collapsed)}
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

// Bottom tab bar (4 most common destinations) + a "More" button opening
// a full-height drawer with every other nav item (see NAV_SECTIONS) —
// reference: design/fleet_overview_zgnis_mobile/code.html's bottom nav
// pattern for the 4-item bar itself, extended here since that mockup
// never accounted for the other ~10 destinations this app actually has.
export function MobileNav() {
  const { session, logout } = useAuth();
  const isOperator = session?.role === "operator";
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <>
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
        <button onClick={() => setMenuOpen(true)} className={mobileNavItemClass({ isActive: false })}>
          <Menu size={20} />
          <span>More</span>
        </button>
      </nav>

      {menuOpen && (
        <div className="md:hidden fixed inset-0 z-[70] bg-black/40" onClick={() => setMenuOpen(false)}>
          <div
            className="absolute right-0 top-0 bottom-0 w-[280px] max-w-[85vw] bg-background overflow-y-auto p-4 shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between mb-4 px-2">
              <span className="font-headline-md text-headline-md font-bold text-on-surface">Menu</span>
              <button onClick={() => setMenuOpen(false)} className="text-on-surface-variant hover:text-on-surface p-1" title="Close menu">
                <X size={22} />
              </button>
            </div>
            <nav className="space-y-0.5 px-1">{renderSections(isOperator, false, () => setMenuOpen(false))}</nav>
            <div className="pt-4 mt-4 border-t border-outline-variant px-1">
              <button
                onClick={logout}
                className="w-full flex items-center gap-3 px-4 py-2.5 rounded-full text-on-surface-variant hover:bg-surface-dim hover:text-on-surface transition-colors font-body-base"
              >
                <LogOut size={20} />
                <span>Logout</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
