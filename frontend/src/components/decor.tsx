import { cn } from "@/lib/format";

export function Blobs({ className }: { className?: string }) {
  return (
    <div className={cn("pointer-events-none absolute inset-0 overflow-hidden", className)} aria-hidden>
      <div className="animate-blob absolute -top-32 -left-24 h-[420px] w-[420px] rounded-full bg-primary/25 blur-[110px]" />
      <div className="animate-blob absolute top-10 right-[-120px] h-[380px] w-[380px] rounded-full bg-accent/25 blur-[110px] [animation-delay:-6s]" />
      <div className="animate-blob absolute top-1/3 left-1/3 h-[320px] w-[320px] rounded-full bg-peach/15 blur-[120px] [animation-delay:-12s]" />
    </div>
  );
}

const drips: [number, number, number][] = [
  [70, 14, 38], [150, 9, 62], [215, 16, 30], [300, 11, 84], [370, 18, 44], [455, 10, 58],
  [520, 15, 26], [600, 12, 96], [680, 17, 40], [760, 9, 66], [830, 14, 34], [905, 12, 78],
  [985, 18, 42], [1060, 10, 60], [1130, 15, 28], [1205, 11, 90], [1285, 16, 46], [1370, 10, 64],
];

const base = 22;

const dripShapes = drips.map(([cx, w, length]) => {
  const bottom = base + length;
  return `M${cx - w - 8} ${base - 1}C${cx - w} ${base - 1} ${cx - w} ${base + 4} ${cx - w} ${base + 10}V${bottom - w}A${w} ${w} 0 0 0 ${cx + w} ${bottom - w}V${base + 10}C${cx + w} ${base + 4} ${cx + w} ${base - 1} ${cx + w + 8} ${base - 1}Z`;
});

export function Drips({ className, fill = "var(--bg)" }: { className?: string; fill?: string }) {
  return (
    <svg viewBox="0 0 1440 124" preserveAspectRatio="none" className={cn("block h-14 w-full sm:h-24", className)} style={{ fill }} aria-hidden>
      <rect x="0" y="0" width="1440" height={base} />
      {dripShapes.map((d) => (
        <path key={d} d={d} />
      ))}
    </svg>
  );
}

export function Squiggle({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 220 18" className={cn("absolute -bottom-3 left-0 w-full", className)} aria-hidden>
      <path
        d="M2 12c18-8 30-8 44 0s28 8 44 0 30-8 44 0 28 8 44 0 26-7 40-2"
        fill="none"
        stroke="currentColor"
        strokeWidth="4"
        strokeLinecap="round"
      />
    </svg>
  );
}
