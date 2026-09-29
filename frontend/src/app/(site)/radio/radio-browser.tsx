"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Headphones, Moon, Pause, Play, Radio, Search, Send, Square, Users, Volume2, VolumeX } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { Badge, Button, Card, EmptyState, Equalizer, Input } from "@/components/ui";
import { useGuilds, useMe } from "@/hooks/use-me";
import { api, ApiError, json, loginUrl } from "@/lib/api";
import { cn, formatNumber } from "@/lib/format";
import type { Song, Station } from "@/lib/types";

type Filter = "all" | "on_air" | "live";
type Sort = "popular" | "name";

const volumeKey = "chilly:radio-volume";

function readVolume() {
  try {
    const saved = Number(localStorage.getItem(volumeKey));
    return Number.isFinite(saved) && saved > 0 && saved <= 1 ? saved : 0.8;
  } catch {
    return 0.8;
  }
}

function songLine(song: Song) {
  if (song.artist && song.title) return `${song.artist} – ${song.title}`;
  return song.text || song.title;
}

export function RadioBrowser({ initial }: { initial: Station[] }) {
  const { data: stations = initial } = useQuery({
    queryKey: ["stations"],
    queryFn: async () => (await api<{ stations: Station[] }>("/radio/stations")).stations,
    initialData: initial,
    refetchInterval: 15_000,
  });
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<Filter>("all");
  const [sort, setSort] = useState<Sort>("popular");
  const [listening, setListening] = useState<string | null>(null);
  const [volume, setVolume] = useState(readVolume);
  const audio = useRef<HTMLAudioElement>(null);

  const current = stations.find((s) => s.shortcode === listening) ?? null;

  useEffect(() => {
    const el = audio.current;
    if (!el) return;
    if (!current) {
      el.pause();
      el.removeAttribute("src");
      return;
    }
    if (el.src !== current.stream_url) {
      el.src = current.stream_url;
    }
    el.play().catch(() => {
      toast.error("Your browser blocked playback. Try again.");
      setListening(null);
    });
  }, [current]);

  useEffect(() => {
    if (audio.current) audio.current.volume = volume;
    try {
      localStorage.setItem(volumeKey, String(volume));
    } catch {}
  }, [volume]);

  const counts = useMemo(
    () => ({
      all: stations.length,
      on_air: stations.filter((s) => s.online).length,
      live: stations.filter((s) => s.online && s.live.is_live).length,
    }),
    [stations],
  );
  const totalListeners = stations.reduce((sum, s) => sum + s.listeners, 0);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return stations
      .filter((s) => (filter === "on_air" ? s.online : filter === "live" ? s.online && s.live.is_live : true))
      .filter((s) => {
        if (!q) return true;
        const song = s.now_playing?.song;
        return [s.name, s.description, s.shortcode, song?.title, song?.artist].some((v) => v?.toLowerCase().includes(q));
      })
      .sort((a, b) => {
        if (sort === "name") return a.name.localeCompare(b.name);
        if (a.online !== b.online) return a.online ? -1 : 1;
        return b.listeners - a.listeners || a.name.localeCompare(b.name);
      });
  }, [stations, query, filter, sort]);

  const toggle = (station: Station) => setListening((cur) => (cur === station.shortcode ? null : station.shortcode));

  if (stations.length === 0) {
    return (
      <EmptyState icon={<Radio className="h-6 w-6" />} title="No stations yet" className="mt-10">
        Stations will show up here once they&apos;re configured.
      </EmptyState>
    );
  }

  return (
    <>
      <audio ref={audio} preload="none" onEnded={() => setListening(null)} />

      <p className="mt-6 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted">
        <span>
          <strong className="font-display text-base text-fg">{counts.on_air}</strong> of {counts.all} stations on air
        </span>
        <span className="h-1 w-1 rounded-full bg-border" />
        <span className="flex items-center gap-1.5">
          <Equalizer className="h-3" paused={totalListeners === 0} />
          <strong className="font-display text-base text-fg">{formatNumber(totalListeners)}</strong> listening now
        </span>
      </p>

      <div className="mt-8 flex flex-col gap-3 md:flex-row md:items-center">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-4 h-4 w-4 -translate-y-1/2 text-muted" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search stations, artists or songs"
            className="pl-11"
            aria-label="Search stations"
          />
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex rounded-xl border border-border bg-surface p-1" role="tablist" aria-label="Filter stations">
            {(
              [
                ["all", "All"],
                ["on_air", "On air"],
                ["live", "Live DJ"],
              ] as const
            ).map(([value, label]) => (
              <button
                key={value}
                role="tab"
                aria-selected={filter === value}
                onClick={() => setFilter(value)}
                className={cn(
                  "flex h-9 cursor-pointer items-center gap-1.5 rounded-lg px-3 text-sm transition",
                  filter === value ? "bg-primary-soft text-fg" : "text-muted hover:text-fg",
                )}
              >
                {label}
                <span className="font-mono text-xs text-muted">{counts[value]}</span>
              </button>
            ))}
          </div>
          <select
            value={sort}
            onChange={(e) => setSort(e.target.value as Sort)}
            aria-label="Sort stations"
            className="h-11 cursor-pointer rounded-xl border border-border bg-surface px-3 text-sm text-fg outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="popular">Most listeners</option>
            <option value="name">A–Z</option>
          </select>
        </div>
      </div>

      {visible.length === 0 ? (
        <EmptyState icon={<Search className="h-6 w-6" />} title="No stations match" className="mt-6">
          Try a different search or filter.
        </EmptyState>
      ) : (
        <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {visible.map((station) => (
            <StationCard
              key={station.shortcode}
              station={station}
              listening={listening === station.shortcode}
              onListen={() => toggle(station)}
            />
          ))}
        </div>
      )}

      {current && (
        <NowListeningBar
          station={current}
          volume={volume}
          onVolume={setVolume}
          onStop={() => setListening(null)}
        />
      )}
      {current && <div className="h-28" aria-hidden />}
    </>
  );
}

