import { useQuery } from "@tanstack/react-query";
import { Download, Clock, AlertCircle, Loader2 } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { DataTable, type Column } from "../components/table/DataTable";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { listExportJobs } from "../api/exportJobs";
import type { ExportJob } from "../api/types";

const JOB_TYPE_LABELS: Record<ExportJob["job_type"], string> = {
  site_telemetry_csv: "Site Telemetry CSV",
  site_summary_csv: "Site Summary CSV",
  fleet_summary_csv: "Fleet Summary CSV",
  site_summary_pdf: "Site Summary PDF",
  fleet_summary_pdf: "Fleet Summary PDF",
};

const STATUS_STYLES: Record<ExportJob["status"], string> = {
  completed: "text-success",
  failed: "text-error",
  running: "text-secondary",
  pending: "text-on-surface-variant",
};

// History of every async export job queued via AsyncExportButton
// (components/export/AsyncExportButton.tsx) elsewhere in the app — sync
// downloads (components/export/ExportButton.tsx) never create a job
// record, so they don't appear here; that's the real difference between
// the two export mechanisms, not a gap in this page.
export function ReportsPage() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ["export-jobs"],
    queryFn: () => listExportJobs(),
  });

  if (isError) {
    return <ErrorState onRetry={() => refetch()} />;
  }

  const columns: Column<ExportJob>[] = [
    { header: "Report", render: (j) => JOB_TYPE_LABELS[j.job_type] },
    { header: "Site", isMono: true, render: (j) => j.site_id ?? "Fleet-wide" },
    {
      header: "Status",
      render: (j) => (
        <span className={`inline-flex items-center gap-1.5 ${STATUS_STYLES[j.status]}`}>
          {j.status === "completed" && <Download size={14} />}
          {j.status === "failed" && <AlertCircle size={14} />}
          {j.status === "running" && <Loader2 size={14} className="animate-spin" />}
          {j.status === "pending" && <Clock size={14} />}
          <span className="capitalize">{j.status}</span>
        </span>
      ),
    },
    { header: "Queued", isMono: true, render: (j) => new Date(j.created_at).toLocaleString() },
    {
      header: "",
      align: "right",
      render: (j) =>
        j.status === "completed" && j.download_url ? (
          <a href={j.download_url} className="text-primary hover:underline text-[12px] font-semibold">
            Download
          </a>
        ) : j.status === "failed" ? (
          <span className="text-[12px] text-error">{j.error ?? "Failed"}</span>
        ) : null,
    },
  ];

  return (
    <>
      <TopNav title="Reports" />
      <div className="flex-1 p-grid-margin space-y-6">
        {isLoading || !data ? (
          <div className="h-64 glass-card rounded-xl animate-pulse" />
        ) : data.length === 0 ? (
          <EmptyState
            title="No reports queued yet"
            body="Async exports queued from a Site Detail or Analytics page will show up here with their status and a download link once ready."
          />
        ) : (
          <DataTable columns={columns} rows={data} rowKey={(j) => String(j.id)} />
        )}
      </div>
    </>
  );
}
