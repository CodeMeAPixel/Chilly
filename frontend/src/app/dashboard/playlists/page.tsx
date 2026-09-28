import type { Metadata } from "next";
import { PlaylistIndex } from "./playlist-index";

export const metadata: Metadata = { title: "Playlists" };

export default function PlaylistsPage() {
  return (
    <div className="space-y-8">
      <div className="space-y-1">
        <h1 className="font-display text-3xl font-semibold tracking-tight">Your playlists</h1>
        <p className="text-muted">Playlists are yours across every server. Queue them from any player.</p>
      </div>
      <PlaylistIndex />
    </div>
  );
}
