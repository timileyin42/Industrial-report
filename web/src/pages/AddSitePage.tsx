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
  const [timezoneTouched, setTimezoneTouched] = useState(false);
  const [cohortId, setCohortId] = useState("");
  // Resolves which grid emission factor this site's CO2-offset reporting
  // uses (backend migrations/0010_site_country.sql) — required, no
  // default. Deliberately not pre-filled off browser locale the way
  // timezone used to be: a browser's locale is a poor proxy for which
  // grid a solar site is actually connected to. Auto-filled from the map
  // pick instead (see LocationPicker's onChange below), same as address.
  const [country, setCountry] = useState("");
  const [countryTouched, setCountryTouched] = useState(false);
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
        country,
      });
      navigate(`/app/sites/${site.site_id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to create site");
    } finally {
      setIsSubmitting(false);
    }
  }

  const inputClass =
    "w-full bg-white/70 border border-outline-variant text-on-surface font-body-base rounded-xl py-2.5 px-4 focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none placeholder:text-on-surface-variant/50";
  const monoInputClass = inputClass;
  const labelClass = "block text-label-caps font-label-caps text-on-surface-variant mb-2";
  const cardClass = "glass-card rounded-2xl p-6";
  const cardHeaderClass = "text-label-caps font-label-caps text-primary border-b border-outline-variant/60 pb-3 mb-6 uppercase tracking-[0.1em]";

  return (
    <>
      <TopNav title="Add Site" />
      <div className="flex-1 p-grid-margin max-w-5xl w-full">
        <form className="space-y-gutter" onSubmit={handleSubmit}>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-gutter">
            <div className={`md:col-span-2 ${cardClass}`}>
              <h3 className={cardHeaderClass}>
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
            <div className={cardClass}>
              <h3 className={cardHeaderClass}>
                Geo-Location
              </h3>
              <LocationPicker
                lat={gpsLat}
                lng={gpsLng}
                onChange={(lat, lng, resolvedAddress, resolvedCountry, resolvedTimezone) => {
                  setGpsLat(lat);
                  setGpsLng(lng);
                  // Only auto-fill if the operator hasn't typed their own
                  // value for that field — a search/click/drag result
                  // shouldn't clobber a deliberate manual entry.
                  if (resolvedAddress && !addressTouched) {
                    setAddress(resolvedAddress);
                  }
                  if (resolvedCountry && !countryTouched) {
                    setCountry(resolvedCountry);
                  }
                  if (resolvedTimezone && !timezoneTouched) {
                    setTimezone(resolvedTimezone);
                  }
                }}
              />
            </div>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-gutter">
            <div className={cardClass}>
              <h3 className={cardHeaderClass}>
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
            <div className={cardClass}>
              <h3 className={cardHeaderClass}>
                System Configuration
              </h3>
              <div className="space-y-5">
                <div>
                  <label className={labelClass}>TIMEZONE</label>
                  <input
                    className={inputClass}
                    value={timezone}
                    onChange={(e) => {
                      setTimezoneTouched(true);
                      setTimezone(e.target.value);
                    }}
                  />
                  <p className="text-[10px] text-on-surface-variant mt-1.5">
                    Auto-fills from the map pick — edit here to override.
                  </p>
                </div>
                <div>
                  <label className={labelClass}>COUNTRY</label>
                  <input
                    className={monoInputClass}
                    placeholder="e.g. NG, GB"
                    required
                    maxLength={2}
                    value={country}
                    onChange={(e) => {
                      setCountryTouched(true);
                      setCountry(e.target.value.toUpperCase());
                    }}
                  />
                  <p className="text-[10px] text-on-surface-variant mt-1.5">
                    2-letter code — auto-fills from the map pick. Determines which grid emission factor this
                    site's CO2-offset reporting uses.
                  </p>
                </div>
                <div>
                  <label className={labelClass}>COHORT / PROJECT (optional)</label>
                  <input className={inputClass} value={cohortId} onChange={(e) => setCohortId(e.target.value)} />
                </div>
              </div>
            </div>
          </div>

          {error && <p className="font-label-caps text-label-caps text-error">{error}</p>}

          <div className="flex items-center justify-end gap-4 pt-4">
            <button
              type="button"
              onClick={() => navigate(-1)}
              className="px-8 py-2.5 glass-card rounded-full text-on-surface hover:text-primary transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="px-10 py-2.5 bg-primary text-on-primary font-semibold rounded-full hover:opacity-90 transition-all disabled:opacity-70 shadow-soft"
            >
              {isSubmitting ? "Saving…" : "Save Site"}
            </button>
          </div>
        </form>
      </div>
    </>
  );
}
