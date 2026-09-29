"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Loader2, Megaphone, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import { toast } from "sonner";
import { Badge, Button, Card, Input } from "@/components/ui";
import { api, ApiError, json } from "@/lib/api";
import { formatRelative } from "@/lib/format";
import type { StatusIncident, SystemStatus } from "@/lib/types";

export function StatusNotices() {
  const queryClient = useQueryClient();
  const { data: status } = useQuery({
    queryKey: ["status"],
    queryFn: () => api<SystemStatus>("/status"),
    refetchInterval: 15_000,
  });
  const [kind, setKind] = useState<"notice" | "maintenance">("notice");
  const [title, setTitle] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState<string | null>(null);

  const open: StatusIncident[] = [...(status?.notices ?? []), ...(status?.incidents ?? []).filter((i) => i.ongoing)];

  const refresh = () => queryClient.invalidateQueries({ queryKey: ["status"] });

  const post = async (e: FormEvent) => {
    e.preventDefault();
    if (!title.trim()) return;
    setBusy("post");
    try {
      await api("/admin/status/notices", { method: "POST", body: json({ kind, title, message }) });
      toast.success("Posted to the status page");
      setTitle("");
      setMessage("");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Couldn't post that");
    } finally {
      setBusy(null);
    }
  };

  const act = async (incident: StatusIncident, action: "resolve" | "delete") => {
    if (action === "delete" && !window.confirm(`Delete "${incident.title}"? It disappears from the status page.`)) return;
    setBusy(`${action}-${incident.id}`);
    try {
      await api(`/admin/status/incidents/${incident.id}${action === "resolve" ? "/resolve" : ""}`, {
        method: action === "resolve" ? "POST" : "DELETE",
      });
      toast.success(action === "resolve" ? "Marked resolved" : "Deleted");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Couldn't update that");
    } finally {
      setBusy(null);
    }
  };

  return (
    <Card className="p-5">
      <h2 className="flex items-center gap-2 font-display text-lg font-semibold">
        <Megaphone className="h-4 w-4 text-primary" /> Status notices
      </h2>
      <p className="mt-1 text-sm text-muted">
        Post announcements or planned maintenance to the public status page. Outages are recorded automatically.
      </p>

      <form onSubmit={post} className="mt-4 grid gap-3 md:grid-cols-[auto_1fr]">
        <select
          value={kind}
          onChange={(e) => setKind(e.target.value as "notice" | "maintenance")}
          aria-label="Notice type"
          className="h-11 cursor-pointer rounded-xl border border-border bg-surface-2 px-3 text-sm"
        >
          <option value="notice">Announcement</option>
          <option value="maintenance">Maintenance</option>
        </select>
        <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Title" maxLength={200} aria-label="Title" />
        <textarea
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder="Details (optional)"
          maxLength={2000}
          rows={3}
          aria-label="Details"
          className="rounded-xl border border-border bg-surface-2 px-4 py-3 text-sm outline-none focus:border-primary/50 focus:ring-2 focus:ring-ring md:col-span-2"
        />
        <Button type="submit" disabled={!title.trim() || busy !== null} className="md:col-span-2 md:justify-self-start">
          {busy === "post" && <Loader2 className="h-4 w-4 animate-spin" />} Post to status page
        </Button>
      </form>

      {open.length > 0 && (
        <ul className="mt-5 divide-y divide-border rounded-2xl border border-border">
          {open.map((i) => (
            <li key={i.id} className="flex flex-wrap items-center gap-3 px-4 py-3 text-sm">
              <div className="min-w-0 flex-1">
                <p className="flex items-center gap-2 font-medium">
                  <Badge tone={i.kind === "auto" ? "accent" : i.kind === "maintenance" ? "default" : "primary"}>
                    {i.kind === "auto" ? "outage" : i.kind}
                  </Badge>
                  <span className="truncate">{i.title}</span>
                </p>
                <p className="text-xs text-muted">started {formatRelative(i.started_at)}</p>
              </div>
              <Button size="sm" variant="secondary" disabled={busy !== null} onClick={() => act(i, "resolve")}>
                <CheckCircle2 className="h-4 w-4" /> Resolve
              </Button>
              <Button size="sm" variant="ghost" disabled={busy !== null} onClick={() => act(i, "delete")} aria-label="Delete">
                <Trash2 className="h-4 w-4" />
              </Button>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
