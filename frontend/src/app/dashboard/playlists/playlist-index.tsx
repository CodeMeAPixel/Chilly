"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ListMusic, Loader2, Plus } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { toast } from "sonner";
import { Button, Card, EmptyState, Input, Skeleton } from "@/components/ui";
import { api, ApiError, json } from "@/lib/api";
import type { Playlist } from "@/lib/types";

export function PlaylistIndex() {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const { data: playlists, isLoading } = useQuery({
    queryKey: ["playlists"],
    queryFn: async () => (await api<{ playlists: Playlist[] }>("/playlists")).playlists,
  });

  const create = useMutation({
    mutationFn: (value: string) => api<Playlist>("/playlists", { method: "POST", body: json({ name: value }) }),
    onSuccess: (playlist) => {
      toast.success(`Created ${playlist.name}`);
      setName("");
      queryClient.invalidateQueries({ queryKey: ["playlists"] });
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : "Couldn't create playlist"),
  });

  return (
    <div className="space-y-6">
      <Card className="p-4">
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (name.trim()) create.mutate(name.trim());
          }}
        >
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="New playlist name" maxLength={100} />
          <Button type="submit" disabled={!name.trim() || create.isPending}>
            {create.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />} Create
          </Button>
        </form>
      </Card>

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-3xl" />
          ))}
        </div>
      ) : !playlists?.length ? (
        <Card>
          <EmptyState icon={<ListMusic className="h-6 w-6" />} title="No playlists yet">
            Create your first playlist above, then add songs to it.
          </EmptyState>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {playlists.map((playlist) => (
            <Link key={playlist.id} href={`/dashboard/playlists/${playlist.id}`}>
              <Card className="flex items-center gap-4 p-5 transition hover:-translate-y-0.5 hover:border-primary/30">
                <div className="grid h-12 w-12 shrink-0 place-items-center rounded-2xl bg-linear-to-br from-primary-soft to-accent-soft text-primary">
                  <ListMusic className="h-5 w-5" />
                </div>
                <div className="min-w-0">
                  <p className="truncate font-medium">{playlist.name}</p>
                  <p className="text-sm text-muted">
                    {playlist.track_count} track{playlist.track_count === 1 ? "" : "s"}
                  </p>
                </div>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
