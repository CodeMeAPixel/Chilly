FROM golang:1.22 AS build

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o chilly .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
	&& addgroup -S chillcord \
	&& adduser -S -G chillcord chillcord \
	&& mkdir -p /app/logs \
	&& chown -R chillcord:chillcord /app

WORKDIR /app
COPY --from=build --chown=chillcord:chillcord /build/chilly /app/chilly

USER chillcord
EXPOSE 8080
ENTRYPOINT ["/app/chilly"]
