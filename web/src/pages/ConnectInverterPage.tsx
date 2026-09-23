import { useState, type FormEvent } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, CheckCircle2, KeyRound, Loader2, Mail, Zap } from "lucide-react";
import { useAuth } from "../auth/AuthContext";
import { listVendorProviders, createVendorConnection } from "../api/vendorConnections";
import { ApiError, type VendorProvider } from "../api/types";
import { LogoMark } from "../components/brand/Logo";
import { ErrorState } from "../components/feedback/ErrorState";
import elinterCspLogo from "../assets/brand/vendors/elinter_csp.png";
import felicitySolarLogo from "../assets/brand/vendors/felicity_solar.png";
import deyeCloudLogo from "../assets/brand/vendors/deye_cloud.png";
import sunsynkConnectLogo from "../assets/brand/vendors/sunsynk_connect.png";

// Each vendor's own real logo, pulled from their official site/app
// store listing (see the commit that added these) — keyed by the same
// provider name the backend uses, so a vendor added later without a
// logo here just falls back to the generic icon below rather than
// breaking.
const VENDOR_LOGOS: Record<string, string> = {
  elinter_csp: elinterCspLogo,
  felicity_solar: felicitySolarLogo,
  deye_cloud: deyeCloudLogo,
  sunsynk_connect: sunsynkConnectLogo,
};

