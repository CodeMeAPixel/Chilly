"use client";

import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { api, ApiError, json } from "@/lib/api";
import type { LoopMode, PlayerState } from "@/lib/types";

type Status = "loading" | "live" | "reconnecting" | "error";

export function usePlayer(guildId: string) {
  const [snapshot, setSnapshot] = useState<{ state: PlayerState; receivedAt: number } | null>(null);
  const [status, setStatus] = useState<Status>("loading");
  const [error, setError] = useState<ApiError | null>(null);
  const state = snapshot?.state ?? null;

  const apply = useCallback((next: PlayerState) => {
    setSnapshot({ state: next, receivedAt: Date.now() });
  }, []);

  useEffect(() => {
    let cancelled = false;
    let source: EventSource | null = null;

    api<PlayerState>(`/guilds/${guildId}/player?queue_limit=200`)
      .then((initial) => {
        if (cancelled) return;
        apply(initial);
        source = new EventSource(`/api/v1/guilds/${guildId}/player/events?queue_limit=200`);
        source.addEventListener("player", (event) => {
          apply(JSON.parse((event as MessageEvent).data));
          setStatus("live");
        });
        source.addEventListener("session_expired", () => {
          source?.close();
          setStatus("error");
          setError(new ApiError(401, "unauthorized", "Your session expired. Please log in again."));
        });
        source.onopen = () => setStatus("live");
        source.onerror = () => setStatus("reconnecting");
      })
      .catch((err: ApiError) => {
        if (cancelled) return;
        setError(err);
        setStatus("error");
      });

    return () => {
      cancelled = true;
      source?.close();
    };
  }, [guildId, apply]);

  const run = useCallback(
    async (request: Promise<PlayerState | void>, success?: string) => {
      try {
        const next = await request;
        if (next) apply(next);
        if (success) toast.success(success);
        return true;
      } catch (err) {
        toast.error(err instanceof ApiError ? err.message : "Something went wrong");
        return false;
      }
    },
    [apply],
  );

  const base = `/guilds/${guildId}`;
  const update = (body: Partial<{ paused: boolean; loop: LoopMode; shuffle: boolean; volume: number; position_ms: number }>) =>
    run(api<PlayerState>(`${base}/player?queue_limit=200`, { method: "PATCH", body: json(body) }));

  const actions = {
    togglePause: () => state && update({ paused: !state.paused }),
    setLoop: (loop: LoopMode) => update({ loop }),
    toggleShuffle: () => state && update({ shuffle: !state.shuffle }),
    setVolume: (volume: number) => update({ volume }),
    seek: (position_ms: number) => update({ position_ms }),
    skip: () => run(api<PlayerState>(`${base}/player/skip?queue_limit=200`, { method: "POST" })),
    previous: () => run(api<PlayerState>(`${base}/player/previous?queue_limit=200`, { method: "POST" })),
    stop: () => run(api<PlayerState>(`${base}/player/stop?queue_limit=200`, { method: "POST" }), "Stopped and cleared the queue"),
    clearQueue: () => run(api<PlayerState>(`${base}/queue?queue_limit=200`, { method: "DELETE" }), "Queue cleared"),
    remove: (index: number) => run(api<PlayerState>(`${base}/queue/${index}?queue_limit=200`, { method: "DELETE" })),
    move: (from: number, to: number) =>
      run(api<PlayerState>(`${base}/queue/move?queue_limit=200`, { method: "POST", body: json({ from, to }) })),
    enqueue: async (body: { query?: string; playlist_id?: number; next?: boolean; play_now?: boolean; shuffle?: boolean }) => {
      try {
        const res = await api<{ added: number; player: PlayerState }>(`${base}/queue?queue_limit=200`, {
          method: "POST",
          body: json(body),
        });
        apply(res.player);
        toast.success(res.added === 1 ? "Added to the queue" : `Added ${res.added} tracks`);
        return true;
      } catch (err) {
        toast.error(err instanceof ApiError ? err.message : "Couldn't add that");
        return false;
      }
    },
  };

  return { state, status, error, receivedAt: snapshot?.receivedAt ?? 0, actions };
}

export function useLivePosition(state: PlayerState | null, receivedAt: number) {
  const [now, setNow] = useState(() => Date.now());
  const ticking = !!state?.current && !state.paused;

  useEffect(() => {
    if (!ticking) return;
    const id = setInterval(() => setNow(Date.now()), 500);
    return () => clearInterval(id);
  }, [ticking]);

  if (!state?.current) {
    return 0;
  }
  const elapsed = ticking ? Math.max(0, now - receivedAt) : 0;
  const position = state.position_ms + elapsed;
  return state.current.is_stream ? position : Math.min(position, state.current.length_ms);
}
