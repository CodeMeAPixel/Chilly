# Chilly Web

The Chilly website and dashboard, built with Next.js (App Router), React 19, Tailwind CSS v4 and TanStack Query.

## How it talks to the backend

Every browser request to `/api/v1/*` is proxied to the Go backend by the route handler in `src/app/api/v1/[...path]/route.ts`. The backend address is read at runtime from `BACKEND_URL`, so one image works in every environment. Because the site and API share an origin, the Discord login cookie needs no CORS or cookie-domain setup, and live player updates (server-sent events) stream through the same proxy.

## Development

```bash
cp .env.example .env.local
bun install
bun dev
```

Run the backend with `API_ENABLED=true` and these settings for local login:

```env
API_PUBLIC_URL=http://localhost:3000
API_DASHBOARD_URL=http://localhost:3000
API_ALLOWED_ORIGINS=http://localhost:3000
API_COOKIE_SECURE=false
```

Add `http://localhost:3000/api/v1/auth/callback` as a redirect URL in the Discord developer portal.

## Environment

| Variable | Used at | Purpose |
| --- | --- | --- |
| `BACKEND_URL` | runtime | Address of the Go API, e.g. `http://chilly-backend:8080` |
| `SITE_URL` | build and runtime | Public URL of the site, used for metadata, `robots.txt` and `sitemap.xml` |

## Scripts

```bash
bun dev        # dev server
bun run build  # production build (standalone output)
bun start      # serve the production build
bun run lint   # eslint
```

## Docker

```bash
docker build --build-arg SITE_URL=https://chilly.example.com -t chilly-web .
docker run -p 3000:3000 -e BACKEND_URL=http://chilly-backend:8080 -e SITE_URL=https://chilly.example.com chilly-web
```

## Structure

```text
src/
├── app/
│   ├── (site)/            Landing page, radio, login
│   ├── dashboard/         Server picker, live player, playlists
│   ├── api/v1/[...path]/  Proxy to the Go API
│   ├── opengraph-image.tsx, twitter-image.tsx, apple-icon.tsx, icon.svg
│   └── manifest.ts, robots.ts, sitemap.ts
├── components/            Shared UI
├── hooks/                 Session, guilds and live player hooks
├── lib/                   API client, server helpers, formatting, OG rendering
└── proxy.ts               Redirects signed-out users away from /dashboard
```
