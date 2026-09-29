"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ExternalLink, Lightbulb } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { TrackArt } from "@/components/track-art";
import { Badge, Button, Card, EmptyState, Skeleton } from "@/components/ui";
import { api, ApiError, json } from "@/lib/api";
import { cn, formatRelative } from "@/lib/format";
import type { SongRequest, SongSuggestion, SuggestionStatus } from "@/lib/types";

const filters: { value: "" | SuggestionStatus; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "reviewing", label: "Reviewing" },
  { value: "added", label: "Added" },
  { value: "declined", label: "Declined" },
  { value: "", label: "All" },
];

const tones: Record<SuggestionStatus, "default" | "primary" | "accent" | "lime"> = {
  pending: "default",
  reviewing: "primary",
  added: "lime",
  declined: "accent",
};

export function RequestsTab() {
  const [status, setStatus] = useState<"" | SuggestionStatus>("pending");
  const queryClient = useQueryClient();
  const suggestions = useQuery({
    queryKey: ["admin", "suggestions", status],
    queryFn: async () =>
      (await api<{ suggestions: SongSuggestion[] }>(`/admin/suggestions${status ? `?status=${status}` : ""}`)).suggestions,
    refetchInterval: 30_000,
  });
  const requests = useQuery({
    queryKey: ["admin", "requests"],
    queryFn: async () => (await api<{ requests: SongRequest[] }>("/admin/requests")).requests,
    refetchInterval: 20_000,
  });

  const review = useMutation({
    mutationFn: ({ id, next, reason }: { id: number; next: SuggestionStatus; reason?: string }) =>
      api<{ suggestion: SongSuggestion }>(`/admin/suggestions/${id}`, {
        method: "PATCH",
        body: json({ status: next, reason: reason ?? "" }),
      }),
    onSuccess: ({ suggestion }) => {
      toast.success(`${suggestion.title} marked ${suggestion.status}`, {
        description: suggestion.status === "added" || suggestion.status === "declined" ? "The suggester was sent a DM." : undefined,
      });
      queryClient.invalidateQueries({ queryKey: ["admin", "suggestions"] });
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : "Couldn't update that suggestion"),
  });

  const decline = (s: SongSuggestion) => {
    const reason = window.prompt(`Why isn't "${s.title}" being added? This is sent to the user (optional).`, "");
    if (reason === null) return;
    review.mutate({ id: s.id, next: "declined", reason });
  };

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border px-5 py-4">
          <h2 className="flex items-center gap-2 font-display text-lg font-semibold">
            <Lightbulb className="h-4 w-4 text-peach" /> Suggestions
          </h2>
          <div className="flex overflow-x-auto rounded-xl border border-border bg-surface p-1">
            {filters.map((f) => (
              <button
                key={f.label}
                onClick={() => setStatus(f.value)}
                className={cn(
                  "h-8 shrink-0 cursor-pointer rounded-lg px-3 text-sm transition",
                  status === f.value ? "bg-primary-soft text-fg" : "text-muted hover:text-fg",
                )}
              >
                {f.label}
              </button>
            ))}
          </div>
        </div>
        {suggestions.isLoading ? (
          <div className="space-y-2 p-4">
            <Skeleton className="h-16" />
            <Skeleton className="h-16" />
          </div>
        ) : !suggestions.data?.length ? (
          <EmptyState icon={<Lightbulb className="h-6 w-6" />} title="Nothing here">
            No suggestions with this status.
          </EmptyState>
        ) : (
          <ul className="divide-y divide-border">
            {suggestions.data.map((s) => (
              <li key={s.id} className="flex flex-col gap-3 px-5 py-4 lg:flex-row lg:items-center">
                <div className="min-w-0 flex-1 space-y-1">
                  <p className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">{s.title}</span>
                    <span className="text-sm text-muted">by {s.artist}</span>
                    <Badge tone={tones[s.status]}>{s.status}</Badge>
                  </p>
                  <p className="text-xs text-muted">
                    <span className="font-mono">{s.user_id}</span> · {formatRelative(s.created_at)}
                    {s.library_key && <span className="font-mono"> · {s.library_key}</span>}
                  </p>
                  {s.note && <p className="text-sm text-muted">&ldquo;{s.note}&rdquo;</p>}
                  {s.reason && <p className="text-xs text-muted">Reason: {s.reason}</p>}
                  {s.link && (
                    <a
                      href={s.link}
                      target="_blank"
                      rel="noreferrer noopener"
                      className="inline-flex max-w-full items-center gap-1 text-xs text-primary hover:underline"
                    >
                      <ExternalLink className="h-3 w-3 shrink-0" /> <span className="truncate">{s.link}</span>
                    </a>
                  )}
                </div>
                <div className="flex flex-wrap gap-2">
                  {s.status === "pending" && (
                    <Button size="sm" variant="secondary" disabled={review.isPending} onClick={() => review.mutate({ id: s.id, next: "reviewing" })}>
                      Start review
                    </Button>
                  )}
                  {(s.status === "pending" || s.status === "reviewing") && (
                    <>
                      <Button size="sm" disabled={review.isPending} onClick={() => review.mutate({ id: s.id, next: "added" })}>
                        Mark added
                      </Button>
                      <Button size="sm" variant="danger" disabled={review.isPending} onClick={() => decline(s)}>
                        Decline
                      </Button>
                    </>
                  )}
                  {(s.status === "added" || s.status === "declined") && (
                    <Button size="sm" variant="ghost" disabled={review.isPending} onClick={() => review.mutate({ id: s.id, next: "pending" })}>
                      Reopen
                    </Button>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
        <p className="border-t border-border px-5 py-3 text-xs text-muted">
          Suggestions are marked added automatically when a matching song shows up in the library after a sync.
        </p>
      </Card>

      <Card className="overflow-hidden">
        <h2 className="border-b border-border px-5 py-4 font-display text-lg font-semibold">Recent requests</h2>
        {requests.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-40" />
          </div>
        ) : !requests.data?.length ? (
          <EmptyState title="No requests yet" />
        ) : (
          <ul className="divide-y divide-border">
            {requests.data.map((r) => (
              <li key={r.id} className="flex items-center gap-3 px-5 py-2.5 text-sm">
                <TrackArt track={{ artwork_url: r.art }} className="h-9 w-9 rounded-lg" />
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium">
                    {r.title} <span className="font-normal text-muted">· {r.artist}</span>
                  </p>
                  <p className="truncate text-xs text-muted">
                    {r.station} · <span className="font-mono">{r.user_id}</span> · {r.source} · {formatRelative(r.created_at)}
                  </p>
                </div>
                <Badge>{r.status}</Badge>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
