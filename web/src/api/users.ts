import { apiRequest } from "./client";
import type { Role } from "./types";

export interface InviteUserInput {
  email: string;
  role: Role;
  site_id?: string;
}

export interface AcceptInviteInput {
  token: string;
  password: string;
}

// Operator-only on the backend (POST /v1/users/invite) — the alternative
// to setting a password directly; the invitee sets their own via
// acceptInvite below.
export async function inviteUser(input: InviteUserInput): Promise<void> {
  await apiRequest<unknown>("/v1/users/invite", { method: "POST", body: input });
}

// Public endpoint — the invitee has no session yet, only the token from
// their email link.
export async function acceptInvite(input: AcceptInviteInput): Promise<void> {
  await apiRequest<unknown>("/v1/invites/accept", { method: "POST", body: input, skipAuthRedirect: true });
}
