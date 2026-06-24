# Cinna Project Notes

## Collaboration

For all future implementation requests, only show the proposed code or patch.
Do not modify files, apply patches, run formatting, or execute any command that
changes the workspace. The user will make all changes manually. Show code changes
in code blocks rather than diffs.

## Stack

- Go
- CloudWeGo Eino
- PostgreSQL
- `pgx`
- `sqlc`
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

The Eino layer may select and orchestrate operations. It must not contain SQL or
own transaction rules.

Before handing off code, run:

```bash
go test ./...
go vet ./...
```
