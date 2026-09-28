import Link from "next/link";
import { Logo } from "./logo";
import { ThemeToggle } from "./theme-toggle";
import { UserNav } from "./user-nav";

const links = [
  { href: "/#features", label: "Features" },
  { href: "/#commands", label: "Commands" },
  { href: "/radio", label: "Radio" },
  { href: "/status", label: "Status" },
  { href: "/dashboard", label: "Dashboard" },
];

export function SiteHeader() {
  return (
    <header className="sticky top-0 z-40 h-0 px-3">
      <div className="mx-auto flex h-14 max-w-6xl translate-y-3 items-center justify-between gap-4 rounded-full border border-border bg-surface/70 pr-2 pl-5 shadow-[0_10px_40px_-20px_rgb(0_0_0/0.5)] backdrop-blur-xl">
        <div className="flex items-center gap-8">
          <Logo />
          <nav className="hidden items-center gap-1 md:flex">
            {links.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className="rounded-full px-3 py-1.5 text-sm text-muted transition hover:bg-surface-2 hover:text-fg"
              >
                {link.label}
              </Link>
            ))}
          </nav>
        </div>
        <div className="flex items-center gap-2">
          <ThemeToggle />
          <UserNav />
        </div>
      </div>
    </header>
  );
}
