# Agent guide

Instructions for AI coding agents (and a quick orientation for humans) working in this repository. Read this before changing anything, then read [CONTRIBUTING.md](CONTRIBUTING.md) for the workflow and review rules that apply to everyone.

## What this project is

Chilly is a self-hosted Discord music bot with a web dashboard.

| Path | Stack | Role |
| --- | --- | --- |
| `backend/` | Go 1.22, disgo, disgolink v3, pgx, PostgreSQL | Discord bot, Lavalink client, HTTP API (`/api/v1`) |
| `frontend/` | Next.js 16 (App Router), React 19, Tailwind v4, TanStack Query, bun | Website, dashboard, status page |

External services: Lavalink v4 nodes (with the youtube-source, LavaSrc and LavaSearch plugins), PostgreSQL, optional AzuraCast, and Discord. Everything is deployed with Docker on Dokploy.

## Ground rules

1. **Do not add code comments.** Code must explain itself through naming and structure. The only exceptions are functional directives such as `//go:embed`. Put explanations in commit messages, pull requests or the README instead.
2. **Never log, print, commit or echo secrets.** This includes the Discord bot token, Lavalink passwords, OAuth client secrets, refresh tokens, session tokens and database credentials. `.env` files are gitignored; keep it that way. When you need a value from `.env` in a command, read it into a variable without printing it.
3. **Keep changes scoped.** Do not reformat, rename or "tidy" code you were not asked to touch.
4. **Verify before you claim success.** Run the checks in [Verification](#verification) and report failures honestly, including the output.
5. **Match the surrounding code.** Follow the idioms, naming and file layout already used in the package you are editing.
6. **Do not guess at library behaviour.** Read the source in the Go module cache, or the bundled Next.js docs, before relying on an API.
7. **Only stop processes you started.** Never kill processes by image name (for example `taskkill /IM node.exe`). Track the PID and stop that one.

## Repository map

```text
backend/
├── main.go          Wiring: config, logging, disgo client, handlers, API, radio
├── api/             HTTP API: auth (Discord OAuth2), sessions, guild player, playlists, radio, status
├── azuracast/       AzuraCast client and now-playing poller
├── commands/        Slash command definitions and handlers
├── handlers/        Discord gateway handlers, Lavalink event handlers, player buttons
├── musicbot/        Core domain: Player, PlayerManager, Searcher, voice, nodes, watchdog, DB
└── db/schema.sql    Embedded schema, applied idempotently on startup
frontend/
├── src/app/(site)/  Public pages: landing, radio, status, login
├── src/app/dashboard/  Authenticated dashboard: servers, live player, playlists
├── src/app/api/v1/[...path]/route.ts  Runtime proxy to the Go API
├── src/proxy.ts     Redirects signed-out users away from /dashboard
├── src/components, src/hooks, src/lib
└── src/app/*-image.tsx, manifest.ts, robots.ts, sitemap.ts  Metadata routes
```

## Commands

Backend (run from `backend/`):

```bash
make run        # run the bot (reads .env)
make check      # gofmt check, go vet, unit tests
make test       # unit tests
make build      # bin/chilly
```

Frontend (run from `frontend/`):

```bash
bun install
bun dev
bun run lint
bun run typecheck   # regenerates route types, then tsc
bun run build
```

## Backend architecture you must respect

- **Player state is guarded by `Player.mu`.** Every method that touches queue, current track or flags takes the lock. Internal helpers that expect the lock to be held end in `Locked`. Do not add unlocked access to player fields.
- **Always resolve the Lavalink player through `lavalinkPlayer` / `PlayerManager.LavalinkPlayer`** and nodes through `musicbot.BestNode`. Never call `link.Player` or `link.BestNode` directly: disgolink's versions can select a disconnected or overloaded node.
- **Voice events must stay synchronous and ordered.** `OnVoiceStateUpdate` and `OnVoiceServerUpdate` run on the gateway goroutine on purpose. Slash commands, autocomplete and component interactions are dispatched through the `concurrent` wrapper in `main.go`. Do not re-enable disgo's async events globally; it reorders voice events and leaves players silent.
- **Wait for voice before playing.** Use `Bot.EnsureVoice` / `Bot.Enqueue`; they wait for Lavalink to receive voice credentials before a track is sent.
- **Stale Lavalink events are ignored** by comparing the ended track's `Encoded` value with the current track. Keep that check when changing track-end handling.
- **Discord limits:** autocomplete choice names and values must be 100 characters or fewer, and autocomplete must answer within 3 seconds (the code budgets 2.5 s). Use `Searcher.Remember` for values.
- **Library loggers are capped at Info.** disgo logs the gateway token at Debug level, so never pass the application log level through to disgo or disgolink.
- **API conventions:** routes live under `/api/v1`, errors are `{"error":{"code","message"}}`, authenticated handlers use `s.authed`, and cookie-authenticated writes are CSRF-checked against allowed origins. When you add or change a route, update the API table in the root README.
- **Schema changes** go in `db/schema.sql` and must be idempotent (`IF NOT EXISTS`), because the file runs on every start.

## Frontend architecture you must respect

- **This is Next.js 16.** APIs differ from older versions: `middleware.ts` is now `proxy.ts`, and `params`, `searchParams`, `cookies()` and `headers()` are async only. Read the bundled docs in `frontend/node_modules/next/dist/docs/` before using a Next.js API. The Next-managed notes live in [frontend/AGENTS.md](frontend/AGENTS.md).
- **All browser API calls go through `/api/v1`** using `api()` from `src/lib/api.ts`. The route handler proxies to `BACKEND_URL` at runtime, which keeps the session cookie same-origin and lets server-sent events stream. Do not call the Go API directly from the browser and do not add CORS.
- **Server-only data access** uses `src/lib/server.ts` (`import "server-only"`). Pages that read runtime data call `await connection()` so `BACKEND_URL` is read at request time rather than baked in at build.
- **React rules are enforced by ESLint:** no synchronous `setState` in effects and no ref reads during render. Derive values, key components to reset state, or use `useEffectEvent`.
- **Styling** uses the design tokens in `src/app/globals.css` (`bg`, `surface`, `surface-2`, `fg`, `muted`, `primary`, `accent`, `peach`, `lime`). Use them instead of hard-coded colours so light and dark mode keep working.
- **Metadata:** Open Graph images use `renderOgImage` from `src/lib/og.tsx`. New public pages should export `metadata` and, where it adds value, their own `opengraph-image.tsx`.

## Verification

Run everything that applies to what you changed before finishing:

```bash
cd backend && make check
cd frontend && bun run lint && bun run typecheck && bun run build
```

For UI changes, also run the app and look at the affected pages in light and dark mode and at a narrow width. For backend changes that affect playback, say clearly which parts were verified by tests and which still need a manual test against a live Lavalink node and Discord.

## Operational notes

- Lavalink needs the youtube-source plugin (with the OAuth `TV` client and a remote cipher server), LavaSrc and LavaSearch. The built-in YouTube source must stay disabled (`lavalink.server.sources.youtube: false`, `plugins.lavasrc.sources.youtube: false`).
- On startup the bot logs the node's sources and plugins and disables search providers the node cannot serve. Check that log line first when searches fail.
- `GET /api/v1/status` and the `/status` page show component health. Uptime history is in memory and resets on restart.
