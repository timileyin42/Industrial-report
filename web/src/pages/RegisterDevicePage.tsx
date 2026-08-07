import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Copy, Check, AlertTriangle } from "lucide-react";
import { TopNav } from "../components/layout/TopNav";
import { registerDevice } from "../api/devices";
import { listSites } from "../api/sites";
import { ApiError, type DeviceWithSecret } from "../api/types";

// Reference: design/register_device_zgnis_industrial_intelligence/code.html.
// The mockup's "Device Secret" input and "Hardware Class" field are both
// omitted: the secret is generated server-side (never client input), and
// there's no backend field for hardware class. The returned plaintext
// secret is shown exactly once, then never re-fetchable — same secrets
// discipline CLAUDE.md requires everywhere else in this system.
export function RegisterDevicePage() {
  const [deviceId, setDeviceId] = useState("");
  const [siteId, setSiteId] = useState("");
  const [installNotes, setInstallNotes] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [result, setResult] = useState<DeviceWithSecret | null>(null);
  const [copied, setCopied] = useState(false);

  const sitesQuery = useQuery({ queryKey: ["sites-for-select"], queryFn: () => listSites(undefined, 200) });

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);
    try {
      const device = await registerDevice({ device_id: deviceId, site_id: siteId, install_notes: installNotes || undefined });
      setResult(device);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to register device");
    } finally {
      setIsSubmitting(false);
    }
  }

  const inputClass =
    "w-full bg-white/70 border border-outline-variant p-3 font-body-base text-on-surface rounded-xl focus:ring-2 focus:ring-primary/20 focus:border-primary outline-none";
  const monoInputClass = inputClass;
  const labelClass = "block font-label-caps text-on-surface-variant mb-2";

  if (result) {
    return (
      <>
        <TopNav title="Device Registered" />
        <div className="flex-1 p-grid-margin max-w-2xl w-full">
          <div className="glass-card rounded-2xl p-6 space-y-6">
            <div className="flex items-start gap-3">
              <AlertTriangle size={20} className="text-secondary mt-0.5" />
              <p className="font-body-base text-on-surface-variant">
                This secret is shown <strong className="text-on-surface">exactly once</strong>. Copy it now and
                sync it into the Mosquitto broker — it will never be shown again. If it's lost, rotate it for a
                new one.
              </p>
            </div>
            <div>
              <p className={labelClass}>DEVICE ID</p>
              <p className="font-data-mono-sm text-data-mono-sm text-primary">{result.device_id}</p>
            </div>
            <div>
              <p className={labelClass}>DEVICE SECRET</p>
              <div className="flex gap-2">
                <code className="flex-1 bg-white/70 border border-outline-variant p-3 rounded-xl font-data-mono-sm text-data-mono-sm text-on-surface break-all">
                  {result.secret}
                </code>
                <button
                  type="button"
                  onClick={() => {
                    navigator.clipboard.writeText(result.secret);
                    setCopied(true);
                  }}
                  className="p-3 glass-card rounded-xl hover:text-primary transition-all"
                  title="Copy to clipboard"
                >
                  {copied ? <Check size={20} className="text-primary" /> : <Copy size={20} />}
                </button>
              </div>
            </div>
            <div className="flex justify-end gap-3 pt-4">
              <Link
                to="/app/devices"
                className="px-6 py-2.5 bg-primary hover:opacity-90 text-on-primary font-semibold rounded-full transition-all shadow-soft"
              >
                Done
              </Link>
            </div>
          </div>
        </div>
      </>
    );
  }

  return (
    <>
      <TopNav title="Register Device" />
      <div className="flex-1 p-grid-margin max-w-2xl w-full">
        <form className="space-y-6" onSubmit={handleSubmit}>
          <section className="glass-card rounded-2xl p-6">
            <h3 className="font-label-caps text-on-surface tracking-wider border-b border-outline-variant/30 pb-4 mb-6">
              DEVICE IDENTITY
            </h3>
            <div className="space-y-6">
              <div>
                <label className={labelClass}>DEVICE ID (SERIAL NUMBER)</label>
                <input
                  className={monoInputClass}
                  placeholder="ZG-0003"
                  required
                  value={deviceId}
                  onChange={(e) => setDeviceId(e.target.value)}
                />
              </div>
              <div>
                <label className={labelClass}>SITE ASSIGNMENT</label>
                <select
                  className={inputClass}
                  required
                  value={siteId}
                  onChange={(e) => setSiteId(e.target.value)}
                >
                  <option value="">Select a site...</option>
                  {sitesQuery.data?.items.map((s) => (
                    <option key={s.site_id} value={s.site_id}>
                      {s.name ?? s.site_id} ({s.site_id})
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </section>
          <section className="glass-card rounded-2xl p-6">
            <h3 className="font-label-caps text-on-surface tracking-wider border-b border-outline-variant/30 pb-4 mb-6">
              DEPLOYMENT CONTEXT
            </h3>
            <div>
              <label className={labelClass}>INSTALLATION NOTES</label>
              <textarea
                className={`${inputClass} resize-none`}
                rows={4}
                placeholder="Mounting details, technician ID, environmental constraints..."
                value={installNotes}
                onChange={(e) => setInstallNotes(e.target.value)}
              />
            </div>
          </section>

          {error && <p className="font-label-caps text-label-caps text-error">{error}</p>}

          <div className="flex items-center justify-end gap-4">
            <Link to="/app/devices" className="px-6 py-2.5 font-label-caps text-on-surface-variant hover:text-on-surface">
              CANCEL
            </Link>
            <button
              type="submit"
              disabled={isSubmitting}
              className="px-8 py-2.5 bg-primary hover:opacity-90 text-on-primary font-bold transition-all rounded-full disabled:opacity-70 shadow-soft"
            >
              {isSubmitting ? "Registering…" : "COMPLETE REGISTRATION"}
            </button>
          </div>
        </form>
      </div>
    </>
  );
}
