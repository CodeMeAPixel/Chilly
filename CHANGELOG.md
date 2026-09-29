# Changelog

All notable changes to Chilly are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). The backend and frontend are released together under one version.

## [Unreleased]

## [2.0.0] - 2026-09-29

Chilly is now a 24/7 radio bot powered entirely by AzuraCast. Stations can stay on around the clock, every song comes from your own AzuraCast library, and listeners can request and suggest songs. Third-party sources such as YouTube and Spotify are gone.

### Upgrading from 1.x

1. In AzuraCast, create a user for the bot with the **Media** and **Broadcasting** permissions on each station, and set its API key as `AZURACAST_API_KEY`.
2. Set `MEDIA_BASE_URL` to an address where every Lavalink node can reach the bot's API directly (not through the website).
3. Deploy, then run **Admin → Tools → Library check** against each node.
4. Remove the youtube-source, LavaSrc and LavaSearch plugins and the yt-cipher service from your Lavalink nodes. Only the HTTP source is needed; the Pterodactyl egg in `extras/` installs that setup.
5. Turn on song requests for your stations and the playlists that should be requestable.
6. Remove `SEARCH_PROVIDERS` from your environment; it's no longer read.

Database changes are applied automatically on startup. Existing playlists are kept, and their songs are matched to the library as it grows.

### Added

#### Radio

- 24/7 radio: `/247 on`, `/247 off` and `/247 status` keep a station playing in a voice channel even when it's empty. The setting survives restarts, and the bot rejoins and restarts the station on its own.
- Station switching: while a station is playing, the player message's previous and next buttons and the dashboard's controls change stations. With 24/7 on, members with Manage Server move the 24/7 station too.
- A redesigned radio page with search, filters, sorting, compact station cards and a sticky player bar with volume control.
- A "Keep it playing 24/7" option when sending a station to a server from the website, and a 24/7 banner with an off switch in the dashboard player.

#### Library and playback

- A music library synced from AzuraCast. `/play`, `/search`, `/list add` and the dashboard search it by song, album, artist or AzuraCast playlist, with instant autocomplete.
- A signed media proxy (`/api/v1/media/...`) that streams library songs from AzuraCast to Lavalink, including seeking. Configure it with `MEDIA_BASE_URL` and optionally `MEDIA_SIGNING_KEY`.
- A public tracks page (`/tracks`) to browse every song, artist, album and station playlist, with search, sorting and paging.
- Personal playlists hold library songs. Songs saved from other services are matched to the library automatically after each sync; songs that aren't in the library yet are marked and skipped when a playlist plays.
- Station playlists from AzuraCast are listed on the playlists page, where you can browse their songs and queue them in a server.
- Lyrics stored on songs in AzuraCast are shown before falling back to LRCLIB, for library songs and station songs.
- An optional lyrics backfill job (`AZURACAST_LYRICS_BACKFILL`) that finds missing lyrics on LRCLIB and saves them to songs in AzuraCast.

#### Requests and suggestions

- Song requests: `/request`, a Request button on the tracks and playlist pages, and `POST /api/v1/requests` ask a station to play a library song soon. Each request is tracked from waiting to on air to played, and expires if it never plays.
- Song suggestions: `/suggest` and the My requests page let people suggest songs that aren't in the library. Suggestions are marked added automatically when a matching song appears in the library, and people get a DM when theirs is added or declined.
- A My requests page (`/dashboard/requests`) and `/requests` command showing each person's requests and suggestions.
- Configurable per-user limits: `REQUEST_COOLDOWN`, `REQUEST_MAX_PENDING` and `SUGGESTION_MAX_OPEN`.

#### Admin

- An admin panel for users in `API_ADMIN_USER_IDS`, with bot, library and node health, every server the bot is in, player actions (disconnect, move node, leave server), a library and playback check, recent logs, a suggestion review queue, recent requests and per-station song skipping.

#### Website and operations

- A rebuilt status page. Uptime history is stored in PostgreSQL for 90 days with a 24-hour and 90-day view, each station and the music library are tracked as components, each audio node has one card with its location and details, and outages are recorded as incidents automatically. Admins can post announcements and maintenance notices from the admin panel.
- A brand page with logo downloads, colours, typography and copy, and a bot list template in `docs/bot-listing.md`.
- Optional `NODE_LOCATION` / `NODE_n_LOCATION` settings, shown on the status page and in the admin panel.
- A mobile navigation menu for the website and dashboard.

