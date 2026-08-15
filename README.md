<div align="center">

<svg width="180" height="180" style="border-radius: 50%;">
    <img src="./assets/cinna-small.jpg" width="180px" height="180px"/>
</svg>

<h1>Cinna</h1>

![CI](https://github.com/akane9506/cinna/actions/workflows/ci.yml/badge.svg)
![Deploy](https://github.com/akane9506/cinna/actions/workflows/deployment.yml/badge.svg)

English | [简体中文](./README.zh.md)

</div>

Cinna is a customizable Telegram ReAct agent bot built with Go and Eino. It supports configurable sub-agents and modules for a variety of use cases, uses database-backed memory to manage daily tasks while retaining long-term context across conversations, and protects historical messages with encryption for improved privacy and security.

## Agent Graph

[Mermaid Graph](https://mermaid.ai/d/a442ee33-6405-4268-84cb-9b0fb390b6fe)

```mermaid
flowchart LR
    Start(["START"]) --> InputProcessLambda["`**Input Process Lambda**<br><br>*Process input and store necessary info in the state*<br><br> In: *GraphInput<br>Out: []*schema.Message`"]
    InputProcessLambda --> IntentClassifier["`**Intent Classification**<br><br>*classifies user intention and possible action*<br><br>In: []*schema.Message<br>Out: *schema.Message*`"]
    IntentClassifier --> IntentLambda["`**Intent Validation Lambda**<br><br>*Validates intent classificaion node output, and feed intention to the output branching*<br><br>In: *schema.Message<br>Out: []*schema.Message`"]
    IntentLambda --> IntentRouter["`**Branch by Intent**<br>Intent:<br>SHOPPING | FEEDBACK | OTHER<br>`"]
    IntentRouter -- SHOPPING --> ListLambda["<b>List Lambda</b><br><br><i>List contents in the shopping list table <br>(note: 1 - list expired items, 2: items are with categories, notify cinna to responde with the corresponding category)</i><br><br>In: []*schema.Message<br>Out: []*schema.Message"]
    ListLambda --> ActionRouter["`**Branch by Action Type**<br>Action:<br>LIST | UPDATE | None<br>`"]
    ActionRouter -->|UPDATE| ShoppingPlanner["`**Shopping Planner**<br>(we don't save this output into history state)<br><br>*Reads user-cinna chat history and decide the Shopping DB operation*<br><br>In: []*schema.Message<br>Out: *schema.Message`"]
    ActionRouter -- LIST --> Passthrough(["Passthrough Node"])
    ShoppingPlanner --> UpdateShoppingListLambda["`**Update Shopping List Lambda**<br><br>Parse the output from the planner into an array of commands, then execute commands to update the DB shopping list<br><br>In:*schema.Message<br>out: []*schema.Message`"]
    IntentRouter -- FEEDBACK --> FeedbackListLambda["`**List Feedback Lambda**<br><br>Prepare pending/in-progress tasks for the Feedback Planner<br><br>In: []*schema.Message<br>Out: []*schema.Message`"]
    FeedbackActionRouter["<b>Branch by Action Type</b><br>Action:<br>List | UPDATE | NONE"] -->|"NONE | LIST (we don't support LIST command for feedbacks)"| Passthrough
    FeedbackListLambda --> FeedbackActionRouter
    FeedbackActionRouter -->|UPDATE| FeedbackPlanner["`**Feedback Planner**<br>(we don't save this output into history state as well)<br><br>Check the current user feedback lists, and decide which items should be added/updated.<br><br>In: []*schema.Message<br>Out: *schema.Message`"]
    FeedbackPlanner --> UpdateFeedbackLambda["<b>Update Feedback Items Lambda</b><br><br>Parse the output from the planner, then update the DB feedback list<br><br>In: *schema.Message<br>out: []*schema.Message"]
    UpdateShoppingListLambda --> Passthrough
    UpdateFeedbackLambda --> Passthrough
    IntentRouter -- OTHER --> Passthrough
    Passthrough --> MemoryCompressionRouter["<b>Conversation size check</b>"]
    MemoryCompressionRouter -- TRUE --> StartOfReplyAndCompressionChain["`**START**<br>Compression+Reply Chain`"]
    StartOfReplyAndCompressionChain --> MemoryCompressor["`**Memory Compressor**<br><br>Compresses chat history into enhanced notes.<br><br>In: []*schema.Message<br>Out: *schema.Message`"] & ReplyGenerator["`****Reply Generator****<br><br>Generates a reply based on the current turn.<br><br>In: []*schema.Message<br>Out: *schema.Message`"]
    ReplyGenerator -->|"key: cinna_reply"| CompressionChainPostProcessor["`**Chain Post Processor**<br><br>Waits for and combines the results from parallel nodes.<br><br>In: map[string]any<br>Out: *ReplyCompressionOutput`"]
    MemoryCompressor -->|"key: memory_compression"| CompressionChainPostProcessor
    MemoryCompressionRouter -- FALSE --> StartOfReplyOnlyChain["`**START**<br>Reply Only Chain`"]
    StartOfReplyOnlyChain --> ReplyGenerator & PassThroughNode(["Passthrough node"])
    PassThroughNode --> CompressionChainPostProcessor
    CompressionChainPostProcessor --> EndOfReplyAndCompressionChain["END\nCompression+Reply Chain"]
    EndOfReplyAndCompressionChain --> ProcessOutputLambda["<b>Process Output Lambda</b><br><br>Organizes and outputs chat history, response, and other metadata<br><br>In: *ReplyCompressionOutput<br>Out: *GraphOutput"]
    ProcessOutputLambda --> End(["END"])
```

Functions:

- Telegram long polling in development and webhooks in production
- Feedback collection with pending and in-progress feedback context
- Commands for help, managing allow-list access, and opting in or out of daily notifications
- Personalized daily notifications generated from each user’s current context

## Chat Commands

- `/help`: Display this help information.
- `/notify on`: Enable daily scheduled notifications.
- `/notify off`: Disable daily scheduled notifications.
- `/addmember <telegram_user_id>`: Add a user to Cinna's allow list (administrators only). The user must have started the Cinna Bot.

## Project Structure

```text
.
├── cmd/cinna/                 # Application entry point
├── db/
│   ├── schema.sql             # PostgreSQL database schema
│   └── queries/               # sqlc query definitions
├── internal/
│   ├── app/
│   │   ├── agent/             # Agent graph, nodes, and prompts
│   │   ├── ports/             # Repository and application interfaces
│   │   ├── telegram/          # Telegram updates, commands, and notification handlers
│   │   └── config.go          # Application configuration loading and validation
│   ├── postgres/              # PostgreSQL repository implementations
│   ├── security/              # Message encryption functionality
│   ├── server/                # Webhook and event-listener HTTP server
│   └── utils/                 # Shared utilities and test helpers
├── assets/                    # Project image assets
├── compose.yaml               # Local PostgreSQL and image-build configuration
├── Dockerfile                 # Container image build definition
├── sqlc.yaml                  # sqlc configuration
└── README.zh.md               # Chinese documentation
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
APP_TIMEZONE=America/Los_Angeles
DATABASE_URL=postgres://cinna:change_me@localhost:5432/cinna?sslmode=disable
DEEPSEEK_API_KEY=your_deepseek_api_key
MESSAGE_ENCRYPTION_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
MESSAGE_ENCRYPTION_KEY_ID=local-v1
POSTGRES_USER=cinna
POSTGRES_PASSWORD=change_me
POSTGRES_DB=cinna
WEBHOOK_SECRET=your_webhook_secret
```

The sample encryption key is for local development only. Generate a unique
AES-256 key for production with `openssl rand -base64 32`. Preserve both that
key and its stable `MESSAGE_ENCRYPTION_KEY_ID` across deployments so existing
messages remain decryptable.

Production webhook mode requires both `WEBHOOK_URL` and `WEBHOOK_SECRET`:

```dotenv
GO_ENV=production
WEBHOOK_URL=https://your-public-host.example
WEBHOOK_SECRET=your_webhook_secret
# Optional; defaults to 8080.
PORT=8080
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
webhook URL, webhook secret, and a database URL reachable from the deployed container.

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
