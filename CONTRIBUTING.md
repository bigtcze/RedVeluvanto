# Contributing to RedVeluvanto

Thanks for your interest in contributing! This guide will help you get set up and submit your first PR.

## Development Setup

### Prerequisites

- **Go 1.26+** — [golang.org/dl](https://go.dev/dl/)
- **Node.js 24+** — [nodejs.org](https://nodejs.org/)
- **Docker & Docker Compose** — for LiteLLM proxy
- A [Gemini API key](https://aistudio.google.com/) (free tier)
- A [Reddit app](https://www.reddit.com/prefs/apps) (type: web app)

### Local Development

1. **Clone the repo:**
   ```bash
   git clone https://github.com/bigtcze/RedVeluvanto.git
   cd RedVeluvanto
   ```

2. **Set up environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your API keys
   ```

3. **Start LiteLLM proxy:**
   ```bash
   docker compose -f docker-compose.dev.yml up litellm -d
   ```

4. **Start the backend:**
   ```bash
   cd backend
   go mod tidy
   go run main.go serve --http=0.0.0.0:8090
   ```
   Open [http://localhost:8090](http://localhost:8090) to access the backend.

5. **Start the frontend:**
   ```bash
   cd frontend
   npm install
   VITE_PB_URL=http://localhost:8090 npm run dev
   ```
   App: [http://localhost:5173](http://localhost:5173)

## Project Structure

```
RedVeluvanto/
├── backend/          # Go + PocketBase
│   ├── ai/           # LiteLLM client, scoring, generation, persona prompt builder
│   ├── reddit/       # Reddit API client + OAuth2 flow
│   ├── routes/       # Custom PocketBase API routes
│   ├── worker/       # Background monitoring goroutine
│   └── migrations/   # PocketBase collection definitions
├── frontend/         # React 19 + Vite 8 + TypeScript
│   └── src/
│       ├── pages/    # Page components (Dashboard, Inbox, ThreadDetail, etc.)
│       ├── components/ # Shared components (Layout, CommentTree)
│       └── lib/      # PocketBase client, auth context
├── docker/           # LiteLLM config
└── docker-compose.yml
```

## Code Style

### Go (Backend)

- Standard Go formatting (`gofmt`)
- Error handling: always handle errors, log and continue in workers, return errors in routes
- No third-party web framework — PocketBase v0.36 has its own router
- Database access via `app.FindRecordsByFilter()`, `app.Save()`, etc. (not raw SQL)
- Custom routes go in `routes/` package, registered via `RegisterXxxRoutes()`

### TypeScript (Frontend)

- **Strict mode** — `strict: true`, `noUnusedLocals`, `noUnusedParameters`
- **verbatimModuleSyntax** — use `import type { X }` for type-only imports
- React function declarations (not `React.FC`)
- shadcn/ui for all UI components — don't add new component libraries
- Tailwind CSS 4 for styling — dark mode default
- PocketBase SDK for collection CRUD, `fetch()` for custom API endpoints

## Making Changes

### Adding a Backend Route

1. Create or edit a file in `backend/routes/`
2. Follow the pattern in existing files (e.g., `routes/reddit.go`)
3. Register in `main.go` inside the `OnServe` hook
4. Run `go build ./...` to verify

### Adding a Frontend Page

1. Create the page in `frontend/src/pages/`
2. Add the route in `frontend/src/App.tsx`
3. Update nav in `frontend/src/components/Layout.tsx` if needed
4. Run `npm run build` to verify

### Adding a PocketBase Collection

1. Create a migration file in `backend/migrations/` (follow the naming pattern: `1700000009_create_xxx.go`)
2. Use PocketBase v0.36 API: `core.NewBaseCollection()`, field types, API rules
3. Run `go build ./...` to verify

## Pull Request Process

1. **Fork and branch** — create a feature branch from `main`
2. **Keep PRs focused** — one feature or fix per PR
3. **Test your changes:**
   - Backend: `go build ./...` must pass
   - Frontend: `npm run build` must pass with 0 errors
   - Manual testing: verify the feature works end-to-end
4. **Write a clear PR description** — what changed and why
5. **No breaking changes** to the Docker Compose setup without discussion

## Reporting Issues

- Use GitHub Issues
- Include: steps to reproduce, expected vs actual behavior, browser/OS info
- For bugs with the AI output: include the persona settings and thread context if possible

## Architecture Decisions

A few decisions worth knowing about:

- **PocketBase as the backend framework** — it provides the database, auth, API, and admin UI in a single Go binary. We extend it with custom routes and hooks.
- **LiteLLM as an AI proxy** — decouples the app from a specific AI provider. Swap models by editing one config file.
- **No external Reddit libraries** — the Reddit API client is hand-written HTTP to minimize dependencies and handle Reddit's specific JSON format (Listings, Things, nested replies as string-or-object).
- **In-memory OAuth state** — simple `sync.Map` with TTL for the OAuth2 CSRF state. Fine for single-instance deployments.
- **Monitoring worker as a goroutine** — runs inside the PocketBase process, no separate scheduler needed.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
