import type { ReactNode } from "react";
import { TrendingUp, TrendingDown } from "lucide-react";

type Tone = "primary" | "secondary" | "error" | "neutral";

const toneText: Record<Tone, string> = {
  primary: "text-primary",
  secondary: "text-secondary",
  error: "text-error",
  neutral: "text-on-surface",
};

// Glass card, soft rounded geometry, big confident numeral — per the
// light/glass redesign. trendPct is optional and renders as muted
// green/red text (not a bold status color) per the brief; omit it when
// there's no real comparison-to-last-period value to show rather than
// inventing one.
export function KpiCard({
  label,
  value,
  unit,
  icon,
  tone = "neutral",
  trendPct,
  footer,
}: {
  label: string;
  value: string | number;
  unit?: string;
  icon?: ReactNode;
  tone?: Tone;
  trendPct?: number | null;
  footer?: ReactNode;
}) {
  return (
    <div className="glass-card rounded-xl p-5 flex flex-col gap-2 hover:shadow-soft transition-shadow">
      <div className="flex justify-between items-start">
        <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">{label}</span>
        {icon && <span className="text-primary">{icon}</span>}
      </div>
      <div className={`flex items-baseline gap-2 mt-2 font-data-display-lg text-data-display-lg ${toneText[tone]}`}>
        <span>{value}</span>
        {unit && <span className="text-headline-md font-headline-md text-on-surface-variant/60">{unit}</span>}
      </div>
      {trendPct != null && (
        <div className={`flex items-center gap-1 text-[12px] font-body-base ${trendPct >= 0 ? "text-success" : "text-error"}`}>
          {trendPct >= 0 ? <TrendingUp size={13} /> : <TrendingDown size={13} />}
          <span>
            {trendPct >= 0 ? "+" : ""}
            {trendPct.toFixed(1)}% vs last period
          </span>
        </div>
      )}
      {footer && <div className="mt-2 pt-2 border-t border-outline-variant/50">{footer}</div>}
    </div>
  );
}
