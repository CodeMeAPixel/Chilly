import type { Metadata } from "next";
import { AdminPanel } from "./admin-panel";

export const metadata: Metadata = { title: "Admin" };

export default function AdminPage() {
  return <AdminPanel />;
}
