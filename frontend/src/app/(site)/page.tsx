import { ArrowRight, ArrowUpRight, Check, ListMusic, Radio, Search, X } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { connection } from "next/server";
import { CommandExplorer } from "@/components/command-explorer";
import { Blobs, Drips, Squiggle } from "@/components/decor";
import { PlayerMock } from "@/components/player-mock";
import { buttonClass, Equalizer } from "@/components/ui";
import { cn, formatNumber } from "@/lib/format";
import { getStations, getStats, getStatus, inviteUrl } from "@/lib/server";
import type { Station, SystemStatus } from "@/lib/types";

export default async function Home() {
  await connection();
  const [stats, stations, status] = await Promise.all([getStats(), getStations(), getStatus()]);
  const invite = inviteUrl(stats?.bot?.id);
  const onAir = (stations ?? []).find((s) => s.online) ?? null;
  const siteHost = new URL(process.env.SITE_URL ?? "https://chillybot.space").host;

  return (
    <>
      <section className="grain relative -mt-18 overflow-hidden">
        <Blobs />
        <div className="pointer-events-none absolute inset-x-0 bottom-0 h-48 bg-linear-to-b from-transparent to-bg" />
        <div className="relative mx-auto grid max-w-6xl items-center gap-16 px-4 pt-34 pb-24 sm:px-6 lg:grid-cols-[1.1fr_1fr] lg:pt-42 lg:pb-32">
          <div className="space-y-8">
            <Link
              href="/status"
              className="inline-flex items-center gap-2 rounded-full border border-border bg-surface/70 py-1 pr-3 pl-1.5 text-xs text-muted backdrop-blur transition hover:text-fg"
            >
              <StatusDot status={status?.status} />
              {statusCopy(status)}
              <ArrowRight className="h-3 w-3" />
            </Link>

            <h1 className="font-display text-[3.4rem] leading-[0.95] font-semibold tracking-tight sm:text-7xl">
              Music that keeps your server{" "}
              <span className="relative inline-block text-accent">
                chill
                <Squiggle className="text-accent/60" />
              </span>
              .
            </h1>

            <p className="max-w-lg text-lg leading-relaxed text-muted">
              Queue songs from anywhere, tune into 24/7 radio and run the whole thing from a live dashboard. Free, fast, and it
              doesn&apos;t go quiet when a track breaks.
            </p>

            <div className="flex flex-wrap items-center gap-3">
              {invite ? (
                <a href={invite} target="_blank" rel="noreferrer" className={buttonClass("primary", "lg", "rounded-full px-7")}>
                  Add to Discord <ArrowUpRight className="h-4 w-4" />
                </a>
              ) : null}
              <Link href="/dashboard" className={buttonClass("ghost", "lg", "rounded-full text-fg")}>
                Open dashboard <ArrowRight className="h-4 w-4" />
              </Link>
            </div>

            {stats && (
              <p className="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-muted">
                <span>
                  <strong className="font-display text-lg text-fg">{formatNumber(stats.guilds)}</strong> servers
                </span>
                <span className="h-1 w-1 rounded-full bg-border" />
                <span className="flex items-center gap-2">
                  <Equalizer className="h-3" paused={stats.playing === 0} />
                  <strong className="font-display text-lg text-fg">{formatNumber(stats.playing)}</strong> playing right now
                </span>
              </p>
            )}
          </div>

          <div className="pt-12 lg:pt-0">
            <PlayerMock avatarUrl={stats?.bot?.avatar_url} siteHost={siteHost} />
          </div>
        </div>
      </section>

      <section id="features" className="scroll-mt-20 bg-surface">
        <Drips className="-mt-px" fill="var(--bg)" />
        <div className="mx-auto max-w-6xl px-4 pt-8 pb-24 sm:px-6">
          <SectionHeading eyebrow="What it does" title="Small bot, big vibes" />

          <div className="mt-12 grid gap-4 md:grid-cols-6">
            <Tile className="md:col-span-4" title="Finds the song you meant" body="Live suggestions from YouTube Music, YouTube, SoundCloud and Spotify. What you pick is exactly what plays.">
              <SearchPreview />
            </Tile>
            <Tile className="md:col-span-2" title="Never goes quiet" body="If a track fails, Chilly grabs the same song from another source.">
              <FallbackPreview />
            </Tile>
            <Tile className="md:col-span-2" title="24/7 radio" body="Always-on stations with live now-playing in your channel.">
              <RadioPreview station={onAir} />
            </Tile>
            <Tile className="md:col-span-4" title="A dashboard that's actually live" body="Reorder the queue, skip, seek and add tracks from your browser. Changes show up instantly for everyone.">
              <QueuePreview />
            </Tile>
            <Tile className="md:col-span-3" title="Playlists that follow you" body="Save songs and whole albums once, queue them in any server.">
              <PlaylistPreview />
            </Tile>
            <Tile className="md:col-span-3" title="Built to stay up" body="Multiple audio nodes with automatic failover and voice recovery.">
              <NodesPreview status={status} />
            </Tile>
          </div>
        </div>
      </section>

      <section id="commands" className="scroll-mt-20">
        <Drips className="-mt-px" fill="var(--surface)" />
        <div className="mx-auto max-w-6xl px-4 pt-8 pb-24 sm:px-6">
          <SectionHeading eyebrow="Commands" title="Type a slash, that's it" />
          <div className="mt-12">
            <CommandExplorer />
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-4 pb-8 sm:px-6">
        <div className="grain relative overflow-hidden rounded-[40px] bg-primary px-8 py-16 text-primary-fg sm:px-16">
          <Image
            src="/logo.svg"
            alt=""
            width={260}
            height={260}
            className="animate-float absolute -right-6 -bottom-14 hidden opacity-95 sm:block"
          />
          <div className="relative max-w-xl space-y-6">
            <h2 className="font-display text-4xl leading-tight font-semibold sm:text-5xl">Press play in under a minute.</h2>
            <p className="text-lg text-primary-fg/70">Invite Chilly, hop into a voice channel and type /play. That&apos;s the whole setup.</p>
            <div className="flex flex-wrap gap-3">
              {invite && (
                <a href={invite} target="_blank" rel="noreferrer" className={buttonClass("secondary", "lg", "rounded-full border-0 bg-primary-fg px-7 text-primary hover:bg-primary-fg/90")}>
                  Add to Discord <ArrowUpRight className="h-4 w-4" />
                </a>
              )}
              <Link href="/radio" className={buttonClass("ghost", "lg", "rounded-full text-primary-fg hover:bg-primary-fg/10 hover:text-primary-fg")}>
                Listen to the radio
              </Link>
            </div>
          </div>
        </div>
      </section>
    </>
  );
}

