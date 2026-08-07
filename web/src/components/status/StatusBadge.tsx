export type Status = "online" | "degraded" | "offline" | "maintenance";

// Pill, solid dot + label, background at the semantic color's 10%
// opacity — per DESIGN.md and confirmed against
// zgnis_style_guide_components/code.html's "Status System" section
// (all 4 states). Strictly primary/secondary/error/on-surface-variant —
// never a 4th ad-hoc color, never repurposed for anything non-status.
const styles: Record<Status, { dot: string; text: string; bg: string; border: string; label: string }> = {
  online: { dot: "bg-primary", text: "text-primary", bg: "bg-primary/10", border: "border-primary/20", label: "ONLINE" },
  degraded: { dot: "bg-secondary", text: "text-secondary", bg: "bg-secondary/10", border: "border-secondary/20", label: "DEGRADED" },
  offline: { dot: "bg-error", text: "text-error", bg: "bg-error-container/20", border: "border-error-container/40", label: "OFFLINE" },
  maintenance: {
    dot: "bg-on-surface-variant",
    text: "text-on-surface-variant",
    bg: "bg-on-surface-variant/10",
    border: "border-on-surface-variant/20",
    label: "MAINTENANCE",
  },
};

export function StatusBadge({ status, label }: { status: Status; label?: string }) {
  const s = styles[status];
  return (
    <span className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full border ${s.bg} ${s.border}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${s.dot}`} />
      <span className={`font-label-caps text-label-caps ${s.text}`}>{label ?? s.label}</span>
    </span>
  );
}
