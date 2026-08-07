import type { ReactNode } from "react";

type Tone = "primary" | "secondary" | "error" | "neutral";

const toneText: Record<Tone, string> = {
  primary: "text-primary",
  secondary: "text-secondary",
  error: "text-error",
  neutral: "text-on-surface",
};

// KPI Card: top-aligned label-caps header, data-display-lg value in
// JetBrains Mono (never Public Sans) — see zgnis_style_guide_components.
export function KpiCard({
  label,
  value,
  unit,
  icon,
  tone = "neutral",
  footer,
}: {
  label: string;
  value: string | number;
  unit?: string;
  icon?: ReactNode;
  tone?: Tone;
  footer?: ReactNode;
}) {
  return (
    <div className="bg-surface-container border border-outline-variant p-5 flex flex-col gap-2 hover:border-primary-container transition-all">
      <div className="flex justify-between items-start">
        <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">{label}</span>
        {icon}
      </div>
      <div className={`flex items-baseline gap-2 mt-2 font-data-display-lg text-data-display-lg ${toneText[tone]}`}>
        <span>{value}</span>
        {unit && <span className="text-headline-md font-headline-md text-on-surface-variant/60">{unit}</span>}
      </div>
      {footer && <div className="mt-2 pt-2 border-t border-outline-variant/30">{footer}</div>}
    </div>
  );
}
