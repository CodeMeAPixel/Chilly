"use client";

import { Loader2, Play } from "lucide-react";
import { type ReactNode, useEffect, useRef, useState } from "react";
import { useGuilds } from "@/hooks/use-me";
import { cn } from "@/lib/format";
import { Button } from "./ui";

export function ServerPicker({
  onPick,
  disabled,
  label = "Play in server",
  icon = <Play className="h-4 w-4" />,
  variant = "primary",
  size = "md",
  align = "right",
}: {
  onPick: (guildId: string, guildName: string) => Promise<boolean>;
  disabled?: boolean;
  label?: string;
  icon?: ReactNode;
  variant?: "primary" | "secondary" | "ghost";
  size?: "sm" | "md";
  align?: "left" | "right";
}) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);
  const { data: guilds, isLoading } = useGuilds(open);
  const ref = useRef<HTMLDivElement>(null);
  const candidates = (guilds ?? []).filter((g) => g.user_voice_channel_id || g.listening_with_bot || g.can_manage);

  useEffect(() => {
    if (!open) return;
    const close = (e: PointerEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, [open]);

  const pick = async (guildId: string, name: string) => {
    setBusy(guildId);
    const ok = await onPick(guildId, name);
    setBusy(null);
    if (ok) setOpen(false);
  };

  return (
    <div ref={ref} className="relative">
      <Button variant={variant} size={size} onClick={() => setOpen((v) => !v)} disabled={disabled} aria-expanded={open}>
        {icon} {label}
      </Button>
      {open && (
        <div
          className={cn(
            "absolute z-30 mt-2 w-72 rounded-2xl border border-border bg-surface p-2 shadow-2xl",
            align === "right" ? "right-0" : "left-0",
          )}
        >
          {isLoading ? (
            <p className="p-3 text-sm text-muted">Loading servers…</p>
          ) : candidates.length === 0 ? (
            <p className="p-3 text-sm text-muted">Join a voice channel in a server with Chilly first.</p>
          ) : (
            candidates.map((g) => (
              <button
                key={g.id}
                onClick={() => pick(g.id, g.name)}
                disabled={busy !== null}
                className="flex w-full cursor-pointer items-center justify-between gap-2 rounded-xl px-3 py-2 text-left text-sm hover:bg-surface-2 disabled:opacity-50"
              >
                <span className="truncate">{g.name}</span>
                {busy === g.id ? (
                  <Loader2 className="h-4 w-4 animate-spin text-muted" />
                ) : g.user_voice_channel_id ? (
                  <span className="shrink-0 text-xs text-muted">In voice</span>
                ) : null}
              </button>
            ))
          )}
        </div>
      )}
    </div>
  );
}
