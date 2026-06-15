# Cinna

Cinna is a modular Telegram assistant written in Go.

## Stack

- Go
- Telegram Bot API
- PostgreSQL
- CloudWeGo Eino
- `pgx`
- `goose`
- `sqlc`

## Current State

The Go service currently supports:

- Environment-based configuration
- Telegram long polling in development
- Telegram webhooks in production
- An environment-based administrator allow list
- Graceful HTTP server shutdown
- Local PostgreSQL using Docker Compose

PostgreSQL application connectivity, migrations, and the database-backed allow
list are still in progress. See `PROGRESS.md`.

## Requirements

- Go 1.26 or later
- Docker Desktop
- A Telegram bot token from BotFather

## Local Environment

Create `.env` from the example:

```bash
cp .env.example .env
```

Configure these values:

```dotenv
GO_ENV=development
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
ALLOWED_ADMIN_USERS=12345678,87654321

POSTGRES_USER=cinna
POSTGRES_PASSWORD=change_me
POSTGRES_DB=cinna

DATABASE_URL=postgres://cinna:change_me@localhost:5432/cinna?sslmode=disable
```

`ALLOWED_ADMIN_USERS` must contain numeric Telegram user IDs separated by
commas.

The webhook variables are required only in production:

```dotenv
WEBHOOK_URL=https://your-service.example.com
WEBHOOK_SECRET=your_webhook_secret
```

Do not commit `.env` because it contains credentials.

## Start PostgreSQL

Docker Compose automatically reads `.env` from the project directory.

Start PostgreSQL:

```bash
docker compose up -d postgres
```

Check its status:

```bash
docker compose ps
```

Follow its logs:

```bash
docker compose logs -f postgres
```

Open a PostgreSQL shell:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
```

Run a connection test:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT current_database(), current_user;"
```

Stop PostgreSQL while preserving its data:

```bash
docker compose down
```

Delete PostgreSQL and its local data:

```bash
docker compose down -v
```

The last command permanently removes the local database volume.

## Run Cinna

Export the environment variables and start the service:

```bash
set -a
source .env
set +a

go run ./cmd/cinna
```

In development, Cinna connects to Telegram using long polling.

Although Docker starts PostgreSQL locally, the Go service does not use it yet.
Database connectivity with `pgx` is part of the current implementation phase.

## Tests

```bash
go test ./...
go vet ./...
```

## Project Structure

```text
cinna/
├── cmd/
│   └── cinna/
├── internal/
│   └── app/
│       └── telegram/
├── db/
│   ├── migrations/
│   └── queries/
├── prompts/
│   ├── core/
│   └── shopping/
├── compose.yaml
├── go.mod
├── CINNA_PLAN.md
└── PROGRESS.md
```

Cinna will remain a single deployable modular monolith until operational
evidence justifies splitting it.
