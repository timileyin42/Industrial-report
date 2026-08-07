import { ShieldAlert } from "lucide-react";
import { EmptyState } from "./EmptyState";

// No canonical "403" screen exists anywhere in the Stitch design export —
// flagged back per the plan: a real access-denied screen should exist in
// the design system before Slice 2. Built here as a minimal reuse of
// EmptyState's shell with error-toned iconography, not a new visual
// language. This is the required outcome for a 403 — never EmptyState,
// never a redirect, never a silent empty list.
export function AccessDenied({ detail }: { detail?: string }) {
  return (
    <EmptyState
      icon={<ShieldAlert size={48} className="text-error" />}
      title="Access denied"
      body={detail ?? "You don't have permission to view this. Your account is scoped to a different site."}
    />
  );
}
