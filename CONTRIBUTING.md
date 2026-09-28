# Contributing to Chilly

Thanks for helping make Chilly better. This guide covers how to set up the project, what we expect from a change, and how reviews work. By taking part you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

If you are an AI coding agent, also read [AGENTS.md](AGENTS.md).

## Ways to contribute

- **Report a bug** with the bug report template. Include steps to reproduce, what you expected and what happened, plus relevant logs with every token and password removed.
- **Suggest a feature** with the feature request template. Explain the problem before the solution.
- **Improve the docs.** Typos, unclear setup steps and missing examples are all welcome fixes.
- **Send code.** For anything bigger than a small fix, open an issue first so we can agree on the approach before you spend time on it.

Security issues are different: please follow [SECURITY.md](SECURITY.md) and do not open a public issue.

## Development setup

You will need Go 1.22+, bun 1.3+, PostgreSQL 14+, a Lavalink v4 node and a Discord application for testing. Use a separate test bot and test server; never develop against a production token.

```bash
git clone https://github.com/CodeMeAPixel/Chilly.git
cd Chilly

cd backend
cp .env.example .env        # fill in a test bot token, Lavalink node and database
make run

cd ../frontend
cp .env.example .env.local
bun install
bun dev
```

The frontend README explains the backend settings needed for Discord login in development.

## Making a change

1. Fork the repository and create a branch from `master`, named after the change, for example `fix/queue-skip-race` or `feat/status-history`.
2. Keep the change focused on one thing. Unrelated refactors belong in their own pull request.
3. Add or update tests when you change behaviour. Player, queue and search logic in `backend/musicbot` are unit tested and changes there must keep those tests meaningful.
4. Update the documentation that your change affects: the root README (features, configuration, API table), `.env.example` files and the frontend README.
5. Run the checks below and make sure they pass.

### Checks

```bash
cd backend && make check
cd frontend && bun run lint && bun run typecheck && bun run build
```

CI runs the same checks on every pull request.

## Code style

These rules apply across the whole repository.

- **No code comments.** Write code that explains itself through clear names and small functions. Functional directives such as `//go:embed` are the only exception. Put the reasoning for a change in the commit message or pull request description.
- **Go:** `gofmt` formatting, `go vet` clean, errors wrapped with context (`fmt.Errorf("...: %w", err)`), and `log/slog` for logging with structured fields. Never log secrets.
- **TypeScript/React:** strict TypeScript, ESLint clean, function components and hooks, TanStack Query for server state, and the design tokens from `globals.css` rather than hard-coded colours.
- **Next.js:** this project runs Next.js 16. Read the bundled docs in `frontend/node_modules/next/dist/docs/` before relying on an API you remember from older versions.
- **Dependencies:** add a new dependency only when it clearly earns its place, and explain why in the pull request.

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <summary>

<optional body explaining what and why>
```

- **Types:** `feat`, `fix`, `perf`, `refactor`, `docs`, `test`, `build`, `ci`, `chore`
- **Scopes:** `backend`, `frontend`, `api`, `player`, `search`, `radio`, `docs`, `ci`
- **Summary:** imperative mood, lower case, no trailing full stop, under 72 characters

Examples:

```text
fix(player): ignore stale track end events after a skip
feat(frontend): add uptime history to the status page
docs: document the Lavalink plugin setup
```

Mark breaking changes with `!` after the scope and a `BREAKING CHANGE:` footer describing the migration.

## Pull requests

- Fill in the pull request template, including how you tested the change.
- Link the issue the pull request resolves (`Closes #123`).
- Include screenshots or a short recording for UI changes, in both light and dark mode.
- Keep pull requests small enough to review in one sitting. Large changes are easier to land as a series.
- Do not commit `.env` files, credentials, build output, `node_modules` or editor settings.

### Review

A maintainer will review your pull request, usually within a week. Expect questions and requests for changes; they are part of the process, not a rejection. Pull requests are squash-merged, so the pull request title becomes the commit message and must follow the Conventional Commits format.

A pull request can be merged when:

- CI passes
- the change is covered by tests or a clear manual test plan
- the docs are updated
- a maintainer has approved it

## Licensing

Chilly is licensed under the [GNU Affero General Public License v3.0](LICENSE). By contributing, you agree that your contributions are licensed under the same terms. Only submit code you wrote yourself or that is available under a compatible license, and say where any third-party code came from in your pull request.
