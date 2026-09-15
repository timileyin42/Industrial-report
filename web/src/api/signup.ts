import { apiRequest } from "./client";
import { SignupResponseSchema, type SignupResponse } from "./types";

// Public self-service account creation — distinct from api/users.ts's
// operator-only createUser (an operator adding someone on their own
// side). See internal/registry/signup.go.
export async function signup(email: string, password: string): Promise<SignupResponse> {
  const data = await apiRequest<unknown>("/v1/signup", {
    method: "POST",
    body: { email, password },
    skipAuthRedirect: true, // no prior session to invalidate, same as login
  });
  return SignupResponseSchema.parse(data);
}
