"use client";

import { useQuery } from "@tanstack/react-query";
import { ChevronRight, RefreshCw, ScrollText } from "lucide-react";
import { useMemo, useState } from "react";
import { Button, Card, EmptyState, Input, Skeleton } from "@/components/ui";
import { api } from "@/lib/api";
import { cn } from "@/lib/format";
import type { LogEntry } from "@/lib/types";

const levels = ["info", "warn", "error"] as const;

const levelStyle: Record<string, string> = {
  ERROR: "bg-danger/15 text-danger",
  WARN: "bg-peach/25 text-fg",
  INFO: "bg-primary-soft text-fg",
  DEBUG: "bg-surface-2 text-muted",
};

export function LogsTab() {
  const [level, setLevel] = useState<(typeof levels)[number]>("warn");
  const [live, setLive] = useState(true);
  const [filter, setFilter] = useState("");
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["admin", "logs", level],
    queryFn: async () => (await api<{ entries: LogEntry[] }>(`/admin/logs?level=${level}&limit=500`)).entries,
    refetchInterval: live ? 5_000 : false,
  });

  const entries = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return data ?? [];
    return (data ?? []).filter(
      (e) => e.message.toLowerCase().includes(q) || Object.values(e.attrs ?? {}).some((v) => v.toLowerCase().includes(q)),
    );
  }, [data, filter]);

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <Input value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="Filter by message, guild ID, node…" className="sm:flex-1" />
        <div className="flex items-center gap-2">
          <div className="flex rounded-xl border border-border bg-surface p-1">
            {levels.map((l) => (
              <button
                key={l}
                onClick={() => setLevel(l)}
                className={cn(
                  "h-9 cursor-pointer rounded-lg px-3 text-sm capitalize transition",
                  level === l ? "bg-primary-soft text-fg" : "text-muted hover:text-fg",
                )}
              >
                {l}+
              </button>
            ))}
          </div>
          <label className="flex cursor-pointer items-center gap-2 text-sm text-muted">
            <input type="checkbox" checked={live} onChange={(e) => setLive(e.target.checked)} className="accent-(--primary)" />
            Live
          </label>
          <Button variant="ghost" size="icon" onClick={() => refetch()} aria-label="Refresh">
            <RefreshCw className={cn("h-4 w-4", isFetching && "animate-spin")} />
          </Button>
        </div>
      </div>

      <p className="text-xs text-muted">The last 1,000 entries at info level and above are kept in memory and cleared when the bot restarts.</p>

      {isLoading ? (
        <Skeleton className="h-96 rounded-3xl" />
      ) : error ? (
        <p className="text-sm text-danger">{error.message}</p>
      ) : entries.length === 0 ? (
        <Card>
          <EmptyState icon={<ScrollText className="h-6 w-6" />} title="Nothing logged">
            No entries at this level yet.
          </EmptyState>
        </Card>
      ) : (
        <Card className="divide-y divide-border overflow-hidden">
          {entries.map((entry, i) => (
            <LogRow key={`${entry.time}-${i}`} entry={entry} />
          ))}
        </Card>
      )}
    </div>
  );
}

function LogRow({ entry }: { entry: LogEntry }) {
  const [open, setOpen] = useState(false);
  const attrs = Object.entries(entry.attrs ?? {});
  const summary = attrs
    .filter(([k]) => ["guild_id", "node", "station", "error", "command"].includes(k))
    .map(([k, v]) => `${k}=${v.length > 60 ? `${v.slice(0, 60)}…` : v}`)
    .join(" ");

  return (
    <div className="text-sm">
      <button
        onClick={() => setOpen((v) => !v)}
        disabled={attrs.length === 0}
        className="flex w-full cursor-pointer items-start gap-3 px-4 py-2.5 text-left hover:bg-surface-2/60 disabled:cursor-default"
      >
        <ChevronRight className={cn("mt-0.5 h-4 w-4 shrink-0 text-muted transition", open && "rotate-90", attrs.length === 0 && "invisible")} />
        <span className="w-20 shrink-0 font-mono text-xs text-muted">{new Date(entry.time).toLocaleTimeString()}</span>
        <span className={cn("shrink-0 rounded-md px-1.5 py-0.5 font-mono text-[10px] font-semibold", levelStyle[entry.level] ?? levelStyle.DEBUG)}>
          {entry.level}
        </span>
        <span className="min-w-0 flex-1">
          <span className="font-medium">{entry.message}</span>
          {summary && <span className="ml-2 font-mono text-xs break-all text-muted">{summary}</span>}
        </span>
      </button>
      {open && (
        <dl className="grid gap-x-4 gap-y-1 bg-surface-2/40 px-4 py-3 pl-11 font-mono text-xs sm:grid-cols-[max-content_1fr]">
          {attrs.map(([k, v]) => (
            <div key={k} className="contents">
              <dt className="text-muted">{k}</dt>
              <dd className="break-all whitespace-pre-wrap">{v}</dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  );
}
