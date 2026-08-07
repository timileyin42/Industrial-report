import { useState, type FormEvent } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { Mail, KeyRound, LogIn, Loader2 } from "lucide-react";
import { useAuth } from "../auth/AuthContext";
import { ApiError } from "../api/types";
import { LogoMark } from "../components/brand/Logo";

// Light/glass redesign — copy softened to match the friendlier tone the
// rest of the app now uses ("Terminal Access Email"/"Initialize Session"
// were leftover industrial-terminal language from the old dark theme).
// A 401 here is expected/inline — never triggers the global "session
// expired" redirect, since there's no prior session to invalidate.
export function LoginPage() {
  const { login, session } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (session) {
    const fallback = session.role === "operator" ? "/app" : `/app/sites/${session.siteId}`;
    return <Navigate to={fallback} replace />;
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);
    try {
      const s = await login(email, password);
      const from = (location.state as { from?: string } | null)?.from;
      const dest = from ?? (s.role === "operator" ? "/app" : `/app/sites/${s.siteId}`);
      navigate(dest, { replace: true });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setError("Invalid email or password");
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
              Clean Energy Analytics
            </h1>
            <p className="font-body-base text-body-base text-on-surface-variant">
              Welcome back
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
                  placeholder="operator@cleanenergyanalytics.co.uk"
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
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••••••"
                  className="w-full bg-white/70 border border-outline-variant text-on-surface font-body-base text-body-base pl-10 pr-4 py-3 rounded-xl focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-all"
                />
              </div>
            </div>
            {error && (
              <p className="font-label-caps text-label-caps text-error text-center">{error}</p>
            )}
            <div className="text-right">
              <Link to="/forgot-password" className="font-body-base text-body-base text-on-surface-variant hover:text-primary transition-colors">
                Forgot password?
              </Link>
            </div>
            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full bg-primary hover:opacity-90 text-on-primary font-bold py-3.5 px-6 rounded-full flex items-center justify-center gap-2 transition-all disabled:opacity-70 shadow-soft"
            >
              {isSubmitting ? (
                <>
                  <Loader2 size={20} className="animate-spin" />
                  <span>Signing in…</span>
                </>
              ) : (
                <>
                  <span>Sign In</span>
                  <LogIn size={20} />
                </>
              )}
            </button>
          </form>
        </div>
      </main>
    </div>
  );
}
