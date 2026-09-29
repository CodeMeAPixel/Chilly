"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { ChevronLeft, ChevronRight, Disc3, ListMusic, Loader2, Mic2, Music2, Play, Search, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { RequestButton } from "@/components/request-button";
import { ServerPicker } from "@/components/server-picker";
import { TrackArt } from "@/components/track-art";
import { Badge, Card, EmptyState, Input, Skeleton } from "@/components/ui";
import { useMe } from "@/hooks/use-me";
import { api, ApiError, json } from "@/lib/api";
import { cn, formatDuration } from "@/lib/format";
import type { LibraryGroupSummary, LibraryPage, LibraryPlaylist } from "@/lib/types";

type Tab = "songs" | "artists" | "albums" | "playlists";
type Filter = { artist?: string; album?: string; playlist?: string };

const perPage = 50;

function useDebounced<T>(value: T, delay: number) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(id);
  }, [value, delay]);
  return debounced;
}

export function TrackBrowser({ initialFilter, initialQuery }: { initialFilter: Filter; initialQuery: string }) {
  const [tab, setTab] = useState<Tab>("songs");
  const [query, setQuery] = useState(initialQuery);
  const [filter, setFilter] = useState<Filter>(initialFilter);
  const [sort, setSort] = useState("artist");
  const [page, setPage] = useState(1);
  const debounced = useDebounced(query.trim(), 300);

  const openSongs = (next: Filter) => {
    setFilter(next);
    setQuery("");
    setPage(1);
    setTab("songs");
  };

  const tabs: { id: Tab; label: string; icon: typeof Music2 }[] = [
    { id: "songs", label: "Songs", icon: Music2 },
    { id: "artists", label: "Artists", icon: Mic2 },
    { id: "albums", label: "Albums", icon: Disc3 },
    { id: "playlists", label: "Playlists", icon: ListMusic },
  ];

  return (
    <div className="mt-10 space-y-5">
      <div className="flex flex-col gap-3 md:flex-row md:items-center">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-4 h-4 w-4 -translate-y-1/2 text-muted" />
          <Input
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setPage(1);
            }}
            placeholder={tab === "songs" ? "Search songs, artists or albums" : `Filter ${tab}`}
            className="pl-11"
            aria-label="Search the library"
          />
        </div>
        <div className="flex overflow-x-auto rounded-xl border border-border bg-surface p-1" role="tablist">
          {tabs.map((t) => (
            <button
              key={t.id}
              role="tab"
              aria-selected={tab === t.id}
              onClick={() => {
                setTab(t.id);
                setPage(1);
              }}
              className={cn(
                "flex h-9 shrink-0 cursor-pointer items-center gap-1.5 rounded-lg px-3 text-sm transition",
                tab === t.id ? "bg-primary-soft text-fg" : "text-muted hover:text-fg",
              )}
            >
              <t.icon className="h-4 w-4" /> {t.label}
            </button>
          ))}
        </div>
      </div>

      {tab === "songs" && (
        <SongList
          query={debounced}
          filter={filter}
          sort={sort}
          page={page}
          onSort={setSort}
          onPage={setPage}
          onClearFilter={() => openSongs({})}
          onArtist={(artist) => openSongs({ artist })}
        />
      )}
      {tab === "artists" && <GroupGrid kind="artists" query={debounced} onOpen={(item) => openSongs({ artist: item.name })} />}
      {tab === "albums" && <GroupGrid kind="albums" query={debounced} onOpen={(item) => openSongs({ album: item.name })} />}
      {tab === "playlists" && <PlaylistGrid query={debounced} onOpen={(name) => openSongs({ playlist: name })} />}
    </div>
  );
}

