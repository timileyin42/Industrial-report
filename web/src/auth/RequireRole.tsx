import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "./AuthContext";
import type { Role } from "../api/types";

// This guard is a UX nicety, not the actual security boundary — the API's
// 403 is. A restricted user is redirected to their own site rather than
// being let through to see AccessDenied, since there's no legitimate
// reason for them to even discover this route exists.
export function RequireRole({ role }: { role: Role }) {
  const { session } = useAuth();

  if (!session) return <Navigate to="/login" replace />;

  if (session.role !== role) {
    if (session.siteId) return <Navigate to={`/app/sites/${session.siteId}`} replace />;
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
