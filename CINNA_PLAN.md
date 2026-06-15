# Cinna Plan

Implementation status is tracked in `PROGRESS.md`.

## Architecture

```text
Telegram
  -> Application
  -> Eino Graph
       -> Chat Model
       -> Tool Validation
       -> Tool Execution
       -> Chat Model
  -> Telegram Response
```

Use Eino `compose.Graph` to implement a bounded ReAct workflow. Do not use ADK,
prebuilt agents, or multiple agents.

## Project Structure

```text
cinna/
├── cmd/cinna/main.go
├── internal/
│   ├── app/                  # configuration and dependency wiring
│   ├── agent/                # Eino graph, state, and tool routing
│   ├── shopping/             # domain services and repository interfaces
│   ├── postygres/
│   │   └── sqlc/             # generated sqlc code
│   └── platform/
│       ├── postgres/         # PostgreSQL repositories
│       └── telegram/         # Telegram adapter
├── db/
│   ├── migrations/
│   └── queries/
├── prompts/
├── deployments/
├── go.mod
└── README.md
```

Create directories only when they are needed.

## Rules

- Keep Telegram types inside the Telegram adapter.
- Keep Eino types inside the agent package.
- Tools call domain services, never SQL directly.
- Validate and authorize every tool call.
- Derive user identity from trusted Telegram context.
- Generate replies from committed operation results.
- Bound graph iterations, execution time, and conversation context.

## Milestones

### 1. Bootstrap

- Initialize the Go service.
- Add environment-based configuration.
- Add structured logging.
- Add signal handling and graceful shutdown.
- Test configuration, startup, and shutdown behavior.

### 2. Telegram

- Connect using long polling in development.
- Configure webhook delivery in production.
- Receive and reply to text messages.
- Safely ignore unsupported or malformed updates.
- Wire the Telegram adapter to the application handler.
- Restrict access using the PostgreSQL-backed allow list.
- Test text messages, authorization, and error handling.
- Verify webhook startup and shutdown in a production-like environment.

### 3. PostgreSQL and Allow List

- Add PostgreSQL configuration and connectivity with `pgx`.
- Add migration tooling with `goose`.
- Create the allowed Telegram users migration.
- Add typed allow-list queries with `sqlc`.
- Define and implement the allow-list repository.
- Wire the allow-list repository into the Telegram adapter.
- Add repository integration and Telegram authorization tests.

### 4. Agent Graph

Build this graph:

```text
START
  -> prepare_messages
  -> chat_model
       -> END             if the model returns a final response
       -> validate_tools  if the model requests tools
  -> execute_tools
  -> chat_model
```

### 5. Shopping Persistence

- Add shopping-list migrations and typed queries.
- Implement shopping repository interfaces and domain services.
- Test services independently from the graph.

### 6. Shopping Tools

Start with:

```text
add_shopping_items
list_shopping_items
```

- Validate tool arguments.
- Use the authenticated Telegram user.
- Call shopping services instead of SQL directly.
- Return committed operation results.

## First Target

```text
Telegram message
  -> Eino Graph
  -> add/list shopping tool
  -> PostgreSQL
  -> Telegram response
```

Defer ADK, prebuilt agents, multiple agents, Redis, queues, and vector databases.
