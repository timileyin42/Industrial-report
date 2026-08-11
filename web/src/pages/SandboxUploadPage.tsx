import { useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { UploadCloud, Loader2, FlaskConical } from "lucide-react";
import { LogoMark } from "../components/brand/Logo";
import { SandboxBadge } from "../components/sandbox/SandboxBadge";
import { uploadSandboxCSV } from "../api/sandbox";
import { ApiError } from "../api/types";

// Public, no-login page — a shareable link for validating a sample of
// someone's own readings against this platform's real ingestion rules
// (see internal/registry/sandbox.go), not an admin tool. Deliberately
// outside the /app tree and RequireAuth entirely.
export function SandboxUploadPage() {
  const navigate = useNavigate();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [systemSizeKW, setSystemSizeKW] = useState("");
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit() {
    if (!file) return;
    setError(null);
    setIsUploading(true);
    try {
      const result = await uploadSandboxCSV(file, systemSizeKW ? Number(systemSizeKW) : undefined);
      navigate(`/sandbox/${result.run_id}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Couldn't reach the server. Try again.");
    } finally {
      setIsUploading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center text-on-surface px-6 py-12">
      <SandboxBadge />
      <main className="w-full max-w-[560px]">
        <div className="glass-card rounded-2xl p-10">
          <div className="mb-8 text-center flex flex-col items-center">
            <Link to="/" aria-label="Back to home">
              <LogoMark size={32} />
            </Link>
            <h1 className="font-headline-lg text-headline-lg font-bold text-primary tracking-tight mb-1 mt-3">
              Clean Energy Analytics
            </h1>
            <p className="font-body-base text-body-base text-on-surface-variant flex items-center gap-2 mt-1">
              <FlaskConical size={16} />
              Data Sandbox
            </p>
          </div>

          <p className="text-[13px] text-on-surface-variant text-center mb-8">
            Upload a CSV of readings and see them validated exactly the way a real
            device's telemetry would be — impossible readings rejected, counter
            resets flagged, nothing silently accepted. No account needed. Nothing
            uploaded here ever touches real fleet data.
          </p>

          <div className="glass-card rounded-xl p-4 mb-6 text-[12px] text-on-surface-variant">
            <p className="font-label-caps text-label-caps text-on-surface uppercase mb-2">Expected CSV columns</p>
            <p className="font-data-mono-sm">ts, power_kw, energy_kwh_total</p>
            <p className="mt-1">Optional: <span className="font-data-mono-sm">voltage_v, status, rssi</span></p>
          </div>

          <div className="space-y-5">
            <div
              className="border-2 border-dashed border-outline-variant rounded-xl p-8 text-center cursor-pointer hover:border-primary transition-colors"
              onClick={() => fileInputRef.current?.click()}
            >
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv,text/csv"
                className="hidden"
                onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              />
              <UploadCloud size={28} className="mx-auto text-on-surface-variant mb-2" />
              <p className="text-[13px] text-on-surface">
                {file ? file.name : "Click to choose a CSV file"}
              </p>
            </div>

            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="system-size">
                System size, kWp (optional)
              </label>
              <input
                id="system-size"
                type="number"
                min="0"
                step="0.1"
                value={systemSizeKW}
                onChange={(e) => setSystemSizeKW(e.target.value)}
                placeholder="e.g. 5"
                className="w-full bg-white/70 border border-outline-variant text-on-surface font-body-base text-body-base px-4 py-3 rounded-xl focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
              />
              <p className="text-[11px] text-on-surface-variant">
                Used only to set the "implausibly high power" check — same as a real site's nameplate capacity. Leave blank to use a generic default.
              </p>
            </div>

            {error && <p className="text-[13px] text-error text-center">{error}</p>}

            <button
              onClick={handleSubmit}
              disabled={!file || isUploading}
              className="w-full flex items-center justify-center gap-2 bg-primary text-on-primary font-semibold py-3 rounded-xl disabled:opacity-50 transition-opacity"
            >
              {isUploading ? <Loader2 size={18} className="animate-spin" /> : <UploadCloud size={18} />}
              {isUploading ? "Validating…" : "Upload & Validate"}
            </button>
          </div>
        </div>
      </main>
    </div>
  );
}
