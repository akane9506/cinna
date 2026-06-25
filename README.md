# Cinna

Cinna is a Telegram assistant written in Go. It uses PostgreSQL for access
control, an Eino chat graph for responses, DeepSeek as the chat model, and
short-term in-memory chat history per Telegram user.

## Agent Graph

```mermaid
flowchart LR
    Start([START]) --> InputProcessLambda["Input Process Lambda<br/><br/>Processes input and stores necessary info in the state<br/><br/>In: TaskInput<br/>Out: schema.Message[]"]

    InputProcessLambda --> IntentClassifier["Intent Classification<br/><br/>Classifies user intention and possible action<br/><br/>In: schema.Message[]<br/>Out: schema.Message"]

    IntentClassifier --> IntentLambda["Intent Validation Lambda<br/><br/>Validates intent classification node output<br/>and feeds intention to output branching<br/><br/>In: schema.Message<br/>Out: schema.Message[]"]

    IntentLambda --> IntentRouter{"Branch by Intent<br/><br/>Intent:<br/>SHOPPING | FEEDBACK | OTHER"}

    IntentRouter -- SHOPPING --> ListLambda["List Lambda<br/><br/>Lists contents in the shopping list table<br/><br/>Note:<br/>1. List expired items<br/>2. Items include categories<br/>3. Notify Cinna to respond with the corresponding category<br/><br/>In: schema.Message[]<br/>Out: schema.Message[]"]

    ListLambda --> ActionRouter{"Branch by Action Type<br/><br/>Action:<br/>LIST | UPDATE | None"}

    ActionRouter -- UPDATE --> ShoppingPlanner["ShoppingPlanner<br/><br/>Reads user-Cinna chat history<br/>and decides the Shopping DB operation<br/><br/>In: schema.Message[]<br/>Out: schema.Message"]

    ActionRouter -- LIST --> CinnaReply["CinnaReply"]

    ShoppingPlanner --> ShoppingOutput["Shopping JSON<br/><br/>{<br/>category: string<br/>action: string<br/>item: string<br/>}"]

    ShoppingOutput --> CinnaReply

    IntentRouter -- OTHER --> CinnaReply

    CinnaReply --> End([END])
```

## Requirements

- Go 1.26+
- Docker
- `sqlc`
- A Telegram bot token
- A DeepSeek API key

## Configuration

Create and configure the local environment:

```bash
cp .env.example .env
```

Edit `.env` before loading it. For local development, set at least:

```dotenv
GO_ENV=development
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
DATABASE_URL=postgres://cinna:change_me@localhost:5432/cinna?sslmode=disable
DEEPSEEK_API_KEY=your_deepseek_api_key
POSTGRES_USER=cinna
POSTGRES_PASSWORD=change_me
POSTGRES_DB=cinna
```

Production webhook mode also requires:

```dotenv
GO_ENV=production
WEBHOOK_URL=https://your-public-host.example
WEBHOOK_SECRET=your_webhook_secret
```

Load the environment:

```bash
set -a
source .env
set +a
```

## Database Setup

Start PostgreSQL:

```bash
docker compose up -d postgres
```

On first startup, PostgreSQL creates the database named by `POSTGRES_DB`.
Validate the server and database from the CLI:

```bash
docker compose exec postgres \
  pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"

docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT current_database(), current_user;"
```

Apply the schema:

```bash
docker compose exec -T postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" < db/schema.sql
```

Validate that the allow-list table exists:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "\d allowed_users"
```

Generate Go code from the SQL queries:

```bash
sqlc generate
```

## Telegram Allow List

Cinna checks every Telegram update against the `allowed_users` table. Add or
reactivate a user before messaging the bot:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "INSERT INTO allowed_users (telegram_user_id, role)
      VALUES (12345678, 'admin')
      ON CONFLICT (telegram_user_id) DO UPDATE
      SET role = EXCLUDED.role, is_active = TRUE, updated_at = NOW();"
```

List allowed users:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT * FROM allowed_users ORDER BY updated_at;"
```

## Run

Run Cinna locally with Telegram long polling:

```bash
go run ./cmd/cinna
```

In production mode, Cinna configures the Telegram webhook, starts the webhook
receiver, and serves it on `PORT` or `8080`.

## Agent

The current agent path is:

```text
Telegram message
  -> allow-list middleware
  -> Cinna Eino graph
  -> DeepSeek chat model
  -> Telegram response
```

The agent injects the Cinna persona prompt from `internal/app/agent/prompt/`,
stores non-system messages in an in-memory per-user history, and sends typing
actions while it waits for a response.

## SQL Commands

Open a PostgreSQL shell:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
```

Regenerate code after changing `db/schema.sql` or files under `db/queries/`:

```bash
sqlc generate
```

Stop PostgreSQL:

```bash
docker compose down
```

Delete PostgreSQL data:

```bash
docker compose down -v
```

## Checks

```bash
go test ./...
go vet ./...
```

## Manual LLM Tests

Manual model and agent tests are guarded by `internal/utils.EnforceManualTest`.
They are skipped unless `RUN_MANUAL_TEST=1` is set, and they require
`DEEPSEEK_API_KEY`.

Run the DeepSeek model checks:

```bash
RUN_MANUAL_TEST=1 go test ./internal/app/agent \
  -run 'TestDeepseekFlash(Model|JSON)Manual' -v
```

Run the Cinna agent chat graph check:

```bash
RUN_MANUAL_TEST=1 go test ./internal/app/agent \
  -run TestCinnaChat -v
```

If you already loaded `.env`, make sure it includes:

```dotenv
DEEPSEEK_API_KEY=your_deepseek_api_key
RUN_MANUAL_TEST=1
```

## Project Structure

```text
cinna/
├── cmd/
│   └── cinna/
├── db/
│   ├── queries/
│   └── schema.sql
├── internal/
│   ├── app/
│   │   ├── agent/
│   │   ├── ports/
│   │   └── telegram/
│   ├── postgres/
│   │   └── sqlc/
│   └── utils/
├── prompts/
│   ├── core/
│   └── shopping/
├── compose.yaml
├── go.mod
└── sqlc.yaml
```
