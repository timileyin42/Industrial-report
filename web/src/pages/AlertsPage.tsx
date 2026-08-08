import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { AlertTriangle, AlertCircle, Info, PowerOff, Gauge, TrendingDown, ShieldOff } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { listFleetAlerts } from "../api/alerts";
import type { Alert } from "../api/types";

const TYPE_ICON: Record<Alert["type"], typeof PowerOff> = {
  device_offline: PowerOff,
  device_fault: AlertTriangle,
  device_revoked: ShieldOff,
  low_coverage: Gauge,
  low_generation: TrendingDown,
};

const SEVERITY_STYLES: Record<Alert["severity"], string> = {
  critical: "text-error",
  warning: "text-secondary",
  info: "text-on-surface-variant",
};

// Every alert here is a real, currently-true (or recently real)
// condition derived from stored data — offline devices, a fault-status
// reading, a revocation, a trailing-baseline anomaly day — never a
// fabricated event. There's no persisted alerts table in this platform
// (see internal/registry/alerts.go); this is computed fresh on each load,
// which is also why it reflects current state rather than a full history
// you could page back through indefinitely.
export function AlertsPage() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ["fleet-alerts"],
    queryFn: () => listFleetAlerts(100),
  });

  if (isError) {
    return <ErrorState onRetry={() => refetch()} />;
  }

  return (
    <>
      <TopNav title="Alerts" />
      <div className="flex-1 p-grid-margin space-y-3">
        {isLoading || !data ? (
          <div className="h-64 glass-card rounded-xl animate-pulse" />
        ) : data.length === 0 ? (
          <EmptyState title="All clear" body="No devices offline, faulted, revoked, or showing an unusual generation drop right now." />
        ) : (
          data.map((alert, i) => {
            const Icon = TYPE_ICON[alert.type];
            return (
              <Link
                key={i}
                to={`/app/sites/${alert.site_id}`}
                className="glass-card rounded-xl px-5 py-4 flex items-start gap-4 hover:bg-white/50 transition-colors"
              >
                <Icon size={20} className={`${SEVERITY_STYLES[alert.severity]} mt-0.5 flex-shrink-0`} />
                <div className="flex-1">
                  <p className="text-on-surface font-body-base text-body-base">{alert.message}</p>
                  <p className="text-on-surface-variant text-[12px] mt-0.5">
                    {alert.site_name ?? alert.site_id}
                    {alert.device_id ? ` · ${alert.device_id}` : ""} · {new Date(alert.occurred_at).toLocaleString()}
                  </p>
                </div>
                {alert.severity === "critical" && <AlertCircle size={16} className="text-error flex-shrink-0" />}
                {alert.severity === "info" && <Info size={16} className="text-on-surface-variant flex-shrink-0" />}
              </Link>
            );
          })
        )}
      </div>
    </>
  );
}
