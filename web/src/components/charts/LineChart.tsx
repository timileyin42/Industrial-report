// Custom hand-rolled SVG line chart, matching the mockups' own approach
// (zgnis_style_guide_components: sharp joins, faint dashed gridlines, no
// area gradients beyond a 5% max fill) rather than pulling in a charting
// library for one simple line.
//
// points2/color2/label2 add an optional second series (e.g. PV output
// alongside AC output) sharing this same chart's single y-axis — both
// series are the same unit (kW), so one shared scale is correct, not a
// dual-axis chart. label/label2 are only rendered (as a small legend)
// when a second series is actually present; a single series needs none,
// since the chart's own heading already names it.
export function LineChart({
  points,
  color = "#2f8fe0",
  height = 220,
  label,
  points2,
  color2 = "#f2a93b",
  label2,
  xAxisLabels,
}: {
  points: { x: number; y: number }[];
  color?: string;
  height?: number;
  label?: string;
  points2?: { x: number; y: number }[];
  color2?: string;
  label2?: string;
  // Optional time-of-day ticks below the chart (e.g. an intraday power
  // curve) — frac is the tick's horizontal position (0 = first point,
  // 1 = last), matching the same index-based x-scale the chart itself
  // uses. Omitted entirely for charts with no natural time-of-day axis
  // (daily/monthly energy trends, which are already index-per-period).
  xAxisLabels?: { frac: number; label: string }[];
}) {
  if (points.length < 2) {
    return (
      <div className="h-full flex items-center justify-center text-on-surface-variant font-body-base text-sm">
        Not enough data yet
      </div>
    );
  }

  const width = 1000;
  const hasSecond = !!points2 && points2.length >= 2;
  const allY = hasSecond ? [...points.map((p) => p.y), ...points2!.map((p) => p.y)] : points.map((p) => p.y);
  // Only fall back to a ceiling of 1 when every real value is exactly 0
  // (avoids a divide-by-zero range below) — flooring the ceiling at 1
  // unconditionally used to squash genuinely small-but-real values (e.g.
  // 0.03 kW just after sunrise) down into the bottom few pixels, reading
  // as "flat" when the data was actually moving.
  const realMaxY = Math.max(...allY);
  const maxY = realMaxY > 0 ? realMaxY : 1;
  const minY = Math.min(...allY, 0);
  const range = maxY - minY || 1;

  const scale = (pts: { x: number; y: number }[]) =>
    pts.map((p, i) => ({
      x: (i / (pts.length - 1)) * width,
      y: height - ((p.y - minY) / range) * height,
    }));

  const toPath = (scaled: { x: number; y: number }[]) =>
    scaled.map((p, i) => `${i === 0 ? "M" : "L"}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(" ");

  const scaled = scale(points);
  const path = toPath(scaled);
  const areaPath = `${path} L${width},${height} L0,${height} Z`;
  const path2 = hasSecond ? toPath(scale(points2!)) : null;

  return (
    <div className="w-full h-full flex flex-col">
      {hasSecond && (label || label2) && (
        <div className="flex items-center gap-4 mb-2 text-[11px] text-on-surface-variant">
          {label && (
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ background: color }} />
              {label}
            </span>
          )}
          {label2 && (
            <span className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full flex-shrink-0" style={{ background: color2 }} />
              {label2}
            </span>
          )}
        </div>
      )}
      <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" className="w-full flex-1">
        <defs>
          <linearGradient id="chart-fill" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.05" />
            <stop offset="100%" stopColor={color} stopOpacity="0" />
          </linearGradient>
        </defs>
        {[0.25, 0.5, 0.75].map((f) => (
          <line
            key={f}
            x1={0}
            x2={width}
            y1={height * f}
            y2={height * f}
            stroke="#404944"
            strokeDasharray="4 4"
            strokeWidth={1}
          />
        ))}
        <path d={areaPath} fill="url(#chart-fill)" />
        <path d={path} fill="none" stroke={color} strokeWidth={2} strokeLinejoin="miter" vectorEffect="non-scaling-stroke" />
        {path2 && (
          <path d={path2} fill="none" stroke={color2} strokeWidth={2} strokeLinejoin="miter" vectorEffect="non-scaling-stroke" strokeDasharray="5 3" />
        )}
      </svg>
      {xAxisLabels && xAxisLabels.length > 0 && (
        <div className="relative h-4 mt-1 text-[10px] text-on-surface-variant">
          {xAxisLabels.map((t, i) => (
            <span
              key={i}
              className="absolute -translate-x-1/2 first:translate-x-0 last:-translate-x-full"
              style={{ left: `${t.frac * 100}%` }}
            >
              {t.label}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
