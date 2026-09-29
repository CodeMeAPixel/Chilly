import type { Metadata } from "next";
import { MyRequests } from "./my-requests";

export const metadata: Metadata = { title: "My requests" };

export default function RequestsPage() {
  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h1 className="font-display text-3xl font-semibold tracking-tight">My requests</h1>
        <p className="text-sm text-muted">Songs you&apos;ve asked the stations to play, and songs you&apos;ve suggested for the library.</p>
      </div>
      <MyRequests />
    </div>
  );
}
