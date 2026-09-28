# Security policy

## Supported versions

Security fixes are made on the `master` branch and released from there. Self-hosted instances should stay up to date with the latest release.

## Reporting a vulnerability

Please do not report security issues in public issues, discussions or pull requests.

Report them privately through GitHub instead: open the repository's **Security** tab and choose **Report a vulnerability**. Include:

- a description of the issue and its impact
- steps to reproduce, or a proof of concept
- the affected component (bot, API, website) and version or commit
- any suggested fix

We aim to acknowledge reports within 3 working days and to share a fix timeline within 10 working days. We will credit you in the release notes unless you would rather stay anonymous.

## Scope

In scope:

- the Go backend: bot, HTTP API, authentication and sessions
- the Next.js website and dashboard
- the Docker images and example configuration in this repository

Out of scope:

- vulnerabilities in Discord, Lavalink, AzuraCast or other third-party services (report those upstream)
- issues that need a compromised host, stolen credentials or physical access
- denial of service through volumetric traffic
- missing hardening on a self-hosted instance that ignores the documented configuration

## Handling secrets

If you find a leaked token, password or other credential in this repository or its history, report it the same way. Do not use it.

For operators: keep `.env` files out of version control, use long random values for `LAVALINK_SERVER_PASSWORD`, `NODE_PASSWORD` and the yt-cipher token, and do not expose Lavalink publicly unless you need to. Rotate the Discord bot token immediately if it ever appears in logs or chat.