function statusCopy(status: SystemStatus | null) {
  if (!status) return "Status unavailable";
  if (status.status === "operational") return "All systems chill";
  if (status.status === "degraded") return "Some services degraded";
  return "We're having issues";
}

function StatusDot({ status }: { status?: SystemStatus["status"] }) {
  const color = status === "operational" ? "bg-lime" : status === "degraded" ? "bg-peach" : status ? "bg-accent" : "bg-muted";
  return (
    <span className="relative grid h-5 w-5 place-items-center rounded-full bg-surface-2">
      <span className={cn("pulse-ring absolute h-2 w-2 rounded-full", color)} />
      <span className={cn("relative h-2 w-2 rounded-full", color)} />
    </span>
  );
}

function SectionHeading({ eyebrow, title }: { eyebrow: string; title: string }) {
  return (
    <div className="space-y-3">
      <p className="font-mono text-xs tracking-widest text-accent uppercase">{eyebrow}</p>
      <h2 className="max-w-2xl font-display text-4xl leading-tight font-semibold tracking-tight sm:text-5xl">{title}</h2>
    </div>
  );
}

function Tile({ className, title, body, children }: { className?: string; title: string; body: string; children: React.ReactNode }) {
  return (
    <div className={cn("group flex flex-col overflow-hidden rounded-[28px] border border-border bg-bg p-6 transition hover:border-primary/30", className)}>
      <div className="mb-6 flex-1">{children}</div>
      <h3 className="font-display text-xl font-semibold">{title}</h3>
      <p className="mt-1.5 text-sm leading-relaxed text-muted">{body}</p>
    </div>
  );
}

