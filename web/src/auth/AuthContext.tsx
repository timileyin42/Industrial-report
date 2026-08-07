import { createContext, useContext, useMemo, useState, type ReactNode } from "react";
import { configureApiClient } from "../api/client";
import { login as loginRequest } from "../api/auth";
import type { Role } from "../api/types";

interface Session {
  token: string;
  expiresAt: string;
  role: Role;
  siteId?: string | null;
}

interface AuthState {
  session: Session | null;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<Session>;
  logout: () => void;
}

const STORAGE_KEY = "cea.session";

// Token storage: in-memory (this context) mirrored to sessionStorage —
// not localStorage (larger XSS exposure window, persists past tab close)
// and not an httpOnly cookie (the backend has no cookie/refresh-token
// support to pair with one). sessionStorage only survives a page refresh
// within the same tab/session. See Frontend Slice 1 plan.
function readStoredSession(): Session | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Session;
    if (new Date(parsed.expiresAt).getTime() <= Date.now()) {
      sessionStorage.removeItem(STORAGE_KEY);
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  // Hydrate synchronously from sessionStorage as the initial state, rather
  // than starting null and hydrating in an effect — that earlier approach
  // left a window where children could mount and fire their first API
  // call before configureApiClient below had been updated with the real
  // token (effects run bottom-up, so a child's mount effect can run before
  // this provider's own effect on the same commit), producing a stray 401
  // on first load. Reading synchronously here removes the window entirely.
  const [session, setSession] = useState<Session | null>(() => readStoredSession());

  // Bridge: apiRequest happens outside React render, so it needs a plain
  // getter/callback rather than useContext. Called directly during render
  // (not in useEffect) so it's always in sync with `session` before any
  // child below renders/fires a request — configureApiClient only
  // reassigns closures, no subscriptions or side effects, so this is safe
  // to do on every render.
  configureApiClient({
    getToken: () => session?.token ?? null,
    onUnauthorized: () => {
      sessionStorage.removeItem(STORAGE_KEY);
      setSession(null);
    },
  });

  const value = useMemo<AuthState>(
    () => ({
      session,
      isLoading: false,
      login: async (email, password) => {
        const res = await loginRequest(email, password);
        const next: Session = {
          token: res.token,
          expiresAt: res.expires_at,
          role: res.role,
          siteId: res.site_id ?? null,
        };
        sessionStorage.setItem(STORAGE_KEY, JSON.stringify(next));
        setSession(next);
        return next;
      },
      logout: () => {
        sessionStorage.removeItem(STORAGE_KEY);
        setSession(null);
      },
    }),
    [session]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
