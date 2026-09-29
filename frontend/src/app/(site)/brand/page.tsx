import { Check, Download, FileText, X } from "lucide-react";
import type { Metadata } from "next";
import Image from "next/image";
import { buttonClass, Card } from "@/components/ui";
import { cn } from "@/lib/format";
import { CopyButton } from "./copy-button";

export const metadata: Metadata = {
  title: "Brand",
  description: "Chilly's logo, colours, typography and copy, ready for bot lists, videos and posts.",
  openGraph: { url: "/brand" },
};

const listingUrl = "https://github.com/CodeMeAPixel/Chilly/blob/master/docs/bot-listing.md";

const colors = [
  { name: "Mint", hex: "#B5EBE5", note: "Primary. Embeds, highlights, dark-mode accents.", text: "#06201D" },
  { name: "Deep mint", hex: "#3FB3A4", note: "Primary on light backgrounds.", text: "#06201D" },
  { name: "Pink", hex: "#FF9AA2", note: "Radio and live states.", text: "#2A0A0E" },
  { name: "Peach", hex: "#FFDAC1", note: "Warm highlights and the logo stick.", text: "#3A2415" },
  { name: "Lime", hex: "#AAD67E", note: "Online and success.", text: "#17240A" },
  { name: "Night", hex: "#0A1014", note: "Background and dark surfaces.", text: "#E6F3F1" },
];

const copy = [
  { label: "Name", value: "Chilly" },
  { label: "Tagline", value: "24/7 radio for your Discord server." },
  {
    label: "Short description",
    value: "24/7 radio for your Discord server. Tune into always-on stations, keep them playing around the clock and play any song on request.",
  },
];

const dos = [
  "Write the name as Chilly, with a capital C.",
  "Give the logo room to breathe: at least a quarter of its width on every side.",
  "Use the logo on Night or plain light backgrounds.",
  "Lead with radio: Chilly is a 24/7 radio bot that also plays songs on request.",
];

const donts = [
  "Don't stretch, recolour, rotate or add effects to the logo.",
  "Don't write CHILLY, chilly or Chilly Bot in running text.",
  "Don't put the logo on busy photos or low-contrast colours.",
  "Don't imply Chilly is made by or affiliated with Discord.",
];