function SearchPreview() {
  const results = [
    { title: "Sunset Lover", author: "Petit Biscuit", source: "YT Music", time: "3:57" },
    { title: "Sunset Lover (Live)", author: "Petit Biscuit", source: "YouTube", time: "4:12" },
    { title: "Sunset Lover", author: "Petit Biscuit", source: "SoundCloud", time: "3:58" },
  ];
  return (
    <div className="rounded-2xl border border-border bg-surface p-3">
      <div className="flex items-center gap-2 rounded-xl bg-surface-2 px-3 py-2.5 text-sm">
        <Search className="h-4 w-4 text-muted" />
        <span className="font-mono text-xs text-primary">/search</span>
        <span>sunset lover</span>
        <span className="h-4 w-px animate-pulse bg-fg" />
      </div>
      <ul className="mt-2 space-y-1">
        {results.map((r, i) => (
          <li key={i} className={cn("flex items-center gap-3 rounded-xl px-3 py-2 text-sm", i === 0 && "bg-primary-soft")}>
            <div className="h-8 w-8 shrink-0 rounded-lg bg-linear-to-br from-peach to-accent" />
            <div className="min-w-0 flex-1">
              <p className="truncate font-medium">{r.title}</p>
              <p className="truncate text-xs text-muted">{r.author}</p>
            </div>
            <span className="hidden rounded-full border border-border px-2 py-0.5 text-[10px] text-muted sm:block">{r.source}</span>
            <span className="font-mono text-xs text-muted">{r.time}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function FallbackPreview() {
  return (
    <div className="space-y-2 font-mono text-xs">
      <p className="truncate pb-1 font-sans text-sm font-medium">Hate Me · Joyner Lucas</p>
      <div className="flex items-center gap-2 rounded-xl border border-border bg-surface px-3 py-2.5 text-muted line-through decoration-accent">
        <X className="h-3.5 w-3.5 shrink-0 text-accent" /> youtube · load failed
      </div>
      <div className="ml-4 h-4 border-l-2 border-dashed border-border" />
      <div className="flex items-center gap-2 rounded-xl border border-primary/30 bg-primary-soft px-3 py-2.5">
        <Check className="h-3.5 w-3.5 shrink-0 text-primary" /> soundcloud · playing
      </div>
    </div>
  );
}

function RadioPreview({ station }: { station: Station | null }) {
  const song = station?.now_playing?.song;
  return (
    <div className="space-y-3">
      <div className="flex h-20 items-end gap-1 rounded-2xl border border-border bg-surface px-4 py-3" aria-hidden>
        {Array.from({ length: 22 }).map((_, i) => (
          <span
            key={i}
            className="eq-bar flex-1 rounded-full bg-linear-to-t from-accent to-peach"
            style={{ height: `${30 + ((i * 37) % 70)}%`, animationDelay: `${(i % 7) * 0.12}s`, animationDuration: `${0.8 + (i % 5) * 0.15}s` }}
          />
        ))}
      </div>
      <RadioNowCard station={station} song={song} />
    </div>
  );
}

function RadioNowCard({ station, song }: { station: Station | null; song: NonNullable<Station["now_playing"]>["song"] | undefined }) {
  return (
    <div className="flex items-center gap-3 rounded-2xl border border-border bg-surface p-3">
      {song?.art ? (
        <img src={song.art} alt="" className="h-14 w-14 rounded-xl object-cover" />
      ) : (
        <div className="grid h-14 w-14 shrink-0 place-items-center rounded-xl bg-accent-soft text-accent">
          <Radio className="h-6 w-6" />
        </div>
      )}
      <div className="min-w-0">
        <p className="flex items-center gap-1.5 text-[10px] font-semibold tracking-wider text-accent uppercase">
          <span className="h-1.5 w-1.5 rounded-full bg-accent" /> {station ? station.name : "On air"}
        </p>
        <p className="truncate text-sm font-medium">{song?.title || "Lo-fi beats to chill to"}</p>
        <p className="truncate text-xs text-muted">{song?.artist || "Chilly Radio"}</p>
      </div>
    </div>
  );
}

function QueuePreview() {
  const queue = [
    { title: "Midnight City", author: "M83", time: "4:03" },
    { title: "Resonance", author: "HOME", time: "3:32" },
    { title: "Tadow", author: "Masego, FKJ", time: "5:01" },
  ];
  return (
    <div className="grid gap-3 sm:grid-cols-[1fr_1.2fr]">
      <div className="rounded-2xl border border-border bg-surface p-4">
        <div className="h-24 w-full rounded-xl bg-linear-to-br from-primary via-peach to-accent" />
        <p className="mt-3 truncate text-sm font-semibold">Sunset Lover</p>
        <p className="text-xs text-muted">Petit Biscuit</p>
        <div className="mt-3 h-1 rounded-full bg-surface-2">
          <div className="h-full w-2/5 rounded-full bg-primary" />
        </div>
      </div>
      <ol className="space-y-1.5 rounded-2xl border border-border bg-surface p-3">
        <li className="px-2 pb-1 text-xs font-medium text-muted">Up next</li>
        {queue.map((t, i) => (
          <li key={t.title} className="flex items-center gap-3 rounded-xl px-2 py-1.5 text-sm hover:bg-surface-2">
            <span className="w-3 font-mono text-xs text-muted">{i + 1}</span>
            <div className="min-w-0 flex-1">
              <p className="truncate font-medium">{t.title}</p>
              <p className="truncate text-xs text-muted">{t.author}</p>
            </div>
            <span className="font-mono text-xs text-muted">{t.time}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}

function PlaylistPreview() {
  return (
    <div className="relative h-32">
      {[
        { name: "Late night drive", count: 42, rotate: "-rotate-6", offset: "left-0", tone: "from-accent to-peach" },
        { name: "Study session", count: 88, rotate: "rotate-3", offset: "left-1/4", tone: "from-primary to-lime" },
        { name: "Gym hype", count: 31, rotate: "-rotate-2", offset: "left-1/2", tone: "from-peach to-primary" },
      ].map((p) => (
        <div
          key={p.name}
          className={cn(
            "absolute top-2 w-44 rounded-2xl border border-border bg-surface p-3 shadow-xl transition group-hover:translate-y-[-4px]",
            p.rotate,
            p.offset,
          )}
        >
          <div className={cn("mb-2 grid h-12 w-12 place-items-center rounded-xl bg-linear-to-br text-bg", p.tone)}>
            <ListMusic className="h-5 w-5" />
          </div>
          <p className="truncate text-sm font-medium">{p.name}</p>
          <p className="text-xs text-muted">{p.count} tracks</p>
        </div>
      ))}
    </div>
  );
}

function NodesPreview({ status }: { status: SystemStatus | null }) {
  const nodes = status?.nodes?.length
    ? status.nodes.map((n) => ({ name: n.name, up: n.status === "CONNECTED", players: n.playing_players }))
    : [
        { name: "node-1", up: true, players: 0 },
        { name: "node-2", up: true, players: 0 },
      ];
  return (
    <Link href="/status" className="block space-y-2">
      {nodes.slice(0, 3).map((n) => (
        <div key={n.name} className="flex items-center gap-3 rounded-xl border border-border bg-surface px-3 py-2.5 text-sm">
          <span className={cn("h-2 w-2 rounded-full", n.up ? "bg-lime" : "bg-accent")} />
          <span className="font-mono text-xs">{n.name}</span>
          <span className="ml-auto text-xs text-muted">{n.up ? `${n.players} playing` : "offline"}</span>
        </div>
      ))}
      <p className="flex items-center gap-1 pt-1 text-xs text-primary">
        View live status <ArrowRight className="h-3 w-3" />
      </p>
    </Link>
  );
}
