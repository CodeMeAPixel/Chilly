import type { Metadata } from "next";
import { PlayerView } from "./player-view";

export const metadata: Metadata = { title: "Player" };

export default async function GuildPage(props: PageProps<"/dashboard/[guildId]">) {
  const { guildId } = await props.params;
  const { lyrics } = await props.searchParams;
  return <PlayerView guildId={guildId} initialTab={lyrics ? "lyrics" : "queue"} />;
}
