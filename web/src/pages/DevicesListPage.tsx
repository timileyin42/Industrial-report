import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, Router, FileClock } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { DataTable, type Column } from "../components/table/DataTable";
import { StatusBadge } from "../components/status/StatusBadge";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { useAuth } from "../auth/AuthContext";
import { listDevices } from "../api/devices";
import { ApiError, type Device } from "../api/types";
import { deriveDeviceStatus } from "../lib/deviceStatus";

export function DevicesListPage() {
  const { session } = useAuth();
  const isOperator = session?.role === "operator";

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["devices"],
    queryFn: () => listDevices(),
  });

  if (isError) {
    if (error instanceof ApiError && error.status === 403) return <AccessDenied />;
    return <ErrorState onRetry={() => refetch()} />;
  }

  const columns: Column<Device>[] = [
    {
      header: "Device ID",
      isMono: true,
      render: (d) => (
        <div className="flex items-center gap-3">
          <Router size={16} className="text-on-surface-variant" />
          <span className="text-primary">{d.device_id}</span>
        </div>
      ),
    },
    { header: "Mapped Site", render: (d) => d.site_id ?? "—" },
    {
      header: "Status",
      render: (d) => <StatusBadge status={deriveDeviceStatus(d.last_seen_at, d.revoked_at)} />,
    },
    {
      header: "Last Seen",
      isMono: true,
      render: (d) => (d.last_seen_at ? new Date(d.last_seen_at).toLocaleString() : "Never"),
    },
  ];

  return (
    <>
      <TopNav title="Device Registry" />
      <div className="flex-1 p-grid-margin space-y-6">
        <div className="flex justify-between items-center">
          {/* Ingestion Log's sidebar entry only shows on desktop (md+) —
              this is the mobile-reachable path to it, on the page both
              roles already land on. */}
          <Link
            to="/app/ingestion-log"
            className="md:hidden flex items-center gap-2 bg-surface-container border border-outline-variant hover:border-primary-container text-on-surface-variant hover:text-on-surface font-body-base text-body-base px-4 py-2 rounded transition-colors"
          >
            <FileClock size={16} />
            <span>Ingestion Log</span>
          </Link>
          {isOperator && (
            <Link
              to="/app/devices/new"
              className="bg-primary-container text-primary font-label-caps py-2.5 px-5 rounded flex items-center gap-2 border border-primary/30 hover:bg-primary/20 transition-all ml-auto"
            >
              <Plus size={18} />
              <span>Register New Device</span>
            </Link>
          )}
        </div>

        {isLoading || !data ? (
          <div className="h-64 bg-surface-container border border-outline-variant animate-pulse" />
        ) : data.items.length === 0 ? (
          <EmptyState
            title="No devices registered yet"
            body="Register your first hardware node to start receiving telemetry."
            action={
              isOperator ? (
                <Link
                  to="/app/devices/new"
                  className="px-8 py-3 bg-primary-container text-primary font-bold rounded-lg border border-primary hover:bg-primary hover:text-on-primary transition-all flex items-center gap-2"
                >
                  <Plus size={20} />
                  <span>Register a device</span>
                </Link>
              ) : undefined
            }
          />
        ) : (
          <DataTable columns={columns} rows={data.items} rowKey={(d) => d.device_id} />
        )}
      </div>
    </>
  );
}
