"use client";

import { useQuery } from "@tanstack/react-query";
import { api, ApiError } from "@/lib/api";
import type { Guild, Me } from "@/lib/types";

export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: async () => {
      try {
        return await api<Me>("/auth/me");
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
          return null;
        }
        throw error;
      }
    },
    staleTime: 60_000,
  });
}

export function useGuilds(enabled = true) {
  return useQuery({
    queryKey: ["guilds"],
    queryFn: async () => (await api<{ guilds: Guild[] }>("/guilds")).guilds,
    enabled,
    refetchInterval: 20_000,
  });
}
