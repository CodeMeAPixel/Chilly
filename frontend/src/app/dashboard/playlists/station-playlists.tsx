"use client";

import { useQuery } from "@tanstack/react-query";
import { ChevronDown, Loader2, Radio } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { ServerPicker } from "@/components/server-picker";
import { TrackArt } from "@/components/track-art";
import { Card, Skeleton } from "@/components/ui";
import { api, ApiError, json } from "@/lib/api";
import { cn, formatDuration } from "@/lib/format";
import type { LibraryPlaylist, SearchResult } from "@/lib/types";

export function StationPlaylists() {
  const { data: playlists, isLoading, error } = useQuery({
    queryKey: ["library", "playlists"],
    queryFn: async () => (await api<{ playlists: LibraryPlaylist[] }>("/library/playlists")).playlists,
    staleTime: 5 * 60_000,
  });

  if (error) {
    return null;
  }

  return (
    <section className="space-y-4">
      <div className="space-y-1">
        <h2 className="flex items-center gap-2 font-display text-xl font-semibold">
          <Radio className="h-5 w-5 text-accent" /> From the stations
        </h2>
        <p className="text-sm text-muted">The playlists our stations play from. Queue one privately in your server.</p>
      </div>
      {isLoading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-28 rounded-3xl" />
          ))}
        </div>
      ) : !playlists?.length ? (
        <p className="text-sm text-muted">No station playlists yet.</p>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {playlists.map((playlist) => (
            <StationPlaylistCard key={playlist.name} playlist={playlist} />
          ))}
        </div>
      )}
    </section>
  );
}

function StationPlaylistCard({ playlist }: { playlist: LibraryPlaylist }) {
  const [open, setOpen] = useState(false);
  const { data: tracks, isLoading } = useQuery({
    queryKey: ["library", "playlist", playlist.name],
    queryFn: async () =>
      (await api<{ tracks: SearchResult[] }>(`/library/playlists/tracks?${new URLSearchParams({ name: playlist.name })}`)).tracks,
    enabled: open,
    staleTime: 5 * 60_000,
  });

  const queue = async (guildId: string, guildName: string) => {
    try {
      const res = await api<{ added: number }>(`/guilds/${guildId}/queue`, {
        method: "POST",
        body: json({ query: playlist.name, type: "playlist", shuffle: true }),
      });
      toast.success(`Queued ${res.added} songs from ${playlist.name} in ${guildName}`);
      return true;
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Couldn't queue the playlist");
      return false;
    }
  };

  return (
    <Card className="flex flex-col">
      <div className="flex gap-4 p-4">
        {playlist.art ? (
          <img src={playlist.art} alt="" className="h-16 w-16 shrink-0 rounded-2xl object-cover" />
        ) : (
          <div className="grid h-16 w-16 shrink-0 place-items-center rounded-2xl bg-accent-soft text-accent">
            <Radio className="h-6 w-6" />
          </div>
        )}
        <div className="min-w-0 flex-1">
          <p className="truncate font-medium">{playlist.name}</p>
          <p className="text-sm text-muted">
            {playlist.track_count} song{playlist.track_count === 1 ? "" : "s"}
          </p>
          {playlist.artists.length > 0 && <p className="truncate text-xs text-muted">{playlist.artists.join(", ")}</p>}
        </div>
      </div>
      <div className="mt-auto flex items-center justify-between gap-2 border-t border-border px-2 py-1.5">
        <button
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          className="inline-flex h-8 cursor-pointer items-center gap-1 rounded-lg px-3 text-sm text-muted hover:bg-surface-2 hover:text-fg"
        >
          Songs <ChevronDown className={cn("h-4 w-4 transition", open && "rotate-180")} />
        </button>
        <ServerPicker onPick={queue} variant="ghost" size="sm" label="Play in server" />
      </div>
      {open && (
        <div className="max-h-72 overflow-y-auto border-t border-border">
          {isLoading ? (
            <div className="flex justify-center p-6 text-muted">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : (
            <ol className="divide-y divide-border">
              {(tracks ?? []).map((t, i) => (
                <li key={t.value} className="flex items-center gap-3 px-4 py-2 text-sm">
                  <span className="w-5 text-right font-mono text-xs text-muted">{i + 1}</span>
                  <TrackArt track={t} className="h-8 w-8 rounded-md" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate">{t.title}</p>
                    <p className="truncate text-xs text-muted">{t.author}</p>
                  </div>
                  <span className="font-mono text-xs text-muted">{formatDuration(t.length_ms)}</span>
                </li>
              ))}
            </ol>
          )}
        </div>
      )}
    </Card>
  );
}
