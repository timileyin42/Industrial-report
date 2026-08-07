import { useState } from "react";
import { Download, Loader2 } from "lucide-react";

// Shared by every CSV export trigger (site telemetry/summary, fleet
// summary) — each just passes its own downloadX() call from api/exports.ts.
export function ExportButton({ label, onExport }: { label: string; onExport: () => Promise<void> }) {
  const [isExporting, setIsExporting] = useState(false);
  const [failed, setFailed] = useState(false);

  async function handleClick() {
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
    <button
      onClick={handleClick}
      disabled={isExporting}
      className="flex items-center gap-2 glass-card rounded-full text-on-surface hover:text-primary font-body-base text-body-base px-4 py-2 transition-colors disabled:opacity-60"
    >
      {isExporting ? <Loader2 size={16} className="animate-spin" /> : <Download size={16} />}
      <span>{failed ? "Export failed — retry" : label}</span>
    </button>
  );
}
