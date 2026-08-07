import { useEffect, useRef, useState } from "react";
import { Download, Loader2, Clock } from "lucide-react";
import { createExportJob, getExportJob, type CreateExportJobInput } from "../../api/exportJobs";
import type { ExportJob } from "../../api/types";

const POLL_INTERVAL_MS = 2000;

// Queues a job via POST /v1/exports and polls GET /v1/exports/:id until
// it's done, rather than blocking on a single synchronous request — for
// exports whose range/row count risks a client-side timeout on the sync
// endpoints (components/export/ExportButton.tsx), which stay the simpler
// default for everyday small exports.
export function AsyncExportButton({ label, input }: { label: string; input: CreateExportJobInput }) {
  const [job, setJob] = useState<ExportJob | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  async function handleClick() {
    setIsCreating(true);
    setError(null);
    try {
      const created = await createExportJob(input);
      setJob(created);
      pollRef.current = setInterval(async () => {
        try {
          const updated = await getExportJob(created.id);
          setJob(updated);
          if (updated.status === "completed" || updated.status === "failed") {
            if (pollRef.current) clearInterval(pollRef.current);
          }
        } catch {
          if (pollRef.current) clearInterval(pollRef.current);
          setError("Lost track of the job's status.");
        }
      }, POLL_INTERVAL_MS);
    } catch {
      setError("Couldn't queue the export job.");
    } finally {
      setIsCreating(false);
    }
  }

  if (job?.status === "completed" && job.download_url) {
    return (
      <a
        href={job.download_url}
        className="flex items-center gap-2 bg-primary text-on-primary font-body-base text-body-base px-4 py-2 rounded-full transition-colors shadow-soft"
      >
        <Download size={16} />
        <span>Download Ready</span>
      </a>
    );
  }

  if (job?.status === "failed" || error) {
    return (
      <button
        onClick={handleClick}
        className="flex items-center gap-2 glass-card rounded-full text-error font-body-base text-body-base px-4 py-2 transition-colors"
      >
        <span>{error ?? job?.error ?? "Export failed"} — retry</span>
      </button>
    );
  }

  if (job && (job.status === "pending" || job.status === "running")) {
    return (
      <span className="flex items-center gap-2 glass-card rounded-full text-on-surface-variant font-body-base text-body-base px-4 py-2">
        <Clock size={16} className="animate-pulse" />
        <span>{job.status === "pending" ? "Queued…" : "Generating…"}</span>
      </span>
    );
  }

  return (
    <button
      onClick={handleClick}
      disabled={isCreating}
      className="flex items-center gap-2 glass-card rounded-full text-on-surface hover:text-primary font-body-base text-body-base px-4 py-2 transition-colors disabled:opacity-60"
    >
      {isCreating ? <Loader2 size={16} className="animate-spin" /> : <Clock size={16} />}
      <span>{label}</span>
    </button>
  );
}
