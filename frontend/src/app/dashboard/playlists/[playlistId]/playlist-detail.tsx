"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Check, Disc3, Info, ListMusic, Loader2, Pencil, Plus, Search, Trash2, X } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { RequestButton } from "@/components/request-button";
import { ServerPicker } from "@/components/server-picker";
import { TrackArt } from "@/components/track-art";
import { Badge, Button, buttonClass, Card, EmptyState, Input, Skeleton } from "@/components/ui";
import { api, ApiError, json } from "@/lib/api";
import { cn, formatDuration } from "@/lib/format";
import type { Playlist, PlaylistTrack, SearchResult } from "@/lib/types";

type Detail = { playlist: Playlist; tracks: PlaylistTrack[]; available: number };

const errorMessage = (err: unknown, fallback: string) => (err instanceof ApiError ? err.message : fallback);

export function PlaylistDetail({ playlistId }: { playlistId: string }) {
  const queryClient = useQueryClient();
  const router = useRouter();
  const key = ["playlist", playlistId];
  const { data, isLoading, error } = useQuery({
    queryKey: key,
    queryFn: () => api<Detail>(`/playlists/${playlistId}`),
  });

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: key });
    queryClient.invalidateQueries({ queryKey: ["playlists"] });
  };

  const removeTrack = useMutation({
    mutationFn: (trackId: number) => api(`/playlists/${playlistId}/tracks/${trackId}`, { method: "DELETE" }),
    onSuccess: refresh,
    onError: (err) => toast.error(errorMessage(err, "Couldn't remove track")),
  });

  const deletePlaylist = useMutation({
    mutationFn: () => api(`/playlists/${playlistId}`, { method: "DELETE" }),
    onSuccess: () => {
      toast.success("Playlist deleted");
      queryClient.invalidateQueries({ queryKey: ["playlists"] });
      router.push("/dashboard/playlists");
    },
    onError: (err) => toast.error(errorMessage(err, "Couldn't delete playlist")),
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-12 w-72" />
        <Skeleton className="h-96 rounded-3xl" />
      </div>
    );
  }

  if (error || !data) {
    return (
      <Card>
        <EmptyState icon={<ListMusic className="h-6 w-6" />} title="Playlist not found">
          <Link href="/dashboard/playlists" className={buttonClass("secondary", "md", "mt-3")}>Back to playlists</Link>
        </EmptyState>
      </Card>
    );
  }

  const { playlist, tracks, available } = data;
  const missing = tracks.length - available;
  const totalMs = tracks.reduce((sum, t) => sum + (t.available && !t.track.is_stream ? t.track.length_ms : 0), 0);

  return (
    <div className="space-y-6">
      <Link href="/dashboard/playlists" className="inline-flex items-center gap-1.5 text-sm text-muted hover:text-fg">
        <ArrowLeft className="h-4 w-4" /> Playlists
      </Link>

      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="flex items-center gap-5">
          <div className="grid h-24 w-24 place-items-center rounded-3xl bg-linear-to-br from-primary-soft to-accent-soft text-primary">
            <ListMusic className="h-10 w-10" />
          </div>
          <div className="space-y-1">
            <RenameTitle playlist={playlist} onRenamed={refresh} />
            <p className="text-sm text-muted">
              {tracks.length} track{tracks.length === 1 ? "" : "s"} · {formatDuration(totalMs)}
              {missing > 0 && ` · ${available} playable`}
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <ServerPicker onPick={(guildId, name) => queuePlaylist(playlist, guildId, name)} disabled={available === 0} />
          <Button
            variant="danger"
            onClick={() => {
              if (confirm(`Delete "${playlist.name}"? This can't be undone.`)) deletePlaylist.mutate();
            }}
          >
            <Trash2 className="h-4 w-4" /> Delete
          </Button>
        </div>
      </div>

      {missing > 0 && (
        <p className="flex items-start gap-2 rounded-2xl border border-border bg-surface-2/60 px-4 py-3 text-sm text-muted">
          <Info className="mt-0.5 h-4 w-4 shrink-0" />
          {missing === 1 ? "1 song isn't" : `${missing} songs aren't`} in the Chilly library yet, so{" "}
          {missing === 1 ? "it's" : "they're"} skipped when you play this playlist. They start working automatically once they&apos;re
          added to the library.
        </p>
      )}

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
        <Card className="overflow-hidden">
          {tracks.length === 0 ? (
            <EmptyState icon={<ListMusic className="h-6 w-6" />} title="This playlist is empty">
              Add songs with the panel on the right.
            </EmptyState>
          ) : (
            <ol className="divide-y divide-border">
              {tracks.map((item, index) => (
                <li key={item.id} className="group flex items-center gap-3 px-5 py-2.5">
                  <span className="w-6 text-right font-mono text-xs text-muted">{index + 1}</span>
                  <TrackArt track={item.track} className={cn("h-10 w-10 rounded-lg", !item.available && "opacity-40 grayscale")} />
                  <div className={cn("min-w-0 flex-1", !item.available && "opacity-60")}>
                    <p className="truncate text-sm font-medium">{item.track.title}</p>
                    <p className="truncate text-xs text-muted">
                      {[item.track.author, item.album].filter(Boolean).join(" · ")}
                    </p>
                  </div>
                  {!item.available && <Badge className="hidden shrink-0 sm:inline-flex">Not in library yet</Badge>}
                  {item.available && <RequestButton songKey={item.track.identifier} title={item.track.title} compact />}
                  <span className="font-mono text-xs text-muted">
                    {item.track.is_stream ? "LIVE" : formatDuration(item.track.length_ms)}
                  </span>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 hover:text-danger sm:opacity-0 sm:group-hover:opacity-100"
                    aria-label="Remove"
                    disabled={removeTrack.isPending}
                    onClick={() => removeTrack.mutate(item.id)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </li>
              ))}
            </ol>
          )}
        </Card>
        <AddToPlaylist playlistId={playlistId} onAdded={refresh} />
      </div>
    </div>
  );
}