function StationArt({ station, className }: { station: Station; className?: string }) {
  const art = station.live.is_live && station.live.art ? station.live.art : station.now_playing?.song.art;
  return art ? (
    <img src={art} alt="" className={cn("rounded-2xl object-cover", className)} />
  ) : (
    <div className={cn("grid place-items-center rounded-2xl bg-accent-soft text-accent", className)}>
      <Radio className="h-7 w-7" />
    </div>
  );
}

function StationCard({ station, listening, onListen }: { station: Station; listening: boolean; onListen: () => void }) {
  const np = station.now_playing;
  const next = station.playing_next?.song;

  return (
    <Card
      className={cn(
        "flex flex-col overflow-hidden transition hover:border-primary/30",
        listening && "border-primary/40 shadow-[0_0_60px_-25px_var(--primary)]",
        !station.online && "opacity-70",
      )}
    >
      <div className="flex gap-4 p-4">
        <button
          onClick={onListen}
          disabled={!station.online}
          aria-label={listening ? `Stop listening to ${station.name}` : `Listen to ${station.name}`}
          className="group relative h-20 w-20 shrink-0 cursor-pointer disabled:cursor-not-allowed"
        >
          <StationArt station={station} className="h-20 w-20" />
          {station.online && (
            <span
              className={cn(
                "absolute inset-0 grid place-items-center rounded-2xl bg-bg/55 backdrop-blur-[2px] transition",
                listening ? "opacity-100" : "opacity-0 group-hover:opacity-100 group-focus-visible:opacity-100",
              )}
            >
              {listening ? <Equalizer className="h-5" /> : <Play className="h-6 w-6 fill-current" />}
            </span>
          )}
        </button>

        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <h2 className="truncate font-display text-base leading-tight font-semibold">{station.name}</h2>
            {station.online ? (
              station.live.is_live ? (
                <Badge tone="accent" className="shrink-0">● Live</Badge>
              ) : (
                <Badge tone="lime" className="shrink-0">On air</Badge>
              )
            ) : (
              <Badge className="shrink-0">Offline</Badge>
            )}
          </div>
          {station.online && np ? (
            <div className="mt-1.5 min-w-0">
              <p className="truncate text-sm font-medium">{np.song.title || np.song.text}</p>
              <p className="truncate text-xs text-muted">
                {station.live.is_live ? `with ${station.live.streamer_name}` : np.song.artist}
              </p>
            </div>
          ) : (
            <p className="mt-1.5 line-clamp-2 text-xs text-muted">{station.description || "Nothing scheduled right now."}</p>
          )}
          {station.online && np?.duration ? (
            <SongProgress key={`${np.song.id}-${np.played_at}`} elapsed={np.elapsed} duration={np.duration} className="mt-3" />
          ) : null}
        </div>
      </div>

      <div className="mt-auto flex items-center gap-3 border-t border-border px-4 py-2.5 text-xs text-muted">
        <span className="flex shrink-0 items-center gap-1.5" title={`${station.listeners} listening on the web and in Discord`}>
          <Users className="h-3.5 w-3.5" />
          <span className="font-mono text-fg">{formatNumber(station.listeners)}</span>
        </span>
        <span className="h-3 w-px shrink-0 bg-border" />
        {station.online && next ? (
          <span className="min-w-0 truncate">
            <span className="text-muted/70">Next</span> <span className="text-fg/80">{songLine(next)}</span>
          </span>
        ) : (
          <span className="truncate">{station.online ? "Always on" : "Back soon"}</span>
        )}
      </div>

      <div className="flex items-center gap-1 border-t border-border bg-surface-2/40 px-2 py-1.5">
        <Button variant={listening ? "primary" : "ghost"} size="sm" onClick={onListen} disabled={!station.online}>
          {listening ? <Pause className="h-4 w-4" /> : <Headphones className="h-4 w-4" />}
          {listening ? "Stop" : "Listen"}
        </Button>
        <SendToServer station={station} />
      </div>
    </Card>
  );
}

