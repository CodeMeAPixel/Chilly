"use client";

import { Menu, X } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { type MouseEvent, type ReactNode, useEffect, useRef, useState } from "react";
import { cn } from "@/lib/format";
import { Button } from "./ui";

type NavLink = { href: string; label: string };

export function MobileNav({ links, children }: { links: NavLink[]; children?: ReactNode }) {
  const pathname = usePathname();
  const [openAt, setOpenAt] = useState<string | null>(null);
  const open = openAt === pathname;
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: PointerEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpenAt(null);
    };
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenAt(null);
    };
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  const closeOnLink = (e: MouseEvent<HTMLDivElement>) => {
    if ((e.target as HTMLElement).closest("a")) setOpenAt(null);
  };

  const isActive = (href: string) => !href.includes("#") && (pathname === href || pathname.startsWith(`${href}/`));

  return (
    <div ref={ref} className={children ? "lg:hidden" : "md:hidden"}>
      <Button
        variant="ghost"
        size="icon"
        aria-label={open ? "Close menu" : "Open menu"}
        aria-expanded={open}
        aria-controls="mobile-menu"
        onClick={() => setOpenAt(open ? null : pathname)}
      >
        {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
      </Button>

      {open && (
        <div
          id="mobile-menu"
          onClick={closeOnLink}
          className="absolute inset-x-0 top-full mt-2 max-h-[calc(100dvh-6rem)] overflow-y-auto rounded-3xl border border-border bg-surface p-2 shadow-2xl"
        >
          <nav className={cn("grid gap-1", children && "md:hidden")}>
            {links.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className={cn(
                  "rounded-2xl px-4 py-3 text-sm font-medium transition",
                  isActive(link.href) ? "bg-primary-soft text-fg" : "text-muted hover:bg-surface-2 hover:text-fg",
                )}
              >
                {link.label}
              </Link>
            ))}
          </nav>
          {children && <div className="mt-2 border-t border-border px-1 pt-4 pb-2 md:mt-0 md:border-0 md:pt-2">{children}</div>}
        </div>
      )}
    </div>
  );
}
