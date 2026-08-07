import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { Mail } from "lucide-react";
import { requestPasswordReset } from "../api/passwordReset";
import { LogoMark } from "../components/brand/Logo";

// Always shows the same "check your email" outcome regardless of whether
// the address matched a user — mirrors the backend's PasswordReset.Request,
// which never reveals account existence via a different response.
export function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [sent, setSent] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await requestPasswordReset(email);
    } finally {
      setIsSubmitting(false);
      setSent(true);
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
              Reset Password
            </h1>
          </div>
          {sent ? (
            <p className="font-body-base text-on-surface text-center">
              If an account exists for that email, a reset link is on its way. Check your inbox.
            </p>
          ) : (
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
                    className="w-full bg-background border border-outline-variant text-on-surface font-data-mono-sm text-data-mono-sm pl-10 pr-4 py-3 rounded focus:border-primary focus:ring-1 focus:ring-primary outline-none transition-all"
                  />
                </div>
              </div>
              <button
                type="submit"
                disabled={isSubmitting}
                className="w-full bg-primary-container hover:bg-on-primary-fixed-variant text-on-primary-container font-bold py-4 px-6 rounded transition-all disabled:opacity-70"
              >
                <span className="font-label-caps text-label-caps uppercase tracking-wider">
                  {isSubmitting ? "Sending..." : "Send Reset Link"}
                </span>
              </button>
            </form>
          )}
          <Link to="/login" className="mt-6 text-on-surface-variant hover:text-primary font-body-base text-body-base transition-colors">
            Back to sign in
          </Link>
        </div>
      </main>
    </div>
  );
}
