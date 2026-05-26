# Cinna Project: Codex Working Notes

This file is the Codex-facing working note for the project. Read it at the
start of a session, then use the existing codebase as the source of truth when
there is any mismatch.

## Project Overview

Cinna is a modular Telegram assistant built with Bun and TypeScript. It uses
Telegraf for Telegram interactions, Hono for the HTTP health/webhook surface,
and Google's modern `@google/genai` SDK for Gemini-powered intent routing.

The current implementation supports:

- Whitelisted Telegram access.
- Text message handling through the core brain.
- Gemini structured JSON responses validated with Zod.
- Short-term in-memory chat history with bounded sessions.
- Firebase Admin initialization for Firestore with explicit project/database
  configuration and startup logging.
- Shopping schemas and a tested Firestore repository foundation for batched add,
  list, update, remove, and clear operations.
- Shopping `add_items` flow uses Brain routing, a second structured shopping
  planner call, category-grouped Firestore writes, then a persona reply generated
  from persisted operation results.
- The Telegram handler currently accepts only `add_items`; other shopping
  command shapes are modeled and repository-tested but still rejected at the
  handler boundary.
- Placeholder voice message handling.
- Placeholder feedback dispatch path.

## Current Architecture

- `src/index.ts`: Runtime entry point. Creates the Hono app, exposes the health
  route, launches the Telegram bot with long polling, and handles shutdown.
- `src/core/bot.ts`: Telegraf bot setup and message handlers.
- `src/core/middleware.ts`: Telegram user whitelist middleware.
- `src/core/brain.ts`: Gemini client, persona loading, structured response
  generation, schema validation, and chat history.
- `src/core/dispatcher.ts`: Intent dispatch boundary. Routes shopping intents to
  the shopping module with the original user text, while general chat still uses
  the Brain reply.
- `src/core/firestore.ts`: Firebase Admin initialization and Firestore database
  selection. Logs project ID, database ID, credential mode, and whether an
  existing Firebase app was reused.
- `src/core/types.ts`: Zod schema and TypeScript types for brain responses.
- `src/core/persona.md`: Local persona/system prompt source.
- `src/modules/shopping/types.ts`: Shopping list, item, planner command, and
  operation result schemas.
- `src/modules/shopping/planner.ts`: Shopping-specific structured LLM planner and
  post-persistence persona reply generator.
- `src/modules/shopping/handler.ts`: Shopping orchestration from planner commands
  through repository writes and final reply.
- `src/modules/shopping/repository.ts`: Firestore-backed shopping repository with
  injected store support for tests.

Future feature modules should live under `src/modules/{feature_name}`.

## Refactoring Direction

The next development phase should prioritize structure before adding more
shopping behavior. The current code works, but some responsibilities are packed
too tightly together. Refactors should make the layers explicit and keep object
construction out of business logic.

Target layers:

- Runtime composition: creates real infrastructure objects and wires the app.
  This belongs near the entry point or a small composition module.
- Transport layer: Telegraf/Hono adapters. This layer extracts request context,
  sends replies, maps framework errors, and delegates work.
- Core orchestration layer: coordinates use cases with plain inputs and outputs.
  It should not know Telegraf, Hono, Firebase document paths, or process startup.
- Domain/module layer: shopping use cases, command execution, validation, and
  deterministic business rules.
- Infrastructure layer: Firestore repositories, Gemini clients, prompt loading,
  logging adapters, and other external services.

Concrete refactoring goals:

- Split app creation from runtime startup. Importing modules should not launch
  Telegram polling or initialize services as side effects.
- Move dependency construction to composition code. Avoid factories with default
  live dependencies such as `repository = new ShoppingRepository()` embedded in
  module handlers.
- Keep handlers thin. A Telegram handler should parse the Telegram context,
  call an application service/use case, then send returned messages.
- Keep use cases framework-agnostic. Shopping orchestration should accept plain
  values such as `userId`, `userText`, and `brainResponse`, and return an
  explicit result instead of calling `ctx.reply` directly.
- Keep repositories behind small interfaces owned by the consuming module. Tests
  should pass fakes or in-memory adapters through composition instead of relying
  on optional constructor defaults inside production factories.
- Separate planning, execution, and reply generation. LLM planning should produce
  validated commands; command execution should produce persisted operation
  results; reply generation should use those results only after side effects
  succeed.
- Prefer named dependency objects over long positional factory parameters. If a
  constructor or factory needs more than two dependencies, pass a single typed
  object and make required dependencies explicit.
- Use narrow, typed result objects for user-facing outcomes. Do not encode flow
  control through thrown errors when the result is an expected unsupported action,
  validation failure, or missing user context.
- Keep module exports intentional. Export public composition/use-case entry
  points; keep helper functions private unless tests need a stable pure unit.

Refactoring should happen in small vertical slices with tests. For example,
extract a shopping application service first, adapt the Telegram handler to it,
then move runtime wiring into composition code. Avoid broad rewrites that change
architecture and feature behavior at the same time.

## Development Commands

- `bun install`: Install dependencies.
- `bun dev`: Start the bot in development with hot reload.
- `bun start`: Start the bot normally.
- `bun test:isolate`: Run the test suite with isolated workers.
- `bun test:coverage`: Run tests with coverage.
- `bun run lint`: Run ESLint.
- `bun run format:check`: Check formatting.
- `bun run format`: Format source files.

