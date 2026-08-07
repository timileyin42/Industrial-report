import { useEffect, useRef, useState, type ReactNode } from "react";
import { ChevronRight } from "lucide-react";

export interface Column<T> {
  header: string;
  align?: "left" | "right";
  // Structural enforcement of DESIGN.md's "every numeric value and every
  // device/serial ID uses JetBrains Mono" rule — a column can't
  // accidentally render in the UI font.
  isMono?: boolean;
  render: (row: T) => ReactNode;
}

// Header: label-caps on a surface-container-highest-toned background.
// Body: bottom-border rows, no full grid lines — per DESIGN.md and
// zgnis_style_guide_components' Data Table Row pattern.
//
// Every table here is wide enough to overflow at mobile widths (5+
// columns is common — Fleet Health, Audit Log, Ingestion Log). The
// container already scrolls horizontally (overflow-x-auto, functionally
// fine), but nothing hinted that more columns existed off-screen — a
// real usability gap caught on mobile testing, not a card-based rebuild
// (that's a much bigger rework across every table usage; this fixes the
// actual problem — discoverability — at a fraction of the cost). The
// hint only renders when there's real overflow left to scroll to, and
// disappears once the user's scrolled to the end.
export function DataTable<T>({
  columns,
  rows,
  rowKey,
}: {
  columns: Column<T>[];
  rows: T[];
  rowKey: (row: T) => string;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [canScrollMore, setCanScrollMore] = useState(false);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;

    function update() {
      setCanScrollMore(el!.scrollLeft + el!.clientWidth < el!.scrollWidth - 1);
    }
    update();

    el.addEventListener("scroll", update, { passive: true });
    const resizeObserver = new ResizeObserver(update);
    resizeObserver.observe(el);
    return () => {
      el.removeEventListener("scroll", update);
      resizeObserver.disconnect();
    };
  }, [columns, rows]);

  return (
    <div className="relative bg-surface-container-low border border-outline-variant overflow-hidden">
      <div ref={scrollRef} className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="bg-surface-container-highest/50 border-b border-outline-variant">
              {columns.map((col) => (
                <th
                  key={col.header}
                  className={`px-6 py-4 font-label-caps text-label-caps text-on-surface-variant uppercase tracking-wider ${
                    col.align === "right" ? "text-right" : ""
                  }`}
                >
                  {col.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-outline-variant/30">
            {rows.map((row) => (
              <tr key={rowKey(row)} className="hover:bg-surface-container-highest/30 transition-colors">
                {columns.map((col) => (
                  <td
                    key={col.header}
                    className={`px-6 py-4 ${col.isMono ? "font-data-mono-sm text-data-mono-sm" : "font-body-base"} ${
                      col.align === "right" ? "text-right" : ""
                    }`}
                  >
                    {col.render(row)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {canScrollMore && (
        <div className="md:hidden pointer-events-none absolute top-0 right-0 h-full w-12 bg-gradient-to-l from-background/80 via-background/30 to-transparent flex items-center justify-end">
          <ChevronRight size={18} className="text-primary animate-pulse mr-1" />
        </div>
      )}
    </div>
  );
}
