// The one place that talks to the backend. Every request goes to /api/…,
// which Vite (dev) or Vercel (prod) forwards to the Go server. Because it's
// the same origin as the page, the browser sends the session cookie
// automatically; no tokens are handled in JavaScript.

/** Thrown for any non-2xx response; carries the backend's error code. */
export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

type RequestOptions = { params?: URLSearchParams; body?: unknown; headers?: Record<string, string> };

export async function apiRequest<T>(method: "GET" | "POST", path: string, options: RequestOptions = {}) {
  const query = options.params?.toString();
  const headers: Record<string, string> = { ...options.headers };
  // The backend only accepts JSON bodies (see httpx.DecodeJSON).
  if (options.body !== undefined) headers["Content-Type"] = "application/json";

  const res = await fetch(`/api${path}${query ? `?${query}` : ""}`, {
    method,
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  });

  // 204 No Content has no body; a proxy error page might not be JSON.
  const body = res.status === 204 ? null : await res.json().catch(() => null);

  if (!res.ok) {
    // Shared error format: { error: { code, message } }
    throw new ApiError(
      res.status,
      body?.error?.code ?? "unknown",
      body?.error?.message ?? `Request failed (${res.status})`,
    );
  }
  return body as T;
}

export const apiGet = <T>(path: string, params?: URLSearchParams) => apiRequest<T>("GET", path, { params });

export const apiPost = <T>(path: string, body?: unknown) => apiRequest<T>("POST", path, { body });
