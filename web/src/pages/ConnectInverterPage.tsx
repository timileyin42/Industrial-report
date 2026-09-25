import { useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { CheckCircle2 } from "lucide-react";
import { useAuth } from "../auth/AuthContext";
import { LogoMark } from "../components/brand/Logo";
import { ConnectVendorFlow } from "../components/vendor/ConnectVendorFlow";

// The step right after signup — see ConnectVendorFlow for the actual
// picker/form. This page owns only what's specific to being a
// mandatory post-signup step: the "connected!" success screen and the
// "Skip for now" escape hatch. A customer who skips (or wants to add
// another vendor later) isn't stuck — the same flow is reachable again
// from Settings → Connect Your Inverter (InverterConnectionsPage).
export function ConnectInverterPage() {
  const { session } = useAuth();
  const navigate = useNavigate();
  const [connected, setConnected] = useState(false);

  // This step only makes sense for a restricted customer who just
  // signed up with their own site — an operator has no single site to
  // connect a vendor account to, and any restricted user without a
  // site_id would mean the self-signup bootstrap itself failed.
  if (!session || session.role !== "restricted" || !session.siteId) {
    return <Navigate to={session ? "/app" : "/login"} replace />;
  }
  const siteId = session.siteId;

  return (
    <div className="min-h-screen flex items-center justify-center text-on-surface px-6 py-12">
      <main className="w-full max-w-[480px]">
        <div className="glass-card rounded-2xl p-10 flex flex-col items-center">
          <div className="mb-8 text-center flex flex-col items-center">
            <LogoMark size={32} />
            <h1 className="font-headline-lg text-headline-lg font-bold text-primary tracking-tight mb-1 mt-3">
              Connect Your Inverter
            </h1>
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
          ) : (
            <ConnectVendorFlow
              siteId={siteId}
              onConnected={() => setConnected(true)}
              pickerFooter={
                <button
                  type="button"
                  onClick={() => navigate(`/app/sites/${siteId}`, { replace: true })}
                  className="w-full text-center font-body-base text-body-base text-on-surface-variant hover:text-primary transition-colors"
                >
                  Skip for now — connect later from Settings
                </button>
              }
            />
          )}
        </div>
      </main>
    </div>
  );
}
