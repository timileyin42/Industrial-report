import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { Factory, BarChart3, Pencil, Star } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { KpiCard } from "../components/kpi/KpiCard";
import { LineChart } from "../components/charts/LineChart";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { ExportButton } from "../components/export/ExportButton";
import { AsyncExportButton } from "../components/export/AsyncExportButton";
import { MapEmbed } from "../components/map/MapEmbed";
import { useAuth } from "../auth/AuthContext";
import { getSite, updateSiteCountry, setSitePrimary } from "../api/sites";
import { listSiteTelemetry } from "../api/telemetry";
import { downloadSiteTelemetryCSV, downloadSiteSummaryCSV } from "../api/exports";
import { ApiError } from "../api/types";

// References: design/site_detail_lagos_central_hub_zgnis/code.html,
// design/site_telemetry_lagos_central_hub/code.html.
export function SiteDetailPage() {
  const { siteId } = useParams<{ siteId: string }>();
  const { session } = useAuth();
  const isOperator = session?.role === "operator";
  const queryClient = useQueryClient();
  const [editingCountry, setEditingCountry] = useState(false);
  const [countryInput, setCountryInput] = useState("");
  const [countryError, setCountryError] = useState<string | null>(null);
  const [primaryError, setPrimaryError] = useState<string | null>(null);

  const siteQuery = useQuery({
    queryKey: ["site", siteId],
    queryFn: () => getSite(siteId!),
    enabled: !!siteId,
  });

  // Every site created before migrations/0010_site_country.sql was
  // backfilled to 'NG' — this is the only place that guess can be
  // corrected, since there's no general site-edit form yet.
  const countryMutation = useMutation({
    mutationFn: (country: string) => updateSiteCountry(siteId!, country),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["site", siteId] });
      setEditingCountry(false);
      setCountryError(null);
    },
    onError: (err) => {
      setCountryError(err instanceof ApiError ? err.message : "Couldn't update the country. Try again.");
    },
  });

  // The fleet's one primary/home site — what the Fleet Dashboard's
  // weather widget resolves its location from (see WeatherWidget wiring
  // in FleetDashboardPage.tsx). Setting a new primary clears the flag
  // from whichever site held it before, server-side.
  const primaryMutation = useMutation({
    mutationFn: () => setSitePrimary(siteId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["site", siteId] });
      setPrimaryError(null);
    },
    onError: (err) => {
      setPrimaryError(err instanceof ApiError ? err.message : "Couldn't set this as the primary site. Try again.");
    },
  });

  const telemetryQuery = useQuery({
    queryKey: ["site-telemetry", siteId],
    queryFn: () => listSiteTelemetry(siteId!, { limit: 200 }),
    enabled: !!siteId,
  });

  const anyError = siteQuery.error ?? telemetryQuery.error;
  if (anyError instanceof ApiError && anyError.status === 403) {
    // Matches the backend's explicit design choice: a site-scope mismatch
    // is always a 403, never a 404 that leaks existence.
    return <AccessDenied detail="This site isn't part of your account's access scope." />;
  }
  if (siteQuery.isError || telemetryQuery.isError) {
    return <ErrorState onRetry={() => { siteQuery.refetch(); telemetryQuery.refetch(); }} />;
  }

  if (siteQuery.isLoading || !siteQuery.data) {
    return (
      <>
        <TopNav title="Site" />
        <div className="flex-1 p-grid-margin">
          <div className="h-40 glass-card rounded-xl animate-pulse" />
        </div>
      </>
    );
  }

  const site = siteQuery.data;
  const points = telemetryQuery.data?.items ?? [];
  const latest = points[points.length - 1];
  const chartPoints = points.map((p, i) => ({ x: i, y: p.power_kw }));

  return (
    <>
      <TopNav title={site.name ?? site.site_id} />
      <div className="flex-1 p-grid-margin space-y-8">
        <div className="glass-card rounded-2xl p-6">
          <div className="flex items-center justify-between gap-4 mb-4">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-xl bg-primary-container flex items-center justify-center text-primary">
                <Factory size={28} />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h4 className="font-headline-md text-headline-md text-on-surface">{site.name ?? site.site_id}</h4>
                  {site.is_primary && (
                    <span className="flex items-center gap-1 bg-primary-container text-on-primary-container text-[10px] font-bold uppercase px-2 py-1 rounded-full">
                      <Star size={10} className="fill-current" />
                      Primary
                    </span>
                  )}
                </div>
                <p className="text-on-surface-variant font-body-base">
                  {site.address ?? "No address on file"} · {site.site_id}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {isOperator && !site.is_primary && (
                <button
                  type="button"
                  disabled={primaryMutation.isPending}
                  onClick={() => primaryMutation.mutate()}
                  title="This is what the Fleet Dashboard's weather widget uses as its location"
                  className="flex items-center gap-2 glass-card rounded-full px-4 py-2 text-on-surface-variant hover:text-primary transition-colors disabled:opacity-60 whitespace-nowrap"
                >
                  <Star size={16} />
                  <span>Set as Primary</span>
                </button>
              )}
              <Link
                to={`/app/sites/${site.site_id}/analytics`}
                className="flex items-center gap-2 bg-primary hover:opacity-90 text-on-primary font-bold px-4 py-2 rounded-full transition-colors shadow-soft whitespace-nowrap"
              >
                <BarChart3 size={16} />
                <span>View Analytics</span>
              </Link>
            </div>
          </div>
          {primaryError && <p className="font-label-caps text-label-caps text-error mb-2">{primaryError}</p>}

          <div className="flex items-center gap-2 text-[12px] text-on-surface-variant">
            <span className="uppercase font-label-caps text-label-caps">Grid country:</span>
            {editingCountry ? (
              <form
                className="flex items-center gap-2"
                onSubmit={(e) => {
                  e.preventDefault();
                  countryMutation.mutate(countryInput.toUpperCase());
                }}
              >
                <input
                  autoFocus
                  maxLength={2}
                  value={countryInput}
                  onChange={(e) => setCountryInput(e.target.value.toUpperCase())}
                  className="w-16 bg-white/70 border border-outline-variant rounded-lg px-2 py-1 font-data-mono-sm text-data-mono-sm uppercase focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none"
                />
                <button
                  type="submit"
                  disabled={countryMutation.isPending}
                  className="text-primary font-semibold disabled:opacity-60"
                >
                  Save
                </button>
                <button type="button" onClick={() => setEditingCountry(false)} className="text-on-surface-variant">
                  Cancel
                </button>
              </form>
            ) : (
              <button
                type="button"
                className="font-data-mono-sm text-data-mono-sm text-on-surface flex items-center gap-1"
                onClick={() => {
                  setCountryInput(site.country);
                  setEditingCountry(true);
                }}
                disabled={!isOperator}
              >
                {site.country}
                {isOperator && <Pencil size={12} className="text-on-surface-variant" />}
              </button>
            )}
            {countryError && <span className="text-error">{countryError}</span>}
          </div>
        </div>

        <div className="flex flex-wrap justify-end gap-2">
          <ExportButton label="Export Telemetry CSV" onExport={() => downloadSiteTelemetryCSV(site.site_id)} />
          <ExportButton label="Export Summary CSV" onExport={() => downloadSiteSummaryCSV(site.site_id)} />
          {/* Async alternative for ranges too large for a synchronous
              request to finish comfortably — same data, queued instead. */}
          <AsyncExportButton label="Queue Telemetry Export" input={{ job_type: "site_telemetry_csv", site_id: site.site_id }} />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-gutter">
          <KpiCard
            label="Current Power"
            value={latest ? latest.power_kw.toFixed(1) : "—"}
            unit="kW"
            tone="primary"
          />
          <KpiCard
            label="System Size"
            value={site.system_size_kw ?? "—"}
            unit={site.system_size_kw != null ? "kWp" : undefined}
          />
          <KpiCard
            label="Last Reading"
            value={latest ? new Date(latest.ts).toLocaleTimeString() : "—"}
          />
        </div>

        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60 flex justify-between items-center">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">
              Power Output
            </span>
          </div>
          <div className="h-[260px] p-6">
            {telemetryQuery.isLoading ? (
              <div className="h-full bg-surface-dim rounded-xl animate-pulse" />
            ) : points.length === 0 ? (
              <EmptyState
                title="No telemetry yet"
                body="This site hasn't reported any readings yet. Once its device starts publishing, data will appear here."
              />
            ) : (
              <LineChart points={chartPoints} color="#f2a93b" />
            )}
          </div>
        </div>

        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-6 border-b border-outline-variant/60">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Location</span>
          </div>
          <div className="h-[300px]">
            {site.gps_lat != null && site.gps_lng != null ? (
              <MapEmbed lat={site.gps_lat} lng={site.gps_lng} label={site.name ?? site.site_id} />
            ) : (
              <EmptyState title="No location set" body="Add latitude/longitude when editing this site to show it on a map." />
            )}
          </div>
        </div>
      </div>
    </>
  );
}
