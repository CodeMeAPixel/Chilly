import type { Metadata } from "next";
import { connection } from "next/server";
import { EmptyState } from "@/components/ui";
import { getStations } from "@/lib/server";
import { RadioBrowser } from "./radio-browser";
import { Radio } from "lucide-react";

export const metadata: Metadata = {
  title: "Radio",
  description: "Browse Chilly's 24/7 radio stations. Listen in your browser, or send a station to your Discord voice channel and keep it playing around the clock.",
  openGraph: { url: "/radio" },
};

export default async function RadioPage() {
  await connection();
  const stations = await getStations();

  return (
    <div className="mx-auto max-w-6xl px-4 py-14 sm:px-6">
      <div className="space-y-2">
        <p className="text-sm font-medium tracking-wide text-primary uppercase">Radio</p>
        <h1 className="font-display text-4xl font-semibold tracking-tight">Always on air</h1>
        <p className="max-w-2xl text-muted">
          Listen right here, send a station to your voice channel, or use <code className="rounded-md bg-surface-2 px-1.5 py-0.5 font-mono text-sm text-fg">/247 on</code> to keep it playing there around the clock.
        </p>
      </div>

      {stations === null ? (
        <EmptyState icon={<Radio className="h-6 w-6" />} title="Radio is offline" className="mt-10">
          The radio service isn&apos;t available right now. Check back soon.
        </EmptyState>
      ) : (
        <RadioBrowser initial={stations} />
      )}

      <p className="mt-12 text-center text-xs text-muted">
        Stations are run by the Chilly team and streamed with{" "}
        <a href="https://www.azuracast.com" target="_blank" rel="noreferrer" className="underline-offset-2 hover:text-fg hover:underline">
          AzuraCast
        </a>
        .
      </p>
    </div>
  );
}
