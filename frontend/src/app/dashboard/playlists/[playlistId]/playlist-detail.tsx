"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Check, ListMusic, Loader2, Pencil, Play, Plus, Trash2, X } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import { TrackArt } from "@/components/track-art";
import { Button, buttonClass, Card, EmptyState, Input, Skeleton } from "@/components/ui";
import { useGuilds } from "@/hooks/use-me";
import { api, ApiError, json } from "@/lib/api";
import { formatDuration } from "@/lib/format";
import type { Playlist, PlaylistTrack } from "@/lib/types";

type Detail = { playlist: Playlist; tracks: PlaylistTrack[] };

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

  const { playlist, tracks } = data;
  const totalMs = tracks.reduce((sum, t) => sum + (t.track.is_stream ? 0 : t.track.length_ms), 0);

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
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <PlayInServer playlist={playlist} />
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

      <div className="grid gap-6 xl:grid-cols-[1fr_380px]">
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
                  <TrackArt track={item.track} className="h-10 w-10 rounded-lg" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{item.track.title}</p>
                    <p className="truncate text-xs text-muted">{item.track.author}</p>
                  </div>
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

function AddToPlaylist({ playlistId, onAdded }: { playlistId: string; onAdded: () => void }) {
  const [query, setQuery] = useState("");
  const add = useMutation({
    mutationFn: (value: string) =>
      api<{ added: number }>(`/playlists/${playlistId}/tracks`, { method: "POST", body: json({ query: value }) }),
    onSuccess: (res) => {
      toast.success(res.added === 1 ? "Track added" : `Added ${res.added} tracks`);
      setQuery("");
      onAdded();
    },
    onError: (err) => toast.error(errorMessage(err, "Couldn't add that")),
  });

  return (
    <Card className="h-fit space-y-3 p-5">
      <h2 className="font-display text-lg font-semibold">Add music</h2>
      <p className="text-sm text-muted">Search for a song, or paste a link to a track, album or playlist.</p>
      <form
        className="space-y-3"
        onSubmit={(e) => {
          e.preventDefault();
          if (query.trim()) add.mutate(query.trim());
        }}
      >
        <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Song name or link" />
        <Button type="submit" className="w-full" disabled={!query.trim() || add.isPending}>
          {add.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />} Add to playlist
        </Button>
      </form>
    </Card>
  );
}

function PlayInServer({ playlist }: { playlist: Playlist }) {
  const [open, setOpen] = useState(false);
  const { data: guilds, isLoading } = useGuilds(open);
  const [busy, setBusy] = useState<string | null>(null);
  const candidates = (guilds ?? []).filter((g) => g.user_voice_channel_id || g.listening_with_bot || g.can_manage);

  const play = async (guildId: string, name: string) => {
    setBusy(guildId);
    try {
      const res = await api<{ added: number }>(`/guilds/${guildId}/queue`, {
        method: "POST",
        body: json({ playlist_id: playlist.id, shuffle: true }),
      });
      toast.success(`Queued ${res.added} tracks in ${name}`);
      setOpen(false);
    } catch (err) {
      toast.error(errorMessage(err, "Couldn't queue the playlist"));
    } finally {
      setBusy(null);
    }
  };

  return (
    <div className="relative">
      <Button onClick={() => setOpen((v) => !v)} disabled={playlist.track_count === 0}>
        <Play className="h-4 w-4" /> Play in server
      </Button>
      {open && (
        <div className="absolute right-0 z-20 mt-2 w-72 rounded-2xl border border-border bg-surface p-2 shadow-2xl">
          {isLoading ? (
            <p className="p-3 text-sm text-muted">Loading servers…</p>
          ) : candidates.length === 0 ? (
            <p className="p-3 text-sm text-muted">Join a voice channel in a server with Chilly first.</p>
          ) : (
            candidates.map((g) => (
              <button
                key={g.id}
                onClick={() => play(g.id, g.name)}
                disabled={busy !== null}
                className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-left text-sm hover:bg-surface-2 disabled:opacity-50 cursor-pointer"
              >
                <span className="truncate">{g.name}</span>
                {busy === g.id && <Loader2 className="h-4 w-4 animate-spin text-muted" />}
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}
