# Cinna

Cinna is being rebuilt as a modular Telegram assistant in Go.

The new implementation will use:

- Go
- CloudWeGo Eino for model and workflow orchestration
- PostgreSQL as the system of record
- Telegram Bot API as the initial transport

The previous TypeScript, Bun, and Firestore implementation has been intentionally
removed. There is no compatibility layer, data migration, or code migration
planned.

## Current State

The repository currently contains product prompts and planning documents only.
Implementation starts with Phase 0 in `CINNA_PLAN.md`.

Preserved prompts:

- `prompts/core/persona.md`
- `prompts/shopping/planner.instruction.md`
- `prompts/shopping/reply.instruction.md`

These prompts are product assets. Changes to them should be reviewed separately
from infrastructure or application code.

## Target Shape

```text
cinna/
├── cmd/
│   └── cinna/
├── internal/
│   ├── app/
│   ├── agent/
│   ├── conversation/
│   ├── shopping/
│   ├── platform/
│   │   ├── postgres/
│   │   └── telegram/
│   └── observability/
├── db/
│   ├── migrations/
│   └── queries/
├── prompts/
│   ├── core/
│   └── shopping/
├── tests/
│   └── integration/
├── deployments/
├── go.mod
└── Makefile
```

This will remain one deployable modular monolith until operational evidence
justifies splitting it.
