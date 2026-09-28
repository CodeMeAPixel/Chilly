"use client";

import { Hash, Slash } from "lucide-react";
import Image from "next/image";
import { useState } from "react";
import { commandGroups } from "@/lib/commands";
import { cn } from "@/lib/format";

export function CommandExplorer() {
  const [group, setGroup] = useState(commandGroups[0].name);
  const [selected, setSelected] = useState(0);
  const active = commandGroups.find((g) => g.name === group) ?? commandGroups[0];
  const command = active.commands[Math.min(selected, active.commands.length - 1)];

  return (
    <div className="grid min-w-0 gap-6 lg:grid-cols-[220px_minmax(0,1fr)]">
      <div className="flex min-w-0 gap-2 overflow-x-auto pb-1 lg:flex-col lg:overflow-visible">
        {commandGroups.map((g) => (
          <button
            key={g.name}
            onClick={() => {
              setGroup(g.name);
              setSelected(0);
            }}
            className={cn(
              "flex shrink-0 items-center justify-between gap-3 rounded-2xl px-4 py-3 text-left text-sm transition cursor-pointer",
              group === g.name ? "bg-fg text-bg" : "text-muted hover:bg-surface hover:text-fg",
            )}
          >
            <span className="font-medium">{g.name}</span>
            <span className={cn("font-mono text-xs", group === g.name ? "text-bg/60" : "text-muted")}>{g.commands.length}</span>
          </button>
        ))}
      </div>

      <div className="min-w-0 overflow-hidden rounded-[28px] border border-border bg-surface">
        <div className="flex items-center gap-2 border-b border-border px-5 py-3 text-sm text-muted">
          <Hash className="h-4 w-4" /> chill-lounge
        </div>

        <div className="space-y-1 p-3">
          <p className="px-2 pt-1 pb-2 text-xs font-medium tracking-wide text-muted uppercase">Commands matching /{active.name.toLowerCase()}</p>
          {active.commands.map((c, i) => (
            <button
              key={c.name}
              onMouseEnter={() => setSelected(i)}
              onFocus={() => setSelected(i)}
              onClick={() => setSelected(i)}
              className={cn(
                "flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left transition cursor-pointer",
                c.name === command.name ? "bg-primary-soft" : "hover:bg-surface-2",
              )}
            >
              <Image src="/logo.svg" alt="" width={22} height={22} className="shrink-0" />
              <span className="shrink-0 font-mono text-sm font-semibold">{c.name}</span>
              <span className="min-w-0 truncate text-sm text-muted">{c.description}</span>
            </button>
          ))}
        </div>

        <div className="border-t border-border p-4">
          <div className="flex items-center gap-2 rounded-xl bg-surface-2 px-4 py-3 text-sm">
            <Slash className="h-4 w-4 text-primary" />
            <span className="font-mono font-semibold">{command.name.slice(1)}</span>
            <span className="h-4 w-px animate-pulse bg-fg" />
            <span className="ml-auto hidden truncate text-xs text-muted sm:block">{command.description}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
