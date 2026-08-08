import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { TopNav } from "../components/layout/TopNav";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { FleetMiniMap } from "../components/dashboard/FleetMiniMap";
import { listSites } from "../api/sites";
import { getFleetHealth } from "../api/fleetHealth";
import { ApiError } from "../api/types";

// Fleet-wide equivalent of LocationPicker/MapEmbed (both single-site) —
// no new backend needed, this just joins two already-fetched lists
// (site locations, per-site health) by site_id and plots one marker per
// located site via the same FleetMiniMap the Dashboard's preview panel
// uses. Sites with no saved gps_lat/gps_lng simply aren't plottable and
// are counted separately rather than silently omitted.
export function MapViewPage() {
  const navigate = useNavigate();
  const sitesQuery = useQuery({ queryKey: ["map-sites"], queryFn: () => listSites(undefined, 200) });
  const healthQuery = useQuery({ queryKey: ["map-health"], queryFn: () => getFleetHealth(undefined, 200) });

  const sites = sitesQuery.data?.items ?? [];
  const healthBySite = new Map((healthQuery.data?.sites.items ?? []).map((s) => [s.site_id, s]));
  const located = sites.filter((s) => s.gps_lat != null && s.gps_lng != null);
  const unlocated = sites.length - located.length;

  const anyError = sitesQuery.error ?? healthQuery.error;
  if (anyError instanceof ApiError && anyError.status === 403) return <AccessDenied />;
  if (sitesQuery.isError) return <ErrorState onRetry={() => sitesQuery.refetch()} />;

  return (
    <>
      <TopNav title="Map View" />
      <div className="flex-1 p-grid-margin space-y-4">
        {unlocated > 0 && (
          <p className="text-[12px] text-on-surface-variant">
            {unlocated} site{unlocated === 1 ? "" : "s"} without a saved location {unlocated === 1 ? "isn't" : "aren't"}{" "}
            shown on the map.
          </p>
        )}
        {sitesQuery.isLoading ? (
          <div className="h-[70vh] glass-card rounded-2xl animate-pulse" />
        ) : (
          <div className="glass-card rounded-2xl p-2">
            <FleetMiniMap
              sites={sites}
              healthBySite={healthBySite}
              height="calc(70vh)"
              zoom={6}
              onSiteClick={(siteId) => navigate(`/app/sites/${siteId}`)}
            />
          </div>
        )}
      </div>
    </>
  );
}
