import Image from "next/image";
import Link from "next/link";
import { cn } from "@/lib/format";

export function Logo({ className, href = "/" }: { className?: string; href?: string }) {
  return (
    <Link href={href} className={cn("group inline-flex items-center gap-2.5", className)}>
      <Image
        src="/logo.svg"
        alt=""
        width={36}
        height={36}
        priority
        className="transition-transform duration-300 group-hover:-rotate-12"
      />
      <span className="font-display text-xl font-semibold tracking-tight">Chilly</span>
    </Link>
  );
}
