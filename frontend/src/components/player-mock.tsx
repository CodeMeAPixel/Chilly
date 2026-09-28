import { CornerUpRight, ExternalLink, Pause, Repeat, Shuffle, SkipBack, SkipForward, Square } from "lucide-react";
import Image from "next/image";

function MockButton({ children, wide }: { children: React.ReactNode; wide?: boolean }) {
  return (
    <span
      className={`inline-flex h-8 items-center justify-center gap-1.5 rounded-md bg-[#4e5058]/60 text-xs font-medium text-white/90 ${wide ? "px-3" : "w-14"}`}
    >
      {children}
    </span>
  );
}

export function PlayerMock({ avatarUrl, siteHost = "chillybot.space" }: { avatarUrl?: string; siteHost?: string }) {
  const avatar = avatarUrl ? (
    <img src={avatarUrl} alt="" className="h-10 w-10 rounded-full" />
  ) : (
    <span className="grid h-10 w-10 place-items-center rounded-full bg-accent-soft">
      <Image src="/logo.svg" alt="" width={26} height={26} />
    </span>
  );

  return (
    <div className="relative mx-auto w-full max-w-md">
      <Image
        src="/logo.svg"
        alt=""
        width={120}
        height={120}
        priority
        className="animate-float absolute -top-16 -right-4 z-10 drop-shadow-2xl sm:-right-10"
      />

      <div className="relative rotate-[-1.5deg] rounded-[28px] border border-border bg-[#313338] p-5 text-[#dbdee1] shadow-[0_40px_120px_-40px_rgb(0_0_0/0.6)]">
        <p className="mb-1.5 ml-12 flex items-center gap-1.5 text-xs text-[#949ba4]">
          <CornerUpRight className="h-3 w-3 -scale-x-100" />
          <span className="h-4 w-4 rounded-full bg-linear-to-br from-peach to-accent" />
          <span className="font-medium text-[#c9cdfb]">you</span> used
          <span className="text-[#00a8fc]">/play</span>
        </p>

        <div className="flex items-start gap-3">
          {avatar}
          <div className="min-w-0 flex-1">
            <p className="flex items-center gap-1.5 text-sm">
              <span className="font-semibold text-white">Chilly</span>
              <span className="rounded bg-[#5865F2] px-1 text-[10px] leading-4 font-semibold text-white">APP</span>
              <span className="text-xs text-[#949ba4]">Today at 9:41 PM</span>
            </p>

            <div className="mt-1.5 flex gap-3 rounded-[4px] border-l-4 border-[#b5ebe5] bg-[#2b2d31] p-3">
              <div className="min-w-0 flex-1 space-y-1.5">
                <p className="flex items-center gap-1.5 text-xs font-semibold text-white">
                  <span className="grid h-5 w-5 place-items-center rounded-full bg-accent-soft">
                    <Image src="/logo.svg" alt="" width={14} height={14} />
                  </span>
                  Now playing
                </p>
                <p className="truncate text-sm font-semibold text-[#00a8fc]">Sunset Lover</p>
                <p className="text-sm">
                  Petit Biscuit · <code className="rounded bg-[#1e1f22] px-1 font-mono text-xs">3:57</code>
                </p>
                <p className="text-sm">
                  Requested by <span className="rounded bg-[#5865f2]/30 px-1 text-[#c9cdfb]">@you</span>
                </p>
                <div className="pt-1">
                  <p className="text-xs font-semibold text-white">Up next</p>
                  <p className="truncate text-sm">
                    <span className="text-[#00a8fc]">Midnight City</span> · +2 more
                  </p>
                </div>
                <p className="pt-1 text-[11px] text-[#949ba4]">YouTube · Loop off · Shuffle off · {siteHost}</p>
              </div>
              <div className="h-20 w-20 shrink-0 rounded-md bg-linear-to-br from-peach via-accent to-primary" />
            </div>

            <div className="mt-2 space-y-1.5">
              <div className="flex gap-1.5">
                <MockButton><SkipBack className="h-3.5 w-3.5" /></MockButton>
                <MockButton><Pause className="h-3.5 w-3.5" /></MockButton>
                <MockButton><SkipForward className="h-3.5 w-3.5" /></MockButton>
              </div>
              <div className="flex flex-wrap gap-1.5">
                <MockButton><Square className="h-3 w-3" /></MockButton>
                <MockButton><Repeat className="h-3.5 w-3.5" /></MockButton>
                <MockButton><Shuffle className="h-3.5 w-3.5" /></MockButton>
                <MockButton wide>
                  Dashboard <ExternalLink className="h-3 w-3" />
                </MockButton>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
