# Cinna Project Notes

## Collaboration

For all future implementation requests, only show the proposed code or patch.
Do not modify files, apply patches, run formatting, or execute any command that
changes the workspace. The user will make all changes manually. Show code changes
in code blocks rather than diffs.

## Status

Cinna is a greenfield rebuild. The previous TypeScript/Bun/Firestore application
was deleted intentionally.

Do not:

- Restore or port the old implementation.
- Add Firestore compatibility.
- Build data migration scripts.
- Treat previous schemas or tests as requirements.

The preserved prompts are the only implementation assets carried forward.

## Stack

- Go
- CloudWeGo Eino
- PostgreSQL
- `pgx`
- `sqlc`
- `goose`
- Telegram Bot API
- Standard library HTTP server unless a concrete need for another framework appears

## Architecture Rules

1. Keep the application a modular monolith.
2. Put dependency construction in `internal/app`.
3. Keep Telegram types inside the Telegram adapter.
4. Keep PostgreSQL details inside repository implementations.
5. Define repository interfaces in the package that consumes them.
6. Use Eino graphs for deterministic workflows with side effects.
7. Never let unvalidated model output reach a database operation.
8. Generate user replies from actual operation results.
9. Keep prompts as external, embedded assets under `prompts/`.
10. Avoid logging raw user messages.

## Required Shopping Flow

```text
Telegram update
  -> normalize input
  -> load relevant context
  -> classify intent
  -> run shopping planner
  -> validate command
  -> execute domain service in a transaction
  -> generate reply from operation result
  -> send Telegram response
```

The Eino layer may select and orchestrate operations. It must not contain SQL or
own transaction rules.

## Testing

- Pure domain rules: table-driven unit tests.
- Repository behavior: PostgreSQL integration tests.
- Eino workflows: fake model and fake tool implementations.
- Telegram adapter: update fixtures with mocked application service.
- End-to-end tests: only for critical vertical slices.

Before handing off code, run:

```bash
go test ./...
go vet ./...
```

Run the configured linter when it is introduced in Phase 0.

## Plans

`CINNA_PLAN.md` is the source of truth for scope and sequencing. Each phase must
end in a working vertical slice. Avoid building generalized agent infrastructure
before the shopping use case requires it.
