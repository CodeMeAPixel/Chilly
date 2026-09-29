import { Library } from "lucide-react";
import type { Metadata } from "next";
import { connection } from "next/server";
import { EmptyState } from "@/components/ui";
import { formatNumber } from "@/lib/format";
import { getLibrarySummary } from "@/lib/server";
import { TrackBrowser } from "./track-browser";

export const metadata: Metadata = {
  title: "Tracks",
  description: "Every song, artist and album in Chilly's library. Request songs on the stations or play them in your server.",
  openGraph: { url: "/tracks" },
};

export default async function TracksPage(props: PageProps<"/tracks">) {
  await connection();
  const [summary, params] = await Promise.all([getLibrarySummary(), props.searchParams]);
  const pick = (key: string) => (typeof params[key] === "string" ? (params[key] as string) : undefined);

  return (
    <div className="mx-auto max-w-6xl px-4 py-14 sm:px-6">
      <div className="space-y-2">
        <p className="text-sm font-medium tracking-wide text-primary uppercase">Library</p>
        <h1 className="font-display text-4xl font-semibold tracking-tight">Every song we&apos;ve got</h1>
        <p className="max-w-2xl text-muted">
          Request a song and it plays on the station soon, or play it privately in your server. Missing something? Suggest it
          from your requests page.
        </p>
        {summary && (
          <p className="flex flex-wrap gap-x-4 gap-y-1 pt-2 text-sm text-muted">
            <span>
              <strong className="font-display text-base text-fg">{formatNumber(summary.tracks)}</strong> songs
            </span>
            <span>
              <strong className="font-display text-base text-fg">{formatNumber(summary.artists)}</strong> artists
            </span>
            <span>
              <strong className="font-display text-base text-fg">{formatNumber(summary.albums)}</strong> albums
            </span>
          </p>
        )}
      </div>

      {summary === null ? (
        <EmptyState icon={<Library className="h-6 w-6" />} title="The library is offline" className="mt-10">
          We can&apos;t reach the music library right now. Check back soon.
        </EmptyState>
      ) : (
        <TrackBrowser
          initialFilter={{ artist: pick("artist"), album: pick("album"), playlist: pick("playlist") }}
          initialQuery={pick("q") ?? ""}
        />
      )}
    </div>
  );
}
