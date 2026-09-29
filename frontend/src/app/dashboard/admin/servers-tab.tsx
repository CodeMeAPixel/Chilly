"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ExternalLink, LogOut, Moon, PhoneOff, Search, Server } from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { Badge, Button, Card, EmptyState, Equalizer, Input, Skeleton } from "@/components/ui";
import { api, ApiError, json } from "@/lib/api";
import { cn, formatNumber, initials } from "@/lib/format";
import type { AdminGuild } from "@/lib/types";
import { useAdminOverview } from "./admin-panel";

type Filter = "all" | "playing" | "stay";

export function ServersTab() {
  const { data: guilds, isLoading, error } = useQuery({
    queryKey: ["admin", "guilds"],
    queryFn: async () => (await api<{ guilds: AdminGuild[] }>("/admin/guilds")).guilds,
    refetchInterval: 15_000,
  });
  const { data: overview } = useAdminOverview();
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<Filter>("all");

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (guilds ?? []).filter((g) => {
      if (filter === "playing" && !g.playing) return false;
      if (filter === "stay" && !g.stay) return false;
      return !q || g.name.toLowerCase().includes(q) || g.id.includes(q);
    });
  }, [guilds, query, filter]);

  if (isLoading) {
    return <Skeleton className="h-96 rounded-3xl" />;
  }
  if (error) {
    return <p className="text-sm text-danger">{error.message}</p>;
  }

  const nodes = (overview?.nodes ?? []).filter((n) => n.status === "CONNECTED").map((n) => n.name);

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-4 h-4 w-4 -translate-y-1/2 text-muted" />
          <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search by name or ID" className="pl-11" />
        </div>
        <div className="flex rounded-xl border border-border bg-surface p-1">
          {(
            [
              ["all", `All ${guilds?.length ?? 0}`],
              ["playing", `Playing ${guilds?.filter((g) => g.playing).length ?? 0}`],
              ["stay", `24/7 ${guilds?.filter((g) => g.stay).length ?? 0}`],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              onClick={() => setFilter(value)}
              className={cn(
                "h-9 cursor-pointer rounded-lg px-3 text-sm transition",
                filter === value ? "bg-primary-soft text-fg" : "text-muted hover:text-fg",
              )}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {visible.length === 0 ? (
        <Card>
          <EmptyState icon={<Server className="h-6 w-6" />} title="No servers match" />
        </Card>
      ) : (
        <div className="space-y-2">
          {visible.map((g) => (
            <GuildRow key={g.id} guild={g} nodes={nodes} />
          ))}
        </div>
      )}
    </div>
  );
}

function GuildRow({ guild, nodes }: { guild: AdminGuild; nodes: string[] }) {
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState<string | null>(null);

  const act = async (name: string, path: string, body?: unknown, success?: string) => {
    setBusy(name);
    try {
      await api(`/admin/guilds/${guild.id}/${path}`, { method: "POST", body: body ? json(body) : undefined });
      if (success) toast.success(success);
      await queryClient.invalidateQueries({ queryKey: ["admin"] });
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Action failed");
    } finally {
      setBusy(null);
    }
  };

  const leave = () => {
    const typed = window.prompt(`This removes Chilly from "${guild.name}". Type the server name to confirm.`);
    if (typed === null) return;
    if (typed.trim() !== guild.name) {
      toast.error("Server name didn't match. Nothing was changed.");
      return;
    }
    act("leave", "leave", undefined, `Left ${guild.name}`);
  };

  return (
    <Card className="flex flex-col gap-4 p-4 lg:flex-row lg:items-center">
      <div className="flex min-w-0 flex-1 items-center gap-3">
        {guild.icon_url ? (
          <img src={guild.icon_url} alt="" className="h-11 w-11 shrink-0 rounded-2xl" />
        ) : (
          <div className="grid h-11 w-11 shrink-0 place-items-center rounded-2xl bg-surface-2 text-xs font-semibold text-muted">
            {initials(guild.name)}
          </div>
        )}
        <div className="min-w-0">
          <p className="flex items-center gap-2 truncate font-medium">
            {guild.name}
            {guild.stay && (
              <Badge tone="primary" className="shrink-0">
                <Moon className="h-3 w-3" /> 24/7
              </Badge>
            )}
          </p>
          <p className="truncate text-xs text-muted">
            {formatNumber(guild.member_count)} members · joined {new Date(guild.joined_at).toLocaleDateString()} ·{" "}
            <span className="font-mono">{guild.id}</span>
          </p>
          {guild.playing ? (
            <p className="mt-1 flex items-center gap-2 truncate text-xs">
              <Equalizer className="h-3" paused={guild.paused} />
              <span className="truncate">{guild.current}</span>
              <span className="shrink-0 text-muted">
                · {guild.listeners} listening{guild.node ? ` · ${guild.node}` : ""}
                {guild.queue_length ? ` · ${guild.queue_length} queued` : ""}
              </span>
            </p>
          ) : guild.voice_channel_id ? (
            <p className="mt-1 text-xs text-muted">In voice, idle · {guild.listeners} listening</p>
          ) : null}
          {guild.stay && guild.stay.health.failures > 0 && (
            <p className="mt-1 truncate text-xs text-danger">24/7 failing: {guild.stay.health.last_error}</p>
          )}
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Link href={`/dashboard/${guild.id}`} className="inline-flex h-8 items-center gap-1.5 rounded-lg px-3 text-sm text-muted hover:bg-surface-2 hover:text-fg">
          <ExternalLink className="h-4 w-4" /> Player
        </Link>
        {guild.playing && nodes.length > 1 && (
          <select
            value={guild.node ?? ""}
            disabled={busy !== null}
            onChange={(e) => act("move", "move", { node: e.target.value }, `Moved to ${e.target.value}`)}
            aria-label="Move to node"
            className="h-8 cursor-pointer rounded-lg border border-border bg-surface px-2 text-sm"
          >
            {nodes.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        )}
        {(guild.voice_channel_id || guild.stay) && (
          <Button
            variant="ghost"
            size="sm"
            disabled={busy !== null}
            onClick={() => act("disconnect", "disconnect", undefined, "Disconnected")}
          >
            <PhoneOff className="h-4 w-4" /> Disconnect
          </Button>
        )}
        <Button variant="danger" size="sm" disabled={busy !== null} onClick={leave}>
          <LogOut className="h-4 w-4" /> Leave
        </Button>
      </div>
    </Card>
  );
}
