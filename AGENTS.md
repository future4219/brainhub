# Brainhub - Codex Agent Guide

## Project Overview

Brainhub connects domain-specific GBrain sources to AI clients. The backend is
Go, the frontend is Vite + React + TypeScript, and local services run through
Docker Compose.

## Commands

- Go tests: `go -C api test ./...`
- Frontend type check: `npm --prefix web run check`
- Frontend tests: `npm --prefix web test`
- Frontend build: `npm --prefix web run build`
- Local stack: `docker compose up -d`
- Rebuild app services: `docker compose up -d --build brainhub web`

## Architecture

### Go backend (`api/`)

- Dependency direction is `api / adapter -> usecase -> domain`.
- Keep GBrain calls inside `adapter/gbrain`.
- Handlers translate HTTP only; business rules belong in use cases or domain.
- Database models and API schemas must not replace domain entities.

### React frontend

- `App.tsx` contains route definitions only.
- Route entry components live in `web/src/components/pages` and stay thin.
- Feature code lives in `web/src/components/features/<Feature>`.
- Containers own data fetching, state, and event handlers.
- Presenters render props and do not call APIs.
- Shared UI lives in `web/src/components/ui`; keep feature-only components with
  their feature.
- Put URL constants and builders in `web/src/config/url.ts`.
- Put domain types in `web/src/entities/<domain>/entity.ts`.
- Put the shared HTTP client and endpoint functions in `web/src/lib`.
- Use the `@/` alias and named exports for components.
- Use PascalCase component filenames and camelCase hook filenames.

When placement or implementation style is unclear, inspect the closest example
in `/home/futur/dev/xroll-frontend` before introducing a new pattern.

## Coding Conventions

- Keep TypeScript strict and use functional components with hooks.
- Use Tailwind classes and the tokens in `web/src/styles/tokens.css`; avoid
  one-off colors and dimensions.
- Do not add an abstraction or dependency for a single speculative use.
- Do not store authentication secrets in browser storage.
- Do not infer API response shapes; verify the running API or Go schema.
- Preserve loading, error, empty, and ready states for data-backed screens.
- When changing shared UI, check every feature that imports it.

## Pull Requests

Every pull request description must include:

### What

A concise summary of the change.

### Why

The reason for the change and its issue link when one exists.

### How

The implementation approach and important tradeoffs.

### Test

Commands and manual checks used to verify the change. At minimum, run the Go
tests and all three frontend checks listed above when both sides are touched.

## Important Notes

- `README.md` rules take priority over convenience.
- Do not modify environment variables, public MCP URLs, authentication behavior,
  migrations, or GBrain source configuration without explicit instruction.
- Edit locally, verify, commit, and push. Do not make normal source changes
  inside running containers.
