import { useEffect, useRef, useState } from "react";
import { UserCircle, ChevronDown, LogOut } from "lucide-react";
import { useAuth } from "../../auth/AuthContext";

// A real dropdown, not a static chip — but only ever shows what's
// actually known about the session (role, and for a restricted account,
// the one site it's scoped to). There's no display-name field anywhere
// in this platform's user model (just email + password_hash + role), so
// this never fabricates a name the way the reference mockup's "John
// Operator" does.
export function ProfileMenu() {
  const { session, logout } = useAuth();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  return (
    <div className="relative" ref={containerRef}>
      <button
        onClick={() => setOpen((v) => !v)}
        className="glass-card rounded-full flex items-center gap-3 pl-4 pr-2 py-2"
      >
        <div className="text-right">
          <p className="font-label-caps text-label-caps text-on-surface leading-none">
            {session?.role === "operator" ? "Operator" : `Site: ${session?.siteId ?? ""}`}
          </p>
          <p className="text-[10px] text-primary uppercase font-bold mt-1">
            {session?.role === "operator" ? "Full Access" : "Restricted"}
          </p>
        </div>
        <UserCircle size={32} className="text-on-surface-variant" />
        <ChevronDown size={14} className={`text-on-surface-variant transition-transform ${open ? "rotate-180" : ""}`} />
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-2 w-56 glass-card rounded-xl overflow-hidden z-50">
          {session?.email && (
            <div className="px-4 py-3 border-b border-outline-variant/60">
              <p className="text-[13px] text-on-surface truncate">{session.email}</p>
            </div>
          )}
          <button
            onClick={logout}
            className="w-full flex items-center gap-2 px-4 py-3 text-[13px] text-on-surface-variant hover:bg-white/50 hover:text-on-surface transition-colors"
          >
            <LogOut size={16} />
            <span>Logout</span>
          </button>
        </div>
      )}
    </div>
  );
}
