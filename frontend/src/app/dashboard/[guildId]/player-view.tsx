"use client";

import { ArrowLeft, Lock, ServerCrash, WifiOff } from "lucide-react";
import Link from "next/link";
import { GuildAvatar } from "@/components/guild-avatar";
import { Badge, buttonClass, Card, EmptyState, Skeleton } from "@/components/ui";
import { useGuilds } from "@/hooks/use-me";
import { usePlayer } from "@/hooks/use-player";
import { loginUrl } from "@/lib/api";
import { AddTracks } from "./add-tracks";
import { NowPlaying } from "./now-playing";
import { QueueList, type QueueTab } from "./queue-list";

export function PlayerView({ guildId, initialTab }: { guildId: string; initialTab?: QueueTab }) {
  const { state, status, error, receivedAt, actions } = usePlayer(guildId);
  const { data: guilds } = useGuilds();
  const guild = guilds?.find((g) => g.id === guildId);

  if (error) {
    const unauthorized = error.status === 401;
    return (
      <Card>
        <EmptyState
          icon={unauthorized ? <Lock className="h-6 w-6" /> : <ServerCrash className="h-6 w-6" />}
          title={unauthorized ? "Session expired" : "Can't open this server"}
        >
          <p>{error.message}</p>
          <div className="mt-4">
            {unauthorized ? (
              <a href={loginUrl(`/dashboard/${guildId}`)} className={buttonClass("primary", "md")}>Log in again</a>
            ) : (
              <Link href="/dashboard" className={buttonClass("secondary", "md")}>Back to servers</Link>
            )}
          </div>
        </EmptyState>
      </Card>
    );
  }

  if (!state) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-72 rounded-3xl" />
        <Skeleton className="h-96 rounded-3xl" />
      </div>
    );
  }

  const canAdd = state.can_control || !!guild?.user_voice_channel_id;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <Link href="/dashboard" className={buttonClass("ghost", "icon", "lg:hidden")} aria-label="Back">
            <ArrowLeft className="h-5 w-5" />
          </Link>
          {guild && <GuildAvatar guild={guild} />}
          <div>
            <h1 className="font-display text-2xl font-semibold tracking-tight">{guild?.name ?? "Player"}</h1>
            <p className="text-sm text-muted">
              {state.playing ? `${state.queue_length} track${state.queue_length === 1 ? "" : "s"} up next` : "Nothing playing"}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {status === "live" ? (
            <Badge tone="lime">● Live</Badge>
          ) : status === "reconnecting" ? (
            <Badge tone="accent"><WifiOff className="h-3 w-3" /> Reconnecting</Badge>
          ) : (
            <Badge>Connecting</Badge>
          )}
          {!state.can_control && <Badge><Lock className="h-3 w-3" /> View only</Badge>}
        </div>
      </div>

      {!state.can_control && (
        <p className="rounded-2xl border border-border bg-surface-2/60 px-4 py-3 text-sm text-muted">
          Join the voice channel Chilly is in to control playback. Server managers can always control it.
        </p>
      )}

      <NowPlaying state={state} receivedAt={receivedAt} actions={actions} />

      <div className="grid gap-6 xl:grid-cols-[1fr_400px]">
        <QueueList guildId={guildId} state={state} receivedAt={receivedAt} actions={actions} initialTab={initialTab} />
        <AddTracks canAdd={canAdd} actions={actions} />
      </div>
    </div>
  );
}
