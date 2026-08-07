import { apiRequest } from "./client";
import { LoginResponseSchema, type LoginResponse } from "./types";

export async function login(email: string, password: string): Promise<LoginResponse> {
  const data = await apiRequest<unknown>("/v1/auth/login", {
    method: "POST",
    body: { email, password },
    skipAuthRedirect: true, // a failed login is inline, not a session-expired redirect
  });
  return LoginResponseSchema.parse(data);
}
