import { ApiError } from "./types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

// Module-level token getter/clear-callback, set once by AuthContext on
// mount. fetch calls happen outside React render, so they can't use
// useContext directly — this is the bridge between the two.
let getToken: () => string | null = () => null;
let onUnauthorized: () => void = () => {};

export function configureApiClient(opts: {
  getToken: () => string | null;
  onUnauthorized: () => void;
}) {
  getToken = opts.getToken;
  onUnauthorized = opts.onUnauthorized;
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  query?: Record<string, string | number | undefined>;
  // Login is the one endpoint where a 401 is expected/inline, not a
  // reason to fire the global "session expired" redirect (see plan:
  // "there's no prior session to invalidate").
  skipAuthRedirect?: boolean;
}

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  const url = new URL(path, API_BASE_URL);
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== "") url.searchParams.set(k, String(v));
    }
  }
  return url.toString();
}

export async function apiRequest<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {};
  const token = getToken();
  if (token) headers["Authorization"] = `Bearer ${token}`;
  if (opts.body !== undefined) headers["Content-Type"] = "application/json";

  const res = await fetch(buildUrl(path, opts.query), {
    method: opts.method ?? "GET",
    headers,
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
  });

  if (res.status === 401 && !opts.skipAuthRedirect) {
    onUnauthorized();
    throw new ApiError(401, "Session expired");
  }

  const contentType = res.headers.get("content-type") ?? "";
  const isJson = contentType.includes("application/json");

  if (!res.ok) {
    let message = res.statusText;
    let body: unknown;
    if (isJson) {
      body = await res.json().catch(() => undefined);
      if (body && typeof body === "object" && "message" in body) {
        message = String((body as { message: unknown }).message);
      }
    }
    throw new ApiError(res.status, message, body);
  }

  if (!isJson) {
    // e.g. CSV export — callers that need raw text/blob use apiRequestRaw
    return undefined as T;
  }
  return (await res.json()) as T;
}

export async function apiRequestRaw(path: string, opts: RequestOptions = {}): Promise<Response> {
  const headers: Record<string, string> = {};
  const token = getToken();
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(buildUrl(path, opts.query), {
    method: opts.method ?? "GET",
    headers,
  });
  if (res.status === 401 && !opts.skipAuthRedirect) {
    onUnauthorized();
    throw new ApiError(401, "Session expired");
  }
  if (!res.ok) {
    throw new ApiError(res.status, res.statusText);
  }
  return res;
}
