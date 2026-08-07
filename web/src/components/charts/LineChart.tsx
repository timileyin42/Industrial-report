// Custom hand-rolled SVG line chart, matching the mockups' own approach
// (zgnis_style_guide_components: sharp joins, faint dashed gridlines, no
// area gradients beyond a 5% max fill) rather than pulling in a charting
// library for one simple line.
export function LineChart({
  points,
  color = "#95d3ba",
  height = 220,
}: {
  points: { x: number; y: number }[];
  color?: string;
  height?: number;
}) {
  if (points.length < 2) {
    return (
      <div className="h-full flex items-center justify-center text-on-surface-variant font-body-base text-sm">
        Not enough data yet
      </div>
    );
  }

  const width = 1000;
  const maxY = Math.max(...points.map((p) => p.y), 1);
  const minY = Math.min(...points.map((p) => p.y), 0);
  const range = maxY - minY || 1;

  const scaled = points.map((p, i) => ({
    x: (i / (points.length - 1)) * width,
    y: height - ((p.y - minY) / range) * height,
  }));

  const path = scaled.map((p, i) => `${i === 0 ? "M" : "L"}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(" ");
  const areaPath = `${path} L${width},${height} L0,${height} Z`;

  return (
    <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" className="w-full h-full">
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
    </svg>
  );
}
