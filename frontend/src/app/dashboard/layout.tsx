import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { SiteHeader } from "@/components/site-header";
import { getSession } from "@/lib/server";
import { DashboardNav } from "./dashboard-nav";

export const metadata: Metadata = {
  title: { default: "Dashboard", template: "%s · Chilly Dashboard" },
  robots: { index: false },
};

export default async function DashboardLayout({ children }: LayoutProps<"/dashboard">) {
  const session = await getSession();
  if (!session) {
    redirect("/login?next=/dashboard");
  }

  return (
    <>
      <SiteHeader menu={<DashboardNav />} />
      <div className="mx-auto flex w-full max-w-7xl flex-1 gap-8 px-4 pt-26 pb-8 sm:px-6">
        <aside className="hidden w-60 shrink-0 lg:block">
          <DashboardNav className="sticky top-24" />
        </aside>
        <main className="min-w-0 flex-1">{children}</main>
      </div>
    </>
  );
}
