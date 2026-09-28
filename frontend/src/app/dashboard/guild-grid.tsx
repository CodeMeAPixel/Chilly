"use client";

import { Headphones, Mic, Server } from "lucide-react";
import Link from "next/link";
import { GuildAvatar } from "@/components/guild-avatar";
import { Badge, Card, EmptyState, Equalizer, Skeleton } from "@/components/ui";
import { useGuilds } from "@/hooks/use-me";

export function GuildGrid() {
  const { data: guilds, isLoading, error } = useGuilds();

  if (isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-28 rounded-3xl" />
        ))}
      </div>
    );
  }

  if (error) {
    return <EmptyState icon={<Server className="h-6 w-6" />} title="Couldn't load your servers">{error.message}</EmptyState>;
  }

  if (!guilds?.length) {
    return (
      <Card>
        <EmptyState icon={<Server className="h-6 w-6" />} title="No servers yet">
          Chilly isn&apos;t in any of your servers. Invite it from the home page, then log out and back in to refresh your server list.
        </EmptyState>
      </Card>
    );
  }

  const sorted = [...guilds].sort((a, b) => Number(b.playing) - Number(a.playing) || a.name.localeCompare(b.name));

  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {sorted.map((guild) => (
        <Link key={guild.id} href={`/dashboard/${guild.id}`}>
          <Card className="group flex h-full items-center gap-4 p-5 transition hover:-translate-y-0.5 hover:border-primary/30">
            <GuildAvatar guild={guild} />
            <div className="min-w-0 flex-1 space-y-1.5">
              <p className="truncate font-medium">{guild.name}</p>
              {guild.playing ? (
                <p className="flex items-center gap-2 truncate text-sm text-muted">
                  <Equalizer className="h-3 shrink-0" /> <span className="truncate">{guild.current_title}</span>
                </p>
              ) : (
                <p className="text-sm text-muted">Idle</p>
              )}
              <div className="flex flex-wrap gap-1.5">
                {guild.listening_with_bot && (
                  <Badge tone="primary"><Headphones className="h-3 w-3" /> Listening</Badge>
                )}
                {!guild.listening_with_bot && guild.user_voice_channel_id && (
                  <Badge tone="lime"><Mic className="h-3 w-3" /> In voice</Badge>
                )}
                {guild.can_manage && <Badge>Manager</Badge>}
              </div>
            </div>
          </Card>
        </Link>
      ))}
    </div>
  );
}
