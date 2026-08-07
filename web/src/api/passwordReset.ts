import { apiRequest } from "./client";

// Both public — no session exists yet for either step.
export async function requestPasswordReset(email: string): Promise<void> {
  await apiRequest<unknown>("/v1/auth/password-reset/request", { method: "POST", body: { email }, skipAuthRedirect: true });
}

export async function confirmPasswordReset(token: string, password: string): Promise<void> {
  await apiRequest<unknown>("/v1/auth/password-reset/confirm", { method: "POST", body: { token, password }, skipAuthRedirect: true });
}
