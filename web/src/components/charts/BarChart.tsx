import { useState } from "react";

// Hand-rolled SVG bar chart, same approach as LineChart.tsx (no charting
// library for one simple chart type) — thin rounded-top bars, a 2px
// surface gap between them, and a hover tooltip per the dataviz skill's
// "ship interactivity by default" rule for bar/dot/cell marks.
export function BarChart({
  points,
  color = "#2f8fe0",
  height = 220,
  valueFormatter = (v: number) => v.toFixed(0),
}: {
  points: { label: string; value: number }[];
  color?: string;
  height?: number;
  valueFormatter?: (value: number) => string;
}) {
  const [hovered, setHovered] = useState<number | null>(null);

  if (points.length === 0) {
    return (
      <div className="h-full flex items-center justify-center text-on-surface-variant font-body-base text-sm">
        Not enough data yet
      </div>
    );
  }

  const width = 1000;
  const yAxisWidth = 70; // reserved for the 0/25%/50%/75%/100% value labels
  const plotWidth = width - yAxisWidth;
  const gap = 6;
  const barWidth = plotWidth / points.length - gap;
  const maxValue = Math.max(...points.map((p) => p.value), 1);
  const plotHeight = height - 24; // reserve room for x-axis labels

  return (
    <div className="relative h-full w-full">
      <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" className="w-full h-full overflow-visible">
        {[0, 0.25, 0.5, 0.75, 1].map((f) => (
          <g key={f}>
            {f > 0 && (
              <line
                x1={yAxisWidth}
                x2={width}
                y1={plotHeight * (1 - f)}
                y2={plotHeight * (1 - f)}
                stroke="#404944"
                strokeDasharray="4 4"
                strokeWidth={1}
              />
            )}
            <text
              x={yAxisWidth - 10}
              y={plotHeight * (1 - f) + 5}
              textAnchor="end"
              fontSize={15}
              fill="#64748b"
              className="font-data-mono-sm"
            >
              {valueFormatter(maxValue * f)}
            </text>
          </g>
        ))}
        {points.map((p, i) => {
          const barHeight = Math.max((p.value / maxValue) * plotHeight, 2);
          const x = yAxisWidth + i * (barWidth + gap) + gap / 2;
          const y = plotHeight - barHeight;
          const isHovered = hovered === i;
          return (
            <g key={i}>
              <rect
                x={x}
                y={y}
                width={barWidth}
                height={barHeight}
                rx={4}
                fill={color}
                opacity={isHovered ? 1 : 0.85}
                onMouseEnter={() => setHovered(i)}
                onMouseLeave={() => setHovered(null)}
                style={{ cursor: "pointer" }}
              />
              <text
                x={x + barWidth / 2}
                y={height - 4}
                textAnchor="middle"
                fontSize={16}
                fill="#64748b"
                className="font-data-mono-sm"
              >
                {p.label}
              </text>
            </g>
          );
        })}
      </svg>
      {hovered !== null && (
        <div
          className="absolute -top-2 -translate-y-full glass-card rounded-lg px-3 py-1.5 text-[12px] text-on-surface pointer-events-none whitespace-nowrap"
          style={{
            left: `${((yAxisWidth + (hovered + 0.5) * (plotWidth / points.length)) / width) * 100}%`,
            transform: "translate(-50%, -100%)",
          }}
        >
          <span className="font-semibold">{points[hovered].label}</span>: {valueFormatter(points[hovered].value)}
        </div>
      )}
    </div>
  );
}
