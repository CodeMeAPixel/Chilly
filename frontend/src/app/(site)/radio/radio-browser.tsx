"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Headphones, History, Moon, Pause, Play, Radio, Search, Send, Square, Users, Volume2, VolumeX } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { Badge, Button, EmptyState, Equalizer, Input } from "@/components/ui";
import { useGuilds, useMe } from "@/hooks/use-me";
import { api, ApiError, json, loginUrl } from "@/lib/api";
import { cn, formatDuration, formatNumber, formatRelative } from "@/lib/format";
import type { Song, Station } from "@/lib/types";

type Live = { stations: Station[]; server_time?: number; received_at: number };

const volumeKey = "chilly:radio-volume";
const liveKey = ["radio-live"];

function readVolume() {
  try {
    const saved = Number(localStorage.getItem(volumeKey));
    return Number.isFinite(saved) && saved > 0 && saved <= 1 ? saved : 0.8;
  } catch {
    return 0.8;
  }
}

function useNow(interval = 1000) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), interval);
    return () => clearInterval(id);
  }, [interval]);
  return now;
}

function withReceipt(data: { stations: Station[]; server_time?: number }): Live {
  return { ...data, received_at: Date.now() };
}

function useLiveStations(initial: Station[]) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: liveKey,
    queryFn: async () => withReceipt(await api<{ stations: Station[]; server_time: number }>("/radio/stations")),
    initialData: () => ({ stations: initial, received_at: Date.now() }),
    refetchInterval: 60_000,
  });

  useEffect(() => {
    const source = new EventSource("/api/v1/radio/events");
    source.addEventListener("stations", (event) => {
      queryClient.setQueryData(liveKey, withReceipt(JSON.parse((event as MessageEvent).data)));
    });
    return () => source.close();
  }, [queryClient]);

  return query.data;
}

function elapsedSeconds(station: Station, live: Live, now: number) {
  const np = station.now_playing;
  if (!np) return 0;
  const offset = live.server_time ? live.server_time * 1000 - live.received_at : 0;
  const elapsed = (now + offset) / 1000 - np.played_at;
  if (!Number.isFinite(elapsed) || elapsed < 0) return np.elapsed;
  return np.duration ? Math.min(elapsed, np.duration) : elapsed;
}

export function RadioBrowser({ initial }: { initial: Station[] }) {
  const live = useLiveStations(initial);
  const stations = live.stations;
  const now = useNow();
  const [selected, setSelected] = useState<string | null>(null);
  const [listening, setListening] = useState<string | null>(null);
  const [volume, setVolume] = useState(readVolume);
  const [query, setQuery] = useState("");
  const [spotlightVisible, setSpotlightVisible] = useState(true);
  const audio = useRef<HTMLAudioElement>(null);
  const spotlightRef = useRef<HTMLDivElement>(null);

  const ranked = useMemo(
    () => [...stations].sort((a, b) => Number(b.online) - Number(a.online) || b.listeners - a.listeners || a.name.localeCompare(b.name)),
    [stations],
  );
  const spotlight = stations.find((s) => s.shortcode === (selected ?? listening)) ?? ranked[0] ?? null;
  const current = stations.find((s) => s.shortcode === listening) ?? null;
  const totalListeners = stations.reduce((sum, s) => sum + s.listeners, 0);
  const onAir = stations.filter((s) => s.online).length;

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

  useEffect(() => {
    const el = spotlightRef.current;
    if (!el) return;
    const observer = new IntersectionObserver(([entry]) => setSpotlightVisible(entry.isIntersecting), { threshold: 0.2 });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const toggle = (code: string) => setListening((cur) => (cur === code ? null : code));

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return ranked;
    return ranked.filter((s) => {
      const song = s.now_playing?.song;
      return [s.name, s.description, song?.title, song?.artist].some((v) => v?.toLowerCase().includes(q));
    });
  }, [ranked, query]);

  if (stations.length === 0 || !spotlight) {
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
          <strong className="font-display text-base text-fg">{onAir}</strong> of {stations.length} stations on air
        </span>
        <span className="h-1 w-1 rounded-full bg-border" />
        <span className="flex items-center gap-1.5">
          <Equalizer className="h-3" paused={totalListeners === 0} />
          <strong className="font-display text-base text-fg">{formatNumber(totalListeners)}</strong> listening now
        </span>
      </p>

      <div ref={spotlightRef} className="mt-8">
        <Spotlight
          station={spotlight}
          elapsed={elapsedSeconds(spotlight, live, now)}
          listening={listening === spotlight.shortcode}
          volume={volume}
          onVolume={setVolume}
          onToggle={() => toggle(spotlight.shortcode)}
        />
      </div>

      <section className="mt-12 space-y-4">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <h2 className="font-display text-2xl font-semibold tracking-tight">All stations</h2>
          {stations.length > 6 && (
            <div className="relative w-full sm:w-72">
              <Search className="pointer-events-none absolute top-1/2 left-4 h-4 w-4 -translate-y-1/2 text-muted" />
              <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search stations or songs" className="pl-11" />
            </div>
          )}
        </div>
        {filtered.length === 0 ? (
          <EmptyState icon={<Search className="h-6 w-6" />} title="No stations match" />
        ) : (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {filtered.map((station) => (
              <StationTile
                key={station.shortcode}
                station={station}
                active={spotlight.shortcode === station.shortcode}
                listening={listening === station.shortcode}
                progress={station.now_playing?.duration ? elapsedSeconds(station, live, now) / station.now_playing.duration : 0}
                onSelect={() => setSelected(station.shortcode)}
                onListen={() => {
                  setSelected(station.shortcode);
                  toggle(station.shortcode);
                }}
              />
            ))}
          </div>
        )}
      </section>

      {current && !spotlightVisible && (
        <MiniPlayer station={current} volume={volume} onVolume={setVolume} onStop={() => setListening(null)} />
      )}
      {current && !spotlightVisible && <div className="h-24" aria-hidden />}
    </>
  );
}

