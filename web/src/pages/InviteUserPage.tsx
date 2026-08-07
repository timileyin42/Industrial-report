import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { TopNav } from "../components/layout/TopNav";
import { inviteUser } from "../api/users";
import { listSites } from "../api/sites";
import { useQuery } from "@tanstack/react-query";
import { ApiError, type Role } from "../api/types";

// No canonical screen exists in design/ for this — built as a minimal
// form reusing AddSitePage's card/input styling rather than inventing a
// new visual language, per CLAUDE.md rule 6.
export function InviteUserPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<Role>("restricted");
  const [siteId, setSiteId] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [sent, setSent] = useState(false);

  const sitesQuery = useQuery({ queryKey: ["sites-for-invite"], queryFn: () => listSites(undefined, 200) });

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);
    try {
      await inviteUser({ email, role, site_id: role === "restricted" ? siteId : undefined });
      setSent(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to send invite");
    } finally {
      setIsSubmitting(false);
    }
  }

  const inputClass =
    "w-full bg-white/70 border border-outline-variant text-on-surface font-body-base rounded-xl py-2.5 px-4 focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none placeholder:text-on-surface-variant/50";
  const labelClass = "block text-label-caps font-label-caps text-on-surface-variant mb-2";

  if (sent) {
    return (
      <>
        <TopNav title="Invite User" />
        <div className="flex-1 p-grid-margin max-w-lg">
          <div className="glass-card rounded-2xl p-6 space-y-4">
            <p className="font-body-base text-on-surface">
              Invite sent to <span className="font-semibold text-primary">{email}</span>. They'll
              receive an email with a link to set their password.
            </p>
            <button
              onClick={() => navigate("/app")}
              className="bg-primary hover:opacity-90 text-on-primary font-semibold px-6 py-2.5 rounded-full transition-all shadow-soft"
            >
              Done
            </button>
          </div>
        </div>
      </>
    );
  }

  return (
    <>
      <TopNav title="Invite User" />
      <div className="flex-1 p-grid-margin max-w-lg">
        <form className="glass-card rounded-2xl p-6 space-y-5" onSubmit={handleSubmit}>
          <div>
            <label className={labelClass}>EMAIL</label>
            <input
              type="email"
              className={inputClass}
              placeholder="name@example.com"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
          <div>
            <label className={labelClass}>ROLE</label>
            <select className={inputClass} value={role} onChange={(e) => setRole(e.target.value as Role)}>
              <option value="restricted">Restricted (one site)</option>
              <option value="operator">Operator (full access)</option>
            </select>
          </div>
          {role === "restricted" && (
            <div>
              <label className={labelClass}>SITE</label>
              <select className={inputClass} required value={siteId} onChange={(e) => setSiteId(e.target.value)}>
                <option value="" disabled>
                  Select a site…
                </option>
                {(sitesQuery.data?.items ?? []).map((s) => (
                  <option key={s.site_id} value={s.site_id}>
                    {s.name ?? s.site_id}
                  </option>
                ))}
              </select>
            </div>
          )}

          {error && <p className="font-label-caps text-label-caps text-error">{error}</p>}

          <div className="flex items-center justify-end gap-4 pt-2">
            <button
              type="button"
              onClick={() => navigate(-1)}
              className="px-8 py-2.5 glass-card rounded-full text-on-surface hover:text-primary transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="px-10 py-2.5 bg-primary hover:opacity-90 text-on-primary font-semibold rounded-full transition-all disabled:opacity-70 shadow-soft"
            >
              {isSubmitting ? "Sending…" : "Send Invite"}
            </button>
          </div>
        </form>
      </div>
    </>
  );
}
