import { ArrowRight, ArrowUpRight, Moon, Radio, Search, Users } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { connection } from "next/server";
import { CommandExplorer } from "@/components/command-explorer";
import { Blobs, Drips, Squiggle } from "@/components/decor";
import { buttonClass, Equalizer } from "@/components/ui";
import { cn, formatNumber } from "@/lib/format";
import { getStations, getStats, getStatus, inviteUrl } from "@/lib/server";
import type { Station, SystemStatus } from "@/lib/types";

export default async function Home() {
  await connection();
  const [stats, stations, status] = await Promise.all([getStats(), getStations(), getStatus()]);
  const invite = inviteUrl(stats?.bot?.id);
  const liveStations = (stations ?? []).filter((s) => s.online).sort((a, b) => b.listeners - a.listeners);
  const listeners = liveStations.reduce((sum, s) => sum + s.listeners, 0);

  return (
    <>
      <section className="grain relative -mt-18 overflow-hidden">
        <Blobs />
        <div className="pointer-events-none absolute inset-x-0 bottom-0 h-48 bg-linear-to-b from-transparent to-bg" />
        <div className="relative mx-auto grid max-w-6xl grid-cols-1 items-center gap-16 px-4 pt-34 pb-24 sm:px-6 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1fr)] lg:pt-42 lg:pb-32">
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
              24/7 radio that keeps your server{" "}
              <span className="relative inline-block text-accent">
                chill
                <Squiggle className="text-accent/60" />
              </span>
              .
            </h1>

            <p className="max-w-lg text-lg leading-relaxed text-muted">
              Tune any voice channel into always-on stations, leave them playing around the clock and run it all from a live
              dashboard. Want a specific song? Chilly plays those too.
            </p>

            <div className="flex flex-wrap items-center gap-3">
              {invite ? (
                <a href={invite} target="_blank" rel="noreferrer" className={buttonClass("primary", "lg", "rounded-full px-7")}>
                  Add to Discord <ArrowUpRight className="h-4 w-4" />
                </a>
              ) : null}
              <Link href="/radio" className={buttonClass("ghost", "lg", "rounded-full text-fg")}>
                Browse stations <ArrowRight className="h-4 w-4" />
              </Link>
            </div>

            {stats && (
              <p className="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-muted">
                <span>
                  <strong className="font-display text-lg text-fg">{formatNumber(stats.guilds)}</strong> servers
                </span>
                <span className="h-1 w-1 rounded-full bg-border" />
                <span className="flex items-center gap-2">
                  <Equalizer className="h-3" paused={listeners === 0} />
                  <strong className="font-display text-lg text-fg">{formatNumber(listeners)}</strong> tuned in now
                </span>
                {liveStations.length > 0 && (
                  <>
                    <span className="h-1 w-1 rounded-full bg-border" />
                    <span>
                      <strong className="font-display text-lg text-fg">{liveStations.length}</strong> stations on air
                    </span>
                  </>
                )}
              </p>
            )}
          </div>

          <div className="pt-12 lg:pt-0">
            <HeroStations stations={liveStations} />
          </div>
        </div>
      </section>

      <section id="features" className="scroll-mt-20 bg-surface">
        <Drips className="-mt-px" fill="var(--bg)" />
        <div className="mx-auto max-w-6xl px-4 pt-8 pb-24 sm:px-6">
          <SectionHeading eyebrow="What it does" title="Press play once. It never stops." />

          <div className="mt-12 grid grid-cols-1 gap-4 md:grid-cols-6">
            <Tile className="md:col-span-4" title="Stations for every mood" body="Lo-fi, hip-hop, late-night mixes and more, streaming around the clock with live now-playing in your channel.">
              <StationsPreview stations={liveStations} />
            </Tile>
            <Tile className="md:col-span-2" title="Truly 24/7" body="/247 on keeps a station playing even when the channel is empty, and it's back on its own after a restart.">
              <StayPreview station={liveStations[0] ?? null} />
            </Tile>
            <Tile className="md:col-span-2" title="Listen anywhere" body="Every station plays in your browser too, no Discord needed.">
              <RadioPreview station={liveStations[0] ?? null} />
            </Tile>
            <Tile className="md:col-span-4" title="A dashboard that's actually live" body="Switch stations, skip, seek and manage the queue from your browser. Changes show up instantly for everyone.">
              <QueuePreview />
            </Tile>
            <Tile className="md:col-span-3" title="Your songs, too" body="Want something specific? /play any song, album or artist from our library, with lyrics and your own playlists.">
              <SearchPreview />
            </Tile>
            <Tile className="md:col-span-3" title="Built to stay up" body="Redundant audio servers with automatic failover, so the music keeps going.">
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
            <h2 className="font-display text-4xl leading-tight font-semibold sm:text-5xl">On air in under a minute.</h2>
            <p className="text-lg text-primary-fg/70">
              Invite Chilly, hop into a voice channel and type /radio play. Add /247 on and it never stops.
            </p>
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
    { title: "Sunset Lover", author: "Petit Biscuit", source: "Presence", time: "3:57" },
    { title: "Problems", author: "Petit Biscuit", source: "Presence", time: "3:33" },
    { title: "Beam Me Up", author: "Petit Biscuit", source: "Presence", time: "3:44" },
  ];
  return (
    <div className="rounded-2xl border border-border bg-surface p-3">
      <div className="flex items-center gap-2 rounded-xl bg-surface-2 px-3 py-2.5 text-sm">
        <Search className="h-4 w-4 text-muted" />
        <span className="font-mono text-xs text-primary">/search</span>
        <span>petit biscuit</span>
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

function HeroStations({ stations }: { stations: Station[] }) {
  const [featured, ...rest] = stations;
  if (!featured) {
    return (
      <div className="rounded-[32px] border border-border bg-surface/80 p-6 shadow-2xl backdrop-blur">
        <RadioNowCard station={null} song={undefined} />
      </div>
    );
  }
  const song = featured.now_playing?.song;
  return (
    <Link href="/radio" className="group relative block">
      <div className="absolute -inset-6 -z-10 rounded-[48px] bg-primary/10 blur-3xl" aria-hidden />
      <div className="overflow-hidden rounded-[32px] border border-border bg-surface/85 shadow-2xl backdrop-blur transition group-hover:border-primary/40">
        <div className="flex items-center gap-2 border-b border-border px-5 py-3 text-xs text-muted">
          <span className="relative grid h-2.5 w-2.5 place-items-center">
            <span className="pulse-ring absolute h-2 w-2 rounded-full bg-accent" />
            <span className="relative h-2 w-2 rounded-full bg-accent" />
          </span>
          <span className="font-semibold tracking-wider text-accent uppercase">On air</span>
          <span className="ml-auto flex items-center gap-1.5">
            <Users className="h-3.5 w-3.5" /> {featured.listeners} listening
          </span>
        </div>
        <div className="flex gap-5 p-5">
          {song?.art ? (
            <img src={song.art} alt="" className="h-28 w-28 shrink-0 rounded-2xl object-cover shadow-xl sm:h-32 sm:w-32" />
          ) : (
            <div className="grid h-28 w-28 shrink-0 place-items-center rounded-2xl bg-accent-soft text-accent sm:h-32 sm:w-32">
              <Radio className="h-8 w-8" />
            </div>
          )}
          <div className="min-w-0 flex-1 space-y-1.5 pt-1">
            <p className="font-display text-sm font-semibold text-primary">{featured.name}</p>
            <p className="line-clamp-2 font-display text-xl leading-tight font-semibold">{song?.title || song?.text || "Live stream"}</p>
            <p className="truncate text-sm text-muted">{song?.artist}</p>
            <div className="flex h-8 items-end gap-[3px] pt-2" aria-hidden>
              {Array.from({ length: 28 }).map((_, i) => (
                <span
                  key={i}
                  className="eq-bar flex-1 rounded-full bg-linear-to-t from-primary to-accent"
                  style={{ height: `${25 + ((i * 41) % 75)}%`, animationDelay: `${(i % 6) * 0.13}s` }}
                />
              ))}
            </div>
          </div>
        </div>
        {rest.length > 0 && (
          <ul className="space-y-1 border-t border-border p-2">
            {rest.slice(0, 3).map((station) => (
              <li key={station.shortcode} className="flex items-center gap-3 rounded-2xl px-3 py-2 text-sm">
                {station.now_playing?.song.art ? (
                  <img src={station.now_playing.song.art} alt="" className="h-9 w-9 rounded-lg object-cover" />
                ) : (
                  <div className="grid h-9 w-9 place-items-center rounded-lg bg-accent-soft text-accent">
                    <Radio className="h-4 w-4" />
                  </div>
                )}
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium">{station.name}</p>
                  <p className="truncate text-xs text-muted">{station.now_playing?.song.text}</p>
                </div>
                <span className="flex items-center gap-1 font-mono text-xs text-muted">
                  <Users className="h-3 w-3" /> {station.listeners}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Link>
  );
}

function StationsPreview({ stations }: { stations: Station[] }) {
  if (stations.length === 0) {
    return (
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="h-[74px] rounded-2xl border border-dashed border-border" />
        ))}
      </div>
    );
  }
  return (
    <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
      {stations.slice(0, 4).map((station) => (
        <div key={station.shortcode} className="flex items-center gap-3 rounded-2xl border border-border bg-surface p-3">
          {station.now_playing?.song.art ? (
            <img src={station.now_playing.song.art} alt="" className="h-12 w-12 rounded-xl object-cover" />
          ) : (
            <div className="grid h-12 w-12 place-items-center rounded-xl bg-accent-soft text-accent">
              <Radio className="h-5 w-5" />
            </div>
          )}
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-semibold">{station.name}</p>
            <p className="truncate text-xs text-muted">{station.now_playing?.song.text || "Live now"}</p>
          </div>
          <Equalizer className="h-3 shrink-0" />
        </div>
      ))}
    </div>
  );
}

function StayPreview({ station }: { station: Station | null }) {
  return (
    <div className="space-y-2 text-sm">
      <div className="flex items-center gap-2 rounded-xl bg-surface-2 px-3 py-2.5">
        <span className="font-mono text-xs text-primary">/247 on</span>
        <span className="truncate text-muted">{station?.name ?? "Chill Vibes"}</span>
      </div>
      <div className="rounded-2xl border border-primary/30 bg-primary-soft p-3">
        <p className="flex items-center gap-1.5 font-medium">
          <Moon className="h-4 w-4 text-primary" /> 24/7 radio is on
        </p>
        <p className="mt-1 text-xs text-muted">Staying in #lounge, even when it&apos;s empty.</p>
      </div>
    </div>
  );
}
