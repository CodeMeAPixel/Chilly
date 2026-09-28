import type { Metadata } from "next";
import { GuildGrid } from "./guild-grid";

export const metadata: Metadata = { title: "Servers" };

export default function DashboardPage() {
  return (
    <div className="space-y-8">
      <div className="space-y-1">
        <h1 className="font-display text-3xl font-semibold tracking-tight">Your servers</h1>
        <p className="text-muted">Pick a server to control the player. Only servers with Chilly are shown.</p>
      </div>
      <GuildGrid />
    </div>
  );
}
