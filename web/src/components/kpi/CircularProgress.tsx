// Circular progress ring — replaces the old system's plain bars for
// capacity/health metrics per the light/glass redesign brief. Pure SVG,
// no chart dependency.
export function CircularProgress({
  percent,
  size = 96,
  strokeWidth = 10,
  color = "#2f8fe0",
  trackColor = "#e3ecf5",
  label,
  value,
}: {
  percent: number;
  size?: number;
  strokeWidth?: number;
  color?: string;
  trackColor?: string;
  label?: string;
  value?: string;
}) {
  const clamped = Math.max(0, Math.min(100, percent));
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (clamped / 100) * circumference;

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="relative" style={{ width: size, height: size }}>
        <svg width={size} height={size} className="-rotate-90">
          <circle cx={size / 2} cy={size / 2} r={radius} fill="none" stroke={trackColor} strokeWidth={strokeWidth} />
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke={color}
            strokeWidth={strokeWidth}
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            strokeLinecap="round"
            style={{ transition: "stroke-dashoffset 0.6s ease" }}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="font-data-display-lg text-[20px] text-on-surface">{value ?? `${Math.round(clamped)}%`}</span>
        </div>
      </div>
      {label && <span className="font-label-caps text-label-caps text-on-surface-variant uppercase text-center">{label}</span>}
    </div>
  );
}
