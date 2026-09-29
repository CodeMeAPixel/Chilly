# Chilly

A self-hosted 24/7 radio bot for Discord, with a web dashboard. It streams your own [AzuraCast](https://www.azuracast.com) stations into voice channels, can keep them playing around the clock, and plays songs on request through [Lavalink](https://lavalink.dev).

This repository contains both halves of the project:

| Path | What it is | Status |
| --- | --- | --- |
| [`backend/`](backend) | Discord bot and HTTP API, written in Go | Active |
| [`frontend/`](frontend) | Website and dashboard, written in Next.js | Active |

## Features

- **Radio.** Stream AzuraCast stations with `/radio`. Now-playing info updates live and dropped streams reconnect automatically.
- **24/7 mode.** `/247 on` keeps a station playing in a voice channel even when it's empty. The setting is saved in PostgreSQL, and the bot rejoins and restarts the station after restarts, disconnects or when a song queue runs out.
- **Web radio.** Browse, search and listen to every station at `/radio` on the website, or send one to a voice channel.
- **Songs on request.** Play from YouTube, SoundCloud, Spotify, Deezer or Apple Music, depending on the Lavalink plugins you run, with queue control, lyrics and saved playlists.
- **Dashboard and admin panel.** A live web player for every server, plus an admin area for bot developers with server management, a track lookup tester and recent logs.
- **Resilient playback.** Multiple Lavalink nodes with automatic failover, voice recovery and a public status page.

## Architecture

```text
                ┌──────────────┐
 Discord  ◄────►│              │◄────► Lavalink node(s)
                │   backend    │
 Dashboard ◄───►│  (Go, :8080) │◄────► PostgreSQL
                │              │
                └──────┬───────┘
                       └────────────► AzuraCast (optional)
```

The backend is a single Go binary. It runs the Discord bot, the Lavalink client and the HTTP API. The Next.js frontend serves the website and dashboard and proxies `/api/v1` to the backend, so login cookies and live updates work without any cross-origin setup.

## Getting started

### Requirements

- Go 1.22+
- PostgreSQL 14+
- One or more [Lavalink v4](https://lavalink.dev) nodes
- A Discord application with a bot token
- Optional: GNU Make, Docker, an AzuraCast instance

### Run the backend

```bash
cd backend
cp .env.example .env    # fill in BOT_TOKEN, NODE_*, DB_*
make run
```

The database schema is applied automatically on startup. Run `make` or `make help` to list every available target.

### Run the website

```bash
cd frontend
cp .env.example .env.local   # BACKEND_URL, SITE_URL
bun install
bun dev
```

See [`frontend/README.md`](frontend/README.md) for the backend settings needed for Discord login in development.

### Run with Docker

```bash
cd backend
make docker-build
make docker-run
```

## Configuration

Configuration comes from environment variables, or from a `.env` file in the working directory. [`backend/.env.example`](backend/.env.example) documents every option. The main groups are:

| Prefix | Purpose |
| --- | --- |
| `BOT_*` | Token, status and activity |
| `NODE_*`, `NODE_1_*` … `NODE_10_*` | Lavalink nodes, including an optional `LOCATION` shown on the status page |
| `SEARCH_PROVIDERS` | Search sources tried, in order, for plain-text queries |
| `DB_*` | PostgreSQL connection |
| `API_*`, `DISCORD_CLIENT_*` | HTTP API, CORS, cookies and OAuth2 |
| `AZURACAST_*` | Radio integration |
| `LOG_*` | Log level, format and output |

### Enabling the API

1. Set `API_ENABLED=true`, `API_PUBLIC_URL`, `API_DASHBOARD_URL` and `API_ALLOWED_ORIGINS`.
2. Copy the OAuth2 client secret from the Discord developer portal into `DISCORD_CLIENT_SECRET`.
3. Add `<API_PUBLIC_URL>/api/v1/auth/callback` as an OAuth2 redirect URL in the developer portal.
4. If the dashboard and API are on different subdomains, set `API_COOKIE_DOMAIN` to the parent domain, for example `.example.com`.

### Enabling radio

1. Set `AZURACAST_ENABLED=true` and `AZURACAST_URL`.
2. Enable Lavalink's HTTP source by setting `lavalink.server.sources.http: true`.
3. If Lavalink reaches AzuraCast over an internal network, set `AZURACAST_STREAM_BASE_URL`, for example `http://azuracast:80`.

## Commands

| Command | Description |
| --- | --- |
| `/radio play`, `/radio now`, `/radio stations` | AzuraCast radio (when enabled) |
| `/247 on`, `/247 off`, `/247 status` | Keep a station playing 24/7 (Manage Server, when radio is enabled) |
| `/play`, `/search` | Play a track, album or playlist from a query or URL |
| `/playlist` | Play one of your saved playlists |
| `/queue`, `/now` | Show the queue or the current track |
| `/lyrics` | Lyrics for the current song, or any song you search for |
| `/pause`, `/resume`, `/skip`, `/stop`, `/seek` | Playback control |
| `/shuffle`, `/loop`, `/remove` | Queue control |
| `/list create`, `/list add`, `/list remove`, `/list delete`, `/list list` | Manage your playlists |
| `/join`, `/leave`, `/help`, `/invite`, `/ping` | General |

To control playback, you need to be in the bot's voice channel. While 24/7 radio is on, only members with Manage Server can stop the bot or make it leave, and doing so turns 24/7 off.

## API overview

All routes live under `/api/v1`. Authenticated routes accept the session cookie or an `Authorization: Bearer <token>` header.

| Method | Route | Auth |
| --- | --- | --- |
| `GET` | `/health`, `/stats`, `/status` | — |
| `GET` | `/radio/stations`, `/radio/stations/{station}` | — |
| `GET` | `/auth/login`, `/auth/callback` | — |
| `POST` | `/auth/logout` | — |
| `GET` | `/auth/me` | ✓ |
| `GET` | `/search?q=&source=` | ✓ |
| `GET` | `/guilds` | ✓ |
| `GET`, `PATCH` | `/guilds/{id}/player` | ✓ |
| `GET` | `/guilds/{id}/player/events` (SSE) | ✓ |
| `GET` | `/guilds/{id}/player/lyrics` | ✓ |
| `POST` | `/guilds/{id}/player/skip`, `/previous`, `/stop` | ✓ |
| `POST`, `DELETE` | `/guilds/{id}/queue` | ✓ |
| `POST` | `/guilds/{id}/queue/move` | ✓ |
| `DELETE` | `/guilds/{id}/queue/{index}` | ✓ |
| `POST` | `/guilds/{id}/radio` | ✓ |
| `GET`, `PUT`, `DELETE` | `/guilds/{id}/radio/247` | ✓ (changes need Manage Server) |
| `GET`, `POST` | `/playlists` | ✓ |
| `GET`, `PATCH`, `DELETE` | `/playlists/{id}` | ✓ |
| `POST` | `/playlists/{id}/tracks` | ✓ |
| `DELETE` | `/playlists/{id}/tracks/{trackId}` | ✓ |
| `GET` | `/admin/overview`, `/admin/guilds`, `/admin/guilds/{id}`, `/admin/logs` | Admin |
| `POST` | `/admin/guilds/{id}/disconnect`, `/move`, `/leave`, `/admin/search` | Admin |

Errors use one shape: `{"error": {"code": "...", "message": "..."}}`.

Admin routes require a user listed in `API_ADMIN_USER_IDS`. Anyone who can see a guild can view its player. To change it, you must be in the bot's voice channel, have *Manage Server*, or be listed in `API_ADMIN_USER_IDS`.

## Development

Run these from `backend/`:

```bash
make check      # gofmt check, go vet, unit tests
make lint       # golangci-lint
make cover      # test coverage report
make build      # build bin/chilly with the version stamped in
```

### Project layout

```text
backend/
├── api/         HTTP API: auth, sessions, player, playlists, radio
├── azuracast/   AzuraCast client and now-playing poller
├── commands/    Slash command definitions and handlers
├── handlers/    Discord and Lavalink event handlers, player buttons
├── musicbot/    Core: player, queue, search, voice, nodes, database
├── db/          SQL schema, embedded into the binary
└── main.go      Wiring and startup
frontend/
├── src/app/     Pages, API proxy, metadata (OG images, icons, sitemap)
├── src/components, src/hooks, src/lib
└── Dockerfile   Standalone Next.js image
```

## Deployment

Both halves ship as small Alpine images that run as a non-root user: the backend exposes port `8080`, the website port `3000`. Only the website needs a public domain; point its `BACKEND_URL` at the backend over the internal network. It runs well on Dokploy, Coolify or plain Docker Compose next to Lavalink, PostgreSQL and AzuraCast. Behind a reverse proxy, set `API_TRUST_PROXY=true` so rate limiting sees the real client IP.

## Contributing

See [CHANGELOG.md](CHANGELOG.md) for release notes.

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) for the development setup, code style and pull request process, and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for community guidelines. AI coding agents should follow [AGENTS.md](AGENTS.md).

Found a security issue? Please report it privately as described in [SECURITY.md](SECURITY.md).

## License

Chilly is licensed under the [GNU Affero General Public License v3.0](LICENSE). If you run a modified version as a network service, the AGPL requires you to make your modified source available to its users.
