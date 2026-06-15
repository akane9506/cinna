# Cinna Progress

This file tracks implementation status and the immediate development focus.
`CINNA_PLAN.md` remains the source of truth for architecture, scope, and
sequencing.

## Current Focus

Set up PostgreSQL and implement a database-backed Telegram allow list.

## Completed

### Bootstrap

- [x] Initialize the Go service.
- [x] Add environment-based configuration.
- [x] Add structured logging.
- [x] Add signal handling and graceful HTTP server shutdown.
- [x] Add configuration tests.

### Telegram

- [x] Connect using long polling in development.
- [x] Configure webhook delivery in production.
- [x] Receive and reply with a temporary echo handler.
- [x] Load the initial allowed-user IDs from configuration.

## In Progress

### PostgreSQL and Allow List

- [ ] Add PostgreSQL configuration and connectivity with `pgx`.
- [ ] Add migration tooling with `goose`.
- [ ] Create the allowed Telegram users migration.
- [ ] Add typed allow-list queries with `sqlc`.
- [ ] Define the allow-list repository interface.
- [ ] Implement the PostgreSQL allow-list repository.
- [ ] Wire the allow-list repository into the Telegram adapter.
- [ ] Add repository integration tests.
- [ ] Add Telegram authorization tests.
