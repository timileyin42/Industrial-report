import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams, Link } from "react-router-dom";
import { KeyRound, LogIn } from "lucide-react";
import { acceptInvite } from "../api/users";
import { ApiError } from "../api/types";
import { LogoMark } from "../components/brand/Logo";

// Public — the invitee has no session yet, only the token from their
// email link (see internal/httpapi/invite_handlers.go's acceptInvite).
export function AcceptInvitePage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const token = params.get("token") ?? "";
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (password !== confirm) {
      setError("Passwords don't match");
      return;
    }
    setIsSubmitting(true);
    try {
      await acceptInvite({ token, password });
      navigate("/login", { replace: true, state: { justAcceptedInvite: true } });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Couldn't accept the invite. The link may have expired.");
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
              Set Your Password
            </h1>
            <p className="font-label-caps text-label-caps text-on-surface-variant uppercase tracking-widest opacity-80">
              Accept Invite
            </p>
          </div>
          {!token ? (
            <p className="font-body-base text-error text-center">
              Missing invite token. Use the link from your invite email, or{" "}
              <Link to="/login" className="text-primary underline">
                sign in
              </Link>
              .
            </p>
          ) : (
            <form className="w-full space-y-6" onSubmit={handleSubmit}>
              <div className="space-y-2">
                <label className="font-label-caps text-label-caps text-on-surface-variant uppercase" htmlFor="password">
                  New Password
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
                    className="w-full bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm pl-10 pr-4 py-3 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all"
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
                    className="w-full bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm pl-10 pr-4 py-3 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all"
                  />
                </div>
              </div>
              {error && <p className="font-label-caps text-label-caps text-error text-center">{error}</p>}
              <button
                type="submit"
                disabled={isSubmitting}
                className="w-full bg-primary-container hover:bg-on-primary-fixed-variant text-on-primary-container font-bold py-4 px-6 rounded flex items-center justify-center gap-2 border border-primary/20 transition-all disabled:opacity-70"
              >
                <span className="font-label-caps text-label-caps uppercase tracking-wider">
                  {isSubmitting ? "Setting password..." : "Set Password & Sign In"}
                </span>
                <LogIn size={20} />
              </button>
            </form>
          )}
        </div>
      </main>
    </div>
  );
}
