# Cinna: Implementation Roadmap

Cinna is a modular, AI-powered Telegram assistant designed to handle voice and text inputs for various tasks, starting with grocery management. Powered by **Gemini 3 Flash**, it leverages frontier reasoning and agentic capabilities for seamless interaction.

## 1. Modular Architecture

The system follows a "Brain & Modules" pattern:

- **Core Brain (Gemini 3 Flash):** Acts as the intent router and transcription engine, utilizing its pro-grade reasoning to handle complex requests.
- **Feature Modules:** Independent logic for specific tasks (Grocery, Reminders, etc.).
- **Domain Planners:** Module-specific LLM services convert conversational intent into validated, storage-safe commands. For grocery, the planner should produce DB operations such as `add_item`, `remove_item`, `update_item`, `list_items`, and `clear_list`.
- **Persistence Layer:** Firestore repositories own reads/writes and keep storage details out of the dispatcher and LLM prompts.
- **Response Engine:** Ensures responses match the user's input language and are grounded in actual operation results when persistence is involved.

### Grocery Dispatch Architecture

Grocery requests use a two-stage LLM pipeline:

```text
Telegram message
  -> Core Brain
     - classify intent as GROCERY / FEEDBACK / OTHER
     - detect language
     - preserve conversational context
  -> Dispatcher
     - route GROCERY only
     - do not write chat content directly to Firestore
  -> Grocery Handler
     - load current list state when needed
     - call Grocery Planner
  -> Grocery Planner
     - convert user text + BrainResponse + current state into strict DB commands
  -> Grocery Service / Repository
     - validate and execute Firestore operations
  -> Reply
     - build confirmation/list output from actual DB results
```

The core rule is: **the Brain routes; the Grocery Planner plans DB operations; the Repository persists.** Gemini-generated chat text must not be treated as the source of truth for Firestore writes.

### Project Structure

```text
cinna/
├── .github/workflows/
│   ├── ci.yml                # CI: Lint, Test
│   └── cd.yml                # CD: Deploy to Cloud Run
├── src/
│   ├── core/
│   │   ├── bot.ts            # Telegraf setup
│   │   ├── brain.ts          # Gemini & Function Calling
│   │   └── config.ts         # Env validation (Zod)
│   ├── modules/
│   │   ├── grocery/          # Grocery module logic
│   │   │   ├── handler.ts
│   │   │   ├── planner.ts
│   │   │   ├── repository.ts
│   │   │   ├── types.ts
│   │   │   └── service.ts
│   │   └── shared/           # Cross-module utils
│   ├── services/
│   │   └── audio.ts          # Telegram voice downloader
│   ├── index.ts              # Entry point (Bun + Hono)
│   └── types.ts              # Global types
├── .env                      # Local secrets
├── .gitignore
├── Dockerfile                # For GCP Cloud Run
├── package.json
├── tsconfig.json
└── README.md
```

## 2. Exhaustive Dependency List

### Production Dependencies

- **`telegraf`**: Modern Telegram Bot Framework.
- **`@google/genai`**: Google's official unified SDK for Gemini 2.0+.
- **`zod`**: TypeScript-first schema validation for environment variables.
- **`hono`**: Ultrafast web framework for the edge and Cloud Run.
- **`firebase-admin`**: Firebase Admin SDK for Firestore persistence.
- **`fetch`**: Built-in Bun API to download voice files from Telegram's file server.

### Development Dependencies

- **`typescript`**: Static typing for the codebase.
- **`@types/bun`**: Type definitions for Bun.
- **`bun:test`**: Built-in Bun testing framework.
- **`eslint`**: Linter to maintain code quality, with TypeScript support.

---

## 3. CI/CD Strategy (GitHub Actions)

### CI (Continuous Integration)

Triggered on every pull request and push to `main`.

- **Lint:** Run `eslint` to ensure style consistency.
- **Test:** Run `bun test` to ensure no regressions.

### CD (Continuous Deployment)

Triggered on merge to `main` or specific tags.

- **Build & Push:** Build Docker image and push to Google Artifact Registry.
- **Deploy:** Deploy the new image to GCP Cloud Run.

---

## 5. Personality & Memory Strategy

### Personality (The "Cinna" Vibe)

- **System Instruction:** All calls to Gemini will include a dedicated `systemInstruction` to define Cinna's persona.
- **Traits:** Helpful, efficient, slightly witty, and context-aware.
- **Language Parity:** Cinna must respond in the same language detected in the user's input.

### Memory Implementation

1. **Short-Term (Conversation Thread):**
   - Use an in-memory `Map` (keyed by `chatId`) with an LRU eviction strategy to store the last 10-20 turns.
   - Pass this history to the Gemini API call.
2. **Long-Term (User Data & Feedback):**
   - Implement "Agentic Memory" via Function Calling.
   - _Storage:_ **Firestore** will be used to persist grocery lists, user preferences, and bot improvement feedback (bugs/suggestions).
   - Cinna will have a dedicated tool to `record_bot_feedback(category, detail)` where `category` is 'bug' or 'improvement'.

---

## 6. Granular Implementation Roadmap

### Phase 1: Infrastructure & Basic Connectivity

