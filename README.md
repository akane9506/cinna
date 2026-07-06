<div align="center">

<img src="./assets/cinna-small.jpg" width="120px" style="border-radius: 50%;" />

# Cinna

![CI](https://github.com/akane9506/cinna/actions/workflows/ci.yml/badge.svg)
![Deploy](https://github.com/akane9506/cinna/actions/workflows/deployment.yml/badge.svg)

</div>

Cinna is a Telegram assistant written in Go. It uses PostgreSQL for access
control and per-user shopping lists, an Eino graph to route and execute tasks,
DeepSeek for classification, planning, and replies, and encrypted PostgreSQL-
backed chat history with a short in-memory working window per Telegram user.

Current capabilities include:

- Telegram long polling in development and webhooks in production
- PostgreSQL-backed Telegram user allow-listing
- Shopping-list listing, addition, removal, and modification
- Batched shopping-list updates with category assignment
- Feedback collection with pending and in-progress feedback context
- General chat with the embedded Cinna persona

## Agent Graph

```mermaid
flowchart LR
    Start([START]) --> InputProcessLambda["Input Process Lambda<br/><br/>Processes input and stores necessary info in the state<br/><br/>In: TaskInput<br/>Out: schema.Message[]"]

    InputProcessLambda --> IntentClassifier["Intent Classification<br/><br/>Classifies user intention and possible action<br/><br/>In: schema.Message[]<br/>Out: schema.Message"]

    IntentClassifier --> IntentLambda["Intent Validation Lambda<br/><br/>Validates intent classification node output<br/>and feeds intention to output branching<br/><br/>In: schema.Message<br/>Out: schema.Message[]"]

    IntentLambda --> IntentRouter{"Branch by Intent<br/><br/>SHOPPING | FEEDBACK | OTHER"}

    IntentRouter -- SHOPPING --> ListLambda["List Shopping Items Lambda<br/><br/>Loads the user's active and expired items<br/>with IDs and categories<br/><br/>In: schema.Message[]<br/>Out: schema.Message[]"]

    ListLambda --> ActionRouter{"Branch by Action<br/><br/>LIST | UPDATE | NONE"}

    ActionRouter -- UPDATE --> ShoppingPlanner["Shopping Planner<br/><br/>Plans batched ADD, REMOVE, and MODIFY commands<br/>from chat history and current items; its raw output<br/>is not retained in conversation history<br/><br/>In: schema.Message[]<br/>Out: schema.Message"]

    ShoppingPlanner --> ShoppingExecutor["Update Shopping List Lambda<br/><br/>Parses and validates commands, then updates PostgreSQL<br/><br/>In: schema.Message<br/>Out: schema.Message[]"]

    ShoppingExecutor --> CinnaReply["Cinna Reply"]
    ActionRouter -- LIST/NONE --> CinnaReply

    IntentRouter -- FEEDBACK --> FeedbackListLambda["List Feedback Items Lambda<br/><br/>Loads pending and in-progress feedback for the planner<br/>when the action is UPDATE<br/><br/>In: schema.Message[]<br/>Out: schema.Message[]"]

    FeedbackListLambda --> FeedbackActionRouter{"Branch by Action<br/><br/>UPDATE | LIST | NONE"}

    FeedbackActionRouter -- UPDATE --> FeedbackPlanner["Feedback Planner<br/><br/>Determines feedback items to add or reopen;<br/>its raw output is not retained in conversation history<br/><br/>In: schema.Message[]<br/>Out: schema.Message"]

    FeedbackPlanner --> FeedbackExecutor["Update Feedback Items Lambda<br/><br/>Parses planner output and writes feedback to PostgreSQL<br/><br/>In: schema.Message<br/>Out: schema.Message[]"]

    FeedbackExecutor --> CinnaReply
    FeedbackActionRouter -- LIST/NONE --> CinnaReply
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
MESSAGE_ENCRYPTION_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
MESSAGE_ENCRYPTION_KEY_ID=local-v1
POSTGRES_USER=cinna
POSTGRES_PASSWORD=change_me
POSTGRES_DB=cinna
```

The sample encryption key is for local development only. Generate a unique
AES-256 key for production with `openssl rand -base64 32`. Preserve both that
key and its stable `MESSAGE_ENCRYPTION_KEY_ID` across deployments so existing
messages remain decryptable.

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

Validate that the required tables and shopping category type exist:

```bash
docker compose exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "\d allowed_users" \
  -c "\d shopping_list" \
  -c "\d agent_memory" \
  -c "\d feedbacks" \
  -c "\dT+ shopping_category"
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

Shopping-list rows reference `allowed_users`, so a Telegram user must be in
the allow list before Cinna can store items for them.

## Shopping Lists

Cinna supports natural-language requests to:

- list the current shopping list;
- add one or more items;
- remove existing items; and
- modify an item's name or category.

The supported categories are `grocery`, `pharmacy`, `pet`, `toy`,
`stationery`, and `other`. Item names are unique per user after trimming and
case normalization. Adding an existing name updates its category and
`updated_at` timestamp instead of creating a duplicate.

When Cinna reads a list, it separates active items from items whose
`updated_at` timestamp is older than one month. Expired items remain in the
database and can still be modified or removed.

Example requests:

```text
Show me my shopping list.
Add milk, cat litter, and a notebook to my shopping list.
Remove the milk.
Change the notebook to an A4 notebook.
```

## Run

Run Cinna locally with Telegram long polling:

```bash
go run ./cmd/cinna
```

In production mode, Cinna configures the Telegram webhook, starts the webhook
receiver, and serves it on `PORT` or `8080`.

## Container Image

The multi-stage `Dockerfile` builds a statically linked Linux `amd64` binary
and copies it into a minimal Alpine runtime image. Build it locally with:

```bash
docker build -t cinna:latest .
```

Run the image in development mode with the host PostgreSQL port exposed by
Compose. On Docker Desktop, `host.docker.internal` resolves to the host:

```bash
docker compose up -d postgres

docker run --rm \
  --env-file .env \
  -e DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@host.docker.internal:5432/${POSTGRES_DB}?sslmode=disable" \
  cinna:latest
```

The `gcp-deployment` Compose service builds and tags the image for Google
Artifact Registry. Set the registry location, Google Cloud project, and an
optional immutable image tag before building and pushing:

```bash
export REGION=us-central1
export PROJECT_ID=your-google-cloud-project
export IMAGE_TAG=$(git rev-parse --short HEAD)

gcloud auth configure-docker "${REGION}-docker.pkg.dev"
gcloud artifacts repositories describe cinna \
  --location "$REGION" \
  --project "$PROJECT_ID"

docker compose build gcp-deployment
docker compose push gcp-deployment
```

The Artifact Registry repository named `cinna` must already exist. This
Compose service builds and publishes the image; deploying it to a runtime such
as Cloud Run is a separate step. Production runtime configuration must provide
the variables listed in [Configuration](#configuration), including an HTTPS
webhook URL and a database URL reachable from the deployed container.

## Agent

The main request path is:

```text
Telegram message
  -> allow-list middleware
  -> intent and action classification
  -> shopping-list or feedback lookup (matching requests only)
  -> shopping or feedback planning and PostgreSQL updates (updates only)
  -> Cinna response generation
  -> Telegram response
```

The prompts under `internal/app/agent/prompt/` are embedded into the binary.
The agent keeps the 30 most recent messages in an in-memory, per-user working
history and periodically flushes buffered user and final assistant messages to
PostgreSQL. Stored message content is encrypted with `MESSAGE_ENCRYPTION_KEY`;
the database retains up to 100 messages per user, allowing recent history to be
restored after a process restart.

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

Run the manual agent graph checks:

```bash
RUN_MANUAL_TEST=1 go test ./internal/app/agent \
  -run 'Test(CinnaChat|IntentLambda|ShoppingTaskPlanningNode|FeedbackPlannerNode)$' -v
```

The model checks only require `DEEPSEEK_API_KEY`. The agent graph checks load
the application configuration, so `.env` must also contain
`TELEGRAM_BOT_TOKEN` and `DATABASE_URL` even though these focused checks do not
connect to Telegram or PostgreSQL.

If you already loaded `.env`, enable manual tests with:

```dotenv
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
│   │   │   ├── memory/
│   │   │   └── prompt/
│   │   ├── ports/
│   │   └── telegram/
│   ├── postgres/
│   │   └── sqlc/
│   └── utils/
├── .dockerignore
├── Dockerfile
├── compose.yaml
├── go.mod
└── sqlc.yaml
```
