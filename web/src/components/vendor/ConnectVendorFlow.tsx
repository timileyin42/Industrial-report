import { useState, type FormEvent, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, KeyRound, Loader2, Mail, Zap } from "lucide-react";
import { listVendorProviders, createVendorConnection } from "../../api/vendorConnections";
import { ApiError, type VendorConnection, type VendorProvider } from "../../api/types";
import { ErrorState } from "../feedback/ErrorState";
import { VENDOR_LOGOS } from "./vendorLogos";

// The vendor picker + password-form flow — pick which app a customer's
// inverter already reports to (no way to auto-detect this, same as
// every comparable product), then authenticate to it. Only a
// password-form vendor has a concrete flow today; an oauth_token
// vendor is a reserved seam with no adapter built yet, so it's shown
// but disabled rather than hidden.
//
// Self-contained (owns its own picker/form state) so it can be reused
// wherever a customer connects a vendor account: right after signup
// (ConnectInverterPage) and later from Settings, when they skipped
// that step or want to add another vendor (InverterConnectionsPage).
// The two call sites differ only in what happens once connected and
// what (if anything) appears below the picker — both handled via
// props rather than duplicating this ~150 lines of picker/form JSX.
export function ConnectVendorFlow({
  siteId,
  onConnected,
  pickerFooter,
}: {
  siteId: string;
  onConnected: (connection: VendorConnection) => void;
  pickerFooter?: ReactNode;
}) {
  const [selected, setSelected] = useState<VendorProvider | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const providersQuery = useQuery({ queryKey: ["vendor-providers"], queryFn: listVendorProviders });

  async function handleConnect(e: FormEvent) {
    e.preventDefault();
    if (!selected) return;
    setError(null);
    setIsSubmitting(true);
    try {
      const connection = await createVendorConnection(siteId, { provider: selected.name, email, password });
      onConnected(connection);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Couldn't reach the server. Try again.");
    } finally {
      setIsSubmitting(false);
    }
  }

  if (providersQuery.isLoading) {
    return (
      <div className="py-8 text-on-surface-variant">
        <Loader2 size={24} className="animate-spin" />
      </div>
    );
  }
  if (providersQuery.isError) {
    return <ErrorState onRetry={() => providersQuery.refetch()} />;
  }

  if (!selected) {
    return (
      <div className="w-full space-y-5">
        <p className="font-body-base text-body-base text-on-surface-variant text-center">
          Which app does your inverter already report to?
        </p>
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
        {pickerFooter}
      </div>
    );
  }

  if (selected.auth_type !== "password") {
    return (
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
    );
  }

  return (
    <form className="w-full space-y-6" onSubmit={handleConnect}>
      <p className="font-body-base text-body-base text-on-surface-variant text-center">
        Sign in with the same details you use in the {selected.display_name} app
      </p>
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
  );
}
