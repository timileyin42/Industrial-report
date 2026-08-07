import { useState, type FormEvent } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import { Mail, KeyRound, LogIn, Loader2 } from "lucide-react";
import { useAuth } from "../auth/AuthContext";
import { ApiError } from "../api/types";
import { LogoMark } from "../components/brand/Logo";

// Reference: design/login_zgnis_industrial_intelligence/code.html.
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
    // <Navigate>, not an imperative navigate() call — calling navigate()
    // directly in a render body updates the router while LoginPage itself
    // is still rendering, which React warns about ("Cannot update a
    // component while rendering a different component").
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
      // operator lands on Fleet Dashboard; restricted goes straight to
      // their own site — sending them to Fleet Dashboard first just to
      // bounce off a 403 is a bad first impression.
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
    <div className="min-h-screen flex items-center justify-center bg-background text-on-surface">
      <main className="w-full max-w-[400px] px-6">
        <div className="bg-surface-container border border-outline-variant rounded-lg p-10 flex flex-col items-center">
          <div className="mb-10 text-center flex flex-col items-center">
            <Link to="/" aria-label="Back to home">
              <LogoMark size={32} />
            </Link>
            <h1 className="font-headline-lg text-headline-lg font-bold text-primary tracking-tight mb-1 mt-3">
              Clean Energy Analytics
            </h1>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase tracking-widest opacity-80">
              Industrial Intelligence
            </p>
          </div>
          <form className="w-full space-y-6" onSubmit={handleSubmit}>
            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="email">
                Terminal Access Email
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
                  className="w-full bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm pl-10 pr-4 py-3 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all"
                />
              </div>
            </div>
            <div className="space-y-2">
              <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="password">
                Secure Credentials
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
                  className="w-full bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm pl-10 pr-4 py-3 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all"
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
              className="w-full bg-primary-container hover:bg-on-primary-fixed-variant text-on-primary-container font-bold py-4 px-6 rounded flex items-center justify-center gap-2 border border-primary/20 transition-all disabled:opacity-70"
            >
              {isSubmitting ? (
                <>
                  <Loader2 size={20} className="animate-spin" />
                  <span className="font-label-caps text-label-caps uppercase tracking-wider">Authenticating...</span>
                </>
              ) : (
                <>
                  <span className="font-label-caps text-label-caps uppercase tracking-wider">Initialize Session</span>
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
