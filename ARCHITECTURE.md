# Architecture

## Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Frontend   │────▶│   Backend    │────▶│   LiteLLM   │
│  React 19    │     │  Go +        │     │   Proxy     │
│  Vite 8      │     │  PocketBase  │     │             │
│  shadcn/ui   │     │  v0.36       │     │  Gemini 2.5 │
│  Tailwind 4  │     │              │     │  Flash      │
│              │     │  :8090       │     │  :4000      │
│  :80 (nginx) │     │              │     │  (internal) │
└─────────────┘     └──────────────┘     └─────────────┘
                           │
                    ┌──────┴──────┐
                    │  SQLite DB  │
                    │  (PocketBase│
                    │   pb_data)  │
                    └─────────────┘
```

## Stack

| Layer | Technology |
|---|---|
| Frontend | React 19, Vite 8, TypeScript, Tailwind CSS 4, shadcn/ui |
| Backend | Go 1.26, PocketBase v0.36 (embedded DB + auth + API) |
| AI | LiteLLM proxy → Google Gemini 2.5 Flash |
| Notifications | Discord webhooks (per-user) |
| Deploy | Docker Compose (3 services) |

## Backend Structure

```
backend/
├── main.go              # App entry, route wiring, worker lifecycle
├── ai/
│   ├── client.go        # LiteLLM HTTP client (OpenAI-compatible)
│   ├── scoring.go       # Thread relevance scoring (0-100)
│   ├── generate.go      # Response generation with full context
│   └── prompt.go        # Persona → system prompt builder
├── reddit/
│   ├── client.go        # Reddit API client (search, comments, rules, post)
│   └── oauth.go         # OAuth2 flow (authorize, token exchange, refresh)
├── routes/
│   ├── admin.go         # /api/setup/* + /api/admin/* endpoints
│   ├── reddit.go        # /api/reddit/* endpoints
│   ├── drafts.go        # /api/drafts/* endpoints
│   └── personas.go      # /api/personas/* endpoints
├── worker/
│   ├── monitor.go       # Background keyword scanner goroutine
│   ├── token.go         # Reddit token auto-refresh
│   └── notify.go        # Discord webhook notifications
└── migrations/          # PocketBase collection definitions (8 collections)
```

## Frontend Structure

```
frontend/src/
├── pages/
│   ├── Dashboard.tsx     # 24h overview with stats and activity timeline
│   ├── Inbox.tsx         # Thread list with filters and swipe gestures
│   ├── ThreadDetail.tsx  # Full thread view with comment tree + reply panel
│   ├── Keywords.tsx      # Keyword management (CRUD + toggle)
│   ├── Settings.tsx      # Reddit OAuth, Discord webhook, persona list, user management
│   ├── PersonaBuilder.tsx # Full persona editor (9 sliders, knowledge base, preview)
│   └── Login.tsx         # Authentication + first-time setup wizard
├── components/
│   ├── Layout.tsx        # App shell (sidebar + bottom nav)
│   ├── CommentTree.tsx   # Recursive Reddit comment renderer
│   └── ProtectedRoute.tsx
└── lib/
    ├── pocketbase.ts     # PocketBase client instance
    └── auth.tsx          # Auth context + provider
```

## Data Model

| Collection | Purpose | Access |
|---|---|---|
| `personas` | Response persona definitions (traits, rules, knowledge) | Per-user |
| `keywords` | Monitored keywords + subreddit filters | Shared read, owner write |
| `threads` | Discovered Reddit threads (OP + comments + rules) | Authenticated read |
| `thread_status` | Per-user thread state (new/reviewed/replied/dismissed) | Per-user |
| `drafts` | AI-generated response drafts + edit history | Per-user |
| `settings` | Global app settings (monitoring interval, thresholds) | Admin write |
| `user_settings` | Per-user settings (Discord webhook URL) | Per-user |
| `reddit_accounts` | Reddit OAuth2 tokens per user | Backend managed |

## API Endpoints

### Setup & Admin
| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/setup/status` | Public | Check if initial setup is needed |
| POST | `/api/setup/init` | Public | Create admin account (one-time) |
| GET | `/api/admin/users` | Superuser | List all users |
| POST | `/api/admin/users` | Superuser | Create user account |
| DELETE | `/api/admin/users/{id}` | Superuser | Delete user account |

### Reddit OAuth
| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/reddit/auth` | User | Initiate Reddit OAuth2 flow |
| GET | `/api/reddit/callback` | — | OAuth2 callback (token exchange) |
| GET | `/api/reddit/status` | User | Connection status |
| POST | `/api/reddit/disconnect` | User | Revoke and disconnect |

### Drafts
| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/drafts/generate` | User | Generate AI reply draft |
| POST | `/api/drafts/{id}/regenerate` | User | Generate alternative draft |
| PATCH | `/api/drafts/{id}` | User | Save edited text |
| POST | `/api/drafts/{id}/approve` | User | Approve and post to Reddit |

### Personas
| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/personas/preview` | User | Live preview with sample thread |

All PocketBase collections are also available via the standard [PocketBase API](https://pocketbase.io/docs/api-records/).

## Key Design Decisions

- **PocketBase as the backend framework** — provides the database, auth, API, and admin UI in a single Go binary. Extended with custom routes and hooks.
- **LiteLLM as an AI proxy** — decouples the app from a specific AI provider. Swap models by editing one config file.
- **No external Reddit libraries** — hand-written HTTP client to minimize dependencies and handle Reddit's specific JSON format (Listings, Things, nested replies as string-or-object).
- **In-memory OAuth state** — simple `sync.Map` with TTL for the OAuth2 CSRF state. Fine for single-instance deployments.
- **Monitoring worker as a goroutine** — runs inside the PocketBase process, no separate scheduler needed.
- **PocketBase admin panel blocked** — all administration goes through the RedVeluvanto UI. The `/_/` route returns 404 in production.
