# Architecture

## Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Frontend   │────▶│   Backend    │────▶│   LiteLLM   │
│  React 19    │     │  Go +        │     │   Proxy     │
│  Vite 8      │     │  PocketBase  │     │             │
│  shadcn/ui   │     │  v0.36       │     │  Vertex AI / │
│  Tailwind 4  │     │              │     │  OpenAI /   │
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
| AI | LiteLLM proxy → Vertex AI / OpenAI / Anthropic (configurable) |
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
│   ├── admin.go         # /api/setup/* + /api/admin/* + /api/ai/* endpoints
│   ├── reddit.go        # /api/reddit/* endpoints
│   ├── drafts.go        # /api/drafts/* endpoints
│   ├── threads.go       # /api/threads/* endpoints
│   ├── knowledge.go     # /api/knowledge/* endpoints
│   └── personas.go      # /api/personas/* endpoints
├── worker/
│   ├── monitor.go       # Background keyword scanner goroutine
│   ├── poster.go        # Post queue worker with rate limiting
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
| POST | `/api/drafts/{id}/approve` | User | Queue draft for posting |
| POST | `/api/drafts/{id}/cancel` | User | Cancel queued draft |

### Threads
| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/threads/{id}/refresh` | User | Refresh thread comments from Reddit |

### AI
| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/ai/models` | User | List available AI models from LiteLLM |

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
- **Post queue with anti-ban throttling** — comments are never posted immediately. A background worker processes a shared queue with configurable cooldowns to prevent Reddit account bans.
- **PocketBase admin panel blocked** — all administration goes through the RedVeluvanto UI. The `/_/` route returns 404 in production.

## Post Queue & Rate Limiting

### Why a queue?

Reddit detects spam through behavioral patterns, not just API rate limits. Rapid posting across subreddits, repeated similar content, or high-volume activity from a single account triggers shadowbans. The post queue ensures all comments are posted at a pace that looks organic.

### How it works

```
User clicks "Approve"
       │
       ▼
  draft.status = "queued"         ← instant response to user
       │
       ▼
  PostWorker (every 30s)
       │
       ├── Pick oldest queued draft
       ├── Pick oldest queued draft
       ├── Check: is this a follow-up in a thread where user already posted?
       │
       ├── YES (follow-up) → only global cooldown (90s) applies
       │                      no account/subreddit/daily limits
       │
       └── NO (new thread) → check all cooldowns:
              ├── Global cooldown      (any post from this instance?)
              ├── Per-account cooldown  (this Reddit account?)
              ├── Per-subreddit cooldown (this subreddit?)
              └── Daily limit           (this account today?)
       │
       ├── All checks pass → status = "posting" → SubmitComment → status = "posted"
       └── Any check fails → skip, try next draft in queue
```

### Cooldown defaults

| Parameter | Setting key | Default | Description |
|---|---|---|---|
| Global delay | `post_delay_global` | 90s | Min time between any two posts from the entire instance |
| Per-account delay | `post_delay_account` | 120s | Min time between posts from the same Reddit account |
| Per-subreddit delay | `post_delay_subreddit` | 600s | Min time between posts to the same subreddit |
| Daily limit | `post_daily_limit` | 50 | Max posts per Reddit account per day (UTC reset) |

All parameters are configurable via the `settings` collection at runtime. Cooldowns apply only to **first posts in new threads**. Follow-up replies in threads where the user already posted are only limited by the global delay (to avoid API overload) — no account, subreddit, or daily limits.

### Draft status flow

```
draft → queued → posting → posted
                         → failed
```

Users can cancel a queued draft (returns to `draft` status). Failed drafts can be re-approved to re-enter the queue.

### Follow-up replies are unrestricted

When a user already has a posted comment in a thread, any subsequent replies in that same thread are treated as a natural conversation. Only the global 90s delay applies (to not overwhelm the Reddit API). Per-account cooldown, per-subreddit cooldown, and the daily limit are all skipped — because replying back and forth in a discussion is normal Reddit behavior, not spam.

The daily limit (50) only counts **first entries into new threads**, not total posts. A user can have 3 new thread entries and 100 follow-up replies in those threads in the same day.

## Reddit API Rate Limiting

### API call budget (free tier: 100 QPM)

The monitoring worker is optimized to minimize API calls:

| Optimization | Impact |
|---|---|
| Subreddit rules/about cached per subreddit per cycle | ~95% reduction in metadata calls |
| Comments lazy-loaded on thread open, not during discovery | Eliminates biggest cost item from monitoring |
| Sleep-and-retry on rate limit (429) with backoff | No data loss on transient limits |

### Typical consumption

| Scenario (5 keywords × 2 subreddits) | Calls/cycle | Calls/min (5min interval) |
|---|---|---|
| Monitoring worker | ~14 | ~2.8 QPM |
| User opens thread | 1 | on-demand |
| User posts comment (via queue) | 1 | throttled |
| **Remaining budget for users** | | **~95 QPM** |

## AI Provider Support

RedVeluvanto uses LiteLLM as a proxy, making it AI provider-agnostic. The Go backend only speaks the OpenAI-compatible `/v1/chat/completions` API.

| Provider | Config prefix | Auth | RPD limit |
|---|---|---|---|
| Google Vertex AI (recommended) | `vertex_ai/` | Service account JSON | None |
| OpenAI | `openai/` | API key | None (pay-per-use) |
| Anthropic | `anthropic/` | API key | None (pay-per-use) |

LiteLLM handles retries (3 attempts, 5s backoff) and cooldowns (30s after 3 failures) for all providers. Configuration in `docker/litellm/config.yaml`.

The available models are dynamically fetched from LiteLLM and displayed in Settings → AI Model.
