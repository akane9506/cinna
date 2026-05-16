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
- Placeholder voice message handling.
- Placeholder grocery and feedback dispatch paths.

## Current Architecture

- `src/index.ts`: Runtime entry point. Creates the Hono app, exposes the health
  route, launches the Telegram bot with long polling, and handles shutdown.
- `src/core/bot.ts`: Telegraf bot setup and message handlers.
- `src/core/middleware.ts`: Telegram user whitelist middleware.
- `src/core/brain.ts`: Gemini client, persona loading, structured response
  generation, schema validation, and chat history.
- `src/core/dispatcher.ts`: Intent dispatch boundary. Currently logs grocery
  and feedback intents, then replies to the user.
- `src/core/types.ts`: Zod schema and TypeScript types for brain responses.
- `src/core/persona.md`: Local persona/system prompt source.

Future feature modules should live under `src/modules/{feature_name}`.

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
6. Do not introduce the deprecated `@google/generative-ai` package. Use
   `@google/genai`.
7. Avoid logging raw personal user content in production paths unless there is a
   clear debugging need and the data is redacted or truncated.
8. Keep tests close to changed behavior. Add narrow unit tests for core logic and
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
unfinished work is Phase 3:

- Firebase/Admin initialization.
- Persistent grocery module.
- Audio-to-brain link.
- Feedback module.
- Multi-language verification.

Phase 4 is deployment-oriented:

- Hono webhook server.
- Docker/Cloud Run packaging.
- Production deployment workflow.

## Coding Style

- TypeScript, ESM, strict mode.
- Prefer existing local patterns over new abstractions.
- Use Zod for boundary validation.
- Use `logger` instead of `console` in runtime code, except for early startup or
  configuration failure paths where the existing project already does so.
- Keep comments sparse and useful.
- Keep persona/prompt changes deliberate; they affect both UX and structured
  output reliability.
