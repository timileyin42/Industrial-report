// Multi-segment donut — unlike CircularProgress (a single-value ring),
// this renders every category as its own proportional arc so a 4-way
// breakdown (online/offline/fault/no-data) is actually visible in the
// ring itself, not just in the text legend beside it.
export interface DonutSegment {
  value: number;
  className: string; // Tailwind stroke-* utility, e.g. "stroke-success"
}

export function StatusDonut({
  segments,
  size = 120,
  strokeWidth = 12,
  centerValue,
  centerLabel,
}: {
  segments: DonutSegment[];
  size?: number;
  strokeWidth?: number;
  centerValue: string;
  centerLabel?: string;
}) {
  const total = segments.reduce((sum, s) => sum + s.value, 0);
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const nonZero = segments.filter((s) => s.value > 0);
  // A visible gap between adjacent segments; skipped entirely when there's
  // only one segment to draw, since there's nothing to separate it from.
  const gap = nonZero.length > 1 ? 3 : 0;

  let cumulative = 0;

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="relative" style={{ width: size, height: size }}>
        <svg width={size} height={size} className="-rotate-90">
          <circle cx={size / 2} cy={size / 2} r={radius} fill="none" className="stroke-outline-variant" strokeWidth={strokeWidth} />
          {total > 0 &&
            nonZero.map((seg, i) => {
              const length = (seg.value / total) * circumference;
              const visibleLength = Math.max(0, length - gap);
              const dashoffset = -cumulative;
              cumulative += length;
              return (
                <circle
                  key={i}
                  cx={size / 2}
                  cy={size / 2}
                  r={radius}
                  fill="none"
                  className={seg.className}
                  strokeWidth={strokeWidth}
                  strokeDasharray={`${visibleLength} ${circumference - visibleLength}`}
                  strokeDashoffset={dashoffset}
                  strokeLinecap="round"
                  style={{ transition: "stroke-dasharray 0.6s ease, stroke-dashoffset 0.6s ease" }}
                />
              );
            })}
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="font-data-display-lg text-[20px] text-on-surface">{centerValue}</span>
        </div>
      </div>
      {centerLabel && <span className="font-label-caps text-label-caps text-on-surface-variant uppercase text-center">{centerLabel}</span>}
    </div>
  );
}
