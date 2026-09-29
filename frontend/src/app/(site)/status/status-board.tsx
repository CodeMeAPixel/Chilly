"use client";

import { useQuery } from "@tanstack/react-query";
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  Clock,
  Cpu,
  Gauge,
  HardDrive,
  Headphones,
  Megaphone,
  MapPin,
  Server,
  Signal,
  Wrench,
  WifiOff,
} from "lucide-react";
import { useEffect, useState } from "react";
import { Blobs } from "@/components/decor";
import { api } from "@/lib/api";
import { cn, formatNumber } from "@/lib/format";
import type { NodeInfo, StatusComponent, StatusIncident, StatusPoint, SystemStatus } from "@/lib/types";

type Range = "day" | "quarter";

const overall = {
  operational: { label: "All systems chill", body: "Everything is running smoothly.", dot: "bg-lime", tint: "from-lime/25" },
  degraded: { label: "Partially degraded", body: "Some parts are having trouble. Music may still play.", dot: "bg-peach", tint: "from-peach/30" },
  outage: { label: "Major outage", body: "We're on it. Playback is likely affected.", dot: "bg-accent", tint: "from-accent/30" },
} as const;

const groupOrder = ["Core", "Stations", "Services", "Audio nodes"];

function formatUptime(seconds: number) {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

function formatLength(seconds: number) {
  if (seconds < 60) return "under a minute";
  const m = Math.round(seconds / 60);
  if (m < 60) return `${m} min`;
  const h = Math.floor(m / 60);
  return `${h}h ${m % 60}m`;
}

function formatBytes(bytes: number) {
  if (!bytes) return "0 MB";
  const mb = bytes / 1024 / 1024;
  return mb >= 1024 ? `${(mb / 1024).toFixed(1)} GB` : `${Math.round(mb)} MB`;
}

function formatPercent(v: number) {
  return `${v.toFixed(v === 100 ? 0 : 2)}%`;
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
  const [range, setRange] = useState<Range>("day");

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
  const groups = groupOrder
    .map((group) => ({ group, components: status.components.filter((c) => c.group === group) }))
    .filter((g) => g.components.length > 0);
  const nodesByName = new Map(status.nodes.map((n) => [n.name, n]));
  const incidents = status.incidents ?? [];
  const notices = status.notices ?? [];
  const recentRecovery =
    status.status === "operational"
      ? incidents.find((i) => !i.ongoing && i.resolved_at && now - new Date(i.resolved_at).getTime() < 3_600_000)
      : undefined;

  return (
    <div className="pb-8">
      <section className="grain relative -mt-18 overflow-hidden">
        <Blobs className="opacity-60" />
        <div className="relative mx-auto max-w-4xl px-4 pt-34 pb-12 sm:px-6">
          <p className="font-mono text-xs tracking-widest text-accent uppercase">System status</p>
          <div className={cn("mt-4 rounded-4xl border border-border bg-linear-to-br to-surface p-8 sm:p-10", o.tint)}>
            <div className="flex items-center gap-4">
              <span className="relative grid h-12 w-12 shrink-0 place-items-center rounded-full bg-surface">
                <span className={cn("pulse-ring absolute h-4 w-4 rounded-full", o.dot)} />
                <span className={cn("relative h-4 w-4 rounded-full", o.dot)} />
              </span>
              <div>
                <h1 className="font-display text-3xl font-semibold tracking-tight sm:text-4xl">{o.label}</h1>
                <p className="text-muted">
                  {recentRecovery
                    ? `Recovered from a brief ${recentRecovery.component_name ?? "service"} outage ${Math.max(
                        1,
                        Math.round((now - new Date(recentRecovery.resolved_at!).getTime()) / 60000),
                      )} min ago.`
                    : o.body}
                </p>
              </div>
            </div>
            <p className="mt-6 flex items-center gap-1.5 text-xs text-muted">
              <Clock className="h-3.5 w-3.5" />
              {status.checked_at && new Date(status.checked_at).getFullYear() > 2000
                ? `Checked ${checkedAgo}s ago · refreshes automatically`
                : "Waiting for the first check…"}
            </p>
          </div>

          {notices.length > 0 && (
            <div className="mt-4 space-y-3">
              {notices.map((n) => (
                <Notice key={n.id} notice={n} />
              ))}
            </div>
          )}
        </div>
      </section>

      <div className="mx-auto max-w-4xl space-y-12 px-4 sm:px-6">
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Metric icon={Server} label="Servers" value={formatNumber(status.bot.guilds)} />
          <Metric
            icon={Headphones}
            label="Listening now"
            value={formatNumber(status.bot.listeners ?? status.bot.playing)}
          />
          <Metric
            icon={Signal}
            label="Discord latency"
            value={status.bot.gateway_latency_ms ? `${status.bot.gateway_latency_ms} ms` : "—"}
          />
          <Metric icon={Activity} label="Bot uptime" value={formatUptime(status.bot.uptime_seconds)} />
        </div>

        <div className="flex items-center justify-between gap-3">
          <h2 className="font-display text-xl font-semibold">Components</h2>
          <div className="flex rounded-xl border border-border bg-surface p-1 text-sm" role="tablist" aria-label="History range">
            {(
              [
                ["day", "24 hours"],
                ["quarter", `${status.history_days ?? 90} days`],
              ] as const
            ).map(([value, label]) => (
              <button
                key={value}
                role="tab"
                aria-selected={range === value}
                onClick={() => setRange(value)}
                className={cn(
                  "h-8 cursor-pointer rounded-lg px-3 transition",
                  range === value ? "bg-primary-soft text-fg" : "text-muted hover:text-fg",
                )}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        {groups.map(({ group, components }) => (
          <section key={group} className="-mt-6 space-y-3">
            <h3 className="text-sm font-medium text-muted">{group}</h3>
            {group === "Audio nodes" ? (
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                {components.map((c) => (
                  <NodeCard key={c.id} component={c} node={nodesByName.get(c.name)} range={range} />
                ))}
              </div>
            ) : (
              <div className="divide-y divide-border overflow-hidden rounded-3xl border border-border bg-surface">
                {components.map((c) => (
                  <ComponentRow key={c.id} component={c} range={range} bucketMinutes={status.bucket_minutes} />
                ))}
              </div>
            )}
          </section>
        ))}

        <section className="space-y-3">
          <h2 className="font-display text-xl font-semibold">Recent incidents</h2>
          {incidents.length === 0 ? (
            <div className="flex items-center gap-2 rounded-3xl border border-border bg-surface p-5 text-sm text-muted">
              <CheckCircle2 className="h-4 w-4 text-lime" /> No incidents in the last 14 days.
            </div>
          ) : (
            <ul className="divide-y divide-border overflow-hidden rounded-3xl border border-border bg-surface">
              {incidents.map((i) => (
                <IncidentRow key={i.id} incident={i} />
              ))}
            </ul>
          )}
        </section>

        <p className="text-center text-xs text-muted">
          History since {new Date(status.tracking_since).toLocaleDateString()} · the last {status.history_days ?? 90} days are
          kept.
        </p>
      </div>
    </div>
  );
}

function Notice({ notice }: { notice: StatusIncident }) {
  const maintenance = notice.kind === "maintenance";
  const Icon = maintenance ? Wrench : Megaphone;
  return (
    <div
      className={cn(
        "flex gap-3 rounded-3xl border p-5",
        maintenance ? "border-peach/40 bg-peach/10" : "border-primary/30 bg-primary-soft",
      )}
    >
      <Icon className={cn("mt-0.5 h-5 w-5 shrink-0", maintenance ? "text-peach" : "text-primary")} />
      <div className="min-w-0 space-y-1">
        <p className="font-medium">
          {maintenance && <span className="mr-2 text-xs font-semibold tracking-wide text-peach uppercase">Maintenance</span>}
          {notice.title}
        </p>
        {notice.message && <p className="text-sm whitespace-pre-line text-muted">{notice.message}</p>}
        <p className="text-xs text-muted">Posted {new Date(notice.started_at).toLocaleString()}</p>
      </div>
    </div>
  );
}

function IncidentRow({ incident }: { incident: StatusIncident }) {
  const started = new Date(incident.started_at);
  return (
    <li className="flex items-start gap-3 p-5">
      {incident.ongoing ? (
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-accent" />
      ) : (
        <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-lime" />
      )}
      <div className="min-w-0 flex-1">
        <p className="font-medium">{incident.title}</p>
        <p className="text-sm text-muted">
          {started.toLocaleDateString([], { month: "short", day: "numeric" })},{" "}
          {started.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })} ·{" "}
          {incident.ongoing ? `ongoing for ${formatLength(incident.duration_seconds)}` : `resolved after ${formatLength(incident.duration_seconds)}`}
        </p>
        {incident.kind !== "auto" && incident.message && (
          <p className="mt-1 text-sm whitespace-pre-line text-muted">{incident.message}</p>
        )}
      </div>
    </li>
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

function UptimeLabel({ component, range }: { component: StatusComponent; range: Range }) {
  const value = range === "day" ? component.uptime_24h : component.uptime;
  if (component.collecting || value === null) {
    return <span className="font-mono text-xs text-muted">collecting data</span>;
  }
  return <span className="font-mono text-xs text-muted">{formatPercent(value)} uptime</span>;
}

function UptimeBars({ points, range, bucketMinutes }: { points: StatusPoint[]; range: Range; bucketMinutes: number }) {
  return (
    <div>
      <div className="flex h-8 items-stretch gap-0.5">
        {points.map((point) => {
          const start = new Date(point.start);
          const when =
            range === "day"
              ? `${start.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}, ${bucketMinutes} min`
              : start.toLocaleDateString([], { month: "short", day: "numeric" });
          const label = point.uptime === null ? `${when} · no data` : `${when} · ${(point.uptime * 100).toFixed(1)}% up`;
          return (
            <span key={point.start} title={label} className={cn("flex-1 rounded-[3px] transition hover:opacity-70", barColor(point.uptime))} />
          );
        })}
      </div>
      <div className="mt-2 flex justify-between text-[11px] text-muted">
        <span>{range === "day" ? "24h ago" : `${points.length} days ago`}</span>
        <span>{range === "day" ? "now" : "today"}</span>
      </div>
    </div>
  );
}

function ComponentRow({ component, range, bucketMinutes }: { component: StatusComponent; range: Range; bucketMinutes: number }) {
  const up = component.status === "operational";
  return (
    <div className="space-y-3 p-5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full", up ? "bg-lime" : "bg-accent")} />
          <span className="truncate font-medium">{component.name}</span>
          <span className="truncate text-sm text-muted">· {component.detail}</span>
        </div>
        <UptimeLabel component={component} range={range} />
      </div>
      <UptimeBars points={range === "day" ? component.history : component.daily} range={range} bucketMinutes={bucketMinutes} />
    </div>
  );
}

function NodeCard({ component, node, range }: { component: StatusComponent; node?: NodeInfo; range: Range }) {
  const [open, setOpen] = useState(false);
  const up = component.status === "operational";
  const memoryLimit = node ? node.memory_reservable || node.memory_allocated : 0;
  const memory = node && memoryLimit ? node.memory_used / memoryLimit : 0;
  const frameLoss = node?.frames_sent ? ((node.frames_nulled + node.frames_deficit) / node.frames_sent) * 100 : 0;

  return (
    <div className="space-y-4 rounded-3xl border border-border bg-surface p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="flex items-center gap-2 font-medium">
            <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full", up ? "bg-lime" : "bg-accent")} />
            <span className="truncate">{node?.location || component.name}</span>
          </p>
          <p className="mt-0.5 flex items-center gap-1 truncate pl-4.5 text-xs text-muted">
            {node?.location && <MapPin className="h-3 w-3 shrink-0" />}
            <span className="font-mono">{component.name}</span>
            <span>· {component.detail}</span>
          </p>
        </div>
        <UptimeLabel component={component} range={range} />
      </div>

      <UptimeBars points={range === "day" ? component.history : component.daily} range={range} bucketMinutes={15} />

      {up && node ? (
        <>
          <div className="grid grid-cols-3 gap-3 text-sm">
            <Stat label="Players" value={`${node.playing_players}/${node.players}`} />
            <Stat label="Uptime" value={formatUptime(node.uptime_ms / 1000)} />
            <Stat label="Frame loss" value={`${frameLoss.toFixed(1)}%`} warn={frameLoss > 2} />
          </div>
          <button
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            className="flex cursor-pointer items-center gap-1 text-xs text-muted hover:text-fg"
          >
            Details <ChevronDown className={cn("h-3.5 w-3.5 transition", open && "rotate-180")} />
          </button>
          {open && (
            <div className="space-y-4">
              <Bar icon={Cpu} label="Lavalink CPU" value={node.cpu_load} detail={`${(node.cpu_load * 100).toFixed(1)}%`} />
              <Bar icon={Gauge} label="Host CPU" value={node.system_load} detail={`${(node.system_load * 100).toFixed(1)}% · ${node.cores} cores`} />
              <Bar
                icon={HardDrive}
                label="Heap memory"
                value={memory}
                detail={`${formatBytes(node.memory_used)} / ${formatBytes(memoryLimit)}`}
              />
            </div>
          )}
        </>
      ) : (
        <p className="text-sm text-muted">This node is unreachable. Players have been moved to a healthy node.</p>
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
