import { useEffect, useRef, useState } from "react";
import { ChevronDown, Download, Loader2 } from "lucide-react";

export interface ExportFormatOption {
  label: string;
  onExport: () => Promise<void>;
}

// One button, format chosen from a menu on click — replaces having a
// separate always-visible button per format (CSV, PDF, ...) cluttering
// the page. Same dropdown pattern as components/layout/NotificationBell.tsx.
export function ExportMenuButton({ label, options }: { label: string; options: ExportFormatOption[] }) {
  const [open, setOpen] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [failed, setFailed] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  async function handleSelect(onExport: () => Promise<void>) {
    setOpen(false);
    setIsExporting(true);
    setFailed(false);
    try {
      await onExport();
    } catch {
      setFailed(true);
    } finally {
      setIsExporting(false);
    }
  }

  return (
    <div className="relative" ref={containerRef}>
      <button
        onClick={() => setOpen((v) => !v)}
        disabled={isExporting}
        className="flex items-center gap-2 glass-card rounded-full text-on-surface hover:text-primary font-body-base text-body-base px-4 py-2 transition-colors disabled:opacity-60"
      >
        {isExporting ? <Loader2 size={16} className="animate-spin" /> : <Download size={16} />}
        <span>{failed ? "Export failed — retry" : label}</span>
        <ChevronDown size={14} />
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-2 min-w-[160px] max-w-[90vw] overlay-panel rounded-xl overflow-hidden z-50">
          {options.map((opt) => (
            <button
              key={opt.label}
              onClick={() => handleSelect(opt.onExport)}
              className="w-full text-left px-4 py-2.5 text-[13px] text-on-surface hover:bg-surface-container-highest transition-colors"
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
