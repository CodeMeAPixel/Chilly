FROM node:22-alpine AS build
COPY --from=oven/bun:1-alpine /usr/local/bin/bun /usr/local/bin/bun
WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .
ARG SITE_URL
ENV SITE_URL=${SITE_URL} \
    NEXT_TELEMETRY_DISABLED=1
RUN bun run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0

RUN addgroup -S chilly && adduser -S -G chilly chilly

COPY --from=build --chown=chilly:chilly /app/.next/standalone ./
COPY --from=build --chown=chilly:chilly /app/.next/static ./.next/static
COPY --from=build --chown=chilly:chilly /app/public ./public
COPY --from=build --chown=chilly:chilly /app/src/assets ./src/assets

USER chilly
EXPOSE 3000
CMD ["node", "server.js"]