function SongList({
  query,
  filter,
  sort,
  page,
  onSort,
  onPage,
  onClearFilter,
  onArtist,
}: {
  query: string;
  filter: Filter;
  sort: string;
  page: number;
  onSort: (sort: string) => void;
  onPage: (page: number) => void;
  onClearFilter: () => void;
  onArtist: (artist: string) => void;
}) {
  const { data: me } = useMe();
  const params = new URLSearchParams({ page: String(page), per_page: String(perPage), sort });
  if (query) params.set("q", query);
  if (filter.artist) params.set("artist", filter.artist);
  if (filter.album) params.set("album", filter.album);
  if (filter.playlist) params.set("playlist", filter.playlist);

  const { data, isLoading, isFetching, error } = useQuery({
    queryKey: ["library-tracks", params.toString()],
    queryFn: () => api<LibraryPage>(`/library/tracks?${params}`),
    placeholderData: keepPreviousData,
    staleTime: 60_000,
  });

  const activeFilter = filter.artist ?? filter.album ?? filter.playlist;
  const pages = data ? Math.max(1, Math.ceil(data.total / perPage)) : 1;

  const play = async (key: string, title: string, guildId: string, guildName: string) => {
    try {
      await api(`/guilds/${guildId}/queue`, { method: "POST", body: json({ query: key }) });
      toast.success(`Playing ${title} in ${guildName}`);
      return true;
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Couldn't play that");
      return false;
    }
  };

  return (
    <Card className="overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border px-4 py-3 text-sm">
        <div className="flex min-w-0 flex-wrap items-center gap-2 text-muted">
          {data && (
            <span>
              {data.total.toLocaleString()} song{data.total === 1 ? "" : "s"}
            </span>
          )}
          {activeFilter && (
            <Badge tone="primary" className="max-w-full">
              <span className="truncate">
                {filter.artist ? "Artist" : filter.album ? "Album" : "Playlist"}: {activeFilter}
              </span>
              <button onClick={onClearFilter} aria-label="Clear filter" className="cursor-pointer hover:text-fg">
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )}
          {isFetching && !isLoading && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
        </div>
        {!query && (
          <select
            value={sort}
            onChange={(e) => {
              onSort(e.target.value);
              onPage(1);
            }}
            aria-label="Sort songs"
            className="h-9 cursor-pointer rounded-lg border border-border bg-surface px-2 text-sm"
          >
            <option value="artist">Artist</option>
            <option value="title">Title</option>
            <option value="album">Album</option>
          </select>
        )}
      </div>

      {isLoading ? (
        <div className="space-y-2 p-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-12" />
          ))}
        </div>
      ) : error ? (
        <EmptyState title="Couldn't load songs">{error.message}</EmptyState>
      ) : !data?.tracks.length ? (
        <EmptyState icon={<Music2 className="h-6 w-6" />} title="No songs found">
          Try another search. If it&apos;s missing from the library, suggest it from your requests page.
        </EmptyState>
      ) : (
        <ul className="divide-y divide-border">
          {data.tracks.map((t) => (
            <li key={t.value} className="flex items-center gap-3 px-4 py-2.5">
              <TrackArt track={t} className="h-11 w-11 rounded-lg" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{t.title}</p>
                <p className="truncate text-xs text-muted">
                  {t.author ? (
                    <button onClick={() => onArtist(t.author)} className="cursor-pointer hover:text-fg hover:underline">
                      {t.author}
                    </button>
                  ) : (
                    "Unknown artist"
                  )}
                  {t.album && ` · ${t.album}`}
                </p>
              </div>
              <span className="hidden font-mono text-xs text-muted sm:block">{formatDuration(t.length_ms)}</span>
              <RequestButton songKey={t.value} title={t.title} compact />
              {me && (
                <ServerPicker
                  onPick={(guildId, guildName) => play(t.value, t.title, guildId, guildName)}
                  variant="ghost"
                  size="sm"
                  label="Play"
                  icon={<Play className="h-4 w-4" />}
                />
              )}
            </li>
          ))}
        </ul>
      )}

      {data && pages > 1 && (
        <div className="flex items-center justify-between border-t border-border px-4 py-3 text-sm text-muted">
          <button
            onClick={() => onPage(page - 1)}
            disabled={page <= 1}
            className="inline-flex cursor-pointer items-center gap-1 rounded-lg px-2 py-1 hover:text-fg disabled:cursor-default disabled:opacity-40"
          >
            <ChevronLeft className="h-4 w-4" /> Previous
          </button>
          <span>
            Page {page} of {pages}
          </span>
          <button
            onClick={() => onPage(page + 1)}
            disabled={page >= pages}
            className="inline-flex cursor-pointer items-center gap-1 rounded-lg px-2 py-1 hover:text-fg disabled:cursor-default disabled:opacity-40"
          >
            Next <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      )}
    </Card>
  );
}

