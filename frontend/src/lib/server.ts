import "server-only";
import { cookies } from "next/headers";
import type { LibrarySummary, Me, Stats, Station, SystemStatus } from "./types";

export const SESSION_COOKIE = "chilly_session";

function backendUrl() {
  return process.env.BACKEND_URL ?? "http://localhost:8080";
}

async function backend<T>(path: string, init: RequestInit = {}): Promise<T | null> {
  try {
    const res = await fetch(`${backendUrl()}/api/v1${path}`, {
      ...init,
      cache: "no-store",
      signal: AbortSignal.timeout(5000),
    });
    if (!res.ok) {
      return null;
    }
    return (await res.json()) as T;
  } catch {
    return null;
  }
}

export function getStats() {
  return backend<Stats>("/stats");
}

export function getStatus() {
  return backend<SystemStatus>("/status");
}

export async function getStations() {
  const res = await backend<{ stations: Station[] }>("/radio/stations");
  return res?.stations ?? null;
}

export function getLibrarySummary() {
  return backend<LibrarySummary>("/library/summary");
}

export async function getSession() {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    return null;
  }
  return backend<Me>("/auth/me", { headers: { Authorization: `Bearer ${token}` } });
}

export function inviteUrl(clientId: string | undefined) {
  if (!clientId) {
    return null;
  }
  const params = new URLSearchParams({
    client_id: clientId,
    scope: "bot applications.commands",
    permissions: "37047296",
  });
  return `https://discord.com/oauth2/authorize?${params}`;
}