export default function BrandPage() {
  return (
    <div className="mx-auto max-w-6xl space-y-16 px-4 py-14 sm:px-6">
      <div className="space-y-2">
        <p className="text-sm font-medium tracking-wide text-primary uppercase">Brand</p>
        <h1 className="font-display text-4xl font-semibold tracking-tight">Chilly brand kit</h1>
        <p className="max-w-2xl text-muted">
          Everything you need to feature Chilly on a bot list, in a video or in a post. Use the assets as they are; if you need
          something that isn&apos;t here, open an issue on GitHub.
        </p>
      </div>

      <section className="space-y-5">
        <SectionTitle title="Logo" body="The popsicle mark works on its own or next to the name." />
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <LogoTile tone="dark" />
          <LogoTile tone="light" />
          <Card className="flex flex-col justify-between gap-4 p-6">
            <div className="space-y-1">
              <p className="font-display text-lg font-semibold">Downloads</p>
              <p className="text-sm text-muted">Transparent backgrounds. Use the PNG where SVG isn&apos;t accepted, like most bot lists.</p>
            </div>
            <div className="grid gap-2">
              <a href="/logo.svg" download="chilly-logo.svg" className={buttonClass("secondary", "md", "justify-start")}>
                <Download className="h-4 w-4" /> Logo · SVG
              </a>
              <a href="/brand/logo.png" download="chilly-logo.png" className={buttonClass("secondary", "md", "justify-start")}>
                <Download className="h-4 w-4" /> Logo · PNG 512×512
              </a>
              <a href="/opengraph-image" download="chilly-banner.png" className={buttonClass("secondary", "md", "justify-start")}>
                <Download className="h-4 w-4" /> Banner · PNG 1200×630
              </a>
            </div>
          </Card>
        </div>
      </section>

      <section className="space-y-5">
        <SectionTitle title="Colours" body="A cool mint base with warm pink and peach accents. Click a value to copy it." />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {colors.map((c) => (
            <Card key={c.name} className="overflow-hidden">
              <div className="flex h-24 items-end p-4" style={{ background: c.hex, color: c.text }}>
                <span className="font-display text-lg font-semibold">{c.name}</span>
              </div>
              <div className="flex items-center justify-between gap-2 p-4">
                <p className="text-sm text-muted">{c.note}</p>
                <CopyButton value={c.hex} label={c.hex} className="shrink-0 font-mono" />
              </div>
            </Card>
          ))}
        </div>
      </section>

      <section className="space-y-5">
        <SectionTitle title="Typography" body="Both families are free on Google Fonts." />
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Card className="space-y-2 p-6">
            <p className="text-xs text-muted">Display · Fredoka SemiBold</p>
            <p className="font-display text-5xl font-semibold tracking-tight">Always on air</p>
          </Card>
          <Card className="space-y-2 p-6">
            <p className="text-xs text-muted">Body · Geist</p>
            <p className="text-lg leading-relaxed">Tune into always-on stations and keep them playing around the clock.</p>
            <p className="font-mono text-sm text-muted">Geist Mono for commands: /radio play</p>
          </Card>
        </div>
      </section>

      <section className="space-y-5">
        <SectionTitle title="Name and copy" body="Use these as they are so Chilly reads the same everywhere." />
        <Card className="divide-y divide-border">
          {copy.map((item) => (
            <div key={item.label} className="flex flex-col gap-2 p-5 sm:flex-row sm:items-start sm:justify-between">
              <div className="space-y-1">
                <p className="text-xs text-muted">{item.label}</p>
                <p>{item.value}</p>
              </div>
              <CopyButton value={item.value} className="self-start" />
            </div>
          ))}
          <div className="flex flex-col gap-3 p-5 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <p className="text-xs text-muted">Bot list template</p>
              <p className="text-sm text-muted">Long description, tags, feature list and commands for top.gg and similar sites.</p>
            </div>
            <a href={listingUrl} target="_blank" rel="noreferrer" className={buttonClass("secondary", "sm")}>
              <FileText className="h-4 w-4" /> Open template
            </a>
          </div>
        </Card>
      </section>

      <section className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <RuleList title="Do" items={dos} good />
        <RuleList title="Don't" items={donts} />
      </section>
    </div>
  );
}

function SectionTitle({ title, body }: { title: string; body: string }) {
  return (
    <div className="space-y-1">
      <h2 className="font-display text-2xl font-semibold tracking-tight">{title}</h2>
      <p className="text-sm text-muted">{body}</p>
    </div>
  );
}

function LogoTile({ tone }: { tone: "dark" | "light" }) {
  const dark = tone === "dark";
  return (
    <div
      className={cn(
        "flex aspect-square flex-col items-center justify-center gap-4 rounded-3xl border border-border",
        dark ? "bg-[#0a1014] text-[#e6f3f1]" : "bg-[#f6fbfa] text-[#10201f]",
      )}
    >
      <Image src="/logo.svg" alt="Chilly logo" width={120} height={120} />
      <span className="font-display text-3xl font-semibold tracking-tight">Chilly</span>
      <span className={cn("text-xs", dark ? "text-[#8da6a3]" : "text-[#5b7472]")}>{dark ? "On Night" : "On light"}</span>
    </div>
  );
}

function RuleList({ title, items, good }: { title: string; items: string[]; good?: boolean }) {
  return (
    <Card className="p-6">
      <h2 className="font-display text-xl font-semibold">{title}</h2>
      <ul className="mt-4 space-y-3">
        {items.map((item) => (
          <li key={item} className="flex gap-3 text-sm">
            {good ? (
              <Check className="mt-0.5 h-4 w-4 shrink-0 text-lime" />
            ) : (
              <X className="mt-0.5 h-4 w-4 shrink-0 text-accent" />
            )}
            <span className="text-muted">{item}</span>
          </li>
        ))}
      </ul>
    </Card>
  );
}
