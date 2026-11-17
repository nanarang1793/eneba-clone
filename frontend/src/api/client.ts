// The one place that talks to the backend. Every request goes to /api/…,
// which Vite (dev) or Vercel (prod) forwards to the Go server.

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

export async function apiGet<T>(path: string, params?: URLSearchParams): Promise<T> {
  const query = params?.toString();
  const res = await fetch(`/api${path}${query ? `?${query}` : ""}`);

  // The backend always answers JSON, but a proxy error page might not be.
  const body = await res.json().catch(() => null);

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
