"use client";

import { CheckCircle2, FlaskConical, XCircle } from "lucide-react";
import { type FormEvent, useState } from "react";
import { TrackArt } from "@/components/track-art";
import { Badge, Button, Card, Input } from "@/components/ui";
import { api, ApiError, json } from "@/lib/api";
import { formatDuration } from "@/lib/format";
import type { AdminSearchResult } from "@/lib/types";
import { useAdminOverview } from "./admin-panel";

export function ToolsTab() {
  const { data: overview } = useAdminOverview();
  const [query, setQuery] = useState("");
  const [node, setNode] = useState("");
  const [result, setResult] = useState<AdminSearchResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;
    setLoading(true);
    setError(null);
    try {
      setResult(await api<AdminSearchResult>("/admin/search", { method: "POST", body: json({ query, node }) }));
    } catch (err) {
      setResult(null);
      setError(err instanceof ApiError ? err.message : "Request failed");
    } finally {
      setLoading(false);
    }
  };

  const test = result?.playback_test;

  return (
    <div className="space-y-4">
      <Card className="p-5">
        <h2 className="flex items-center gap-2 font-display text-lg font-semibold">
          <FlaskConical className="h-4 w-4 text-primary" /> Library check
        </h2>
        <p className="mt-1 text-sm text-muted">
          Search the library, then ask a node to load the top result through the media proxy. A failed load usually means
          the node can&apos;t reach <code>MEDIA_BASE_URL</code> or its HTTP source is off.
        </p>
        <form onSubmit={submit} className="mt-4 grid gap-3 md:grid-cols-[1fr_auto_auto]">
          <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Song, artist or album" />
          <select
            value={node}
            onChange={(e) => setNode(e.target.value)}
            aria-label="Node"
            className="h-11 cursor-pointer rounded-xl border border-border bg-surface-2 px-3 text-sm"
          >
            <option value="">Best node</option>
            {(overview?.nodes ?? []).map((n) => (
              <option key={n.name} value={n.name} disabled={n.status !== "CONNECTED"}>
                {n.name}
                {n.location ? ` (${n.location})` : ""}
              </option>
            ))}
          </select>
          <Button type="submit" disabled={loading || !query.trim()} className="h-11">
            {loading ? "Checking…" : "Run"}
          </Button>
        </form>
      </Card>

      {error && <p className="text-sm text-danger">{error}</p>}

      {result && (
        <Card className="space-y-4 p-5">
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <Badge tone={result.library.enabled ? "primary" : "accent"}>
              {result.library.enabled ? `${result.library.tracks} songs in library` : "Library disabled"}
            </Badge>
            {result.library.last_sync && (
              <span className="text-xs text-muted">synced {new Date(result.library.last_sync).toLocaleTimeString()}</span>
            )}
            {result.library.media_url && <span className="font-mono text-xs text-muted">{result.library.media_url}</span>}
          </div>
          {result.library.error && <p className="text-xs text-danger">{result.library.error}</p>}

          {test && (
            <div
              className={`rounded-2xl border p-4 text-sm ${test.ok ? "border-lime/30 bg-lime/10" : "border-danger/30 bg-danger/10"}`}
            >
              <p className="flex items-center gap-2 font-medium">
                {test.ok ? <CheckCircle2 className="h-4 w-4 text-lime" /> : <XCircle className="h-4 w-4 text-danger" />}
                {test.ok ? "Playback test passed" : "Playback test failed"}
                <span className="font-normal text-muted">
                  · {test.track} on {test.node} · {test.took_ms}ms
                  {test.length_ms ? ` · ${formatDuration(test.length_ms)}` : ""}
                </span>
              </p>
              {test.error && (
                <pre className="mt-2 max-h-48 overflow-auto font-mono text-xs whitespace-pre-wrap text-danger">
                  {test.error}
                  {test.cause ? `\n\n${test.cause}` : ""}
                </pre>
              )}
            </div>
          )}

          {result.tracks.length === 0 ? (
            <p className="text-sm text-muted">No songs match that search.</p>
          ) : (
            <ul className="divide-y divide-border rounded-2xl border border-border">
              {result.tracks.map((t) => (
                <li key={t.value} className="flex items-center gap-3 px-4 py-2.5 text-sm">
                  <TrackArt track={t} className="h-9 w-9 rounded-lg" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-medium">{t.title}</p>
                    <p className="truncate text-xs text-muted">{[t.author, t.album].filter(Boolean).join(" · ")}</p>
                  </div>
                  <Badge>{t.station}</Badge>
                  <span className="w-12 text-right font-mono text-xs text-muted">{formatDuration(t.length_ms)}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>
      )}
    </div>
  );
}
