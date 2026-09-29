"use client";

import { FlaskConical } from "lucide-react";
import { type FormEvent, useState } from "react";
import { Badge, Button, Card, Input } from "@/components/ui";
import { api, ApiError, json } from "@/lib/api";
import { formatDuration } from "@/lib/format";
import type { AdminSearchResult } from "@/lib/types";
import { useAdminOverview } from "./admin-panel";

const sources = [
  { value: "ytsearch", label: "YouTube" },
  { value: "ytmsearch", label: "YouTube Music" },
  { value: "scsearch", label: "SoundCloud" },
  { value: "spsearch", label: "Spotify" },
];

export function ToolsTab() {
  const { data: overview } = useAdminOverview();
  const [query, setQuery] = useState("");
  const [source, setSource] = useState("ytsearch");
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
      setResult(await api<AdminSearchResult>("/admin/search", { method: "POST", body: json({ query, source, node }) }));
    } catch (err) {
      setResult(null);
      setError(err instanceof ApiError ? err.message : "Request failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <Card className="p-5">
        <h2 className="flex items-center gap-2 font-display text-lg font-semibold">
          <FlaskConical className="h-4 w-4 text-primary" /> Track lookup
        </h2>
        <p className="mt-1 text-sm text-muted">
          Load a search or URL on a specific node to see exactly what it returns. Links are loaded as-is; the source only applies
          to searches.
        </p>
        <form onSubmit={submit} className="mt-4 grid gap-3 md:grid-cols-[1fr_auto_auto_auto]">
          <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Song name or URL" />
          <select
            value={source}
            onChange={(e) => setSource(e.target.value)}
            aria-label="Search source"
            className="h-11 cursor-pointer rounded-xl border border-border bg-surface-2 px-3 text-sm"
          >
            {sources.map((s) => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
          </select>
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
            {loading ? "Loading…" : "Run"}
          </Button>
        </form>
      </Card>

      {error && <p className="text-sm text-danger">{error}</p>}

      {result && (
        <Card className="p-5">
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <Badge tone={result.error ? "accent" : result.tracks.length ? "lime" : "default"}>{result.load_type}</Badge>
            <span className="font-mono text-xs text-muted">{result.identifier}</span>
            <span className="text-xs text-muted">
              on {result.node} · {result.took_ms}ms
              {result.total !== undefined && ` · ${result.total} results`}
            </span>
          </div>
          {result.playlist && <p className="mt-3 text-sm">Playlist: {result.playlist}</p>}
          {result.error && (
            <pre className="mt-3 max-h-64 overflow-auto rounded-xl bg-surface-2 p-3 font-mono text-xs whitespace-pre-wrap text-danger">
              {result.error}
              {result.cause ? `\n\n${result.cause}` : ""}
            </pre>
          )}
          {result.tracks.length > 0 && (
            <ul className="mt-3 divide-y divide-border rounded-2xl border border-border">
              {result.tracks.map((t, i) => (
                <li key={i} className="flex items-center gap-3 px-4 py-2.5 text-sm">
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-medium">
                      {t.uri ? (
                        <a href={t.uri} target="_blank" rel="noreferrer" className="hover:underline">
                          {t.title}
                        </a>
                      ) : (
                        t.title
                      )}
                    </p>
                    <p className="truncate text-xs text-muted">{t.author}</p>
                  </div>
                  <Badge>{t.source}</Badge>
                  <span className="w-14 text-right font-mono text-xs text-muted">
                    {t.is_stream ? "live" : formatDuration(t.length_ms)}
                  </span>
                </li>
              ))}
            </ul>
          )}
          <p className="mt-3 text-xs text-muted">
            A successful lookup doesn&apos;t guarantee playback. Stream errors, such as a failing cipher server, show up in Logs when
            a track is played.
          </p>
        </Card>
      )}
    </div>
  );
}
