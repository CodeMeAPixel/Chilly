"use client";

import { useQuery } from "@tanstack/react-query";
import { Activity, Clock, Cpu, Gauge, HardDrive, Server, Signal, Users, WifiOff } from "lucide-react";
import { useEffect, useState } from "react";
import { Blobs } from "@/components/decor";
import { api } from "@/lib/api";
import { cn, formatNumber } from "@/lib/format";
import type { NodeInfo, StatusComponent, SystemStatus } from "@/lib/types";

const overall = {
  operational: { label: "All systems chill", body: "Everything is running smoothly.", dot: "bg-lime", tint: "from-lime/25" },
  degraded: { label: "Partially degraded", body: "Some parts are having trouble. Music may still play.", dot: "bg-peach", tint: "from-peach/30" },
  outage: { label: "Major outage", body: "We're on it. Playback is likely affected.", dot: "bg-accent", tint: "from-accent/30" },
} as const;

function formatUptime(seconds: number) {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 MB";
  const mb = bytes / 1024 / 1024;
  return mb >= 1024 ? `${(mb / 1024).toFixed(1)} GB` : `${Math.round(mb)} MB`;
}

function useNow(interval = 1000) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), interval);
    return () => clearInterval(id);
  }, [interval]);
  return now;
}

export function StatusBoard({ initial }: { initial: SystemStatus | null }) {
  const { data: status, isError } = useQuery({
    queryKey: ["status"],
    queryFn: () => api<SystemStatus>("/status"),
    initialData: initial ?? undefined,
    refetchInterval: 15_000,
  });
  const now = useNow();

  if (!status) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-24 sm:px-6">
        <div className="rounded-[28px] border border-border bg-surface p-10 text-center">
          <WifiOff className="mx-auto mb-4 h-8 w-8 text-accent" />
          <h1 className="font-display text-3xl font-semibold">Status unavailable</h1>
          <p className="mt-2 text-muted">We can&apos;t reach the Chilly API right now{isError ? "" : ", retrying"}…</p>
        </div>
      </div>
    );
  }

  const o = overall[status.status];
  const checkedAgo = Math.max(0, Math.round((now - new Date(status.checked_at).getTime()) / 1000));
  const groups = status.components.reduce<Record<string, StatusComponent[]>>((acc, c) => {
    (acc[c.group] ??= []).push(c);
    return acc;
  }, {});

  return (
    <div className="pb-8">
      <section className="grain relative -mt-18 overflow-hidden">
        <Blobs className="opacity-60" />
        <div className="relative mx-auto max-w-4xl px-4 pt-34 pb-12 sm:px-6">
          <p className="font-mono text-xs tracking-widest text-accent uppercase">System status</p>
          <div className={cn("mt-4 rounded-[32px] border border-border bg-linear-to-br to-surface p-8 sm:p-10", o.tint)}>
            <div className="flex items-center gap-4">
              <span className="relative grid h-12 w-12 shrink-0 place-items-center rounded-full bg-surface">
                <span className={cn("pulse-ring absolute h-4 w-4 rounded-full", o.dot)} />
                <span className={cn("relative h-4 w-4 rounded-full", o.dot)} />
              </span>
              <div>
                <h1 className="font-display text-3xl font-semibold tracking-tight sm:text-4xl">{o.label}</h1>
                <p className="text-muted">{o.body}</p>
              </div>
            </div>
            <p className="mt-6 flex items-center gap-1.5 text-xs text-muted">
              <Clock className="h-3.5 w-3.5" />
              {status.checked_at && new Date(status.checked_at).getFullYear() > 2000
                ? `Checked ${checkedAgo}s ago · refreshes automatically`
                : "Waiting for the first check…"}
            </p>
          </div>
        </div>
      </section>

      <div className="mx-auto max-w-4xl space-y-12 px-4 sm:px-6">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Metric icon={Server} label="Servers" value={formatNumber(status.bot.guilds)} />
          <Metric icon={Users} label="Playing now" value={formatNumber(status.bot.playing)} />
          <Metric
            icon={Signal}
            label="Discord latency"
            value={status.bot.gateway_latency_ms ? `${status.bot.gateway_latency_ms} ms` : "—"}
          />
          <Metric icon={Activity} label="Bot uptime" value={formatUptime(status.bot.uptime_seconds)} />
        </div>

        {Object.entries(groups).map(([group, components]) => (
          <section key={group} className="space-y-3">
            <h2 className="font-display text-xl font-semibold">{group}</h2>
            <div className="divide-y divide-border overflow-hidden rounded-[24px] border border-border bg-surface">
              {components.map((c) => (
                <ComponentRow key={c.id} component={c} bucketMinutes={status.bucket_minutes} />
              ))}
            </div>
          </section>
        ))}

        {status.nodes.length > 0 && (
          <section className="space-y-3">
            <h2 className="font-display text-xl font-semibold">Audio node details</h2>
            <div className="grid gap-4 md:grid-cols-2">
              {status.nodes.map((node) => (
                <NodeCard key={node.name} node={node} />
              ))}
            </div>
          </section>
        )}

        <p className="text-center text-xs text-muted">
          Tracking since {new Date(status.tracking_since).toLocaleString()} · history covers the last 24 hours and resets when the bot restarts.
        </p>
      </div>
    </div>
  );
}

