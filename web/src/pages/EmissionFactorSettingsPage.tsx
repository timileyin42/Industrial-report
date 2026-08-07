import { useState, type FormEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Leaf } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { EmptyState } from "../components/feedback/EmptyState";
import { ErrorState } from "../components/feedback/ErrorState";
import { getEmissionFactor, setEmissionFactor } from "../api/emissions";
import { ApiError } from "../api/types";

// Reference: no dedicated screen exists in design/ for this — flagged per
// CLAUDE.md's rule 6 (don't improvise a new visual language for a
// screen the design system doesn't cover). Built as a minimal form
// reusing existing tokens/components rather than inventing new ones.
// Operator-only: enforced both by the route guard and by the backend
// (POST /v1/config/emission-factor is operatorOnly in router.go).
export function EmissionFactorSettingsPage() {
  const queryClient = useQueryClient();
  const currentQuery = useQuery({ queryKey: ["emission-factor"], queryFn: getEmissionFactor, retry: false });

  const [kgCo2PerKwh, setKgCo2PerKwh] = useState("");
  // No regional default — the operator must pick their own country/grid
  // explicitly rather than silently inheriting a value that made sense
  // for this platform's first deployment.
  const [country, setCountry] = useState("");
  const [source, setSource] = useState("");
  const [effectiveFrom, setEffectiveFrom] = useState(() => new Date().toISOString().slice(0, 10));
  const [formError, setFormError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () =>
      setEmissionFactor({
        kg_co2_per_kwh: Number(kgCo2PerKwh),
        country,
        source,
        effective_from: new Date(effectiveFrom).toISOString(),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["emission-factor"] });
      setSource("");
    },
    onError: (err) => {
      setFormError(err instanceof ApiError ? err.message : "Couldn't save the emission factor. Try again.");
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setFormError(null);
    mutation.mutate();
  }

  const notConfigured = currentQuery.error instanceof ApiError && currentQuery.error.status === 409;
  if (currentQuery.isError && !notConfigured) {
    return <ErrorState onRetry={() => currentQuery.refetch()} />;
  }

  return (
    <>
      <TopNav title="Emission Factor" />
      <div className="flex-1 p-grid-margin space-y-8 max-w-2xl">
        <div className="bg-surface-container border border-outline-variant p-6">
          <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Current Factor</span>
          <div className="mt-4">
            {currentQuery.isLoading ? (
              <div className="h-16 bg-surface-container-high animate-pulse" />
            ) : notConfigured ? (
              <EmptyState
                icon={<Leaf size={40} />}
                title="No factor configured yet"
                body="Set the grid emission factor below to enable emissions-avoided figures fleet-wide."
              />
            ) : currentQuery.data ? (
              <div className="flex items-baseline gap-3">
                <span className="font-data-display-lg text-data-display-lg text-primary">
                  {currentQuery.data.kg_co2_per_kwh}
                </span>
                <span className="text-on-surface-variant font-data-mono-sm text-data-mono-sm">kg CO2/kWh</span>
                <span className="text-[10px] text-on-surface-variant ml-4">
                  {currentQuery.data.country} · {currentQuery.data.source} · effective{" "}
                  {new Date(currentQuery.data.effective_from).toLocaleDateString()}
                </span>
              </div>
            ) : null}
          </div>
        </div>

        <form className="bg-surface-container border border-outline-variant p-6 space-y-5" onSubmit={handleSubmit}>
          <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Set New Factor</span>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-gutter">
            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="kgco2">
                kg CO2 per kWh
              </label>
              <input
                id="kgco2"
                type="number"
                step="0.001"
                min="0"
                required
                value={kgCo2PerKwh}
                onChange={(e) => setKgCo2PerKwh(e.target.value)}
                className="w-full bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm px-4 py-3 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none"
              />
            </div>
            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="country">
                Country
              </label>
              <input
                id="country"
                type="text"
                required
                placeholder="e.g. NG, US, GB"
                value={country}
                onChange={(e) => setCountry(e.target.value)}
                className="w-full bg-background border border-outline-variant text-on-surface font-body-base text-body-base px-4 py-3 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none"
              />
            </div>
            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="source">
                Source
              </label>
              <input
                id="source"
                type="text"
                required
                placeholder="e.g. National grid average, IEA 2025"
                value={source}
                onChange={(e) => setSource(e.target.value)}
                className="w-full bg-background border border-outline-variant text-on-surface font-body-base text-body-base px-4 py-3 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none"
              />
            </div>
            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="effective">
                Effective From
              </label>
              <input
                id="effective"
                type="date"
                required
                value={effectiveFrom}
                onChange={(e) => setEffectiveFrom(e.target.value)}
                className="w-full bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm px-4 py-3 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none"
              />
            </div>
          </div>
          {formError && <p className="font-label-caps text-label-caps text-error">{formError}</p>}
          <button
            type="submit"
            disabled={mutation.isPending}
            className="bg-primary-container hover:bg-on-primary-fixed-variant text-on-primary-container font-bold py-3 px-6 rounded transition-all disabled:opacity-70"
          >
            {mutation.isPending ? "Saving..." : "Save Emission Factor"}
          </button>
        </form>
      </div>
    </>
  );
}
