"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { HandHeart, Lightbulb, Loader2, Send, Trash2 } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";
import { toast } from "sonner";
import { TrackArt } from "@/components/track-art";
import { Badge, Button, buttonClass, Card, EmptyState, Input, Skeleton } from "@/components/ui";
import { api, ApiError, json } from "@/lib/api";
import { cn, formatRelative } from "@/lib/format";
import type { RequestStatus, SongRequest, SongSuggestion, SuggestionStatus } from "@/lib/types";

type Data = { requests: SongRequest[]; suggestions: SongSuggestion[] };

const requestStatus: Record<RequestStatus, { label: string; tone: "default" | "primary" | "accent" | "lime" }> = {
  queued: { label: "Waiting", tone: "primary" },
  playing: { label: "On air now", tone: "accent" },
  played: { label: "Played", tone: "lime" },
  expired: { label: "Expired", tone: "default" },
};

const suggestionStatus: Record<SuggestionStatus, { label: string; tone: "default" | "primary" | "accent" | "lime" }> = {
  pending: { label: "Waiting for review", tone: "default" },
  reviewing: { label: "Being reviewed", tone: "primary" },
  added: { label: "Added", tone: "lime" },
  declined: { label: "Not added", tone: "accent" },
};

export function MyRequests() {
  const [tab, setTab] = useState<"requests" | "suggestions">("requests");
  const { data, isLoading, error } = useQuery({
    queryKey: ["my-requests"],
    queryFn: () => api<Data>("/me/requests"),
    refetchInterval: 20_000,
  });

  return (
    <div className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
      <Card className="overflow-hidden">
        <div className="flex gap-1 border-b border-border p-3">
          {(
            [
              ["requests", "Requests", HandHeart, data?.requests.length],
              ["suggestions", "Suggestions", Lightbulb, data?.suggestions.length],
            ] as const
          ).map(([id, label, Icon, count]) => (
            <button
              key={id}
              onClick={() => setTab(id)}
              className={cn(
                "flex flex-1 cursor-pointer items-center justify-center gap-1.5 rounded-xl py-2 text-sm transition",
                tab === id ? "bg-primary-soft text-fg" : "text-muted hover:bg-surface-2",
              )}
            >
              <Icon className="h-4 w-4" /> {label}
              {count ? <span className="font-mono text-xs text-muted">{count}</span> : null}
            </button>
          ))}
        </div>
        {isLoading ? (
          <div className="space-y-2 p-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-14" />
            ))}
          </div>
        ) : error ? (
          <EmptyState title="Couldn't load your requests">{error.message}</EmptyState>
        ) : tab === "requests" ? (
          <RequestList requests={data?.requests ?? []} />
        ) : (
          <SuggestionList suggestions={data?.suggestions ?? []} />
        )}
      </Card>
      <SuggestForm />
    </div>
  );
}

function RequestList({ requests }: { requests: SongRequest[] }) {
  if (requests.length === 0) {
    return (
      <EmptyState icon={<HandHeart className="h-6 w-6" />} title="No requests yet">
        <p>Find a song in the library and request it. It&apos;ll play on the station soon.</p>
        <Link href="/tracks" className={buttonClass("secondary", "md", "mt-4")}>
          Browse the library
        </Link>
      </EmptyState>
    );
  }
  return (
    <ul className="divide-y divide-border">
      {requests.map((r) => {
        const status = requestStatus[r.status] ?? requestStatus.expired;
        return (
          <li key={r.id} className="flex items-center gap-3 px-4 py-3">
            <TrackArt track={{ artwork_url: r.art }} className="h-11 w-11 rounded-lg" />
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium">{r.title}</p>
              <p className="truncate text-xs text-muted">
                {r.artist} · {r.station} · {formatRelative(r.created_at)}
                {r.source === "discord" && " · from Discord"}
              </p>
            </div>
            <Badge tone={status.tone} className="shrink-0">
              {status.label}
            </Badge>
          </li>
        );
      })}
    </ul>
  );
}

