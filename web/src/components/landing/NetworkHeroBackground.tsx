import { useMemo } from "react";

interface Node {
  x: number;
  y: number;
  r: number;
  hub: boolean;
}

const WIDTH = 1200;
const HEIGHT = 700;
const NODE_COUNT = 46;
const MAX_LINK_DISTANCE = 170;

function generateNetwork() {
  const nodes: Node[] = Array.from({ length: NODE_COUNT }, () => ({
    x: Math.random() * WIDTH,
    y: Math.random() * HEIGHT,
    r: 1.5 + Math.random() * 2,
    hub: false,
  }));

  // A handful of larger, pulsing "hub" nodes — echoes the bar-chart mark
  // (a few taller bars standing out among smaller ones) rather than a
  // uniform dot grid.
  const hubIndexes = new Set<number>();
  while (hubIndexes.size < 6) {
    hubIndexes.add(Math.floor(Math.random() * NODE_COUNT));
  }
  hubIndexes.forEach((i) => {
    nodes[i].r = 4 + Math.random() * 2;
    nodes[i].hub = true;
  });

  // Sparse network, not a dense mesh: each node links to its nearest 1-2
  // neighbors within range, echoing the reference's grid/route lines
  // without pretending to be a literal geographic map (this platform
  // doesn't have "sites" placed on this backdrop — it's an abstract
  // network motif, not a claim about coverage).
  const edges: [Node, Node][] = [];
  nodes.forEach((node, i) => {
    const distances = nodes
      .map((other, j) => ({ j, d: Math.hypot(node.x - other.x, node.y - other.y) }))
      .filter(({ j, d }) => j !== i && d < MAX_LINK_DISTANCE)
      .sort((a, b) => a.d - b.d)
      .slice(0, 2);
    distances.forEach(({ j }) => {
      if (i < j) edges.push([node, nodes[j]]);
    });
  });

  return { nodes, edges };
}

// Abstract network/graph motif for the marketing hero — inspired by
// dashboard map-visualization references, reinterpreted in this
// platform's own palette (primary green / secondary amber) rather than
// lifted wholesale. Deliberately not a literal map: this product doesn't
// have a real global site map to show on a public marketing page, and
// implying one would be exactly the kind of fabricated-looking claim this
// pass is fixing elsewhere on the page.
export function NetworkHeroBackground() {
  const { nodes, edges } = useMemo(() => generateNetwork(), []);

  return (
    <div className="absolute inset-0 overflow-hidden pointer-events-none">
      <svg viewBox={`0 0 ${WIDTH} ${HEIGHT}`} preserveAspectRatio="xMidYMid slice" className="w-full h-full">
        <defs>
          <filter id="hero-glow" x="-100%" y="-100%" width="300%" height="300%">
            <feGaussianBlur stdDeviation="4" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        {edges.map(([a, b], i) => (
          <line
            key={i}
            x1={a.x}
            y1={a.y}
            x2={b.x}
            y2={b.y}
            stroke={a.hub || b.hub ? "#ffb95f" : "#95d3ba"}
            strokeOpacity={a.hub || b.hub ? 0.25 : 0.12}
            strokeWidth={1}
          />
        ))}

        {nodes.map((n, i) => (
          <circle
            key={i}
            cx={n.x}
            cy={n.y}
            r={n.r}
            fill={n.hub ? "#ffb95f" : "#95d3ba"}
            fillOpacity={n.hub ? 0.85 : 0.5}
            filter={n.hub ? "url(#hero-glow)" : undefined}
            className={n.hub ? "animate-pulse" : undefined}
            style={n.hub ? { animationDuration: `${2.5 + (i % 4) * 0.6}s` } : undefined}
          />
        ))}
      </svg>
      {/* Fades the network toward the background color under the text
          column (left) while keeping it visible further right, and
          keeps a dark floor so text always passes contrast regardless of
          where nodes happen to land. */}
      <div className="absolute inset-0 bg-gradient-to-r from-background via-background/70 to-background/20" />
      <div className="absolute inset-0 bg-gradient-to-t from-background via-transparent to-background/40" />
    </div>
  );
}
