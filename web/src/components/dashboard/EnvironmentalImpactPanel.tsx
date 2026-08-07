import { TreeDeciduous, Leaf, Car } from "lucide-react";

// EPA GHG Equivalencies Calculator's published constants — real,
// citable conversion factors applied to our own measured CO2-avoided
// figure, not invented numbers. (1 metric ton CO2 ≈ CO2 sequestered by
// 16.5 tree seedlings grown for 10 years; average passenger vehicle
// emits ≈4.6 metric tons CO2/year.)
const TREE_SEEDLINGS_PER_TONNE = 16.5;
const TONNES_PER_CAR_YEAR = 4.6;

export function EnvironmentalImpactPanel({ cumulativeTonnesCO2 }: { cumulativeTonnesCO2: number | null }) {
  const rows = cumulativeTonnesCO2 != null
    ? [
        {
          icon: TreeDeciduous,
          label: "Tree Seedlings Equivalent",
          value: `${Math.round(cumulativeTonnesCO2 * TREE_SEEDLINGS_PER_TONNE).toLocaleString()}`,
          color: "text-success",
        },
        {
          icon: Leaf,
          label: "CO2 Offset",
          value: `${cumulativeTonnesCO2.toFixed(2)} t`,
          color: "text-success",
        },
        {
          icon: Car,
          label: "Cars Off Road (1yr equiv.)",
          value: cumulativeTonnesCO2 >= TONNES_PER_CAR_YEAR ? (cumulativeTonnesCO2 / TONNES_PER_CAR_YEAR).toFixed(1) : "< 1",
          color: "text-success",
        },
      ]
    : [];

  return (
    <div className="glass-card rounded-xl p-5">
      <span className="font-label-caps text-label-caps text-on-surface-variant uppercase">Environmental Impact</span>
      {cumulativeTonnesCO2 == null ? (
        <p className="mt-4 font-body-base text-body-base text-on-surface-variant">
          Set a grid emission factor to see impact figures.
        </p>
      ) : (
        <div className="mt-4 space-y-3">
          {rows.map((row) => (
            <div key={row.label} className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-on-surface-variant">
                <row.icon size={16} className={row.color} />
                <span className="font-body-base text-body-base">{row.label}</span>
              </div>
              <span className="font-data-display-lg text-[16px] text-on-surface">{row.value}</span>
            </div>
          ))}
          <p className="text-[10px] text-on-surface-variant pt-1">
            Based on EPA GHG equivalency factors applied to measured energy output.
          </p>
        </div>
      )}
    </div>
  );
}
