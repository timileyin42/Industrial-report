import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Leaf } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { KpiCard } from "../components/kpi/KpiCard";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { AccessDenied } from "../components/feedback/AccessDenied";
import { getFleetEmissions } from "../api/emissions";
import { ApiError } from "../api/types";

// Fleet-wide CO2-avoided view — split out of the former combined
// FleetAnalyticsPage. Site-level emissions still live on SiteAnalyticsPage.
export function EmissionsPage() {
  const emissionsQuery = useQuery({ queryKey: ["fleet-emissions"], queryFn: () => getFleetEmissions(), retry: false });

  if (emissionsQuery.error instanceof ApiError && emissionsQuery.error.status === 403) {
    return <AccessDenied />;
  }
  const emissionsUnconfigured = emissionsQuery.error instanceof ApiError && emissionsQuery.error.status === 409;
  if (emissionsQuery.isError && !emissionsUnconfigured) {
    return <ErrorState onRetry={() => emissionsQuery.refetch()} />;
  }

  return (
    <>
      <TopNav title="Emissions" />
      <div className="flex-1 p-grid-margin space-y-8">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-gutter">
          <KpiCard
            label="Emissions Avoided (lifetime)"
            value={emissionsQuery.data ? emissionsQuery.data.cumulative_lifetime_co2_tonnes.toFixed(2) : "—"}
            unit={emissionsQuery.data ? "t CO2" : undefined}
          />
        </div>

        {/* Fleet spans more than one grid — no single factor represents
            all of it, so the backend returns a per-country breakdown
            instead of one emission_factor (see registry.CountryEmissions).
            An unconfigured country's sites generate real energy that's
            excluded from the total above until someone sets its factor
            — flagged here rather than left invisible. */}
        {emissionsQuery.data?.country_breakdown && (
          <div className="glass-card rounded-2xl p-6 space-y-2">
            <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Emissions by grid</span>
            <div className="flex flex-wrap gap-4 pt-2">
              {emissionsQuery.data.country_breakdown.map((c) => (
                <div key={c.country} className="flex items-center gap-2 text-body-base">
                  <span className="font-semibold text-on-surface">{c.country}</span>
                  {c.unconfigured ? (
                    <span className="text-[11px] text-secondary">
                      No emission factor set —{" "}
                      <Link to="/app/settings/emissions" className="underline">
                        configure
                      </Link>
                    </span>
                  ) : (
                    <span className="text-on-surface-variant font-data-mono-sm text-data-mono-sm">
                      {c.cumulative_lifetime_co2_tonnes.toFixed(2)} t CO2
                    </span>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {emissionsUnconfigured && (
          <EmptyState
            icon={<Leaf size={48} />}
            title="No grid emission factor configured"
            body="Emissions-avoided figures need a grid emission factor set first."
            action={
              <Link to="/app/settings/emissions" className="bg-primary text-on-primary font-bold px-4 py-2 rounded-full shadow-soft">
                Configure emission factor
              </Link>
            }
          />
        )}
      </div>
    </>
  );
}
