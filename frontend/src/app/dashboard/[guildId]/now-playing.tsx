"use client";

import {
  ExternalLink,
  Pause,
  Play,
  Repeat,
  Repeat1,
  Shuffle,
  SkipBack,
  SkipForward,
  Square,
  Volume1,
  Volume2,
  VolumeX,
} from "lucide-react";
import { useEffect, useEffectEvent, useState } from "react";
import { TrackArt } from "@/components/track-art";
import { Badge, Button, Card, Equalizer } from "@/components/ui";
import { useLivePosition, type usePlayer } from "@/hooks/use-player";
import { cn, formatDuration, sourceLabels } from "@/lib/format";
import type { LoopMode, PlayerState } from "@/lib/types";

type Actions = ReturnType<typeof usePlayer>["actions"];

const nextLoop: Record<LoopMode, LoopMode> = { none: "queue", queue: "track", track: "none" };

export function NowPlaying({
  state,
  receivedAt,
  actions,
}: {
  state: PlayerState;
  receivedAt: number;
  actions: Actions;
}) {
  const position = useLivePosition(state, receivedAt);
  const track = state.current;
  const disabled = !state.can_control || !track;

  return (
    <Card className="relative overflow-hidden">
      {track?.artwork_url && (
        <img src={track.artwork_url} alt="" className="pointer-events-none absolute inset-0 h-full w-full scale-125 object-cover opacity-20 blur-3xl" />
      )}
      <div className="relative flex flex-col gap-6 p-6 sm:flex-row sm:items-center sm:p-8">
        <TrackArt track={track} className="h-40 w-40 rounded-3xl shadow-2xl sm:h-48 sm:w-48" />

        <div className="min-w-0 flex-1 space-y-5">
          {track ? (
            <div className="space-y-2">
              <div className="flex flex-wrap items-center gap-2">
                {!state.paused && <Equalizer className="h-3.5" />}
                <Badge tone="primary">{sourceLabels[track.source] ?? track.source}</Badge>
                {track.is_stream && <Badge tone="accent">Live</Badge>}
                {track.playlist_name && <Badge>From {track.playlist_name}</Badge>}
              </div>
              <h2 className="line-clamp-2 font-display text-2xl font-semibold sm:text-3xl">{track.title}</h2>
              <p className="flex items-center gap-2 text-muted">
                <span className="truncate">{track.author}</span>
                {track.uri && (
                  <a href={track.uri} target="_blank" rel="noreferrer" className="shrink-0 hover:text-fg" aria-label="Open source">
                    <ExternalLink className="h-4 w-4" />
                  </a>
                )}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              <h2 className="font-display text-2xl font-semibold">Nothing playing</h2>
              <p className="text-muted">Add something from the search panel to get the party started.</p>
            </div>
          )}

          <Progress state={state} position={position} disabled={disabled} onSeek={actions.seek} />

          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="icon"
                aria-label="Shuffle"
                disabled={!state.can_control}
                onClick={actions.toggleShuffle}
                className={cn(state.shuffle && "text-primary")}
              >
                <Shuffle className="h-5 w-5" />
              </Button>
              <Button variant="ghost" size="icon" aria-label="Previous" disabled={disabled} onClick={actions.previous}>
                <SkipBack className="h-5 w-5" />
              </Button>
              <Button size="icon" aria-label={state.paused ? "Play" : "Pause"} disabled={disabled} onClick={actions.togglePause} className="h-12 w-12 rounded-2xl">
                {state.paused ? <Play className="h-5 w-5 fill-current" /> : <Pause className="h-5 w-5 fill-current" />}
              </Button>
              <Button variant="ghost" size="icon" aria-label="Skip" disabled={disabled} onClick={actions.skip}>
                <SkipForward className="h-5 w-5" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                aria-label={`Loop: ${state.loop}`}
                disabled={!state.can_control}
                onClick={() => actions.setLoop(nextLoop[state.loop])}
                className={cn(state.loop !== "none" && "text-primary")}
              >
                {state.loop === "track" ? <Repeat1 className="h-5 w-5" /> : <Repeat className="h-5 w-5" />}
              </Button>
              <Button variant="ghost" size="icon" aria-label="Stop" disabled={disabled} onClick={actions.stop}>
                <Square className="h-4 w-4 fill-current" />
              </Button>
            </div>
            <VolumeControl volume={state.volume} disabled={!state.can_control} onChange={actions.setVolume} />
          </div>
        </div>
      </div>
    </Card>
  );
}

function Progress({
  state,
  position,
  disabled,
  onSeek,
}: {
  state: PlayerState;
  position: number;
  disabled: boolean;
  onSeek: (ms: number) => void;
}) {
  const [dragging, setDragging] = useState<number | null>(null);
  const track = state.current;
  const length = track?.length_ms ?? 0;
  const live = !track || track.is_stream || length <= 0;
  const value = dragging ?? position;

  return (
    <div className="space-y-1.5">
      <input
        type="range"
        min={0}
        max={live ? 1 : length}
        value={live ? (track ? 1 : 0) : value}
        disabled={disabled || live}
        onChange={(e) => setDragging(Number(e.target.value))}
        onPointerUp={() => {
          if (dragging !== null) {
            onSeek(dragging);
            setDragging(null);
          }
        }}
        onKeyUp={() => {
          if (dragging !== null) {
            onSeek(dragging);
            setDragging(null);
          }
        }}
        aria-label="Seek"
        className="h-1.5 w-full cursor-pointer appearance-none rounded-full bg-surface-2 accent-primary disabled:cursor-default"
        style={{
          background: live
            ? undefined
            : `linear-gradient(to right, var(--primary) ${(value / Math.max(length, 1)) * 100}%, var(--surface-2) 0)`,
        }}
      />
      <div className="flex justify-between font-mono text-xs text-muted">
        <span>{track ? formatDuration(value) : "0:00"}</span>
        <span>{live ? (track ? "LIVE" : "0:00") : formatDuration(length)}</span>
      </div>
    </div>
  );
}

function VolumeControl({
  volume,
  disabled,
  onChange,
}: {
  volume: number;
  disabled: boolean;
  onChange: (v: number) => Promise<unknown> | unknown;
}) {
  const [draft, setDraft] = useState<number | null>(null);
  const local = draft ?? volume;
  const commit = useEffectEvent((value: number) => onChange(value));

  useEffect(() => {
    if (draft === null) return;
    const id = setTimeout(async () => {
      await commit(draft);
      setDraft(null);
    }, 300);
    return () => clearTimeout(id);
  }, [draft]);

  const Icon = local === 0 ? VolumeX : local < 60 ? Volume1 : Volume2;

  return (
    <div className="flex items-center gap-2">
      <button
        className="text-muted hover:text-fg disabled:opacity-50 cursor-pointer"
        disabled={disabled}
        onClick={() => setDraft(local === 0 ? 100 : 0)}
        aria-label="Mute"
      >
        <Icon className="h-5 w-5" />
      </button>
      <input
        type="range"
        min={0}
        max={150}
        value={local}
        disabled={disabled}
        onChange={(e) => setDraft(Number(e.target.value))}
        aria-label="Volume"
        className="h-1.5 w-28 cursor-pointer appearance-none rounded-full accent-primary"
        style={{ background: `linear-gradient(to right, var(--primary) ${(local / 150) * 100}%, var(--surface-2) 0)` }}
      />
      <span className="w-9 text-right font-mono text-xs text-muted">{local}%</span>
    </div>
  );
}
