"use client";

import { useQuery } from "@tanstack/react-query";
import { Headphones, Pause, Play, Radio, Send, Users } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { Badge, Button, Card, EmptyState, Equalizer } from "@/components/ui";
import { useGuilds, useMe } from "@/hooks/use-me";
import { api, ApiError, json, loginUrl } from "@/lib/api";
import { cn } from "@/lib/format";
import type { Station } from "@/lib/types";

export function RadioBrowser({ initial }: { initial: Station[] }) {
  const { data: stations = initial } = useQuery({
    queryKey: ["stations"],
    queryFn: async () => (await api<{ stations: Station[] }>("/radio/stations")).stations,
    initialData: initial,
    refetchInterval: 15_000,
  });
  const [listening, setListening] = useState<string | null>(null);
  const audio = useRef<HTMLAudioElement>(null);

  useEffect(() => {
    const el = audio.current;
    if (!el) return;
    const station = stations.find((s) => s.shortcode === listening);
    if (!station) {
      el.pause();
      el.removeAttribute("src");
      return;
    }
    if (el.src !== station.stream_url) {
      el.src = station.stream_url;
    }
    el.play().catch(() => {
      toast.error("Your browser blocked playback. Try again.");
      setListening(null);
    });
  }, [listening, stations]);

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
      <div className="mt-10 grid gap-5 lg:grid-cols-2">
        {stations.map((station) => (
          <StationCard
            key={station.shortcode}
            station={station}
            listening={listening === station.shortcode}
            onListen={() => setListening((cur) => (cur === station.shortcode ? null : station.shortcode))}
          />
        ))}
      </div>
    </>
  );
}

function StationCard({ station, listening, onListen }: { station: Station; listening: boolean; onListen: () => void }) {
  const np = station.now_playing;

  return (
    <Card className={cn("overflow-hidden transition", listening && "border-primary/40 shadow-[0_0_60px_-25px_var(--primary)]")}>
      <div className="flex gap-5 p-5">
        <div className="relative shrink-0">
          {np?.song.art ? (
            <img src={np.song.art} alt="" className="h-28 w-28 rounded-2xl object-cover" />
          ) : (
            <div className="grid h-28 w-28 place-items-center rounded-2xl bg-accent-soft text-accent">
              <Radio className="h-8 w-8" />
            </div>
          )}
          {listening && (
            <span className="absolute right-2 bottom-2 rounded-lg bg-bg/80 p-1.5 backdrop-blur">
              <Equalizer />
            </span>
          )}
        </div>
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="font-display text-lg font-semibold">{station.name}</h2>
            {station.online ? (
              station.live.is_live ? <Badge tone="accent">● Live DJ</Badge> : <Badge tone="lime">On air</Badge>
            ) : (
              <Badge>Offline</Badge>
            )}
          </div>
          {np ? (
            <div className="min-w-0">
              <p className="truncate font-medium">{np.song.title || np.song.text}</p>
              <p className="truncate text-sm text-muted">{np.song.artist}</p>
            </div>
          ) : (
            <p className="text-sm text-muted">{station.description || "Nothing scheduled right now."}</p>
          )}
          {np?.duration ? (
            <SongProgress key={`${np.song.id}-${np.played_at}-${np.elapsed}`} elapsed={np.elapsed} duration={np.duration} />
          ) : null}
          <p className="flex items-center gap-1.5 text-xs text-muted">
            <Users className="h-3.5 w-3.5" /> {station.listeners} listening
            {station.playing_next?.song.text && <span className="truncate">· Up next: {station.playing_next.song.text}</span>}
          </p>
        </div>
      </div>
      <div className="flex flex-wrap gap-2 border-t border-border bg-surface-2/40 px-5 py-3">
        <Button variant={listening ? "primary" : "secondary"} size="sm" onClick={onListen} disabled={!station.online}>
          {listening ? <Pause className="h-4 w-4" /> : <Headphones className="h-4 w-4" />}
          {listening ? "Stop listening" : "Listen here"}
        </Button>
        <SendToServer station={station} />
      </div>
    </Card>
  );
}

function SongProgress({ elapsed, duration }: { elapsed: number; duration: number }) {
  const [seconds, setSeconds] = useState(elapsed);

  useEffect(() => {
    const id = setInterval(() => setSeconds((s) => Math.min(s + 1, duration)), 1000);
    return () => clearInterval(id);
  }, [duration]);

  return (
    <div className="h-1.5 overflow-hidden rounded-full bg-surface-2">
      <div
        className="h-full rounded-full bg-linear-to-r from-primary to-accent transition-[width] duration-1000 ease-linear"
        style={{ width: `${Math.min(100, (seconds / duration) * 100)}%` }}
      />
    </div>
  );
}

function SendToServer({ station }: { station: Station }) {
  const { data: me } = useMe();
  const [open, setOpen] = useState(false);
  const { data: guilds, isLoading } = useGuilds(open && !!me);
  const [sending, setSending] = useState<string | null>(null);

  if (!me) {
    return (
      <a href={loginUrl("/radio")} className="inline-flex h-8 items-center gap-1.5 rounded-lg px-3 text-sm text-muted hover:bg-surface-2 hover:text-fg">
        <Send className="h-4 w-4" /> Log in to play in your server
      </a>
    );
  }

  const send = async (guildId: string) => {
    setSending(guildId);
    try {
      await api(`/guilds/${guildId}/radio`, { method: "POST", body: json({ station: station.shortcode }) });
      toast.success(`Tuned into ${station.name}`);
      setOpen(false);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Couldn't start the station");
    } finally {
      setSending(null);
    }
  };

  const candidates = (guilds ?? []).filter((g) => g.user_voice_channel_id || g.listening_with_bot || g.can_manage);

  return (
    <div className="relative">
      <Button variant="ghost" size="sm" onClick={() => setOpen((v) => !v)} disabled={!station.online}>
        <Play className="h-4 w-4" /> Play in my server
      </Button>
      {open && (
        <div className="absolute bottom-full left-0 z-20 mb-2 w-72 rounded-2xl border border-border bg-surface p-2 shadow-2xl">
          {isLoading ? (
            <p className="p-3 text-sm text-muted">Loading servers…</p>
          ) : candidates.length === 0 ? (
            <p className="p-3 text-sm text-muted">Join a voice channel in a server that has Chilly, then try again.</p>
          ) : (
            candidates.map((g) => (
              <button
                key={g.id}
                onClick={() => send(g.id)}
                disabled={sending !== null}
                className="flex w-full items-center justify-between rounded-xl px-3 py-2 text-left text-sm hover:bg-surface-2 disabled:opacity-50 cursor-pointer"
              >
                <span className="truncate">{g.name}</span>
                {sending === g.id && <span className="text-xs text-muted">Starting…</span>}
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}