function SongProgress({ elapsed, duration, className }: { elapsed: number; duration: number; className?: string }) {
  const [seconds, setSeconds] = useState(elapsed);

  useEffect(() => {
    const id = setInterval(() => setSeconds((s) => Math.min(s + 1, duration)), 1000);
    return () => clearInterval(id);
  }, [duration]);

  return (
    <div className={cn("h-1 overflow-hidden rounded-full bg-surface-2", className)}>
      <div
        className="h-full rounded-full bg-linear-to-r from-primary to-accent transition-[width] duration-1000 ease-linear"
        style={{ width: `${Math.min(100, (seconds / duration) * 100)}%` }}
      />
    </div>
  );
}

function NowListeningBar({
  station,
  volume,
  onVolume,
  onStop,
}: {
  station: Station;
  volume: number;
  onVolume: (v: number) => void;
  onStop: () => void;
}) {
  const song = station.now_playing?.song;
  const muted = volume === 0;

  return (
    <div className="fixed inset-x-3 bottom-3 z-30 sm:inset-x-6">
      <div className="mx-auto flex max-w-4xl items-center gap-3 rounded-3xl border border-border bg-surface/90 p-2.5 pr-3 shadow-2xl backdrop-blur-xl sm:gap-4">
        <StationArt station={station} className="h-14 w-14 rounded-2xl" />
        <div className="min-w-0 flex-1">
          <p className="flex items-center gap-1.5 text-[11px] font-semibold tracking-wider text-accent uppercase">
            <Equalizer className="h-2.5" /> {station.name}
          </p>
          <p className="truncate text-sm font-medium">{song ? song.title || song.text : "Live stream"}</p>
          <p className="truncate text-xs text-muted">{song?.artist}</p>
        </div>
        <div className="hidden items-center gap-2 sm:flex">
          <button
            onClick={() => onVolume(muted ? 0.8 : 0)}
            aria-label={muted ? "Unmute" : "Mute"}
            className="cursor-pointer text-muted hover:text-fg"
          >
            {muted ? <VolumeX className="h-4 w-4" /> : <Volume2 className="h-4 w-4" />}
          </button>
          <input
            type="range"
            min={0}
            max={1}
            step={0.01}
            value={volume}
            onChange={(e) => onVolume(Number(e.target.value))}
            aria-label="Volume"
            className="w-24 accent-(--primary)"
          />
        </div>
        <SendToServer station={station} compact />
        <Button variant="primary" size="icon" onClick={onStop} aria-label="Stop listening" className="h-11 w-11 rounded-2xl">
          <Square className="h-4 w-4 fill-current" />
        </Button>
      </div>
    </div>
  );
}

