import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, Router, FileClock, RefreshCw, Copy, Check, AlertTriangle } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { DataTable, type Column } from "../components/table/DataTable";
import { StatusBadge } from "../components/status/StatusBadge";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { useAuth } from "../auth/AuthContext";
import { listDevices, rotateDeviceSecret } from "../api/devices";
import { ApiError, type Device, type DeviceWithSecret } from "../api/types";
import { deriveDeviceStatus } from "../lib/deviceStatus";

export function DevicesListPage() {
  const { session } = useAuth();
  const isOperator = session?.role === "operator";
  const queryClient = useQueryClient();
  const [rotatedSecret, setRotatedSecret] = useState<DeviceWithSecret | null>(null);
  const [copied, setCopied] = useState(false);
  const [rotateError, setRotateError] = useState<string | null>(null);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["devices"],
    queryFn: () => listDevices(),
  });

  // The old secret is invalidated server-side the instant this succeeds
  // (see internal/registry/devices.go RotateSecret) — this is the ONLY
  // moment the new one is ever visible, same discipline as registration.
  const rotateMutation = useMutation({
    mutationFn: (deviceId: string) => rotateDeviceSecret(deviceId),
    onSuccess: (device) => {
      setRotatedSecret(device);
      setCopied(false);
      setRotateError(null);
      queryClient.invalidateQueries({ queryKey: ["devices"] });
    },
    onError: (err) => {
      setRotateError(err instanceof ApiError ? err.message : "Couldn't rotate the secret. Try again.");
    },
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
    ...(isOperator
      ? [
          {
            header: "",
            align: "right" as const,
            render: (d: Device) => (
              <button
                type="button"
                disabled={rotateMutation.isPending}
                onClick={(e) => {
                  e.stopPropagation();
                  if (window.confirm(`Regenerate the secret for ${d.device_id}? The current secret stops working immediately.`)) {
                    rotateMutation.mutate(d.device_id);
                  }
                }}
                className="inline-flex items-center gap-1.5 text-on-surface-variant hover:text-primary transition-colors disabled:opacity-60"
                title="Regenerate secret"
              >
                <RefreshCw size={14} />
                <span className="text-[12px]">Regenerate Secret</span>
              </button>
            ),
          },
        ]
      : []),
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
            className="md:hidden flex items-center gap-2 glass-card rounded-full text-on-surface-variant hover:text-primary font-body-base text-body-base px-4 py-2 transition-colors"
          >
            <FileClock size={16} />
            <span>Ingestion Log</span>
          </Link>
          {isOperator && (
            <Link
              to="/app/devices/new"
              className="bg-primary hover:opacity-90 text-on-primary font-semibold py-2.5 px-5 rounded-full flex items-center gap-2 transition-all shadow-soft ml-auto"
            >
              <Plus size={18} />
              <span>Register New Device</span>
            </Link>
          )}
        </div>

        {rotatedSecret && (
          <div className="glass-card rounded-2xl p-6 space-y-4">
            <div className="flex items-start gap-3">
              <AlertTriangle size={20} className="text-secondary mt-0.5" />
              <p className="font-body-base text-on-surface-variant">
                New secret for <strong className="text-on-surface">{rotatedSecret.device_id}</strong> — shown{" "}
                <strong className="text-on-surface">exactly once</strong>. Copy it now and sync it into the
                Mosquitto broker; the old secret no longer works.
              </p>
            </div>
            <div className="flex gap-2">
              <code className="flex-1 bg-white/70 border border-outline-variant p-3 rounded-xl font-data-mono-sm text-data-mono-sm text-on-surface break-all">
                {rotatedSecret.secret}
              </code>
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard.writeText(rotatedSecret.secret);
                  setCopied(true);
                }}
                className="p-3 glass-card rounded-xl hover:text-primary transition-all"
                title="Copy to clipboard"
              >
                {copied ? <Check size={20} className="text-primary" /> : <Copy size={20} />}
              </button>
            </div>
            <div className="flex justify-end">
              <button
                type="button"
                onClick={() => setRotatedSecret(null)}
                className="px-6 py-2.5 bg-primary hover:opacity-90 text-on-primary font-semibold rounded-full transition-all shadow-soft"
              >
                Done
              </button>
            </div>
          </div>
        )}
        {rotateError && <p className="font-label-caps text-label-caps text-error">{rotateError}</p>}

        {isLoading || !data ? (
          <div className="h-64 glass-card rounded-xl animate-pulse" />
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
