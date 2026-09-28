"use client";

import { useQuery } from "@tanstack/react-query";
import { Loader2, MicVocal } from "lucide-react";
import { useEffect, useRef } from "react";
import { EmptyState } from "@/components/ui";
import { useLivePosition, type usePlayer } from "@/hooks/use-player";
import { api, ApiError } from "@/lib/api";
import { cn } from "@/lib/format";
import type { LyricsResult, PlayerState } from "@/lib/types";

type Actions = ReturnType<typeof usePlayer>["actions"];

const leadMs = 300;

export function LyricsPanel({
  guildId,
  state,
  receivedAt,
  actions,
}: {
  guildId: string;
  state: PlayerState;
  receivedAt: number;
  actions: Actions;
}) {
  const track = state.current;
  const trackKey = track ? `${track.source}:${track.identifier}:${track.title}` : "none";

  const { data, isLoading, error } = useQuery({
    queryKey: ["lyrics", guildId, trackKey],
    queryFn: () => api<LyricsResult>(`/guilds/${guildId}/player/lyrics`),
    enabled: !!track,
    staleTime: Infinity,
    retry: false,
  });

  if (!track) {
    return <EmptyState icon={<MicVocal className="h-6 w-6" />} title="Nothing playing">Lyrics show up here when a song is playing.</EmptyState>;
  }
  if (isLoading) {
    return (
      <div className="flex justify-center p-14 text-muted">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    );
  }
  if (error || !data) {
    const notFound = error instanceof ApiError && error.status === 404;
    return (
      <EmptyState icon={<MicVocal className="h-6 w-6" />} title={notFound ? "No lyrics found" : "Lyrics unavailable"}>
        {notFound ? "We couldn't find lyrics for this song." : "The lyrics service isn't responding right now."}
      </EmptyState>
    );
  }

  const { lyrics } = data;
  if (lyrics.instrumental) {
    return <EmptyState icon={<MicVocal className="h-6 w-6" />} title="Instrumental">This track has no lyrics.</EmptyState>;
  }

  const canSync = !!lyrics.synced?.length && !track.is_stream && !track.radio_station;

  return (
    <div className="flex flex-col">
      <div className="flex items-center justify-between gap-3 border-b border-border px-5 py-3 text-xs text-muted">
        <span className="truncate">
          <span className="font-medium text-fg">{lyrics.title}</span> · {lyrics.artist}
        </span>
        <span className="shrink-0">{canSync ? "Synced" : "Lyrics"} · {lyrics.source}</span>
      </div>
      {canSync ? (
        <SyncedLyrics lines={lyrics.synced!} state={state} receivedAt={receivedAt} onSeek={state.can_control ? actions.seek : undefined} />
      ) : (
        <p className="max-h-[560px] overflow-y-auto px-6 py-5 text-sm leading-7 whitespace-pre-line text-fg/90">{lyrics.plain}</p>
      )}
    </div>
  );
}

function SyncedLyrics({
  lines,
  state,
  receivedAt,
  onSeek,
}: {
  lines: NonNullable<LyricsResult["lyrics"]["synced"]>;
  state: PlayerState;
  receivedAt: number;
  onSeek?: (ms: number) => void;
}) {
  const position = useLivePosition(state, receivedAt);
  const container = useRef<HTMLDivElement>(null);
  const lineRefs = useRef<(HTMLButtonElement | null)[]>([]);

  let active = -1;
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].time_ms <= position + leadMs) active = i;
    else break;
  }

  useEffect(() => {
    const box = container.current;
    const line = active >= 0 ? lineRefs.current[active] : null;
    if (!box || !line) return;
    const target = line.offsetTop - box.clientHeight / 2 + line.clientHeight / 2;
    box.scrollTo({ top: target, behavior: "smooth" });
  }, [active]);

  return (
    <div ref={container} className="relative max-h-[560px] overflow-y-auto px-6 py-10 [mask-image:linear-gradient(to_bottom,transparent,black_12%,black_88%,transparent)]">
      {lines.map((line, i) => (
        <button
          key={`${i}-${line.time_ms}`}
          ref={(el) => {
            lineRefs.current[i] = el;
          }}
          disabled={!onSeek}
          onClick={() => onSeek?.(line.time_ms)}
          className={cn(
            "block w-full py-1.5 text-left font-display text-xl leading-snug transition-all duration-300 sm:text-2xl",
            onSeek && "cursor-pointer hover:text-fg",
            i === active ? "scale-[1.02] text-primary" : i < active ? "text-muted/60" : "text-muted",
            !line.text && "h-6",
          )}
        >
          {line.text || "♪"}
        </button>
      ))}
    </div>
  );
}
