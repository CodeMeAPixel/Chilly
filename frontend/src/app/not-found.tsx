import Image from "next/image";
import Link from "next/link";
import { buttonClass } from "@/components/ui";

export default function NotFound() {
  return (
    <main className="flex flex-1 flex-col items-center justify-center gap-6 px-4 py-24 text-center">
      <Image src="/logo.svg" alt="" width={120} height={120} className="animate-float opacity-90" />
      <div className="space-y-2">
        <h1 className="font-display text-4xl font-semibold">This page melted</h1>
        <p className="text-muted">We couldn&apos;t find what you were looking for.</p>
      </div>
      <Link href="/" className={buttonClass("primary", "md")}>
        Back home
      </Link>
    </main>
  );
}