function Metric({ icon: Icon, label, value }: { icon: typeof Server; label: string; value: string }) {
  return (
    <div className="rounded-[20px] border border-border bg-surface p-4">
      <p className="flex items-center gap-1.5 text-xs text-muted">
        <Icon className="h-3.5 w-3.5" /> {label}
      </p>
      <p className="mt-1 font-display text-2xl font-semibold">{value}</p>
    </div>
  );
}

function barColor(uptime: number | null) {
  if (uptime === null) return "bg-surface-2";
  if (uptime >= 0.999) return "bg-lime";
  if (uptime >= 0.9) return "bg-peach";
  return "bg-accent";
}

function ComponentRow({ component, bucketMinutes }: { component: StatusComponent; bucketMinutes: number }) {
  const up = component.status === "operational";
  return (
    <div className="space-y-3 p-5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2.5">
          <span className={cn("h-2.5 w-2.5 rounded-full", up ? "bg-lime" : "bg-accent")} />
          <span className="font-medium">{component.name}</span>
          <span className="text-sm text-muted">· {component.detail}</span>
        </div>
        <span className="font-mono text-xs text-muted">
          {component.uptime === null ? "no data" : `${component.uptime.toFixed(component.uptime === 100 ? 0 : 2)}% uptime`}
        </span>
      </div>
      <div className="flex h-8 items-stretch gap-[2px]">
        {component.history.map((point) => {
          const start = new Date(point.start);
          const label =
            point.uptime === null
              ? `${start.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })} · no data`
              : `${start.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })} · ${(point.uptime * 100).toFixed(1)}% up over ${bucketMinutes}m`;
          return (
            <span
              key={point.start}
              title={label}
              className={cn("flex-1 rounded-[3px] transition hover:opacity-70", barColor(point.uptime))}
            />
          );
        })}
      </div>
      <div className="flex justify-between text-[11px] text-muted">
        <span>24h ago</span>
        <span>now</span>
      </div>
    </div>
  );
}

function NodeCard({ node }: { node: NodeInfo }) {
  const up = node.status === "CONNECTED";
  const memoryLimit = node.memory_reservable || node.memory_allocated;
  const memory = memoryLimit ? node.memory_used / memoryLimit : 0;
  const frameLoss = node.frames_sent ? ((node.frames_nulled + node.frames_deficit) / node.frames_sent) * 100 : 0;

  return (
    <div className="rounded-[24px] border border-border bg-surface p-5">
      <div className="flex items-center justify-between">
        <p className="font-mono text-sm font-semibold">{node.name}</p>
        <span
          className={cn(
            "rounded-full px-2.5 py-0.5 text-xs font-medium",
            up ? "bg-lime/15 text-lime" : "bg-accent-soft text-accent",
          )}
        >
          {up ? "Online" : node.status.toLowerCase()}
        </span>
      </div>

      {up ? (
        <div className="mt-5 space-y-4">
          <div className="grid grid-cols-3 gap-3 text-sm">
            <Stat label="Players" value={`${node.playing_players}/${node.players}`} />
            <Stat label="Uptime" value={formatUptime(node.uptime_ms / 1000)} />
            <Stat label="Frame loss" value={`${frameLoss.toFixed(1)}%`} warn={frameLoss > 2} />
          </div>
          <Bar icon={Cpu} label="Lavalink CPU" value={node.cpu_load} detail={`${(node.cpu_load * 100).toFixed(1)}%`} />
          <Bar icon={Gauge} label="Host CPU" value={node.system_load} detail={`${(node.system_load * 100).toFixed(1)}% · ${node.cores} cores`} />
          <Bar icon={HardDrive} label="Heap memory" value={memory} detail={`${formatBytes(node.memory_used)} / ${formatBytes(memoryLimit)}`} />
        </div>
      ) : (
        <p className="mt-5 text-sm text-muted">This node is unreachable. Players have been moved to a healthy node.</p>
      )}
    </div>
  );
}

function Stat({ label, value, warn }: { label: string; value: string; warn?: boolean }) {
  return (
    <div>
      <p className="text-xs text-muted">{label}</p>
      <p className={cn("font-display text-lg font-semibold", warn && "text-accent")}>{value}</p>
    </div>
  );
}

function Bar({ icon: Icon, label, value, detail }: { icon: typeof Cpu; label: string; value: number; detail: string }) {
  const pct = Math.min(100, Math.max(0, value * 100));
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between text-xs">
        <span className="flex items-center gap-1.5 text-muted">
          <Icon className="h-3.5 w-3.5" /> {label}
        </span>
        <span className="font-mono">{detail}</span>
      </div>
      <div className="h-1.5 overflow-hidden rounded-full bg-surface-2">
        <div
          className={cn("h-full rounded-full", pct > 85 ? "bg-accent" : pct > 60 ? "bg-peach" : "bg-primary")}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}
