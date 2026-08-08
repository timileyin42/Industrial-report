import { useQuery } from "@tanstack/react-query";
import { Layers } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { DataTable, type Column } from "../components/table/DataTable";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { listCohorts } from "../api/cohorts";
import type { Cohort } from "../api/types";

// Cohorts have no dedicated management UI beyond this list — a cohort is
// just whatever value operators have typed into a site's "Cohort /
// Project" field (AddSitePage.tsx), so there's nothing to create/delete
// here independently of the sites that reference it.
export function CohortsListPage() {
  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ["cohorts"],
    queryFn: () => listCohorts(),
  });

  if (isError) {
    return <ErrorState onRetry={() => refetch()} />;
  }

  const columns: Column<Cohort>[] = [
    {
      header: "Cohort / Project",
      render: (c) => (
        <div className="flex items-center gap-3">
          <Layers size={16} className="text-on-surface-variant" />
          <span className="text-on-surface">{c.cohort_id}</span>
        </div>
      ),
    },
    { header: "Sites", isMono: true, align: "right", render: (c) => c.site_count },
    { header: "Total Capacity", isMono: true, align: "right", render: (c) => `${(c.total_capacity_kw / 1000).toFixed(2)} MWp` },
  ];

  return (
    <>
      <TopNav title="Cohorts / Projects" />
      <div className="flex-1 p-grid-margin space-y-6">
        {isLoading || !data ? (
          <div className="h-64 glass-card rounded-xl animate-pulse" />
        ) : data.length === 0 ? (
          <EmptyState
            title="No cohorts yet"
            body="Assign a Cohort / Project value when adding a site to group sites together here."
          />
        ) : (
          <DataTable columns={columns} rows={data} rowKey={(c) => c.cohort_id} />
        )}
      </div>
    </>
  );
}
