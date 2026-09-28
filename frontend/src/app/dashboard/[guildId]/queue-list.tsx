"use client";

import { ChevronDown, ChevronUp, History, ListMusic, MicVocal, Trash2, X } from "lucide-react";
import { useState } from "react";
import { TrackArt } from "@/components/track-art";
import { Button, Card, EmptyState } from "@/components/ui";
import type { usePlayer } from "@/hooks/use-player";
import { cn, formatDuration } from "@/lib/format";
import type { PlayerState, Track } from "@/lib/types";
import { LyricsPanel } from "./lyrics-panel";

type Actions = ReturnType<typeof usePlayer>["actions"];

export type QueueTab = "queue" | "history" | "lyrics";

export function QueueList({
  guildId,
  state,
  receivedAt,
  actions,
  initialTab = "queue",
}: {
  guildId: string;
  state: PlayerState;
  receivedAt: number;
  actions: Actions;
  initialTab?: QueueTab;
}) {
  const [tab, setTab] = useState<QueueTab>(initialTab);
  const [dragFrom, setDragFrom] = useState<number | null>(null);
  const [dragOver, setDragOver] = useState<number | null>(null);
  const editable = state.can_control;
  const totalMs = state.queue.reduce((sum, t) => sum + (t.is_stream ? 0 : t.length_ms), 0);

  const drop = (to: number) => {
    if (dragFrom !== null && dragFrom !== to) {
      actions.move(dragFrom, to);
    }
    setDragFrom(null);
    setDragOver(null);
  };

  return (
    <Card className="overflow-hidden">
      <div className="flex items-center justify-between gap-3 border-b border-border px-5 py-3">
        <div className="flex gap-1 rounded-xl bg-surface-2 p-1">
          <TabButton active={tab === "queue"} onClick={() => setTab("queue")}>
            <ListMusic className="h-4 w-4" /> Up next <span className="text-muted">{state.queue_length}</span>
          </TabButton>
          <TabButton active={tab === "history"} onClick={() => setTab("history")}>
            <History className="h-4 w-4" /> History
          </TabButton>
          <TabButton active={tab === "lyrics"} onClick={() => setTab("lyrics")}>
            <MicVocal className="h-4 w-4" /> Lyrics
          </TabButton>
        </div>
        {tab === "queue" && state.queue.length > 0 && (
          <div className="flex items-center gap-3">
            <span className="hidden text-xs text-muted sm:block">{formatDuration(totalMs)} total</span>
            {editable && (
              <Button variant="danger" size="sm" onClick={actions.clearQueue}>
                <Trash2 className="h-4 w-4" /> Clear
              </Button>
            )}
          </div>
        )}
      </div>

      {tab === "lyrics" ? (
        <LyricsPanel guildId={guildId} state={state} receivedAt={receivedAt} actions={actions} />
      ) : tab === "queue" ? (
        state.queue.length === 0 ? (
          <EmptyState icon={<ListMusic className="h-6 w-6" />} title="The queue is empty">
            Search for something on the right to add it.
          </EmptyState>
        ) : (
          <ol className="max-h-[560px] divide-y divide-border overflow-y-auto">
            {state.queue.map((track, index) => (
              <li
                key={`${index}-${track.identifier}`}
                draggable={editable}
                onDragStart={() => setDragFrom(index)}
                onDragOver={(e) => {
                  e.preventDefault();
                  setDragOver(index);
                }}
                onDragLeave={() => setDragOver((cur) => (cur === index ? null : cur))}
                onDrop={() => drop(index)}
                onDragEnd={() => {
                  setDragFrom(null);
                  setDragOver(null);
                }}
                className={cn(
                  "group flex items-center gap-3 px-5 py-2.5 transition",
                  editable && "cursor-grab active:cursor-grabbing",
                  dragFrom === index && "opacity-40",
                  dragOver === index && dragFrom !== index && "bg-primary-soft",
                )}
              >
                <span className="w-6 text-right font-mono text-xs text-muted">{index + 1}</span>
                <QueueRow track={track} />
                {editable && (
                  <div className="flex items-center gap-0.5 opacity-100 transition sm:opacity-0 sm:group-hover:opacity-100">
                    <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="Move up" disabled={index === 0} onClick={() => actions.move(index, index - 1)}>
                      <ChevronUp className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="Move down" disabled={index === state.queue.length - 1} onClick={() => actions.move(index, index + 1)}>
                      <ChevronDown className="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" className="h-8 w-8 hover:text-danger" aria-label="Remove" onClick={() => actions.remove(index)}>
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                )}
              </li>
            ))}
            {state.queue_length > state.queue.length && (
              <li className="px-5 py-3 text-center text-sm text-muted">
                +{state.queue_length - state.queue.length} more
              </li>
            )}
          </ol>
        )
      ) : state.history.length === 0 ? (
        <EmptyState icon={<History className="h-6 w-6" />} title="Nothing played yet" />
      ) : (
        <ol className="max-h-[560px] divide-y divide-border overflow-y-auto">
          {state.history.map((track, index) => (
            <li key={`${index}-${track.identifier}`} className="flex items-center gap-3 px-5 py-2.5">
              <QueueRow track={track} />
            </li>
          ))}
        </ol>
      )}
    </Card>
  );
}

function QueueRow({ track }: { track: Track }) {
  return (
    <>
      <TrackArt track={track} className="h-10 w-10 rounded-lg" />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{track.title}</p>
        <p className="truncate text-xs text-muted">{track.author}</p>
      </div>
      <span className="font-mono text-xs text-muted">{track.is_stream ? "LIVE" : formatDuration(track.length_ms)}</span>
    </>
  );
}

function TabButton({ active, children, onClick }: { active: boolean; children: React.ReactNode; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition cursor-pointer",
        active ? "bg-surface text-fg shadow-sm" : "text-muted hover:text-fg",
      )}
    >
      {children}
    </button>
  );
}
