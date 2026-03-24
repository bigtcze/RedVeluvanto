# RedVeluvanto

**Open-source Reddit copilot with a persona engine.** Monitor keywords, get AI-scored threads with full context, craft replies in your custom persona, and post — all with a human in the loop.

Built by the team behind [Veluvanto](https://veluvanto.com) — AI-native document management.

## Why RedVeluvanto?

Existing Reddit AI tools either:
- Are fully automated bots (result: banned accounts)
- Offer 4 preset tones ("Friendly / Casual / Technical")
- Sound like AI wrote them

**RedVeluvanto is a copilot, not a bot.** You're always in the loop.

## Features

- **Keyword Monitoring** — Track keywords across subreddits. New matching threads appear in your inbox automatically.
- **AI Relevance Scoring** — Every thread gets a 0-100 relevance score so you focus on what matters.
- **Full Thread Context** — OP, all comments (nested tree), subreddit rules and description — the AI sees everything you'd see.
- **Persona Engine** — 9 personality sliders (formality, humor, empathy...), custom instructions, behavior rules, competitor stance, knowledge base, few-shot examples. Your replies sound like *you*.
- **Draft → Edit → Post** — AI generates a draft, you edit it, then approve to post directly from the app.
- **Draft History** — Every generated draft is saved. Try different approaches, compare, pick the best.
- **Multi-User** — Team-friendly. Shared keywords, per-user personas, per-user Reddit accounts.
- **Discord Notifications** — Get notified in Discord when relevant threads are found.
- **Mobile-First** — Responsive dark UI with swipe gestures on mobile.
- **Self-Hosted** — Your data stays on your server. One `docker compose up` and you're running.

## Quick Start

### Prerequisites

- Docker & Docker Compose
- A [Google Gemini API key](https://aistudio.google.com/) (free tier works)
- A [Reddit app](https://www.reddit.com/prefs/apps) (type: web app)

### 3 Steps

```bash
# 1. Clone
git clone https://github.com/bigtcze/RedVeluvanto.git
cd RedVeluvanto

# 2. Configure
cp .env.example .env
# Edit .env — add your GEMINI_API_KEY, REDDIT_CLIENT_ID, REDDIT_CLIENT_SECRET

# 3. Launch (uses pre-built images from GitHub Container Registry)
docker compose up -d
```

Open [http://localhost](http://localhost) in your browser.

> **Building from source?** Use `docker compose -f docker-compose.dev.yml up --build` instead.

### First-Time Setup

1. Open [http://localhost](http://localhost) — you'll see the setup wizard
2. Create your admin account (email + password)
3. You'll be automatically logged in
4. Go to Settings → Connect your Reddit account
5. Go to Keywords → Add your first keyword
6. Wait for the monitoring worker to find threads (default: every 5 minutes)

To add more users, go to Settings → User Management (admin only).

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `GEMINI_API_KEY` | Yes | — | Google Gemini API key |
| `LITELLM_MASTER_KEY` | No | `sk-redveluvanto-change-me` | LiteLLM proxy auth key |
| `LITELLM_URL` | No | `http://litellm:4000` | LiteLLM proxy URL |
| `AI_FAST_MODEL` | No | `gemini-2.5-flash` | Model name for scoring + generation |
| `REDDIT_CLIENT_ID` | Yes | — | Reddit app client ID |
| `REDDIT_CLIENT_SECRET` | Yes | — | Reddit app client secret |
| `REDDIT_REDIRECT_URI` | No | `http://localhost:8090/api/reddit/callback` | OAuth2 redirect URI |

### Reddit App Setup

1. Go to [reddit.com/prefs/apps](https://www.reddit.com/prefs/apps)
2. Click "create another app..."
3. Select **web app**
4. Set redirect URI to `http://localhost:8090/api/reddit/callback` (or your domain)
5. Copy the client ID (under the app name) and secret

### Using a Different AI Model

RedVeluvanto uses LiteLLM as a proxy, so you can use any model it supports. Edit `docker/litellm/config.yaml`:

```yaml
model_list:
  - model_name: my-model
    litellm_params:
      model: openai/gpt-4o
      api_key: os.environ/OPENAI_API_KEY
```

Then set `AI_FAST_MODEL=my-model` in your `.env` file.

## Development

### Backend (Go)

```bash
cd backend
go mod tidy
go run main.go serve --http=0.0.0.0:8090
```

### Frontend (React)

```bash
cd frontend
npm install
npm run dev
```

Set `VITE_PB_URL=http://localhost:8090` for local development (or proxy via Vite config).

### Docker (full stack, from source)

```bash
docker compose -f docker-compose.dev.yml up --build
```

## Persona Engine

The persona engine is the core differentiator. Each persona has:

**9 Personality Sliders (0-10)**
| Trait | Low (0) | High (10) |
|---|---|---|
| Formality | Slang, casual | Professional, proper |
| Verbosity | Brief, 1-2 sentences | Detailed, thorough |
| Humor | Completely serious | Witty, sarcastic |
| Empathy | Factual, objective | Understanding, warm |
| Confidence | Hedging ("maybe", "perhaps") | Direct, certain |
| Expertise | Curious learner | Deep expert |
| Controversy | Always agreeable | Challenges ideas |
| Emoji Usage | Never | Frequent |
| Typo Tolerance | Perfect grammar | Casual with typos |

**Plus:**
- Custom text instructions
- Reply goal (help / promote / reputation / traffic / educate)
- Behavior rules ("Never promise features", "Don't mention pricing")
- Competitor stance (ignore / acknowledge / compare / differentiate)
- Product knowledge base (text + URLs)
- Few-shot example responses
- Automatic language detection (replies in the same language as the target comment)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style guidelines, and PR process.

For architecture details, data model, and API reference, see [ARCHITECTURE.md](ARCHITECTURE.md).

## License

[MIT](LICENSE) — Built by the [Veluvanto](https://veluvanto.com) team.
