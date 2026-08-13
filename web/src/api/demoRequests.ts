import { ApiError } from "./types";

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "http://localhost:8080";

// Public, no-login endpoint — same reasoning as api/sandbox.ts for not
// using client.ts's apiRequest wrapper (no Authorization header to
// inject, no 401 redirect that makes sense here).
export async function submitDemoRequest(organization: string, email: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/v1/demo-requests`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ organization, email }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => undefined);
    throw new ApiError(res.status, body?.message ?? res.statusText, body);
  }
}
