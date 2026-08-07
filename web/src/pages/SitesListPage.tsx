import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, MapPin } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { DataTable, type Column } from "../components/table/DataTable";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { useAuth } from "../auth/AuthContext";
import { listSites } from "../api/sites";
import type { Site } from "../api/types";

export function SitesListPage() {
  const { session } = useAuth();
  const isOperator = session?.role === "operator";

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ["sites"],
    queryFn: () => listSites(),
  });

  const columns: Column<Site>[] = [
    {
      header: "Site",
      render: (s) => (
        <div className="flex items-center gap-3">
          <MapPin size={18} className="text-on-surface-variant" />
          <div>
            <p className="font-body-base font-bold text-on-surface">{s.name ?? s.site_id}</p>
            <p className="text-xs font-data-mono-sm text-on-surface-variant">{s.site_id}</p>
          </div>
        </div>
      ),
    },
    { header: "Timezone", render: (s) => s.timezone },
    {
      header: "System Size",
      isMono: true,
      render: (s) => (s.system_size_kw != null ? `${s.system_size_kw} kWp` : "—"),
    },
    {
      header: "Actions",
      align: "right",
      render: (s) => (
        <Link to={`/app/sites/${s.site_id}`} className="text-on-surface-variant hover:text-primary transition-colors">
          View
        </Link>
      ),
    },
  ];

  return (
    <>
      <TopNav title="Sites" />
      <div className="flex-1 p-grid-margin space-y-6">
        {isOperator && (
          <div className="flex justify-end">
            <Link
              to="/app/sites/new"
              className="bg-primary-container text-primary font-label-caps py-2.5 px-5 rounded flex items-center gap-2 border border-primary/30 hover:bg-primary/20 transition-all"
            >
              <Plus size={18} />
              <span>Add Site</span>
            </Link>
          </div>
        )}

        {isError ? (
          <ErrorState onRetry={() => refetch()} />
        ) : isLoading || !data ? (
          <div className="h-64 bg-surface-container border border-outline-variant animate-pulse" />
        ) : data.items.length === 0 ? (
          <EmptyState
            title="No sites registered yet"
            body="Begin monitoring your renewable assets by initializing your first fleet location."
            action={
              isOperator ? (
                <Link
                  to="/app/sites/new"
                  className="px-8 py-3 bg-primary-container text-primary font-bold rounded-lg border border-primary hover:bg-primary hover:text-on-primary transition-all flex items-center gap-2"
                >
                  <Plus size={20} />
                  <span>Add your first site</span>
                </Link>
              ) : undefined
            }
          />
        ) : (
          <DataTable columns={columns} rows={data.items} rowKey={(s) => s.site_id} />
        )}
      </div>
    </>
  );
}
