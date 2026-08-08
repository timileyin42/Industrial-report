import type { ReactNode } from "react";
import { Factory } from "lucide-react";

// Reuses the three dedicated empty-state screens' actual layout
// (fleet_dashboard_empty_state, device_registry_empty_state,
// site_telemetry_pending_data) — icon + heading + body + primary action.
//
// compact drops the same content into a much smaller footprint (a chart
// slot inside a fixed h-[XXXpx] card, not a whole page) — same tokens,
// no new colors, just sized to actually fit instead of overflowing/
// overlapping the caption text below it the way the full-size version
// does once squeezed under ~250px tall.
export function EmptyState({
  icon,
  title,
  body,
  action,
  compact,
}: {
  icon?: ReactNode;
  title: string;
  body: string;
  action?: ReactNode;
  compact?: boolean;
}) {
  if (compact) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center text-center px-4 gap-1.5 overflow-hidden">
        <div className="mb-1 w-10 h-10 glass-card flex items-center justify-center rounded-xl text-on-surface-variant flex-shrink-0">
          {icon ?? <Factory size={18} />}
        </div>
        <h3 className="font-headline-md text-[14px] font-bold text-on-background leading-tight">{title}</h3>
        <p className="font-body-base text-[11px] text-on-surface-variant max-w-xs leading-snug">{body}</p>
        {action}
      </div>
    );
  }

  return (
    <div className="flex-1 flex items-center justify-center p-grid-margin">
      <div className="w-full max-w-lg text-center flex flex-col items-center">
        <div className="mb-8 w-24 h-24 glass-card flex items-center justify-center rounded-2xl text-on-surface-variant">
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
