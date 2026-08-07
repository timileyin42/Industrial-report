import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { Factory, BarChart3 } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { KpiCard } from "../components/kpi/KpiCard";
import { LineChart } from "../components/charts/LineChart";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { ExportButton } from "../components/export/ExportButton";
import { AsyncExportButton } from "../components/export/AsyncExportButton";
import { MapEmbed } from "../components/map/MapEmbed";
import { getSite } from "../api/sites";
import { listSiteTelemetry } from "../api/telemetry";
import { downloadSiteTelemetryCSV, downloadSiteSummaryCSV } from "../api/exports";
import { ApiError } from "../api/types";

// References: design/site_detail_lagos_central_hub_zgnis/code.html,
// design/site_telemetry_lagos_central_hub/code.html.
export function SiteDetailPage() {
  const { siteId } = useParams<{ siteId: string }>();

  const siteQuery = useQuery({
    queryKey: ["site", siteId],
    queryFn: () => getSite(siteId!),
    enabled: !!siteId,
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
          <div className="h-40 bg-surface-container border border-outline-variant animate-pulse" />
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
        <div className="bg-surface-container-high p-6 rounded-lg border border-primary/30">
          <div className="flex items-center justify-between gap-4 mb-4">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded bg-primary-container flex items-center justify-center text-primary">
                <Factory size={28} />
              </div>
              <div>
                <h4 className="font-headline-md text-headline-md text-on-surface">{site.name ?? site.site_id}</h4>
                <p className="text-on-surface-variant font-body-base">
                  {site.address ?? "No address on file"} · {site.site_id}
                </p>
              </div>
            </div>
            <Link
              to={`/app/sites/${site.site_id}/analytics`}
              className="flex items-center gap-2 bg-primary-container hover:bg-on-primary-fixed-variant text-on-primary-container font-bold px-4 py-2 rounded transition-colors whitespace-nowrap"
            >
              <BarChart3 size={16} />
              <span>View Analytics</span>
            </Link>
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

        <div className="bg-surface-container-lowest rounded-xl border border-outline-variant overflow-hidden">
          <div className="p-6 border-b border-outline-variant flex justify-between items-center">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">
              Power Output
            </span>
          </div>
          <div className="h-[260px] p-6">
            {telemetryQuery.isLoading ? (
              <div className="h-full bg-surface-container animate-pulse" />
            ) : points.length === 0 ? (
              <EmptyState
                title="No telemetry yet"
                body="This site hasn't reported any readings yet. Once its device starts publishing, data will appear here."
              />
            ) : (
              <LineChart points={chartPoints} color="#ffb95f" />
            )}
          </div>
        </div>

        <div className="bg-surface-container-lowest rounded-xl border border-outline-variant overflow-hidden">
          <div className="p-6 border-b border-outline-variant">
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
