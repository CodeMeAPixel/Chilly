export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(`/api/v1${path}`, {
    ...init,
    headers,
    credentials: "same-origin",
    cache: "no-store",
  });

  if (res.status === 204) {
    return undefined as T;
  }

  const body = await res.json().catch(() => null);
  if (!res.ok) {
    const error = body?.error;
    throw new ApiError(res.status, error?.code ?? "unknown", error?.message ?? res.statusText);
  }
  return body as T;
}

export const json = (value: unknown) => JSON.stringify(value);

export function loginUrl(redirect = "/dashboard") {
  return `/api/v1/auth/login?redirect=${encodeURIComponent(redirect)}`;
}
