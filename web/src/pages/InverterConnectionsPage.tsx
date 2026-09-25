import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Navigate } from "react-router-dom";
import { Plus, Trash2, Zap } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { StatusBadge, type Status } from "../components/status/StatusBadge";
import { ConnectVendorFlow } from "../components/vendor/ConnectVendorFlow";
import { VENDOR_LOGOS } from "../components/vendor/vendorLogos";
import { useAuth } from "../auth/AuthContext";
import { listVendorConnections, listVendorProviders, revokeVendorConnection } from "../api/vendorConnections";
import { ApiError, type VendorConnection } from "../api/types";

// Reached from Settings (a restricted-only nav item — see Sidebar) —
// the place a customer who skipped Connect Your Inverter during signup
// (or wants to add a second vendor) comes back to. Genuinely missing
// before this: /connect-inverter existed but only as a one-time
// post-signup step with no link back to it anywhere in the app.
const STATUS_MAP: Record<VendorConnection["status"], { status: Status; label: string }> = {
  active: { status: "online", label: "CONNECTED" },
  pending: { status: "maintenance", label: "CONNECTING" },
  error: { status: "degraded", label: "RETRYING" },
  invalid_credentials: { status: "offline", label: "RECONNECT NEEDED" },
  revoked: { status: "maintenance", label: "DISCONNECTED" },
};

export function InverterConnectionsPage() {
  const { session } = useAuth();
  const queryClient = useQueryClient();
  const [isAdding, setIsAdding] = useState(false);
  const [revokeError, setRevokeError] = useState<string | null>(null);

  // Same scope as ConnectInverterPage — this is a customer managing
  // their own single site's vendor connections, not a fleet-wide admin
  // page, so an operator (no personal site_id) has nothing to see here.
  if (!session || session.role !== "restricted" || !session.siteId) {
    return <Navigate to={session ? "/app" : "/login"} replace />;
  }
  const siteId = session.siteId;

  const connectionsQuery = useQuery({
    queryKey: ["vendor-connections", siteId],
    queryFn: () => listVendorConnections(siteId),
  });
  // Connections only carry the provider's stable key (e.g. "deye_cloud")
  // — this looks up the friendly display_name the picker already shows,
  // so a connected row reads "Deye Inverter" rather than the raw key.
  const providersQuery = useQuery({ queryKey: ["vendor-providers"], queryFn: listVendorProviders });
  const displayName = (provider: string) =>
    providersQuery.data?.find((p) => p.name === provider)?.display_name ?? provider;

  const revokeMutation = useMutation({
    mutationFn: (connectionId: number) => revokeVendorConnection(siteId, connectionId),
    onSuccess: () => {
      setRevokeError(null);
      queryClient.invalidateQueries({ queryKey: ["vendor-connections", siteId] });
    },
    onError: (err) => {
      setRevokeError(err instanceof ApiError ? err.message : "Couldn't disconnect this vendor. Try again.");
    },
  });

  function handleConnected() {
    setIsAdding(false);
    queryClient.invalidateQueries({ queryKey: ["vendor-connections", siteId] });
  }

  return (
    <>
      <TopNav title="Inverter Connections" />
      <div className="flex-1 p-grid-margin space-y-6">
        {connectionsQuery.isLoading ? (
          <div className="h-40 glass-card rounded-xl animate-pulse" />
        ) : connectionsQuery.isError ? (
          <ErrorState onRetry={() => connectionsQuery.refetch()} />
        ) : (
          <>
            {revokeError && <p className="font-label-caps text-label-caps text-error">{revokeError}</p>}

            {connectionsQuery.data && connectionsQuery.data.length > 0 ? (
              <div className="space-y-3">
                {connectionsQuery.data.map((conn) => {
                  const meta = STATUS_MAP[conn.status];
                  return (
                    <div
                      key={conn.id}
                      className="glass-card rounded-xl px-5 py-4 flex items-center gap-4"
                    >
                      <div className="w-12 h-12 flex-shrink-0 rounded-lg bg-white border border-outline-variant/60 flex items-center justify-center overflow-hidden p-1.5">
                        {VENDOR_LOGOS[conn.provider] ? (
                          <img src={VENDOR_LOGOS[conn.provider]} alt="" className="max-w-full max-h-full object-contain" />
                        ) : (
                          <Zap size={18} className="text-primary" />
                        )}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="font-body-base text-body-base font-bold text-on-surface truncate">{displayName(conn.provider)}</p>
                        {conn.status === "invalid_credentials" ? (
                          <p className="font-body-base text-[12px] text-error">
                            Password no longer works — disconnect and reconnect to fix it.
                          </p>
                        ) : conn.last_error && conn.status === "error" ? (
                          <p className="font-body-base text-[12px] text-on-surface-variant truncate">{conn.last_error}</p>
                        ) : conn.last_synced_at ? (
                          <p className="font-body-base text-[12px] text-on-surface-variant">
                            Last synced {new Date(conn.last_synced_at).toLocaleString()}
                          </p>
                        ) : (
                          <p className="font-body-base text-[12px] text-on-surface-variant">Waiting for first sync…</p>
                        )}
                      </div>
                      <StatusBadge status={meta.status} label={meta.label} />
                      {conn.status !== "revoked" && (
                        <button
                          type="button"
                          onClick={() => revokeMutation.mutate(conn.id)}
                          disabled={revokeMutation.isPending}
                          className="p-2 rounded-full text-on-surface-variant hover:text-error hover:bg-error-container/20 transition-colors disabled:opacity-50"
                          title="Disconnect"
                        >
                          <Trash2 size={16} />
                        </button>
                      )}
                    </div>
                  );
                })}
              </div>
            ) : !isAdding ? (
              <EmptyState
                icon={<Zap size={48} />}
                title="No inverter connected yet"
                body="Connect your existing smart inverter's cloud account (PV Pro, Deye, Felicity Solar, Sunsynk) and your real data will start showing up on your dashboard."
                action={
                  <button
                    type="button"
                    onClick={() => setIsAdding(true)}
                    className="bg-primary hover:opacity-90 text-on-primary font-bold py-3 px-6 rounded-full flex items-center gap-2 transition-all shadow-soft"
                  >
                    <Plus size={18} /> Connect Your Inverter
                  </button>
                }
              />
            ) : null}

            {connectionsQuery.data && connectionsQuery.data.length > 0 && !isAdding && (
              <button
                type="button"
                onClick={() => setIsAdding(true)}
                className="flex items-center gap-2 font-body-base text-body-base text-primary hover:underline"
              >
                <Plus size={18} /> Connect another inverter
              </button>
            )}

            {isAdding && (
              <div className="glass-card rounded-2xl p-8 max-w-[480px]">
                <ConnectVendorFlow siteId={siteId} onConnected={handleConnected} />
                <button
                  type="button"
                  onClick={() => setIsAdding(false)}
                  className="mt-4 w-full text-center font-body-base text-body-base text-on-surface-variant hover:text-primary transition-colors"
                >
                  Cancel
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </>
  );
}
