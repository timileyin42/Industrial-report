import { apiRequest } from "./client";
import { PageSchema, UserSchema, type Role, type User } from "./types";

export async function listUsers(cursor?: string, limit = 50): Promise<{ items: User[]; nextCursor?: string }> {
  const data = await apiRequest<unknown>("/v1/users", { query: { cursor, limit } });
  const parsed = PageSchema(UserSchema).parse(data);
  return { items: parsed.items, nextCursor: parsed.next_cursor };
}

export async function setUserDisabled(userId: number, disabled: boolean): Promise<User> {
  const data = await apiRequest<unknown>(`/v1/users/${userId}/disabled`, { method: "PATCH", body: { disabled } });
  return UserSchema.parse(data);
}

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
