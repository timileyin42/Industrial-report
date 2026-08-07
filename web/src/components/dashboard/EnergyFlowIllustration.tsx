interface FlowNode {
  label: string;
  value: string;
  available: boolean;
}

interface EnergyFlowIllustrationProps {
  solar: FlowNode;
  battery: FlowNode;
  grid: FlowNode;
  consumption: FlowNode;
  animated?: boolean;
  height?: number;
}

// A simplified isometric-style house + solar-panel illustration with
// animated flow particles along 3 paths (solar->house->battery,
// ->grid, ->consumption). Used two ways: on the real dashboard with
// real data (unavailable flows — this platform has no battery/grid
// telemetry — render as a muted "Not tracked" node rather than an
// invented number), and on the marketing hero as a fully-animated
// illustrative example. Same component, same visual language, so the
// product and the marketing site don't look like they came from two
// different design systems.
export function EnergyFlowIllustration({
  solar,
  battery,
  grid,
  consumption,
  animated = true,
  height = 280,
}: EnergyFlowIllustrationProps) {
  const flowPath = (d: string, color: string, available: boolean, delay: number) => (
    <>
      <path d={d} fill="none" stroke={available ? color : "#c7d7e6"} strokeWidth={2} strokeOpacity={available ? 0.35 : 0.5} />
      {available && animated && (
        <circle r={3.5} fill={color}>
          <animateMotion dur="3.2s" repeatCount="indefinite" path={d} begin={`${delay}s`} />
        </circle>
      )}
    </>
  );

  const node = (x: number, y: number, n: FlowNode, color: string, icon: string) => (
    <g transform={`translate(${x}, ${y})`}>
      <circle r={26} fill={n.available ? color : "#eef4fb"} fillOpacity={n.available ? 0.15 : 1} stroke={n.available ? color : "#c7d7e6"} strokeWidth={2} />
      <text textAnchor="middle" y={6} fontSize={20}>{icon}</text>
      <text textAnchor="middle" y={44} fontSize={11} fontWeight={600} fill="#1e2a3a">{n.label}</text>
      <text textAnchor="middle" y={60} fontSize={13} fontWeight={700} fill={n.available ? color : "#94a3b8"}>
        {n.available ? n.value : "Not tracked"}
      </text>
    </g>
  );

  return (
    <svg viewBox="0 0 400 260" width="100%" height={height} style={{ overflow: "visible" }}>
      {/* House silhouette, simple isometric-ish roof with panel grid lines */}
      <g transform="translate(140, 60)">
        <polygon points="0,50 60,10 120,50 120,120 0,120" fill="#dceefc" stroke="#2f8fe0" strokeWidth={1.5} />
        <polygon points="0,50 60,10 120,50 60,80" fill="#bfe0f7" stroke="#2f8fe0" strokeWidth={1.5} />
        {[0, 1, 2, 3].map((i) => (
          <line key={i} x1={15 + i * 22} y1={50 - i * 2} x2={45 + i * 22} y2={35 - i * 2} stroke="#2f8fe0" strokeWidth={1} opacity={0.5} />
        ))}
      </g>

      {flowPath("M60,60 C110,60 150,90 200,110", "#f2a93b", solar.available, 0)}
      {flowPath("M200,110 C230,140 260,150 320,60", "#2f8fe0", battery.available, 0.8)}
      {flowPath("M200,110 C230,160 260,190 320,200", "#1a9c6b", grid.available, 1.6)}
      {flowPath("M200,150 C200,180 200,200 200,220", "#2f8fe0", consumption.available, 2.4)}

      {node(60, 60, solar, "#f2a93b", "☀️")}
      {node(320, 60, battery, "#2f8fe0", "🔋")}
      {node(320, 200, grid, "#1a9c6b", "🔌")}
      {node(200, 220, consumption, "#2f8fe0", "🏠")}
    </svg>
  );
}
