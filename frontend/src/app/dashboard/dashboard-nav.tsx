"use client";

import { LayoutGrid, ListMusic } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { GuildAvatar } from "@/components/guild-avatar";
import { Equalizer, Skeleton } from "@/components/ui";
import { useGuilds } from "@/hooks/use-me";
import { cn } from "@/lib/format";

export function DashboardNav() {
  const pathname = usePathname();
  const { data: guilds, isLoading } = useGuilds();

  const item = (href: string, active: boolean) =>
    cn(
      "flex items-center gap-3 rounded-xl px-3 py-2 text-sm transition",
      active ? "bg-primary-soft text-fg" : "text-muted hover:bg-surface-2 hover:text-fg",
    );

  return (
    <nav className="sticky top-24 space-y-6">
      <div className="space-y-1">
        <Link href="/dashboard" className={item("/dashboard", pathname === "/dashboard")}>
          <LayoutGrid className="h-4 w-4" /> Servers
        </Link>
        <Link href="/dashboard/playlists" className={item("/dashboard/playlists", pathname.startsWith("/dashboard/playlists"))}>
          <ListMusic className="h-4 w-4" /> Playlists
        </Link>
      </div>
      <div className="space-y-1">
        <p className="px-3 text-xs font-medium tracking-wide text-muted uppercase">Your servers</p>
        {isLoading &&
          Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="mx-3 h-9" />)}
        {guilds?.map((guild) => (
          <Link key={guild.id} href={`/dashboard/${guild.id}`} className={item(`/dashboard/${guild.id}`, pathname === `/dashboard/${guild.id}`)}>
            <GuildAvatar guild={guild} size="sm" />
            <span className="min-w-0 flex-1 truncate">{guild.name}</span>
            {guild.playing && <Equalizer className="h-3" />}
          </Link>
        ))}
      </div>
    </nav>
  );
}