// The step right after signup: pick which vendor cloud a customer's
// existing smart inverter already reports to (no way to auto-detect
// this — see the plan's discussion of why it has to be a manual pick,
// same as every comparable product), then authenticate to it. Only a
// password-form vendor has a concrete flow today (E-linter CSP/PV Pro);
// an oauth_token vendor is a reserved seam with no adapter built yet,
// so it's shown but disabled rather than hidden — customers can see
// what's coming without it looking abandoned.
export function ConnectInverterPage() {
  const { session } = useAuth();
  const navigate = useNavigate();
  const [selected, setSelected] = useState<VendorProvider | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [connected, setConnected] = useState(false);

  const providersQuery = useQuery({ queryKey: ["vendor-providers"], queryFn: listVendorProviders });

  // This step only makes sense for a restricted customer who just
  // signed up with their own site — an operator has no single site to
  // connect a vendor account to, and any restricted user without a
  // site_id would mean the self-signup bootstrap itself failed.
  if (!session || session.role !== "restricted" || !session.siteId) {
    return <Navigate to={session ? "/app" : "/login"} replace />;
  }
  const siteId = session.siteId;

  async function handleConnect(e: FormEvent) {
    e.preventDefault();
    if (!selected) return;
    setError(null);
    setIsSubmitting(true);
    try {
      await createVendorConnection(siteId, { provider: selected.name, email, password });
      setConnected(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Couldn't reach the server. Try again.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center text-on-surface px-6 py-12">
      <main className="w-full max-w-[480px]">
        <div className="glass-card rounded-2xl p-10 flex flex-col items-center">
          <div className="mb-8 text-center flex flex-col items-center">
            <LogoMark size={32} />
            <h1 className="font-headline-lg text-headline-lg font-bold text-primary tracking-tight mb-1 mt-3">
              Connect Your Inverter
            </h1>
            <p className="font-body-base text-body-base text-on-surface-variant">
              {connected
                ? "You're all set."
                : selected
                  ? `Sign in with the same details you use in the ${selected.display_name} app`
                  : "Which app does your inverter already report to?"}
            </p>
          </div>

          {connected ? (
            <div className="w-full flex flex-col items-center text-center gap-6">
              <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center text-primary">
                <CheckCircle2 size={32} />
              </div>
              <p className="font-body-base text-body-base text-on-surface-variant">
                We're syncing your data now — it can take a minute or two for your first reading to
                appear.
              </p>
              <button
                type="button"
                onClick={() => navigate(`/app/sites/${siteId}`, { replace: true })}
                className="w-full bg-primary hover:opacity-90 text-on-primary font-bold py-3.5 px-6 rounded-full transition-all shadow-soft"
              >
                Go to my dashboard
              </button>
            </div>
          ) : providersQuery.isLoading ? (
            <div className="py-8 text-on-surface-variant">
              <Loader2 size={24} className="animate-spin" />
            </div>
          ) : providersQuery.isError ? (
            <ErrorState onRetry={() => providersQuery.refetch()} />
          ) : !selected ? (
            <div className="w-full space-y-5">
              <div className="grid grid-cols-2 gap-3">
                {providersQuery.data?.map((p) => (
                  <button
                    key={p.name}
                    type="button"
                    onClick={() => setSelected(p)}
                    className="flex flex-col items-center gap-3 bg-white/70 hover:bg-white border border-outline-variant rounded-xl px-4 py-5 text-center transition-all"
                  >
                    <div className="w-16 h-16 flex-shrink-0 rounded-lg bg-white border border-outline-variant/60 flex items-center justify-center overflow-hidden p-2">
                      {VENDOR_LOGOS[p.name] ? (
                        <img src={VENDOR_LOGOS[p.name]} alt={`${p.display_name} logo`} className="max-w-full max-h-full object-contain" />
                      ) : (
                        <Zap size={24} className="text-primary" />
                      )}
                    </div>
                    <div>
                      <p className="font-body-base text-body-base font-bold text-on-surface">{p.display_name}</p>
                      <p className="font-body-base text-[12px] text-on-surface-variant">
                        {p.auth_type === "password" ? "Email & password" : "Coming soon"}
                      </p>
                    </div>
                  </button>
                ))}
              </div>
              <button
                type="button"
                onClick={() => navigate(`/app/sites/${siteId}`, { replace: true })}
                className="w-full text-center font-body-base text-body-base text-on-surface-variant hover:text-primary transition-colors"
              >
                Skip for now
              </button>
            </div>
          ) : selected.auth_type !== "password" ? (
            <div className="w-full flex flex-col items-center text-center gap-6">
              <p className="font-body-base text-body-base text-on-surface-variant">
                {selected.display_name} isn't connectable yet — we'll email you the moment it's ready.
              </p>
              <button
                type="button"
                onClick={() => setSelected(null)}
                className="font-body-base text-body-base text-primary hover:underline flex items-center gap-1"
              >
                <ArrowLeft size={16} /> Choose a different vendor
              </button>
            </div>
          ) : (
            <form className="w-full space-y-6" onSubmit={handleConnect}>
              {VENDOR_LOGOS[selected.name] && (
                <div className="w-16 h-12 mx-auto rounded-lg bg-white border border-outline-variant/60 flex items-center justify-center overflow-hidden p-2">
                  <img
                    src={VENDOR_LOGOS[selected.name]}
                    alt={`${selected.display_name} logo`}
                    className="max-w-full max-h-full object-contain"
                  />
                </div>
              )}
              <div className="space-y-2">
                <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="vendor-email">
                  {selected.display_name} Email
                </label>
                <div className="relative">
                  <Mail size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant" />
                  <input
                    id="vendor-email"
                    type="email"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full bg-white/70 border border-outline-variant text-on-surface font-body-base text-body-base pl-10 pr-4 py-3 rounded-xl focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
                  />
                </div>
              </div>
              <div className="space-y-2">
                <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="vendor-password">
                  {selected.display_name} Password
                </label>
                <div className="relative">
                  <KeyRound size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant" />
                  <input
                    id="vendor-password"
                    type="password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full bg-white/70 border border-outline-variant text-on-surface font-body-base text-body-base pl-10 pr-4 py-3 rounded-xl focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
                  />
                </div>
              </div>
              <p className="font-body-base text-[12px] text-on-surface-variant text-center">
                Your {selected.display_name} password is encrypted and only ever used to fetch your own
                inverter's readings.
              </p>
              {error && <p className="font-label-caps text-label-caps text-error text-center">{error}</p>}
              <button
                type="submit"
                disabled={isSubmitting}
                className="w-full bg-primary hover:opacity-90 text-on-primary font-bold py-3.5 px-6 rounded-full flex items-center justify-center gap-2 transition-all disabled:opacity-70 shadow-soft"
              >
                {isSubmitting ? (
                  <>
                    <Loader2 size={20} className="animate-spin" />
                    <span>Connecting…</span>
                  </>
                ) : (
                  <span>Connect {selected.display_name}</span>
                )}
              </button>
              <button
                type="button"
                onClick={() => setSelected(null)}
                className="w-full text-center font-body-base text-body-base text-on-surface-variant hover:text-primary transition-colors flex items-center justify-center gap-1"
              >
                <ArrowLeft size={16} /> Choose a different vendor
              </button>
            </form>
          )}
        </div>
      </main>
    </div>
  );
}
