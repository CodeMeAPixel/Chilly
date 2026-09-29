"use client";

import { useQueryClient } from "@tanstack/react-query";
import { HandHeart, LayoutDashboard, ListMusic, LogOut, ShieldCheck } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useMe } from "@/hooks/use-me";
import { api, loginUrl } from "@/lib/api";
import { userAvatar } from "@/lib/format";
import { buttonClass, Skeleton } from "./ui";

export function UserNav() {
  const { data: me, isLoading } = useMe();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const pathname = usePathname();
  const router = useRouter();
  const queryClient = useQueryClient();

  useEffect(() => {
    const close = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  if (isLoading) {
    return <Skeleton className="h-10 w-10 rounded-full" />;
  }

  if (!me) {
    return (
      <a href={loginUrl(pathname.startsWith("/dashboard") ? pathname : "/dashboard")} className={buttonClass("primary", "md", "rounded-full")}>
        Log in
      </a>
    );
  }

  const logout = async () => {
    await api("/auth/logout", { method: "POST" }).catch(() => undefined);
    queryClient.clear();
    router.push("/");
    router.refresh();
  };

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-2 rounded-full border border-border bg-surface p-1 pr-3 transition hover:border-primary/40 cursor-pointer"
        aria-expanded={open}
      >
        <img src={userAvatar(me.id, me.avatar)} alt="" className="h-8 w-8 rounded-full" />
        <span className="hidden max-w-32 truncate text-sm font-medium sm:block">{me.global_name ?? me.username}</span>
      </button>
      {open && (
        <div className="absolute right-0 z-50 mt-2 w-52 overflow-hidden rounded-2xl border border-border bg-surface p-1.5 shadow-2xl">
          <Link href="/dashboard" onClick={() => setOpen(false)} className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm hover:bg-surface-2">
            <LayoutDashboard className="h-4 w-4 text-muted" /> Dashboard
          </Link>
          <Link href="/dashboard/playlists" onClick={() => setOpen(false)} className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm hover:bg-surface-2">
            <ListMusic className="h-4 w-4 text-muted" /> Playlists
          </Link>
          <Link href="/dashboard/requests" onClick={() => setOpen(false)} className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm hover:bg-surface-2">
            <HandHeart className="h-4 w-4 text-muted" /> My requests
          </Link>
          {me.admin && (
            <Link href="/dashboard/admin" onClick={() => setOpen(false)} className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm hover:bg-surface-2">
              <ShieldCheck className="h-4 w-4 text-muted" /> Admin
            </Link>
          )}
          <button onClick={logout} className="flex w-full items-center gap-2 rounded-xl px-3 py-2 text-sm text-danger hover:bg-danger/10 cursor-pointer">
            <LogOut className="h-4 w-4" /> Log out
          </button>
        </div>
      )}
    </div>
  );
}
