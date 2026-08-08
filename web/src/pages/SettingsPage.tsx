import { Link } from "react-router-dom";
import { UserPlus, Leaf, ChevronRight } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";

const SETTINGS_LINKS = [
  {
    to: "/app/users/invite",
    icon: UserPlus,
    title: "Invite User",
    body: "Give someone operator or restricted access to the dashboard.",
  },
  {
    to: "/app/settings/emissions",
    icon: Leaf,
    title: "Emission Factor",
    body: "Set the grid CO2 emission factor CO2-offset reporting resolves per-site country against.",
  },
];

// Settings landing page — a real hub rather than one nav item jumping
// straight to the single Emission Factor form it used to point at
// directly. Invite User lives here now instead of as its own top-level
// dashboard quick-link.
export function SettingsPage() {
  return (
    <>
      <TopNav title="Settings" />
      <div className="flex-1 p-grid-margin max-w-2xl space-y-3">
        {SETTINGS_LINKS.map(({ to, icon: Icon, title, body }) => (
          <Link
            key={to}
            to={to}
            className="glass-card rounded-2xl p-5 flex items-center gap-4 hover:bg-white/50 transition-colors"
          >
            <div className="w-11 h-11 rounded-xl bg-primary-container flex items-center justify-center text-primary flex-shrink-0">
              <Icon size={20} />
            </div>
            <div className="flex-1">
              <h3 className="font-headline-md text-[16px] font-bold text-on-surface">{title}</h3>
              <p className="text-on-surface-variant text-[13px] mt-0.5">{body}</p>
            </div>
            <ChevronRight size={18} className="text-on-surface-variant flex-shrink-0" />
          </Link>
        ))}
      </div>
    </>
  );
}
