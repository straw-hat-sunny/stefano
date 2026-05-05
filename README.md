# Stefano (ai-assistant)

A small full-stack app: a **Go** HTTP server embeds the **React + Vite** frontend, exposes REST APIs for health checks, selectable chat models, and in-memory chat sessions (wired to a pluggable LLM client—currently a fake implementation suitable for development).

## Tech stack

| Layer | Technologies |
|--------|----------------|
| Backend | Go 1.26, [gorilla/mux](https://github.com/gorilla/mux), `embed` for static assets |
| Frontend | React 19, TypeScript, Vite 8 |
| Tooling | [Task](https://taskfile.dev/) (`task` CLI), npm |

## Repository layout

- `cmd/server` — HTTP entrypoint; registers routes and serves the SPA + `/api/*`.
- `internal/chat` — Chat domain, HTTP handlers (`RegisterRoutes`), in-memory repository.
- `internal/model` — Selectable model catalog and handlers (`/api/models`, `/api/model`).
- `internal/llm` — LLM abstraction (e.g. fake client used by default in `main`).
- `web/` — Vite app; production build output goes to `web/dist` and is embedded into the binary.

## Prerequisites

- **Go** — version aligned with `go.mod` (currently 1.26.2).
- **Node.js** — use **25.x** if you follow `.nvmrc` (`nvm use`). `web/package.json` also documents a minimum Node version for the frontend.
- **Task** (optional but recommended) — [install Task](https://taskfile.dev/installation/) to use the `Taskfile.yml` shortcuts below.

## How to run

### Full build and run (frontend + server)

Builds the Vite app, compiles the Go binary with embedded `web/dist`, then starts the server:

```bash
task run
```

Default listen address: **`http://127.0.0.1:8080`**. Override the port:

```bash
set PORT=3000
task run
```

(On Unix-style shells: `PORT=3000 task run`.)

### Backend only (skip npm/Vite)

Useful when you only changed Go code and `web/dist` already exists from a previous frontend build:

```bash
task run-backend
```

If `web/dist` is missing or stale, run `task build` or `task frontend` first so `embed` has assets to bundle.

### Frontend dev server (Vite)

For fast UI iteration with HMR (API calls still expect the Go server if you use same origin or configure a proxy):

```bash
cd web
npm install
npm run dev
```

Adjust Vite config if you need to proxy `/api` to the Go process during development.

## Other tasks

| Command | Purpose |
|---------|---------|
| `task` or `task build` | Build `web/dist` and compile `bin/server` (or `bin/server.exe` on Windows) |
| `task test` | Run `go test ./...` |
| `task clean` | Remove `bin/` and `web/dist` |

### Without Task

```bash
cd web && npm install && npm run build
go build -o bin/server ./cmd/server   # or bin/server.exe on Windows
./bin/server
```

## HTTP API (overview)

- `GET /api/health` — Liveness JSON.
- `GET /api/models` — Lists models and current selection.
- `POST /api/model` — Select a model (`{"id":"..."}`).
- `POST /api/chat` — Create a chat session.
- `GET /api/chat/{chat_id}` — Fetch a session.
- `POST /api/chat/{chat_id}` — Send a user message (`{"content":"..."}`); response includes the assistant reply.

Chat data is **in-memory** and is lost when the process exits.
