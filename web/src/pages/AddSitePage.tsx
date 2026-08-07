import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { TopNav } from "../components/layout/TopNav";
import { LocationPicker } from "../components/map/LocationPicker";
import { createSite } from "../api/sites";
import { ApiError } from "../api/types";

// Reference: design/add_site_zgnis_industrial_intelligence/code.html.
// Deliberate deviations from the mockup, all flagged in the plan:
//  - adds a Site ID field the mockup doesn't have (mockup only shows
//    Name; the backend requires a distinct site_id primary key).
//  - omits "Install Date" (no backend field to bind it to).
//  - replaces the mockup's plain lat/lng text fields with LocationPicker
//    (search-or-click-to-pin) — typing coordinates by hand isn't how an
//    operator should have to place a site when a map is right there.
export function AddSitePage() {
  const navigate = useNavigate();
  const [siteId, setSiteId] = useState("");
  const [name, setName] = useState("");
  const [address, setAddress] = useState("");
  const [addressTouched, setAddressTouched] = useState(false);
  const [gpsLat, setGpsLat] = useState<number | null>(null);
  const [gpsLng, setGpsLng] = useState<number | null>(null);
  const [inverterMakeModel, setInverterMakeModel] = useState("");
  const [systemSizeKw, setSystemSizeKw] = useState("");
  // Defaults off the browser's own timezone rather than any region —
  // matches the backend's UTC fallback for "not specified at all"
  // (see internal/httpapi/site_handlers.go), but a real operator adding
  // a real site almost always wants their own timezone pre-filled.
  const [timezone, setTimezone] = useState(() => Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC");
  const [cohortId, setCohortId] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);
    try {
      const site = await createSite({
        site_id: siteId,
        name,
        address: address || undefined,
        gps_lat: gpsLat ?? undefined,
        gps_lng: gpsLng ?? undefined,
        inverter_make_model: inverterMakeModel || undefined,
        system_size_kw: systemSizeKw ? Number(systemSizeKw) : undefined,
        timezone,
        cohort_id: cohortId || undefined,
      });
      navigate(`/app/sites/${site.site_id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to create site");
    } finally {
      setIsSubmitting(false);
    }
  }

  const inputClass =
    "w-full bg-surface-dim border border-outline-variant text-on-surface font-body-base rounded-sm py-2.5 px-4 focus:border-primary focus:ring-1 focus:ring-primary outline-none placeholder:text-outline-variant/50";
  const monoInputClass =
    "w-full bg-surface-dim border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm rounded-sm py-2.5 px-4 focus:border-primary focus:ring-1 focus:ring-primary outline-none placeholder:text-outline-variant/50";
  const labelClass = "block text-label-caps font-label-caps text-on-surface-variant mb-2";

  return (
    <>
      <TopNav title="Add Site" />
      <div className="flex-1 p-grid-margin max-w-5xl w-full">
        <form className="space-y-gutter" onSubmit={handleSubmit}>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-gutter">
            <div className="md:col-span-2 bg-surface-container p-6 border border-outline-variant rounded-lg">
              <h3 className="text-label-caps font-label-caps text-primary border-b border-outline-variant pb-3 mb-6 uppercase tracking-[0.1em]">
                General Information
              </h3>
              <div className="space-y-5">
                <div>
                  <label className={labelClass}>SITE ID</label>
                  <input
                    className={monoInputClass}
                    placeholder="e.g. SITE-0003"
                    required
                    value={siteId}
                    onChange={(e) => setSiteId(e.target.value)}
                  />
                </div>
                <div>
                  <label className={labelClass}>SITE NAME</label>
                  <input
                    className={inputClass}
                    placeholder="e.g. West Cluster A"
                    required
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                  />
                </div>
                <div>
                  <label className={labelClass}>SITE ADDRESS</label>
                  <textarea
                    className={`${inputClass} resize-none`}
                    rows={3}
                    placeholder="Street name, District, State..."
                    value={address}
                    onChange={(e) => {
                      setAddressTouched(true);
                      setAddress(e.target.value);
                    }}
                  />
                </div>
              </div>
            </div>
            <div className="bg-surface-container p-6 border border-outline-variant rounded-lg">
              <h3 className="text-label-caps font-label-caps text-primary border-b border-outline-variant pb-3 mb-6 uppercase tracking-[0.1em]">
                Geo-Location
              </h3>
              <LocationPicker
                lat={gpsLat}
                lng={gpsLng}
                onChange={(lat, lng, resolvedAddress) => {
                  setGpsLat(lat);
                  setGpsLng(lng);
                  // Only auto-fill if the operator hasn't typed their own
                  // address — a search/click result shouldn't clobber a
                  // deliberate manual entry.
                  if (resolvedAddress && !addressTouched) {
                    setAddress(resolvedAddress);
                  }
                }}
              />
            </div>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-gutter">
            <div className="bg-surface-container p-6 border border-outline-variant rounded-lg">
              <h3 className="text-label-caps font-label-caps text-primary border-b border-outline-variant pb-3 mb-6 uppercase tracking-[0.1em]">
                Technical Specs
              </h3>
              <div className="grid grid-cols-2 gap-4">
                <div className="col-span-2">
                  <label className={labelClass}>INVERTER MAKE / MODEL</label>
                  <input
                    className={inputClass}
                    placeholder="e.g. Huawei SUN2000-100KTL"
                    value={inverterMakeModel}
                    onChange={(e) => setInverterMakeModel(e.target.value)}
                  />
                </div>
                <div>
                  <label className={labelClass}>SYSTEM SIZE (kWp)</label>
                  <input
                    className={monoInputClass}
                    type="number"
                    step="0.1"
                    placeholder="0.0"
                    value={systemSizeKw}
                    onChange={(e) => setSystemSizeKw(e.target.value)}
                  />
                </div>
              </div>
            </div>
            <div className="bg-surface-container p-6 border border-outline-variant rounded-lg">
              <h3 className="text-label-caps font-label-caps text-primary border-b border-outline-variant pb-3 mb-6 uppercase tracking-[0.1em]">
                System Configuration
              </h3>
              <div className="space-y-5">
                <div>
                  <label className={labelClass}>TIMEZONE</label>
                  <input className={inputClass} value={timezone} onChange={(e) => setTimezone(e.target.value)} />
                </div>
                <div>
                  <label className={labelClass}>COHORT / PROJECT (optional)</label>
                  <input className={inputClass} value={cohortId} onChange={(e) => setCohortId(e.target.value)} />
                </div>
              </div>
            </div>
          </div>

          {error && <p className="font-label-caps text-label-caps text-error">{error}</p>}

          <div className="flex items-center justify-end gap-4 pt-4 border-t border-outline-variant">
            <button
              type="button"
              onClick={() => navigate(-1)}
              className="px-8 py-2.5 border border-outline-variant text-on-surface font-label-caps tracking-widest rounded-sm hover:bg-surface-container-highest uppercase"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="px-10 py-2.5 bg-primary-container text-primary font-label-caps tracking-widest rounded-sm border border-primary/20 hover:brightness-110 uppercase disabled:opacity-70"
            >
              {isSubmitting ? "Saving..." : "Save Site"}
            </button>
          </div>
        </form>
      </div>
    </>
  );
}
