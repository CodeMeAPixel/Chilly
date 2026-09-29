"use client";

import { useQuery } from "@tanstack/react-query";
import { Activity, Cpu, MapPin, Moon, Radio, Server, ShieldAlert, Users } from "lucide-react";
import { useState } from "react";
import { Badge, Card, EmptyState, Skeleton } from "@/components/ui";
import { useMe } from "@/hooks/use-me";
import { api } from "@/lib/api";
import { cn, formatNumber } from "@/lib/format";
import type { AdminOverview } from "@/lib/types";
import { LogsTab } from "./logs-tab";
import { RequestsTab } from "./requests-tab";
import { ServersTab } from "./servers-tab";
import { StationControls } from "./station-controls";
import { ToolsTab } from "./tools-tab";
import { formatBytes, formatSeconds } from "./format";

const tabs = [
  { id: "overview", label: "Overview" },
  { id: "servers", label: "Servers" },
  { id: "requests", label: "Requests" },
  { id: "tools", label: "Tools" },
  { id: "logs", label: "Logs" },
] as const;

type Tab = (typeof tabs)[number]["id"];

export function AdminPanel() {
  const { data: me, isLoading } = useMe();
  const [tab, setTab] = useState<Tab>("overview");

  if (isLoading) {
    return <Skeleton className="h-96 rounded-3xl" />;
  }
  if (!me?.admin) {
    return (
      <Card>
        <EmptyState icon={<ShieldAlert className="h-6 w-6" />} title="Admins only">
          This area is for Chilly&apos;s developers. Ask an admin to add your Discord ID to <code>API_ADMIN_USER_IDS</code>.
        </EmptyState>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h1 className="font-display text-3xl font-semibold tracking-tight">Admin</h1>
        <p className="text-sm text-muted">Inspect the bot, every server it&apos;s in, and debug playback.</p>
      </div>

      <div className="flex gap-1 overflow-x-auto rounded-2xl border border-border bg-surface p-1" role="tablist">
        {tabs.map((t) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={tab === t.id}
            onClick={() => setTab(t.id)}
            className={cn(
              "h-9 shrink-0 cursor-pointer rounded-xl px-4 text-sm transition",
              tab === t.id ? "bg-primary-soft font-medium text-fg" : "text-muted hover:text-fg",
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === "overview" && <OverviewTab />}
      {tab === "servers" && <ServersTab />}
      {tab === "requests" && <RequestsTab />}
      {tab === "tools" && <ToolsTab />}
      {tab === "logs" && <LogsTab />}
    </div>
  );
}

export function useAdminOverview() {
  return useQuery({
    queryKey: ["admin", "overview"],
    queryFn: () => api<AdminOverview>("/admin/overview"),
    refetchInterval: 10_000,
  });
}

function OverviewTab() {
  const { data, isLoading, error } = useAdminOverview();

  if (isLoading) {
    return <Skeleton className="h-96 rounded-3xl" />;
  }
  if (error || !data) {
    return <p className="text-sm text-danger">{error?.message ?? "Couldn't load the overview."}</p>;
  }

  const stats = [
    { label: "Servers", value: formatNumber(data.guilds), detail: `${formatNumber(data.members)} members`, icon: Server },
    { label: "Players", value: `${data.playing}/${data.players}`, detail: "playing / total", icon: Activity },
    { label: "24/7 servers", value: String(data.stays.length), detail: `${data.stays.filter((s) => s.health.failures > 0).length} failing`, icon: Moon },
    {
      label: "Gateway",
      value: `${data.gateway_latency_ms}ms`,
      detail: data.gateway_status.toLowerCase(),
      icon: Users,
    },
    { label: "Uptime", value: formatSeconds(data.uptime_seconds), detail: `since ${new Date(data.started_at).toLocaleString()}`, icon: Activity },
    { label: "Memory", value: formatBytes(data.heap_bytes), detail: `${formatBytes(data.sys_bytes)} from OS · ${data.goroutines} goroutines`, icon: Cpu },
  ];

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {stats.map((s) => (
          <Card key={s.label} className="p-4">
            <p className="flex items-center gap-1.5 text-xs text-muted">
              <s.icon className="h-3.5 w-3.5" /> {s.label}
            </p>
            <p className="mt-1 font-display text-2xl font-semibold">{s.value}</p>
            <p className="truncate text-xs text-muted">{s.detail}</p>
          </Card>
        ))}
      </div>

      <Card className="p-5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="flex items-center gap-2 font-display text-lg font-semibold">
            <Radio className="h-4 w-4 text-accent" /> Radio
          </h2>
          {data.radio.enabled ? (
            <Badge tone={data.radio.healthy ? "lime" : "accent"}>{data.radio.healthy ? "Polling OK" : "Polling failing"}</Badge>
          ) : (
            <Badge>Disabled</Badge>
          )}
        </div>
        {data.radio.enabled && (
          <p className="mt-2 text-sm text-muted">
            {data.radio.online} of {data.radio.stations} stations online
            {data.radio.last_poll && ` · last poll ${new Date(data.radio.last_poll).toLocaleTimeString()}`}
          </p>
        )}
        <p className="mt-1 text-sm text-muted">
          {data.library.enabled
            ? `Library: ${formatNumber(data.library.tracks)} songs${
                data.library.last_sync ? ` · synced ${new Date(data.library.last_sync).toLocaleTimeString()}` : " · not synced yet"
              }`
            : "Library: disabled (needs AzuraCast, an API key and MEDIA_BASE_URL)"}
        </p>
        {data.library.error && <p className="mt-1 text-xs text-danger">{data.library.error}</p>}
        <p className="mt-1 text-sm text-muted">
          {data.lyrics_backfill.enabled
            ? `Lyrics backfill: ${data.lyrics_backfill.written} saved, ${data.lyrics_backfill.not_found} not found, ${data.lyrics_backfill.pending} waiting${
                data.lyrics_backfill.last_run ? ` · last run ${new Date(data.lyrics_backfill.last_run).toLocaleTimeString()}` : " · first run pending"
              }`
            : "Lyrics backfill: off (set AZURACAST_LYRICS_BACKFILL=true)"}
        </p>
        {data.lyrics_backfill.last_error && <p className="mt-1 text-xs text-danger">{data.lyrics_backfill.last_error}</p>}
        {data.radio.enabled && <StationControls />}
        {data.stays.length > 0 && (
          <ul className="mt-4 divide-y divide-border rounded-2xl border border-border">
            {data.stays.map((stay) => (
              <li key={stay.guild_id} className="flex flex-wrap items-center gap-x-4 gap-y-1 px-4 py-2.5 text-sm">
                <span className="font-mono text-xs text-muted">{stay.guild_id}</span>
                <span className="font-medium">{stay.station_name}</span>
                <span className="text-xs text-muted">since {new Date(stay.enabled_at).toLocaleDateString()}</span>
                {stay.health.failures > 0 && (
                  <span className="basis-full truncate text-xs text-danger">
                    {stay.health.failures} failures · {stay.health.last_error}
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}
      </Card>

      <Card className="overflow-hidden">
        <h2 className="px-5 pt-5 font-display text-lg font-semibold">Audio nodes</h2>
        <div className="mt-3 overflow-x-auto">
          <table className="w-full min-w-160 text-sm">
            <thead className="text-left text-xs text-muted">
              <tr className="border-b border-border">
                <th className="px-5 py-2 font-medium">Node</th>
                <th className="px-3 py-2 font-medium">Status</th>
                <th className="px-3 py-2 font-medium">Players</th>
                <th className="px-3 py-2 font-medium">Lavalink CPU</th>
                <th className="px-3 py-2 font-medium">Host CPU</th>
                <th className="px-3 py-2 font-medium">Heap</th>
                <th className="px-5 py-2 font-medium">Uptime</th>
              </tr>
            </thead>
            <tbody>
              {data.nodes.map((n) => (
                <tr key={n.name} className="border-b border-border last:border-0">
                  <td className="px-5 py-2.5">
                    <p className="font-mono text-xs font-semibold">{n.name}</p>
                    {n.location && (
                      <p className="flex items-center gap-1 text-xs text-muted">
                        <MapPin className="h-3 w-3" /> {n.location}
                      </p>
                    )}
                  </td>
                  <td className="px-3 py-2.5">
                    <Badge tone={n.status === "CONNECTED" ? "lime" : "accent"}>{n.status.toLowerCase()}</Badge>
                  </td>
                  <td className="px-3 py-2.5 font-mono text-xs">
                    {n.playing_players}/{n.players}
                  </td>
                  <td className="px-3 py-2.5 font-mono text-xs">{(n.cpu_load * 100).toFixed(1)}%</td>
                  <td className="px-3 py-2.5 font-mono text-xs">
                    {(n.system_load * 100).toFixed(1)}% · {n.cores}c
                  </td>
                  <td className="px-3 py-2.5 font-mono text-xs">
                    {formatBytes(n.memory_used)} / {formatBytes(n.memory_reservable || n.memory_allocated)}
                  </td>
                  <td className="px-5 py-2.5 font-mono text-xs">{n.uptime_ms ? formatSeconds(n.uptime_ms / 1000) : "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <p className="text-xs text-muted">
        Chilly {data.version} · {data.go_version}
      </p>
    </div>
  );
}