function SuggestionList({ suggestions }: { suggestions: SongSuggestion[] }) {
  const queryClient = useQueryClient();
  const withdraw = useMutation({
    mutationFn: (id: number) => api(`/suggestions/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      toast.success("Suggestion withdrawn");
      queryClient.invalidateQueries({ queryKey: ["my-requests"] });
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : "Couldn't withdraw that"),
  });

  if (suggestions.length === 0) {
    return (
      <EmptyState icon={<Lightbulb className="h-6 w-6" />} title="No suggestions yet">
        Missing a song? Suggest it with the form and we&apos;ll let you know when it&apos;s added.
      </EmptyState>
    );
  }
  return (
    <ul className="divide-y divide-border">
      {suggestions.map((s) => {
        const status = suggestionStatus[s.status] ?? suggestionStatus.pending;
        return (
          <li key={s.id} className="space-y-1.5 px-4 py-3">
            <div className="flex items-center gap-3">
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{s.title}</p>
                <p className="truncate text-xs text-muted">
                  {s.artist} · suggested {formatRelative(s.created_at)}
                </p>
              </div>
              <Badge tone={status.tone} className="shrink-0">
                {status.label}
              </Badge>
              {s.status === "pending" && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 hover:text-danger"
                  aria-label="Withdraw suggestion"
                  disabled={withdraw.isPending}
                  onClick={() => withdraw.mutate(s.id)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              )}
            </div>
            {s.status === "declined" && s.reason && <p className="text-xs text-muted">&ldquo;{s.reason}&rdquo;</p>}
            {s.status === "added" && (
              <Link href={`/tracks?q=${encodeURIComponent(s.title)}`} className="text-xs text-primary hover:underline">
                Find it in the library →
              </Link>
            )}
          </li>
        );
      })}
    </ul>
  );
}

function SuggestForm() {
  const queryClient = useQueryClient();
  const router = useRouter();
  const [form, setForm] = useState({ artist: "", title: "", link: "", note: "" });
  const suggest = useMutation({
    mutationFn: () => api<{ suggestion: SongSuggestion }>("/suggestions", { method: "POST", body: json(form) }),
    onSuccess: ({ suggestion }) => {
      toast.success(`Thanks! ${suggestion.title} is in the review queue`, { description: "We'll DM you when it's reviewed." });
      setForm({ artist: "", title: "", link: "", note: "" });
      queryClient.invalidateQueries({ queryKey: ["my-requests"] });
    },
    onError: (err) => {
      if (err instanceof ApiError && err.code === "in_library") {
        toast.success(err.message, {
          action: { label: "Open", onClick: () => router.push(`/tracks?q=${encodeURIComponent(form.title)}`) },
        });
        return;
      }
      toast.error(err instanceof ApiError ? err.message : "Couldn't send that suggestion");
    },
  });

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (form.artist.trim() && form.title.trim()) suggest.mutate();
  };
  const set = (key: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) => setForm((f) => ({ ...f, [key]: e.target.value }));

  return (
    <Card className="h-fit p-5">
      <h2 className="flex items-center gap-2 font-display text-lg font-semibold">
        <Lightbulb className="h-5 w-5 text-peach" /> Suggest a song
      </h2>
      <p className="mt-1 text-sm text-muted">Can&apos;t find something in the library? Tell us and we&apos;ll look into adding it.</p>
      <form onSubmit={submit} className="mt-4 space-y-3">
        <Input value={form.artist} onChange={set("artist")} placeholder="Artist" maxLength={200} required aria-label="Artist" />
        <Input value={form.title} onChange={set("title")} placeholder="Song title" maxLength={200} required aria-label="Song title" />
        <Input value={form.link} onChange={set("link")} placeholder="Link (optional)" maxLength={500} type="url" aria-label="Link" />
        <Input value={form.note} onChange={set("note")} placeholder="Note (optional)" maxLength={500} aria-label="Note" />
        <Button type="submit" className="w-full" disabled={!form.artist.trim() || !form.title.trim() || suggest.isPending}>
          {suggest.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />} Send suggestion
        </Button>
      </form>
    </Card>
  );
}
