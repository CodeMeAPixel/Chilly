"use client";

import { useQuery } from "@tanstack/react-query";
import { Disc3, Link2, ListMusic, ListPlus, Loader2, Play, Plus, Search } from "lucide-react";
import { useEffect, useState } from "react";
import { TrackArt } from "@/components/track-art";
import { Button, Card, EmptyState, Input } from "@/components/ui";
import type { usePlayer } from "@/hooks/use-player";
import { api } from "@/lib/api";
import { cn, formatDuration } from "@/lib/format";
import type { Playlist, SearchResult } from "@/lib/types";

type Actions = ReturnType<typeof usePlayer>["actions"];

const isUrl = (value: string) => /^https?:\/\//i.test(value.trim());

function useDebounced<T>(value: T, delay: number) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(id);
  }, [value, delay]);
  return debounced;
}

export function AddTracks({ canAdd, actions }: { canAdd: boolean; actions: Actions }) {
  const [tab, setTab] = useState<"search" | "playlists">("search");

  return (
    <Card className="flex flex-col overflow-hidden">
      <div className="flex gap-1 border-b border-border p-3">
        {(["search", "playlists"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={cn(
              "flex flex-1 items-center justify-center gap-1.5 rounded-xl py-2 text-sm capitalize transition cursor-pointer",
              tab === t ? "bg-primary-soft text-fg" : "text-muted hover:bg-surface-2",
            )}
          >
            {t === "search" ? <Search className="h-4 w-4" /> : <ListMusic className="h-4 w-4" />} {t}
          </button>
        ))}
      </div>
      {!canAdd && (
        <p className="border-b border-border bg-surface-2/60 px-4 py-2.5 text-xs text-muted">
          Join a voice channel in this server to add music.
        </p>
      )}
      {tab === "search" ? <SearchPanel canAdd={canAdd} actions={actions} /> : <PlaylistPanel canAdd={canAdd} actions={actions} />}
    </Card>
  );
}

function SearchPanel({ canAdd, actions }: { canAdd: boolean; actions: Actions }) {
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const debounced = useDebounced(query.trim(), 400);
  const url = isUrl(debounced);

  const { data: results, isFetching, error } = useQuery({
    queryKey: ["search", debounced],
    queryFn: async () => {
      const params = new URLSearchParams({ q: debounced, limit: "20" });
      return (await api<{ results: SearchResult[] }>(`/search?${params}`)).results;
    },
    enabled: debounced.length > 1 && !url,
    staleTime: 60_000,
  });

  const add = async (key: string, body: Parameters<Actions["enqueue"]>[0]) => {
    setBusy(key);
    await actions.enqueue(body);
    setBusy(null);
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="space-y-3 p-4">
        <div className="relative">
          <Search className="pointer-events-none absolute top-1/2 left-3.5 h-4 w-4 -translate-y-1/2 text-muted" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search the library by song, artist or album"
            className="pl-10"
          />
        </div>
      </div>

      <div className="max-h-[480px] flex-1 overflow-y-auto border-t border-border">
        {url ? (
          <EmptyState icon={<Link2 className="h-6 w-6" />} title="Links aren't supported">
            Chilly plays songs from its own library. Search by song, artist or album name instead.
          </EmptyState>
        ) : debounced.length < 2 ? (
          <EmptyState icon={<Search className="h-6 w-6" />} title="Find something to play">
            Search Chilly&apos;s library by song, artist or album.
          </EmptyState>
        ) : isFetching && !results ? (
          <div className="flex justify-center p-10 text-muted">
            <Loader2 className="h-5 w-5 animate-spin" />
          </div>
        ) : error ? (
          <EmptyState title="Search failed">{error.message}</EmptyState>
        ) : !results?.length ? (
          <EmptyState title="Not in the library">Try a different search.</EmptyState>
        ) : (
          <ul className="divide-y divide-border">
            {results.map((result) => (
              <li key={result.value} className="group flex items-center gap-3 px-4 py-2.5">
                <TrackArt track={result} className="h-11 w-11 rounded-lg" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{result.title}</p>
                  <p className="truncate text-xs text-muted">
                    {[result.author, result.album].filter(Boolean).join(" · ")} · {formatDuration(result.length_ms)}
                  </p>
                </div>
                <div className="flex gap-0.5">
                  {result.album && (
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      aria-label={`Queue the album ${result.album}`}
                      title={`Queue the album ${result.album}`}
                      disabled={!canAdd || busy !== null}
                      onClick={() => add(`${result.value}-album`, { query: result.album!, type: "album" })}
                    >
                      {busy === `${result.value}-album` ? <Loader2 className="h-4 w-4 animate-spin" /> : <Disc3 className="h-4 w-4" />}
                    </Button>
                  )}
                  <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="Play now" disabled={!canAdd || busy !== null} onClick={() => add(`${result.value}-now`, { query: result.value, play_now: true })}>
                    <Play className="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="Play next" disabled={!canAdd || busy !== null} onClick={() => add(`${result.value}-next`, { query: result.value, next: true })}>
                    <ListPlus className="h-4 w-4" />
                  </Button>
                  <Button variant="secondary" size="icon" className="h-8 w-8" aria-label="Add to queue" disabled={!canAdd || busy !== null} onClick={() => add(result.value, { query: result.value })}>
                    {busy === result.value ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function PlaylistPanel({ canAdd, actions }: { canAdd: boolean; actions: Actions }) {
  const [busy, setBusy] = useState<number | null>(null);
  const { data: playlists, isLoading } = useQuery({
    queryKey: ["playlists"],
    queryFn: async () => (await api<{ playlists: Playlist[] }>("/playlists")).playlists,
  });

  if (isLoading) {
    return (
      <div className="flex justify-center p-10 text-muted">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    );
  }

  if (!playlists?.length) {
    return (
      <EmptyState icon={<ListMusic className="h-6 w-6" />} title="No playlists yet">
        Create one on the Playlists page or with <code>/list create</code>.
      </EmptyState>
    );
  }

  return (
    <ul className="max-h-[520px] divide-y divide-border overflow-y-auto">
      {playlists.map((playlist) => (
        <li key={playlist.id} className="flex items-center gap-3 px-4 py-3">
          <div className="grid h-11 w-11 shrink-0 place-items-center rounded-lg bg-accent-soft text-accent">
            <ListMusic className="h-5 w-5" />
          </div>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium">{playlist.name}</p>
            <p className="text-xs text-muted">{playlist.track_count} tracks</p>
          </div>
          <Button
            size="sm"
            variant="secondary"
            disabled={!canAdd || busy !== null || playlist.track_count === 0}
            onClick={async () => {
              setBusy(playlist.id);
              await actions.enqueue({ playlist_id: playlist.id, shuffle: true });
              setBusy(null);
            }}
          >
            {busy === playlist.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} Queue
          </Button>
        </li>
      ))}
    </ul>
  );
}
