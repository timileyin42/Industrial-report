import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams, Link } from "react-router-dom";
import { KeyRound, LogIn } from "lucide-react";
import { confirmPasswordReset } from "../api/passwordReset";
import { ApiError } from "../api/types";
import { LogoMark } from "../components/brand/Logo";

export function ResetPasswordPage() {
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
      await confirmPasswordReset(token, password);
      navigate("/login", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Couldn't reset the password. The link may have expired.");
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
              New Password
            </h1>
          </div>
          {!token ? (
            <p className="font-body-base text-error text-center">
              Missing reset token. Use the link from your reset email, or{" "}
              <Link to="/forgot-password" className="text-primary underline">
                request a new one
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
              {error && <p className="font-label-caps text-label-caps text-error text-center">{error}</p>}
              <button
                type="submit"
                disabled={isSubmitting}
                className="w-full bg-primary hover:opacity-90 text-on-primary font-bold py-3.5 px-6 rounded-full flex items-center justify-center gap-2 transition-all disabled:opacity-70 shadow-soft"
              >
                <span>{isSubmitting ? "Saving…" : "Set Password & Sign In"}</span>
                <LogIn size={20} />
              </button>
            </form>
          )}
        </div>
      </main>
    </div>
  );
}