function SendToServer({ station, compact }: { station: Station; compact?: boolean }) {
  const { data: me } = useMe();
  const [open, setOpen] = useState(false);
  const [stay, setStay] = useState(false);
  const { data: guilds, isLoading } = useGuilds(open && !!me);
  const [sending, setSending] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const close = (e: PointerEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, [open]);

  if (!me) {
    return (
      <a
        href={loginUrl("/radio")}
        className="inline-flex h-8 items-center gap-1.5 rounded-lg px-3 text-sm text-muted hover:bg-surface-2 hover:text-fg"
        title="Log in to play in your server"
      >
        <Send className="h-4 w-4" /> {compact ? <span className="hidden sm:inline">Play in server</span> : "Play in my server"}
      </a>
    );
  }

  const send = async (guildId: string, canManage: boolean) => {
    setSending(guildId);
    const keep = stay && canManage;
    try {
      if (keep) {
        await api(`/guilds/${guildId}/radio/247`, { method: "PUT", body: json({ station: station.shortcode }) });
        toast.success(`${station.name} is now playing 24/7`);
      } else {
        await api(`/guilds/${guildId}/radio`, { method: "POST", body: json({ station: station.shortcode }) });
        toast.success(`Tuned into ${station.name}`);
      }
      setOpen(false);
      queryClient.invalidateQueries({ queryKey: ["guilds"] });
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Couldn't start the station");
    } finally {
      setSending(null);
    }
  };

  const candidates = (guilds ?? []).filter((g) => g.user_voice_channel_id || g.listening_with_bot || g.can_manage);

  return (
    <div ref={ref} className="relative">
      <Button variant="ghost" size="sm" onClick={() => setOpen((v) => !v)} disabled={!station.online} aria-expanded={open}>
        <Send className="h-4 w-4" /> {compact ? <span className="hidden sm:inline">Play in server</span> : "Play in my server"}
      </Button>
      {open && (
        <div className="absolute right-0 bottom-full z-40 mb-2 w-72 rounded-2xl border border-border bg-surface p-2 shadow-2xl sm:right-auto sm:left-0">
          <label className="flex cursor-pointer items-start gap-3 rounded-xl px-3 py-2.5 hover:bg-surface-2">
            <input
              type="checkbox"
              checked={stay}
              onChange={(e) => setStay(e.target.checked)}
              className="mt-0.5 h-4 w-4 accent-(--primary)"
            />
            <span className="text-sm">
              <span className="flex items-center gap-1.5 font-medium">
                <Moon className="h-3.5 w-3.5 text-primary" /> Keep it playing 24/7
              </span>
              <span className="block text-xs text-muted">Stays in the channel even when it&apos;s empty. Needs Manage Server.</span>
            </span>
          </label>
          <div className="my-1 border-t border-border" />
          {isLoading ? (
            <p className="p-3 text-sm text-muted">Loading servers…</p>
          ) : candidates.length === 0 ? (
            <p className="p-3 text-sm text-muted">Join a voice channel in a server that has Chilly, then try again.</p>
          ) : (
            candidates.map((g) => {
              const blocked = stay && !g.can_manage;
              return (
                <button
                  key={g.id}
                  onClick={() => send(g.id, g.can_manage)}
                  disabled={sending !== null || blocked}
                  className="flex w-full cursor-pointer items-center justify-between gap-2 rounded-xl px-3 py-2 text-left text-sm hover:bg-surface-2 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <span className="truncate">{g.name}</span>
                  <span className="shrink-0 text-xs text-muted">
                    {sending === g.id ? "Starting…" : blocked ? "No permission" : g.user_voice_channel_id ? "In voice" : ""}
                  </span>
                </button>
              );
            })
          )}
        </div>
      )}
    </div>
  );
}