- **Step 1.1: Project Initialization**
  - Initialize `bun`, install `telegraf`, `hono`, `typescript`, `zod`.
  - _Verification:_ [x] `bun test` succeeds.
- **Step 1.2: CI/CD Setup**
  - Create `.github/workflows/ci.yml` using `oven-sh/setup-bun`.
  - _Verification:_ [x] Push to GitHub and see the green checkmark on the commit.
- **Step 1.3: Basic Bot "Heartbeat"**
  - Create `src/index.ts` with a simple Telegraf handler and Hono health check.
  - _Verification:_
    - [x] Manual: Send text to bot, get reply.
    - [x] Test: Unit test the message handler logic using `bun:test`.
- **Step 1.4: Audio Reception Verification**
  - Update bot to handle `voice` updates.
  - _Verification:_
    - [x] Manual: Send voice note, bot replies "Voice received".

### Phase 2: Core Brain (Gemini Integration)

- **Step 2.1: Simple Gemini Text Completion**
  - Integration with `@google/genai`.
  - Wire up `src/core/bot.ts` to use Gemini for text replies.
  - _Verification:_ [x] Unit tests for brain.ts pass.
- **Step 2.2: Text Intent Routing & Memory**
  - Implement `Brain` service for structured intent classification (JSON output).
  - Integrate `systemInstruction` for personality.
  - Implement basic in-memory session tracking for short-term context.
  - _Verification:_ [x] Unit tests with various intents and mocked Gemini output pass.

### Phase 3: Persistence (Firestore) & Modules

- **Step 3.1: Firebase Initialization**
  - Install `firebase-admin` and configure credentials.
  - Add a Firestore initialization module that can be mocked in tests.
  - _Verification:_ [ ] `firebase-admin` successfully initializes in development and is bypassed/mocked in tests.
- **Step 3.2: Grocery Domain Model & Firestore Repository**
  - Implement `src/modules/grocery/types.ts` with Zod schemas for grocery items, planner commands, and operation results.
  - Implement a Firestore repository that stores active grocery items under user-scoped lists.
  - Recommended collection shape:
    ```text
    users/{telegramUserId}/groceryLists/{shopOrDefault}/items/{itemId}
    ```
  - Store structured item fields such as display name, canonical name, quantity/notes when available, shop, status, and timestamps.
  - _Verification:_ [ ] Repository unit tests cover add, list, update, remove, and clear using a mocked Firestore adapter.
- **Step 3.3: Grocery LLM Planner**
  - Implement `src/modules/grocery/planner.ts` as a grocery-specific LLM service using `@google/genai`.
  - Input should include the original user text, the core `BrainResponse`, user/chat identifiers, optional shop, and current list state when needed.
  - Output must be strict JSON validated by Zod before any side effect. The planner should return DB commands, not user-facing prose.
  - Supported planner commands:
    - `add_item`
    - `remove_item`
    - `update_item`
    - `list_items`
    - `clear_list`
  - _Verification:_ [ ] Planner tests mock Gemini output and reject malformed or unsupported commands before execution.
- **Step 3.4: Persistent Grocery Handler**
  - Implement `src/modules/grocery/handler.ts` to orchestrate current state loading, planner invocation, service execution, and final reply composition.
  - Update `Dispatcher` so the `GROCERY` case calls the grocery handler instead of writing `brainResponse.item` directly to storage.
  - Build `list` responses from Firestore results, not from Gemini-generated text.
  - For add/update/remove/clear, reply from operation results and preserve the user's detected language/persona tone.
  - _Verification:_ [ ] Items added/updated/removed via Telegram persist correctly in Firestore, and list replies reflect actual stored data.
- **Step 3.5: Audio-to-Brain Link** (Moved)
  - Implement `audio.ts` to fetch and pass audio bytes to Gemini.
  - _Verification:_ [ ] Send voice note and verify transcription and intent routing.
- **Step 3.6: Feedback & Bug Tracker**
  - Implement a `feedback` module.
  - Define a tool for Gemini: `record_bot_feedback(category, detail)`.
  - _Verification:_ [ ] Manual: Tell the bot "I found a bug: the audio is too slow", and verify it appears in the Firestore `feedback` collection.
- **Step 3.7: Multi-Language Loop**
  - Refine system prompts for language parity.
  - _Verification:_ [ ] Manual: Multi-language voice note testing (FR, EN, ES, etc.).

### Phase 4: Deployment & Webhooks

- **Step 4.1: Hono Webhook Server**
  - Expose webhook endpoint using Hono.
  - _Verification:_
    - [ ] Test: Unit test for the Hono route handling Telegram updates.
- **Step 4.2: GCP Containerization**
  - Finalize `Dockerfile`.
  - _Verification:_
    - [ ] Manual: `docker build` succeeds; container starts locally.
- **Step 4.3: Cloud Run Deployment**
  - Setup `cd.yml` for automated deployment.
  - _Verification:_
    - [ ] Manual: Bot works in production via the Cloud Run URL.

## 4. Multi-Language Strategy

- **Prompting:** The System Instruction will state: _"You are a helpful assistant. Detect the user's language. If a tool is called, process the data. Always output your final text response in the detected language."_
- **Voice:** **Gemini 3 Flash** natively understands audio in multiple languages with high fidelity.