function GroupGrid({
  kind,
  query,
  onOpen,
}: {
  kind: "artists" | "albums";
  query: string;
  onOpen: (item: LibraryGroupSummary) => void;
}) {
  const { data, isLoading } = useQuery({
    queryKey: ["library-groups", kind],
    queryFn: async () => (await api<{ items: LibraryGroupSummary[] }>(`/library/${kind}`)).items,
    staleTime: 5 * 60_000,
  });
  const items = useMemo(() => {
    const q = query.toLowerCase();
    return (data ?? []).filter((i) => !q || i.name.toLowerCase().includes(q) || i.artist?.toLowerCase().includes(q));
  }, [data, query]);

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        {Array.from({ length: 10 }).map((_, i) => (
          <Skeleton key={i} className="aspect-square rounded-3xl" />
        ))}
      </div>
    );
  }
  if (items.length === 0) {
    return (
      <Card>
        <EmptyState title={`No ${kind} found`} />
      </Card>
    );
  }
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
      {items.map((item) => (
        <button
          key={`${item.name}-${item.artist ?? ""}`}
          onClick={() => onOpen(item)}
          className="group cursor-pointer text-left"
        >
          <div className="aspect-square overflow-hidden rounded-3xl border border-border bg-surface transition group-hover:border-primary/40">
            {item.art ? (
              <img src={item.art} alt="" className="h-full w-full object-cover transition group-hover:scale-105" />
            ) : (
              <div className="grid h-full place-items-center bg-linear-to-br from-primary-soft to-accent-soft text-primary">
                {kind === "artists" ? <Mic2 className="h-8 w-8" /> : <Disc3 className="h-8 w-8" />}
              </div>
            )}
          </div>
          <p className="mt-2 truncate text-sm font-medium">{item.name}</p>
          <p className="truncate text-xs text-muted">
            {kind === "albums" && item.artist ? `${item.artist} · ` : ""}
            {item.track_count} song{item.track_count === 1 ? "" : "s"}
          </p>
        </button>
      ))}
    </div>
  );
}

function PlaylistGrid({ query, onOpen }: { query: string; onOpen: (name: string) => void }) {
  const { data, isLoading } = useQuery({
    queryKey: ["library", "playlists"],
    queryFn: async () => (await api<{ playlists: LibraryPlaylist[] }>("/library/playlists")).playlists,
    staleTime: 5 * 60_000,
  });
  const items = (data ?? []).filter((p) => !query || p.name.toLowerCase().includes(query.toLowerCase()));

  if (isLoading) {
    return <Skeleton className="h-40 rounded-3xl" />;
  }
  if (items.length === 0) {
    return (
      <Card>
        <EmptyState title="No playlists found" />
      </Card>
    );
  }
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {items.map((p) => (
        <button key={p.name} onClick={() => onOpen(p.name)} className="cursor-pointer text-left">
          <Card className="flex items-center gap-4 p-4 transition hover:border-primary/40">
            {p.art ? (
              <img src={p.art} alt="" className="h-14 w-14 shrink-0 rounded-2xl object-cover" />
            ) : (
              <div className="grid h-14 w-14 shrink-0 place-items-center rounded-2xl bg-accent-soft text-accent">
                <ListMusic className="h-6 w-6" />
              </div>
            )}
            <div className="min-w-0">
              <p className="truncate font-medium">{p.name}</p>
              <p className="truncate text-xs text-muted">
                {p.track_count} songs{p.artists.length ? ` · ${p.artists.join(", ")}` : ""}
              </p>
            </div>
          </Card>
        </button>
      ))}
    </div>
  );
}
