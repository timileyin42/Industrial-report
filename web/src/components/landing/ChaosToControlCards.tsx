import { useState } from "react";
import { motion } from "framer-motion";
import { Layers, ShieldCheck, FileCheck2, type LucideIcon } from "lucide-react";

interface CardData {
  icon: LucideIcon;
  from: string;
  to: string;
  body: string;
}

const CARDS: CardData[] = [
  {
    icon: Layers,
    from: "Fragmented Data",
    to: "Unified Fleet View",
    body: "Aggregate inverter, meter, and weather station data across multiple OEMs into one normalized schema.",
  },
  {
    icon: ShieldCheck,
    from: "Opaque Status",
    to: "Verified Reporting",
    body: "Real-time device-level status monitoring with data provenance for full auditability.",
  },
  {
    icon: FileCheck2,
    from: "Unstructured Data",
    to: "Audit-Ready ESG",
    body: "Generate structured reports for carbon avoidance and energy yield tailored for regulatory compliance.",
  },
];

// "Music deck" hover: the hovered card lifts (scale + glow); its
// neighbors nudge away horizontally, like a carousel making room for the
// focused item. Both driven off one piece of state (hoveredIndex) so the
// whole row reacts together rather than each card animating in isolation.
const NUDGE_PX = 18;

export function ChaosToControlCards() {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
      {CARDS.map((item, index) => {
        const isHovered = hoveredIndex === index;
        const nudgeDirection = hoveredIndex === null ? 0 : index < hoveredIndex ? -1 : index > hoveredIndex ? 1 : 0;

        return (
          <motion.div
            key={item.from}
            onMouseEnter={() => setHoveredIndex(index)}
            onMouseLeave={() => setHoveredIndex(null)}
            animate={{
              scale: isHovered ? 1.05 : 1,
              x: nudgeDirection * NUDGE_PX,
              boxShadow: isHovered
                ? "0 20px 45px -12px rgb(47 143 224 / 0.35), 0 0 0 1px rgb(47 143 224 / 0.25)"
                : "0 1px 2px rgb(30 58 95 / 0.06)",
            }}
            transition={{ type: "spring", stiffness: 300, damping: 24, mass: 0.6 }}
            style={{ transformOrigin: "center center", zIndex: isHovered ? 10 : 0 }}
            className="relative bg-surface-container-low p-6 rounded-lg border border-outline-variant flex flex-col gap-4 cursor-default"
          >
            <div className="w-10 h-10 rounded bg-primary-container/20 flex items-center justify-center border border-primary-container">
              <item.icon size={20} className="text-primary" />
            </div>
            <div className="flex justify-between items-center pb-4 border-b border-outline-variant">
              <span className="text-error/80 font-semibold line-through">{item.from}</span>
              <span className="text-outline">→</span>
              <span className="text-primary font-bold">{item.to}</span>
            </div>
            <p className="text-on-surface-variant text-sm mt-2">{item.body}</p>
          </motion.div>
        );
      })}
    </div>
  );
}
