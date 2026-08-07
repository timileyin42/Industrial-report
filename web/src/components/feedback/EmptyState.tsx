import type { ReactNode } from "react";
import { Factory } from "lucide-react";

// Reuses the three dedicated empty-state screens' actual layout
// (fleet_dashboard_empty_state, device_registry_empty_state,
// site_telemetry_pending_data) — icon + heading + body + primary action.
export function EmptyState({
  icon,
  title,
  body,
  action,
}: {
  icon?: ReactNode;
  title: string;
  body: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex-1 flex items-center justify-center p-grid-margin">
      <div className="w-full max-w-lg text-center flex flex-col items-center">
        <div className="mb-8 w-24 h-24 bg-surface-container border border-outline-variant flex items-center justify-center rounded-xl text-outline">
          {icon ?? <Factory size={48} />}
        </div>
        <h2 className="font-headline-lg text-headline-lg text-on-background mb-3">{title}</h2>
        <p className="font-body-base text-body-base text-on-surface-variant max-w-sm mb-10 leading-relaxed">
          {body}
        </p>
        {action}
      </div>
    </div>
  );
}
