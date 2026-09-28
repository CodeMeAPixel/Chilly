import type { Metadata } from "next";
import Image from "next/image";
import { redirect } from "next/navigation";
import { buttonClass, Card } from "@/components/ui";
import { loginUrl } from "@/lib/api";
import { getSession } from "@/lib/server";

export const metadata: Metadata = {
  title: "Log in",
  robots: { index: false },
};

const errors: Record<string, string> = {
  access_denied: "You cancelled the Discord login.",
};

function safeNext(next: string | string[] | undefined) {
  const value = Array.isArray(next) ? next[0] : next;
  return value && value.startsWith("/") && !value.startsWith("//") ? value : "/dashboard";
}

export default async function LoginPage(props: PageProps<"/login">) {
  const params = await props.searchParams;
  const next = safeNext(params.next);
  if (await getSession()) {
    redirect(next);
  }
  const error = typeof params.error === "string" ? errors[params.error] ?? "Login failed. Please try again." : null;

  return (
    <div className="mx-auto flex max-w-md flex-col items-center px-4 py-24">
      <Card className="w-full space-y-6 p-8 text-center">
        <Image src="/logo.svg" alt="" width={72} height={72} className="mx-auto animate-float" />
        <div className="space-y-2">
          <h1 className="font-display text-2xl font-semibold">Welcome back</h1>
          <p className="text-sm text-muted">Log in with Discord to control Chilly and manage your playlists.</p>
        </div>
        {error && <p className="rounded-xl bg-danger/10 px-4 py-2 text-sm text-danger">{error}</p>}
        <a href={loginUrl(next)} className={buttonClass("primary", "lg", "w-full bg-[#5865F2] text-white shadow-none")}>
          <svg viewBox="0 0 24 24" className="h-5 w-5 fill-current" aria-hidden>
            <path d="M20.3 4.4A19.8 19.8 0 0 0 15.4 3l-.6 1.3a18.4 18.4 0 0 0-5.6 0L8.6 3a19.7 19.7 0 0 0-4.9 1.5C.6 9.1-.3 13.6.1 18.1a19.9 19.9 0 0 0 6 3l1.3-2a13 13 0 0 1-2-1l.5-.4a14.2 14.2 0 0 0 12.2 0l.5.4-2 1 1.3 2a19.8 19.8 0 0 0 6-3c.5-5.2-.8-9.7-3.6-13.7ZM8 15.4c-1.2 0-2.2-1.1-2.2-2.4s1-2.4 2.2-2.4 2.2 1.1 2.2 2.4-1 2.4-2.2 2.4Zm8 0c-1.2 0-2.2-1.1-2.2-2.4s1-2.4 2.2-2.4 2.2 1.1 2.2 2.4-1 2.4-2.2 2.4Z" />
          </svg>
          Continue with Discord
        </a>
        <p className="text-xs text-muted">We only ask for your username and server list.</p>
      </Card>
    </div>
  );
}
