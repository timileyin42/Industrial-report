import { useState, type FormEvent } from "react";
import { Link, Navigate, useNavigate } from "react-router-dom";
import { Mail, KeyRound, UserPlus, Loader2 } from "lucide-react";
import { useAuth } from "../auth/AuthContext";
import { ApiError } from "../api/types";
import { LogoMark } from "../components/brand/Logo";

// Public self-service signup — creates a brand-new restricted account
// and an initially empty site (internal/registry/signup.go), then signs
// the customer straight in and sends them to /connect-inverter to link
// their existing smart-inverter vendor account. Mirrors LoginPage's
// layout exactly (same glass-card auth-page shell every page in this
// family uses) rather than inventing a new visual language for it.
export function SignupPage() {
  const { signup, session } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  // Captured once, at mount — this is "did someone already-authenticated
  // land here by mistake," not "is there a session right now." A live
  // `session` check would also fire the instant signup() below succeeds
  // (that sets a brand-new session too), redirecting to the dashboard
  // and racing the explicit navigate to /connect-inverter that's
  // supposed to happen next — confirmed happening in browser testing.
  const [hadSessionOnMount] = useState(() => session);

  if (hadSessionOnMount) {
    const fallback = hadSessionOnMount.role === "operator" ? "/app" : `/app/sites/${hadSessionOnMount.siteId}`;
    return <Navigate to={fallback} replace />;
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (password !== confirm) {
      setError("Passwords don't match");
      return;
    }
    setIsSubmitting(true);
    try {
      await signup(email, password);
      navigate("/connect-inverter", { replace: true });
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setError("An account already exists for that email");
      } else if (err instanceof ApiError && err.status === 400) {
        setError(err.message);
      } else {
        setError("Couldn't reach the server. Try again.");
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center text-on-surface px-6">
      <main className="w-full max-w-[400px]">
        <div className="glass-card rounded-2xl p-10 flex flex-col items-center">
          <div className="mb-10 text-center flex flex-col items-center">
            <Link to="/" aria-label="Back to home">
              <LogoMark size={32} />
            </Link>
            <h1 className="font-headline-lg text-headline-lg font-bold text-primary tracking-tight mb-1 mt-3">
              Create Your Account
            </h1>
            <p className="font-body-base text-body-base text-on-surface-variant">
              Already have a smart inverter? Connect it in the next step.
            </p>
          </div>
          <form className="w-full space-y-6" onSubmit={handleSubmit}>
            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="email">
                Email
              </label>
              <div className="relative">
                <Mail size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant" />
                <input
                  id="email"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                  className="w-full bg-white/70 border border-outline-variant text-on-surface font-body-base text-body-base pl-10 pr-4 py-3 rounded-xl focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
                />
              </div>
            </div>
            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="password">
                Password
              </label>
              <div className="relative">
                <KeyRound size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant" />
                <input
                  id="password"
                  type="password"
                  required
                  minLength={8}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="At least 8 characters"
                  className="w-full bg-white/70 border border-outline-variant text-on-surface font-body-base text-body-base pl-10 pr-4 py-3 rounded-xl focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
                />
              </div>
            </div>
            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="confirm">
                Confirm Password
              </label>
              <div className="relative">
                <KeyRound size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-on-surface-variant" />
                <input
                  id="confirm"
                  type="password"
                  required
                  minLength={8}
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  className="w-full bg-white/70 border border-outline-variant text-on-surface font-body-base text-body-base pl-10 pr-4 py-3 rounded-xl focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
                />
              </div>
            </div>
            {error && (
              <p className="font-label-caps text-label-caps text-error text-center">{error}</p>
            )}
            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full bg-primary hover:opacity-90 text-on-primary font-bold py-3.5 px-6 rounded-full flex items-center justify-center gap-2 transition-all disabled:opacity-70 shadow-soft"
            >
              {isSubmitting ? (
                <>
                  <Loader2 size={20} className="animate-spin" />
                  <span>Creating account…</span>
                </>
              ) : (
                <>
                  <span>Create Account</span>
                  <UserPlus size={20} />
                </>
              )}
            </button>
          </form>
          <p className="mt-6 font-body-base text-body-base text-on-surface-variant">
            Already have an account?{" "}
            <Link to="/login" className="text-primary hover:underline">
              Sign in
            </Link>
          </p>
        </div>
      </main>
    </div>
  );
}
