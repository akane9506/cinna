# Cinna Rebuild Plan

## Decision Record

This is a greenfield rebuild.

- No data migration from Firestore.
- No code migration from TypeScript.
- No backward-compatible API or schema.
- Existing prompts are retained as product assets.
- PostgreSQL is the only system of record.
- Eino is used for model integration and orchestration, not domain persistence.

## Target Structure

```text
cinna/
├── cmd/
│   └── cinna/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   └── dependencies.go
│   ├── agent/
│   │   ├── model.go
│   │   ├── router.go
│   │   └── callbacks.go
│   ├── conversation/
│   │   ├── domain.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── shopping/
│   │   ├── domain.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── planner.go
│   │   └── workflow.go
│   ├── platform/
│   │   ├── postgres/
│   │   │   ├── db.go
│   │   │   └── repositories.go
│   │   └── telegram/
│   │       ├── bot.go
│   │       ├── handler.go
│   │       └── update.go
│   └── observability/
│       ├── logging.go
│       └── tracing.go
├── db/
│   ├── migrations/
│   ├── queries/
│   └── sqlc.yaml
├── prompts/
│   ├── core/
│   │   └── persona.md
│   └── shopping/
│       ├── planner.instruction.md
│       └── reply.instruction.md
├── tests/
│   └── integration/
├── deployments/
│   ├── Dockerfile
│   └── compose.yaml
├── .env.example
├── go.mod
├── Makefile
└── README.md
```

Directories should be created when their first real file is needed. Do not add
empty packages to make the tree look complete.

## Phase 0: Bootstrap

Goal: a boring, testable Go service that starts and shuts down cleanly.

- Initialize the Go module.
- Pin Eino and provider integrations to explicit versions.
- Add configuration parsing with strict validation.
- Add structured logging with `log/slog`.
- Add `/healthz` and `/readyz`.
- Add graceful shutdown.
- Add `Makefile`, Dockerfile, and local PostgreSQL Compose service.
- Add CI for formatting, tests, vet, and lint.

Exit criteria:

- `go test ./...` passes.
- The service starts without Telegram or model credentials in test mode.
- Health endpoints work.
- PostgreSQL readiness is observable.

## Phase 1: PostgreSQL Foundation

Goal: establish the persistence boundary before adding agent behavior.

- Add `pgxpool`.
- Add `goose` migrations.
- Add `sqlc` query generation.
- Create `users`, `shopping_items`, and `feedback` tables.
- Use UUID or generated numeric identifiers consistently.
- Store timestamps as `TIMESTAMPTZ`.
- Enforce active-item uniqueness at the database level.
- Add repository integration tests against real PostgreSQL.

Initial shopping item fields:

```text
id
user_id
name
normalized_name
category
quantity
unit
note
completed_at
created_at
updated_at
```

Do not add vector columns, Redis, event buses, or audit tables in this phase.

Exit criteria:

- Migrations apply to an empty database.
- Repository tests cover add, list, rename, complete/remove, and clear.
- Transactions enforce deterministic shopping behavior.

## Phase 2: Shopping Application Slice

Goal: complete shopping behavior without an LLM or Telegram dependency.

- Define shopping commands and operation results.
- Implement category validation.
- Implement item normalization behind an interface.
- Implement add, list, update, remove, and clear services.
- Treat expected outcomes as typed results, not generic errors.
- Generate deterministic fallback replies.

Exit criteria:

- Domain and service tests cover all operations.
- A command-line or test harness can exercise the full shopping lifecycle.
- No framework types appear in the shopping package.

## Phase 3: Eino Integration

Goal: convert natural language into validated application commands.

- Configure the selected chat model through an Eino component.
- Load prompts from embedded files under `prompts/`.
- Split persona/routing concerns from shopping planning concerns.
- Build a deterministic Eino graph:

```text
input
  -> classify
  -> route
  -> shopping plan
  -> schema validation
  -> shopping service
  -> grounded reply
```

- Add callbacks for latency, token use, failures, and trace correlation.
- Use fake Eino components in workflow tests.
- Reject invalid, unsupported, or ambiguous commands before side effects.

Prompt refactor rule:

- Preserve the original prompt files.
- Introduce revised versions only through explicit prompt changes.
- Separate persona text from machine-output contracts over time.
- Add prompt fixtures/evaluations before materially changing behavior.

Exit criteria:

- Shopping requests produce validated commands.
- Invalid model output cannot write to PostgreSQL.
- Replies describe only committed operation results.
- Multi-language behavior is covered by evaluation fixtures.

## Phase 4: Telegram MVP

Goal: ship a usable private Telegram assistant.

- Add allowed-user enforcement with strict Telegram ID parsing.
- Normalize Telegram text updates into application inputs.
- Connect the shopping workflow.
- Add idempotency using Telegram update IDs.
- Start with long polling locally.
- Add webhook support only for deployment.
- Return safe user-facing errors without leaking model or database details.

Exit criteria:

- Add, list, update, remove, and clear work end to end.
- Duplicate Telegram updates do not duplicate writes.
- Unauthorized users are rejected.
- A manual smoke test passes in at least two languages.

## Phase 5: Conversation and Feedback

Goal: add only the memory needed for coherent interactions.

- Add conversations and messages if real product behavior needs persisted history.
- Bound context by turn count or token budget.
- Add feedback recording as a normal application service and Eino tool.
- Define retention and deletion behavior before storing broad chat history.

Do not add semantic memory by default. If retrieval quality later requires it,
add a separate `memories` table with `pgvector` and measure the benefit.

Exit criteria:

- Context loading is bounded and observable.
- Feedback writes are validated and attributable.
- Retention behavior is documented and tested.

## Phase 6: Voice and Production

Goal: support voice input and deploy a production-ready service.

- Download and validate Telegram voice files.
- Add transcription/model processing with size and timeout limits.
- Containerize and deploy.
- Use managed PostgreSQL with backups and point-in-time recovery.
- Add metrics, tracing, error reporting, and alerting.
- Add deployment smoke tests and rollback instructions.

Exit criteria:

- Voice and text use the same application workflows after normalization.
- Production deployment is reproducible.
- Database backup and restore procedures are verified.

## Deferred Decisions

These require evidence before adoption:

- Redis
- Dedicated vector database
- Message queue
- Multiple deployable services
- Multi-agent architecture
- Event sourcing
- Kubernetes

## First Implementation Milestone

The first milestone is intentionally narrow:

1. Bootstrap the Go service.
2. Start PostgreSQL locally.
3. Add the initial schema.
4. Implement and test `add_items` and `list_items` without Eino.
5. Add the Eino planner for those two commands.
6. Connect Telegram.

Only after that slice works should update, remove, clear, feedback, memory, or
voice work begin.
