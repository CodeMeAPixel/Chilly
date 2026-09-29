# Changelog

All notable changes to Chilly are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). The backend and frontend are released together under one version.

## [Unreleased]

### Added

- 24/7 radio: `/247 on`, `/247 off` and `/247 status` keep a station playing in a voice channel even when it's empty. The setting survives restarts, and the bot rejoins and restarts the station on its own.
- 24/7 controls on the website: a "Keep it playing 24/7" option when sending a station to a server, and a 24/7 banner with an off switch in the dashboard player.
- A redesigned radio page with search, filters, sorting, compact station cards and a sticky player bar with volume control.
- An admin panel for users in `API_ADMIN_USER_IDS`, with bot and node health, every server the bot is in, player actions (disconnect, move node, leave server), a track lookup tester and recent logs.
- A brand page with logo downloads, colours, typography and copy, and a bot list template in `docs/bot-listing.md`.
- Optional `NODE_LOCATION` / `NODE_n_LOCATION` settings, shown on the status page and in the admin panel.
- A mobile navigation menu for the website and dashboard.

### Changed

- Chilly is now presented as a 24/7 radio bot first, across the website, help and invite embeds, the default presence and the README. Song playback is still fully supported.
- The invite link now asks only for the permissions Chilly uses, and no longer requests Manage Server, Manage Channels, Manage Messages, View Audit Log or Add Reactions.
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

[Unreleased]: https://github.com/CodeMeAPixel/Chilly/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/CodeMeAPixel/Chilly/releases/tag/v1.0.0