function Art({ station, className, iconClass }: { station: Station; className?: string; iconClass?: string }) {
  const art = station.live.is_live && station.live.art ? station.live.art : station.now_playing?.song.art;
  return art ? (
    <img src={art} alt="" className={cn("object-cover", className)} />
  ) : (
    <div className={cn("grid place-items-center bg-accent-soft text-accent", className)}>
      <Radio className={cn("h-1/3 w-1/3", iconClass)} />
    </div>
  );
}

function StatusBadge({ station }: { station: Station }) {
  if (!station.online) return <Badge>Offline</Badge>;
  if (station.live.is_live) return <Badge tone="accent">● Live · {station.live.streamer_name}</Badge>;
  return <Badge tone="lime">On air</Badge>;
}

function Spotlight({
  station,
  elapsed,
  listening,
  volume,
  onVolume,
  onToggle,
}: {
  station: Station;
  elapsed: number;
  listening: boolean;
  volume: number;
  onVolume: (v: number) => void;
  onToggle: () => void;
}) {
  const np = station.now_playing;
  const song = np?.song;
  const next = station.playing_next?.song;
  const history = (station.song_history ?? []).slice(0, 4);
  const art = station.live.is_live && station.live.art ? station.live.art : song?.art;
  const pct = np?.duration ? Math.min(100, (elapsed / np.duration) * 100) : 0;

  return (
    <div className="relative z-10 rounded-4xl border border-border bg-surface">
      {art && (
        <div className="pointer-events-none absolute inset-0 overflow-hidden rounded-4xl" aria-hidden>
          <img src={art} alt="" className="h-full w-full scale-125 object-cover opacity-25 blur-3xl" />
        </div>
      )}
      <div className="relative grid grid-cols-1 gap-8 p-6 sm:p-8 lg:grid-cols-[minmax(0,1fr)_300px]">
        <div className="flex flex-col gap-6 sm:flex-row sm:items-center">
          <div className="relative mx-auto shrink-0 sm:mx-0">
            <Art station={station} className="h-48 w-48 rounded-3xl shadow-2xl sm:h-56 sm:w-56" />
            {listening && (
              <span className="absolute right-3 bottom-3 rounded-xl bg-bg/80 p-2 backdrop-blur">
                <Equalizer className="h-5" />
              </span>
            )}
          </div>

          <div className="min-w-0 flex-1 space-y-4">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs font-semibold tracking-wider text-primary uppercase">{station.name}</span>
              <StatusBadge station={station} />
              <span className="flex items-center gap-1 text-xs text-muted">
                <Users className="h-3.5 w-3.5" /> {formatNumber(station.listeners)}
              </span>
            </div>

            {station.online && song ? (
              <div className="min-w-0 space-y-1">
                <h2 className="line-clamp-2 font-display text-3xl leading-tight font-semibold tracking-tight sm:text-4xl">
                  {song.title || song.text}
                </h2>
                <p className="truncate text-muted">{[song.artist, song.album].filter(Boolean).join(" · ")}</p>
              </div>
            ) : (
              <div className="space-y-1">
                <h2 className="font-display text-3xl font-semibold tracking-tight">{station.name}</h2>
                <p className="text-muted">{station.online ? station.description : "This station is off air right now. Check back soon."}</p>
              </div>
            )}

            {station.online && np?.duration ? (
              <div className="space-y-1.5">
                <div className="h-1.5 overflow-hidden rounded-full bg-surface-2">
                  <div
                    className="h-full rounded-full bg-linear-to-r from-primary to-accent transition-[width] duration-1000 ease-linear"
                    style={{ width: `${pct}%` }}
                  />
                </div>
                <div className="flex justify-between font-mono text-xs text-muted">
                  <span>{formatDuration(elapsed * 1000)}</span>
                  <span>{formatDuration(np.duration * 1000)}</span>
                </div>
              </div>
            ) : null}

            <div className="flex flex-wrap items-center gap-3">
              <Button size="lg" variant={listening ? "secondary" : "primary"} onClick={onToggle} disabled={!station.online} className="rounded-full px-6">
                {listening ? <Pause className="h-5 w-5 fill-current" /> : <Headphones className="h-5 w-5" />}
                {listening ? "Stop" : "Listen"}
              </Button>
              <VolumeControl volume={volume} onVolume={onVolume} />
              <SendToServer station={station} placement="down" />
            </div>
          </div>
        </div>

        <div className="space-y-5 border-t border-border pt-6 lg:border-t-0 lg:border-l lg:pt-0 lg:pl-8">
          {next && station.online && (
            <div className="space-y-2">
              <p className="text-xs font-medium tracking-wide text-muted uppercase">Up next</p>
              <SongRow song={next} />
            </div>
          )}
          <div className="space-y-2">
            <p className="flex items-center gap-1.5 text-xs font-medium tracking-wide text-muted uppercase">
              <History className="h-3.5 w-3.5" /> Recently played
            </p>
            {history.length === 0 ? (
              <p className="text-sm text-muted">Nothing yet.</p>
            ) : (
              <ul className="space-y-2">
                {history.map((h) => (
                  <li key={`${h.played_at}-${h.song.id}`}>
                    <SongRow song={h.song} meta={formatRelative(new Date(h.played_at * 1000).toISOString())} />
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function SongRow({ song, meta }: { song: Song; meta?: string }) {
  return (
    <div className="flex items-center gap-3">
      {song.art ? (
        <img src={song.art} alt="" className="h-10 w-10 shrink-0 rounded-lg object-cover" />
      ) : (
        <div className="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-surface-2 text-muted">
          <Radio className="h-4 w-4" />
        </div>
      )}
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{song.title || song.text}</p>
        <p className="truncate text-xs text-muted">
          {song.artist}
          {meta && ` · ${meta}`}
        </p>
      </div>
    </div>
  );
}

function VolumeControl({ volume, onVolume }: { volume: number; onVolume: (v: number) => void }) {
  const muted = volume === 0;
  return (
    <div className="flex items-center gap-2">
      <button onClick={() => onVolume(muted ? 0.8 : 0)} aria-label={muted ? "Unmute" : "Mute"} className="cursor-pointer text-muted hover:text-fg">
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
        className="w-24 accent-primary"
      />
    </div>
  );
}

function StationTile({
  station,
  active,
  listening,
  progress,
  onSelect,
  onListen,
}: {
  station: Station;
  active: boolean;
  listening: boolean;
  progress: number;
  onSelect: () => void;
  onListen: () => void;
}) {
  const song = station.now_playing?.song;
  return (
    <div
      className={cn(
        "group relative flex items-center gap-3 overflow-hidden rounded-3xl border bg-surface p-3 transition",
        active ? "border-primary/50 shadow-[0_0_40px_-20px_var(--primary)]" : "border-border hover:border-primary/30",
        !station.online && "opacity-60",
      )}
    >
      <button onClick={onSelect} className="absolute inset-0 cursor-pointer" aria-label={`Show ${station.name}`} />
      <Art station={station} className="h-16 w-16 shrink-0 rounded-2xl" />
      <div className="pointer-events-none min-w-0 flex-1">
        <p className="flex items-center gap-1.5 truncate text-sm font-semibold">
          <span className={cn("h-1.5 w-1.5 shrink-0 rounded-full", station.online ? "bg-lime" : "bg-muted")} />
          <span className="truncate">{station.name}</span>
        </p>
        <p className="truncate text-sm">{station.online && song ? song.title || song.text : "Off air"}</p>
        <p className="flex items-center gap-2 truncate text-xs text-muted">
          <span className="truncate">{station.online && song ? song.artist : station.description}</span>
          <span className="flex shrink-0 items-center gap-1">
            <Users className="h-3 w-3" /> {station.listeners}
          </span>
        </p>
      </div>
      <Button
        variant={listening ? "primary" : "secondary"}
        size="icon"
        onClick={onListen}
        disabled={!station.online}
        aria-label={listening ? `Stop ${station.name}` : `Listen to ${station.name}`}
        className="relative h-10 w-10 shrink-0 rounded-full"
      >
        {listening ? <Pause className="h-4 w-4 fill-current" /> : <Play className="h-4 w-4 fill-current" />}
      </Button>
      {station.online && progress > 0 && (
        <span className="pointer-events-none absolute right-0 bottom-0 left-0 h-0.5 bg-surface-2">
          <span className="block h-full bg-linear-to-r from-primary to-accent" style={{ width: `${Math.min(100, progress * 100)}%` }} />
        </span>
      )}
    </div>
  );
}

function MiniPlayer({
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
  return (
    <div className="fixed inset-x-3 bottom-3 z-30 sm:inset-x-6">
      <div className="mx-auto flex max-w-4xl items-center gap-3 rounded-3xl border border-border bg-surface/90 p-2.5 pr-3 shadow-2xl backdrop-blur-xl sm:gap-4">
        <Art station={station} className="h-14 w-14 shrink-0 rounded-2xl" />
        <div className="min-w-0 flex-1">
          <p className="flex items-center gap-1.5 text-[11px] font-semibold tracking-wider text-accent uppercase">
            <Equalizer className="h-2.5" /> {station.name}
          </p>
          <p className="truncate text-sm font-medium">{song ? song.title || song.text : "Live stream"}</p>
          <p className="truncate text-xs text-muted">{song?.artist}</p>
        </div>
        <div className="hidden sm:block">
          <VolumeControl volume={volume} onVolume={onVolume} />
        </div>
        <SendToServer station={station} placement="up" />
        <Button variant="primary" size="icon" onClick={onStop} aria-label="Stop listening" className="h-11 w-11 rounded-2xl">
          <Square className="h-4 w-4 fill-current" />
        </Button>
      </div>
    </div>
  );
}

function SendToServer({ station, placement }: { station: Station; placement: "up" | "down" }) {
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
        className="inline-flex h-10 items-center gap-1.5 rounded-xl px-3 text-sm text-muted hover:bg-surface-2 hover:text-fg"
        title="Log in to play in your server"
      >
        <Send className="h-4 w-4" /> <span className="hidden sm:inline">Play in server</span>
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
      <Button variant="ghost" onClick={() => setOpen((v) => !v)} disabled={!station.online} aria-expanded={open}>
        <Send className="h-4 w-4" /> <span className="hidden sm:inline">Play in server</span>
      </Button>
      {open && (
        <div
          className={cn(
            "absolute z-40 w-72 max-w-[calc(100vw-2rem)] rounded-2xl border border-border bg-surface p-2 shadow-2xl",
            placement === "up" ? "right-0 bottom-full mb-3" : "top-full left-0 mt-2 sm:right-auto",
          )}
        >
          <label className="flex cursor-pointer items-start gap-3 rounded-xl px-3 py-2.5 hover:bg-surface-2">
            <input type="checkbox" checked={stay} onChange={(e) => setStay(e.target.checked)} className="mt-0.5 h-4 w-4 accent-primary" />
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
