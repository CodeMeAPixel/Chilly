import type { Metadata } from "next";
import { PlaylistDetail } from "./playlist-detail";

export const metadata: Metadata = { title: "Playlist" };

export default async function PlaylistPage(props: PageProps<"/dashboard/playlists/[playlistId]">) {
  const { playlistId } = await props.params;
  return <PlaylistDetail playlistId={playlistId} />;
}