function RenameTitle({ playlist, onRenamed }: { playlist: Playlist; onRenamed: () => void }) {
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(playlist.name);
  const rename = useMutation({
    mutationFn: (value: string) => api(`/playlists/${playlist.id}`, { method: "PATCH", body: json({ name: value }) }),
    onSuccess: () => {
      setEditing(false);
      onRenamed();
    },
    onError: (err) => toast.error(errorMessage(err, "Couldn't rename playlist")),
  });

  if (!editing) {
    return (
      <h1 className="group flex items-center gap-2 font-display text-3xl font-semibold tracking-tight">
        {playlist.name}
        <button onClick={() => setEditing(true)} className="text-muted opacity-0 transition group-hover:opacity-100 hover:text-fg cursor-pointer" aria-label="Rename">
          <Pencil className="h-4 w-4" />
        </button>
      </h1>
    );
  }

  return (
    <form
      className="flex items-center gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        if (name.trim() && name.trim() !== playlist.name) rename.mutate(name.trim());
        else setEditing(false);
      }}
    >
      <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus maxLength={100} className="h-10 w-64" />
      <Button type="submit" size="icon" aria-label="Save" disabled={rename.isPending}>
        <Check className="h-4 w-4" />
      </Button>
      <Button variant="ghost" size="icon" aria-label="Cancel" onClick={() => setEditing(false)}>
        <X className="h-4 w-4" />
      </Button>
    </form>
  );
}

function useDebounced<T>(value: T, delay: number) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(id);
  }, [value, delay]);
  return debounced;
}

function AddToPlaylist({ playlistId, onAdded }: { playlistId: string; onAdded: () => void }) {
  const [query, setQuery] = useState("");
  const debounced = useDebounced(query.trim(), 300);
  const { data: results, isFetching } = useQuery({
    queryKey: ["search", debounced],
    queryFn: async () =>
      (await api<{ results: SearchResult[] }>(`/search?${new URLSearchParams({ q: debounced, limit: "12" })}`)).results,
    enabled: debounced.length > 1,
    staleTime: 60_000,
  });

  const add = useMutation({
    mutationFn: (body: { query: string; type?: "album" }) =>
      api<{ added: number }>(`/playlists/${playlistId}/tracks`, { method: "POST", body: json(body) }),
    onSuccess: (res) => {
      toast.success(res.added === 1 ? "Song added" : `Added ${res.added} songs`);
      onAdded();
    },
    onError: (err) => toast.error(errorMessage(err, "Couldn't add that")),
  });

  return (
    <Card className="flex h-fit flex-col overflow-hidden">
      <div className="space-y-3 p-5 pb-4">
        <h2 className="font-display text-lg font-semibold">Add music</h2>
        <div className="relative">
          <Search className="pointer-events-none absolute top-1/2 left-3.5 h-4 w-4 -translate-y-1/2 text-muted" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search the library"
            className="pl-10"
            aria-label="Search the library"
          />
        </div>
      </div>
      {debounced.length > 1 && (
        <div className="max-h-110 overflow-y-auto border-t border-border">
          {isFetching && !results ? (
            <div className="flex justify-center p-8 text-muted">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : !results?.length ? (
            <p className="p-5 text-sm text-muted">Nothing in the library matches that yet.</p>
          ) : (
            <ul className="divide-y divide-border">
              {results.map((result) => (
                <li key={result.value} className="flex items-center gap-3 px-4 py-2.5">
                  <TrackArt track={result} className="h-10 w-10 rounded-lg" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{result.title}</p>
                    <p className="truncate text-xs text-muted">{[result.author, result.album].filter(Boolean).join(" · ")}</p>
                  </div>
                  {result.album && (
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      aria-label={`Add the album ${result.album}`}
                      title={`Add the album ${result.album}`}
                      disabled={add.isPending}
                      onClick={() => add.mutate({ query: result.album!, type: "album" })}
                    >
                      <Disc3 className="h-4 w-4" />
                    </Button>
                  )}
                  <Button
                    variant="secondary"
                    size="icon"
                    className="h-8 w-8"
                    aria-label={`Add ${result.title}`}
                    disabled={add.isPending}
                    onClick={() => add.mutate({ query: result.value })}
                  >
                    <Plus className="h-4 w-4" />
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </Card>
  );
}

async function queuePlaylist(playlist: Playlist, guildId: string, guildName: string) {
  try {
    const res = await api<{ added: number; missing: number }>(`/guilds/${guildId}/queue`, {
      method: "POST",
      body: json({ playlist_id: playlist.id, shuffle: true }),
    });
    toast.success(`Queued ${res.added} songs in ${guildName}`, {
      description: res.missing > 0 ? `${res.missing} skipped because they aren't in the library yet.` : undefined,
    });
    return true;
  } catch (err) {
    toast.error(errorMessage(err, "Couldn't queue the playlist"));
    return false;
  }
}
