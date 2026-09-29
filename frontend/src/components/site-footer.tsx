import Link from "next/link";
import { Logo } from "./logo";

export function SiteFooter() {
  return (
    <footer className="mt-24 border-t border-border">
      <div className="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-10 sm:flex-row sm:items-center sm:justify-between sm:px-6">
        <div className="space-y-2">
          <Logo />
          <p className="text-sm text-muted">24/7 radio for your Discord server.</p>
        </div>
        <nav className="flex flex-wrap gap-x-6 gap-y-2 text-sm text-muted">
          <Link href="/#features" className="hover:text-fg">Features</Link>
          <Link href="/#commands" className="hover:text-fg">Commands</Link>
          <Link href="/radio" className="hover:text-fg">Radio</Link>
          <Link href="/tracks" className="hover:text-fg">Tracks</Link>
          <Link href="/status" className="hover:text-fg">Status</Link>
          <Link href="/brand" className="hover:text-fg">Brand</Link>
          <Link href="/dashboard" className="hover:text-fg">Dashboard</Link>
          <a href="https://github.com/CodeMeAPixel/Chilly" target="_blank" rel="noreferrer" className="hover:text-fg">GitHub</a>
        </nav>
      </div>
    </footer>
  );
}
