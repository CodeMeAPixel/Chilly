"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2, SkipForward } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { Badge, Button } from "@/components/ui";
import { api, ApiError } from "@/lib/api";
import type { Station } from "@/lib/types";

export function StationControls() {
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState<string | null>(null);
  const { data: stations } = useQuery({
    queryKey: ["stations"],
    queryFn: async () => (await api<{ stations: Station[] }>("/radio/stations")).stations,
    refetchInterval: 15_000,
  });

  if (!stations?.length) {
    return null;
  }

  const skip = async (station: Station) => {
    const song = station.now_playing?.song.text || "the current song";
    if (!window.confirm(`Skip ${song} on ${station.name}? This skips it for every listener.`)) return;
    setBusy(station.shortcode);
    try {
      await api(`/admin/stations/${encodeURIComponent(station.shortcode)}/skip`, { method: "POST" });
      toast.success(`Skipped on ${station.name}`);
      setTimeout(() => queryClient.invalidateQueries({ queryKey: ["stations"] }), 3000);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Couldn't skip that song");
    } finally {
      setBusy(null);
    }
  };

  return (
    <ul className="mt-4 divide-y divide-border rounded-2xl border border-border">
      {stations.map((station) => (
        <li key={station.shortcode} className="flex items-center gap-3 px-4 py-2.5 text-sm">
          <div className="min-w-0 flex-1">
            <p className="flex items-center gap-2 font-medium">
              {station.name}
              {station.online ? <Badge tone="lime">On air</Badge> : <Badge>Offline</Badge>}
            </p>
            <p className="truncate text-xs text-muted">
              {station.now_playing?.song.text ?? "Nothing playing"} · {station.listeners} listening
            </p>
          </div>
          <Button
            variant="secondary"
            size="sm"
            disabled={!station.online || busy !== null}
            onClick={() => skip(station)}
            aria-label={`Skip the current song on ${station.name}`}
          >
            {busy === station.shortcode ? <Loader2 className="h-4 w-4 animate-spin" /> : <SkipForward className="h-4 w-4" />} Skip
          </Button>
        </li>
      ))}
    </ul>
  );
}
