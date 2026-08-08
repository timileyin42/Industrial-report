import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Bell, AlertTriangle, PowerOff, ShieldOff, Info } from "lucide-react";
import { listFleetAlerts } from "../../api/alerts";
import { useAuth } from "../../auth/AuthContext";

const TYPE_ICON = {
  device_fault: AlertTriangle,
  device_offline: PowerOff,
  device_revoked: ShieldOff,
  low_coverage: Info,
  low_generation: Info,
} as const;

// Real badge count AND real preview list from the same Alerts feed
// /app/alerts shows in full (internal/registry/alerts.go) — never a
// decorative fixed number or fabricated entries.
export function NotificationBell() {
  const { session } = useAuth();
  const isOperator = session?.role === "operator";
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const { data } = useQuery({
    queryKey: ["nav-alerts"],
    queryFn: () => listFleetAlerts(100),
    enabled: isOperator,
    refetchInterval: 60_000,
  });

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  if (!isOperator) return null;

  const count = data?.length ?? 0;
  const preview = (data ?? []).slice(0, 5);

  return (
    <div className="relative" ref={containerRef}>
      <button
        onClick={() => setOpen((v) => !v)}
        className="relative glass-card rounded-full p-2.5 text-on-surface-variant hover:text-primary transition-colors"
        title="Alerts"
      >
        <Bell size={18} />
        {count > 0 && (
          <span className="absolute -top-1 -right-1 min-w-[18px] h-[18px] px-1 rounded-full bg-error text-white text-[10px] font-bold flex items-center justify-center">
            {count > 9 ? "9+" : count}
          </span>
        )}
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-2 w-80 glass-card rounded-xl overflow-hidden z-50">
          <div className="px-4 py-3 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Alerts</span>
          </div>
          {preview.length === 0 ? (
            <p className="px-4 py-6 text-center text-[13px] text-on-surface-variant">All clear — no active alerts.</p>
          ) : (
            <div className="divide-y divide-outline-variant/40 max-h-80 overflow-y-auto">
              {preview.map((alert, i) => {
                const Icon = TYPE_ICON[alert.type];
                return (
                  <Link
                    key={i}
                    to={`/app/sites/${alert.site_id}`}
                    onClick={() => setOpen(false)}
                    className="flex items-start gap-3 px-4 py-3 hover:bg-white/50 transition-colors"
                  >
                    <Icon size={16} className={`mt-0.5 flex-shrink-0 ${alert.severity === "critical" ? "text-error" : alert.severity === "warning" ? "text-secondary" : "text-on-surface-variant"}`} />
                    <div>
                      <p className="text-[13px] text-on-surface leading-tight">{alert.message}</p>
                      <p className="text-[11px] text-on-surface-variant mt-0.5">{alert.site_name ?? alert.site_id}</p>
                    </div>
                  </Link>
                );
              })}
            </div>
          )}
          <Link
            to="/app/alerts"
            onClick={() => setOpen(false)}
            className="block text-center px-4 py-3 text-[13px] font-semibold text-primary hover:underline border-t border-outline-variant/60"
          >
            View All Alerts
          </Link>
        </div>
      )}
    </div>
  );
}
