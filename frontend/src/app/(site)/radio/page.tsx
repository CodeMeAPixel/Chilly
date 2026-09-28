import type { Metadata } from "next";
import { connection } from "next/server";
import { EmptyState } from "@/components/ui";
import { getStations } from "@/lib/server";
import { RadioBrowser } from "./radio-browser";
import { Radio } from "lucide-react";

export const metadata: Metadata = {
  title: "Radio",
  description: "Tune into Chilly's 24/7 radio stations in your browser or straight into your Discord voice channel.",
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
          Listen right here or send a station to your voice channel. Now-playing info updates live.
        </p>
      </div>

      {stations === null ? (
        <EmptyState icon={<Radio className="h-6 w-6" />} title="Radio is offline" className="mt-10">
          The radio service isn&apos;t available right now. Check back soon.
        </EmptyState>
      ) : (
        <RadioBrowser initial={stations} />
      )}
    </div>
  );
}
