import type { Metadata } from "next";
import { connection } from "next/server";
import { getStatus } from "@/lib/server";
import { StatusBoard } from "./status-board";

export const metadata: Metadata = {
  title: "Status",
  description: "Live status of Chilly's Discord connection, database, audio nodes and radio.",
  openGraph: { url: "/status" },
};

export default async function StatusPage() {
  await connection();
  const status = await getStatus();
  return <StatusBoard initial={status} />;
}