Before handing off meaningful code changes, run at least:

```bash
bun test:isolate
bun run lint
```

## Implementation Rules

1. Keep the brain as the single intent-classification path for user input.
2. Keep feature behavior modular; new product logic belongs in `src/modules/*`.
3. Preserve multi-language behavior: Cinna should reply in the user's detected
   language.
4. Keep Gemini output schema-driven and validate all model responses before
   performing side effects.
5. Treat dispatcher/module boundaries carefully: classification is not the same
   as persistence or action execution.
6. Do not create live services in default function parameters or module-level
   handler exports. Build production dependencies in composition/runtime code and
   inject them explicitly.
7. Do not let framework types leak into use cases. `Context`, Hono request
   objects, and Firebase document references belong at the adapter or
   infrastructure boundary.
8. Keep side-effect order obvious: classify, plan, validate, execute, then reply.
   Persisted state should be the source for success replies.
9. Do not introduce the deprecated `@google/generative-ai` package. Use
   `@google/genai`.
10. Avoid logging raw personal user content in production paths unless there is a
   clear debugging need and the data is redacted or truncated.
11. Keep tests close to changed behavior. Add narrow unit tests for core logic and
   broader tests when changing cross-module flows.

## Review Notes From Initial Read

These are current improvement candidates, not blockers for every change:

- `ALLOWED_USERS` should reject malformed values instead of allowing `NaN`.
- Brain parse failures should not send raw invalid model output to users.
- Conversation history currently stores only the assistant reply text, which may
  conflict with the JSON-only system prompt over long chats.
- `src/index.ts` launches the bot as an import side effect; splitting app
  creation from runtime startup would make tests and webhook deployment easier.
- The dummy `src/index.test.ts` should eventually become a real Hono health
  route test.
- Voice handling is still a placeholder and should eventually download Telegram
  voice files and route transcription through the brain.

## Roadmap Alignment

Follow the roadmap and status checklist in `CINNA_PLAN.md`. The next major
unfinished work is Phase 3, implemented as vertical end-to-end shopping action
slices:

- First: finish hardening `add_items` from Telegram text through planner,
  repository, Firestore writes, and user reply. The unit-tested implementation
  already supports multi-item and multi-category add batches.
- Then: wire `list_items`, `remove_items`, `update_items`, and `clear_list` into
  the handler/reply path as separate verified slices.
- Finish with a full shopping-list smoke flow against the configured development
  Firestore database.

After the shopping-list MVP works locally, Phase 4 is the online launch track:

- Hono webhook server.
- Docker/Cloud Run packaging.
- Production deployment workflow.
- Production shopping-list smoke test.

Audio, feedback, and broader multi-language polish are Phase 5 post-launch work.

## Shopping LLM Pattern

Do not write shopping-list data directly from the core Brain response. The expected
flow is:

```text
user text -> Brain intent reply -> shopping planner structured commands
-> repository writes -> persona reply from persisted results
```

This lets one user message produce multiple storage operations, such as adding
several shopping items across different categories at once, while keeping replies
grounded in saved data. For now, non-add shopping commands must stay modeled as
planner/repository capabilities until the handler has a complete response flow
for each action.

## Firestore Shape

Start with the simplest useful shopping-list model: one list document per
planner-controlled category, scoped under the Telegram user. If the planner does
not provide a category, use `grocery`; if it provides an unknown category, fall
back to `other`.

```text
users/{userId}/shoppingLists/{category}
```

Use one of the controlled category ids as the document id:

- No category provided: `grocery`
- Supported categories: `grocery`, `pharmacy`, `pet_store`, `toy_shop`,
  `stationery`, `other`
- Unknown category: `other`

Keep the document shape compact:

```ts
export interface ShoppingListItem {
  name: string; // "两箱全脂牛奶(milk)"
  addedAt: number; // epoch milliseconds
}

export interface ShoppingListDoc {
  category:
    | "grocery"
    | "pharmacy"
    | "pet_store"
    | "toy_shop"
    | "stationery"
    | "other";
  items: ShoppingListItem[];
  lastUpdated: number; // epoch milliseconds
}
```

Normalize category ids before writing:

```ts
function normalizeShoppingCategory(category?: string): ShoppingCategory {
  if (!category?.trim()) return "grocery";
  return ShoppingCategorySchema.parse(category); // invalid values fall back to "other"
}
```

## Functional Requirements

- Shopping list retrieval: when the user asks to view a shopping list, Cinna
  should first reply with the requested list contents.
- Shopping retention prompt: after replying with a requested shopping list, Cinna
  should check whether the list or any items are older than one month. If stale
  data exists, send a separate follow-up message asking whether the user wants
  to delete it. Do not delete stale data automatically.

## Coding Style

- TypeScript, ESM, strict mode.
- Prefer existing local patterns over new abstractions.
- Use Zod for boundary validation.
- Use `logger` instead of `console` in runtime code, except for early startup or
  configuration failure paths where the existing project already does so.
- Keep comments sparse and useful.
- Keep persona/prompt changes deliberate; they affect both UX and structured
  output reliability.