### Changed

- **Breaking:** AzuraCast is now the only music source. YouTube, SoundCloud, Spotify and other third-party sources, `SEARCH_PROVIDERS`, the alternative-source fallback and the Lavalink plugins are removed, and links to other services are no longer accepted. Lavalink only needs its HTTP source.
- **Breaking:** `/play`, `/search` and `/list add` take a `type` option (song, album, artist or playlist) instead of `source`, and the API's `source` field on queue and playlist requests is now `type`.
- Chilly is presented as a 24/7 radio bot first across the website, help and invite embeds, the default presence and the README.
- The invite link asks only for the permissions Chilly uses, and no longer requests Manage Server, Manage Channels, Manage Messages, View Audit Log or Add Reactions.
- The dashboard's search panel searches the library and can queue a song's whole album.
- Track exception logs include the node name and no longer repeat the error twice.

### Fixed

- AzuraCast polling no longer fails when a track reports a fractional duration.

## [1.0.0] - 2026-09-28

The first public release of Chilly: a self-hosted Discord music bot with a web dashboard, live status page and AzuraCast radio.

### Added

#### Bot

- Playback from YouTube, SoundCloud, Spotify, Deezer and Apple Music through Lavalink v4, depending on the plugins the node runs.
- `/play` and `/search` with live autocomplete that queues the exact track you pick, including albums and playlists.
- Queue control with `/queue`, `/now`, `/skip`, `/pause`, `/resume`, `/seek`, `/stop`, `/shuffle`, `/loop` and `/remove`, plus previous and history.
- A live player message with a progress bar, "up next" preview and buttons for previous, play/pause, skip, lyrics, stop, loop, shuffle and the dashboard.
- `/lyrics` for the current song or any search, using LRCLIB with synced lyrics where available.
- Personal playlists stored in PostgreSQL, managed with `/list` and played with `/playlist`.
- AzuraCast radio with `/radio play`, `/radio now` and `/radio stations`, live now-playing updates and automatic reconnects.
- `/join`, `/leave`, `/help`, `/invite` and `/ping`, with links to the website, dashboard and status page.
- Automatic fallback to another source when a track cannot be played from its original one.

#### Lavalink

- Support for multiple Lavalink nodes. New players go to the least loaded connected node.
- Node supervision with automatic failover: players move to a healthy node when theirs goes down and are restored after it restarts.
- A startup check that logs each node's sources and plugins and disables search providers a node cannot serve.
- A voice watchdog that recovers players left silent after voice server changes.

#### API

- REST API under `/api/v1` with Discord OAuth2 login, server-side sessions and bearer token support.
- Live player updates over server-sent events.
- Player, queue, radio, lyrics, search and playlist endpoints for the dashboard.
- Public `/health`, `/stats` and `/status` endpoints, with 24 hours of per-component uptime history.
- CSRF origin checks on cookie-authenticated writes and per-client rate limiting.

#### Website and dashboard

- Landing page with a feature overview, command explorer and a player preview that matches the real Discord embed.
- Dashboard with a server picker, live player, queue, history and synced lyrics, track search and playlist management.
- Radio page listing every AzuraCast station with what is on air.
- Status page showing Discord, database, radio and Lavalink node health, uptime history and per-node resource usage.
- Light and dark themes, Open Graph and Twitter images, web manifest, sitemap and robots rules.

#### Operations

- Docker images for the backend and frontend, and Nixpacks support for the frontend.
- A Pterodactyl egg for Lavalink nodes preconfigured with youtube-source, LavaSrc, LavaSearch and a remote cipher server.
- Makefile targets for running, testing, linting and building the backend.
- Contributor documentation: agent guide, contributing guide, code of conduct, security policy, issue forms and CI.

### Security

- Secrets are never logged: the Discord and Lavalink client libraries are capped at Info level because the gateway token is logged at Debug.
- Session tokens are stored hashed, and cookies are `HttpOnly` and `Secure` by default.

[Unreleased]: https://github.com/CodeMeAPixel/Chilly/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/CodeMeAPixel/Chilly/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/CodeMeAPixel/Chilly/releases/tag/v1.0.0
