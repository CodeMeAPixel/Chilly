import { Music2, Radio } from "lucide-react";
import { cn } from "@/lib/format";
import type { Track } from "@/lib/types";

export function TrackArt({ track, className }: { track: Pick<Track, "artwork_url" | "radio_station"> | null; className?: string }) {
  if (track?.artwork_url) {
    return <img src={track.artwork_url} alt="" className={cn("shrink-0 object-cover", className)} />;
  }
  const Icon = track?.radio_station ? Radio : Music2;
  return (
    <div className={cn("grid shrink-0 place-items-center bg-linear-to-br from-primary-soft to-accent-soft text-primary", className)}>
      <Icon className="h-1/3 w-1/3" />
    </div>
  );
}
