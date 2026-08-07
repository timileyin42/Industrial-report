import { Link, NavLink } from "react-router-dom";
import { Grid2x2, MapPin, Radio, LogOut, BarChart3, HeartPulse, ScrollText, FileClock } from "lucide-react";
import { useAuth } from "../../auth/AuthContext";
import { LogoMark } from "../brand/Logo";

const navItemClass = ({ isActive }: { isActive: boolean }) =>
  [
    "flex items-center gap-3 px-3 py-2 rounded transition-colors font-body-base",
    isActive
      ? "text-primary font-bold bg-surface-container-high border-r-4 border-primary"
      : "text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface",
  ].join(" ");

// General Settings/Reports are still omitted — no export-job system or
// settings screen beyond the one emission-factor form exists yet.
// Analytics was added in Slice 2 (fleet-wide, operator-only nav item
// here; the site-scoped equivalent is reached from Site Detail, not this
// sidebar). Fleet Health and Audit Log were added in Slice 3 — both
// operator-only, kept off MobileNav (below) to avoid crowding the bottom
// bar past 4 items; reachable on mobile via links from Fleet/Analytics.
//
// Responsive per DESIGN.md's documented breakpoint (mobile < 768px: "side
// nav collapses to a bottom bar") — the mobile screens in design/ weren't
// separate pages to build, they're the reference for exactly this
// collapse behavior. Below md, this component renders nothing; MobileNav
// (same file) renders the bottom bar instead, used together in AppLayout.
export function Sidebar() {
  const { session, logout } = useAuth();
  const isOperator = session?.role === "operator";

  return (
    <aside className="hidden md:flex fixed left-0 top-0 h-screen w-[240px] bg-surface-container-low border-r border-outline-variant flex-col py-grid-margin z-50">
      <Link to="/" className="px-6 mb-10 flex items-start gap-2 hover:opacity-90 transition-opacity" aria-label="Back to home">
        <LogoMark size={20} />
        <div>
          <h1 className="font-headline-md text-headline-md font-bold text-on-surface leading-tight">
            Clean Energy Analytics
          </h1>
          <p className="font-label-caps text-label-caps text-primary uppercase tracking-widest mt-1">Fleet Management</p>
        </div>
      </Link>
      <nav className="flex-1 px-3 space-y-1">
        {isOperator && (
          <NavLink to="/app" className={navItemClass} end>
            <Grid2x2 size={20} />
            <span>Fleet</span>
          </NavLink>
        )}
        <NavLink to="/app/sites" className={navItemClass}>
          <MapPin size={20} />
          <span>Sites</span>
        </NavLink>
        <NavLink to="/app/devices" className={navItemClass}>
          <Radio size={20} />
          <span>Devices</span>
        </NavLink>
        <NavLink to="/app/ingestion-log" className={navItemClass}>
          <FileClock size={20} />
          <span>Ingestion Log</span>
        </NavLink>
        {isOperator && (
          <NavLink to="/app/analytics" className={navItemClass}>
            <BarChart3 size={20} />
            <span>Analytics</span>
          </NavLink>
        )}
        {isOperator && (
          <NavLink to="/app/fleet-health" className={navItemClass}>
            <HeartPulse size={20} />
            <span>Fleet Health</span>
          </NavLink>
        )}
        {isOperator && (
          <NavLink to="/app/audit" className={navItemClass}>
            <ScrollText size={20} />
            <span>Audit Log</span>
          </NavLink>
        )}
      </nav>
      <div className="px-3 pt-6 border-t border-outline-variant">
        <button
          onClick={logout}
          className="w-full flex items-center gap-3 px-3 py-2 text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors rounded font-body-base"
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
// bottom nav pattern, adapted to this app's actual 3 sections (Fleet/
// Sites/Devices) rather than that mockup's own Home/Alerts/Profile, since
// this app has no alerts feed or profile page to link to.
export function MobileNav() {
  const { session } = useAuth();
  const isOperator = session?.role === "operator";

  return (
    <nav className="md:hidden fixed bottom-0 left-0 w-full z-50 flex justify-around items-center h-16 px-density-base bg-surface-container-low border-t border-outline-variant">
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
        <NavLink to="/app/analytics" className={mobileNavItemClass}>
          <BarChart3 size={20} />
          <span>Analytics</span>
        </NavLink>
      )}
    </nav>
  );
}
