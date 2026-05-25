# Cinna: Implementation Roadmap

Cinna is a modular, AI-powered Telegram assistant designed to handle voice and text inputs for various tasks, starting with shopping-list management. Powered by **Gemini 3 Flash**, it leverages frontier reasoning and agentic capabilities for seamless interaction.

## 1. Modular Architecture

The system follows a "Brain & Modules" pattern:

- **Core Brain (Gemini 3 Flash):** Acts as the intent router and transcription engine, utilizing its pro-grade reasoning to handle complex requests.
- **Feature Modules:** Independent logic for specific tasks (Shopping, Reminders, etc.).
- **Domain Planners:** Module-specific LLM services convert conversational intent into validated, storage-safe commands. For shopping lists, the planner should produce DB operations such as `add_items`, `remove_items`, `update_items`, `list_items`, and `clear_list`.
- **Persistence Layer:** Firestore repositories own reads/writes and keep storage details out of the dispatcher and LLM prompts.
- **Response Engine:** Ensures responses match the user's input language and are grounded in actual operation results when persistence is involved.

### Shopping Dispatch Architecture

Shopping requests use a staged LLM pipeline:

```text
Telegram message
  -> Core Brain
     - classify intent as SHOPPING / FEEDBACK / OTHER
     - detect language
     - preserve conversational context
  -> Dispatcher
     - route SHOPPING only
     - do not write chat content directly to Firestore
  -> Shopping Handler
     - load current list state when needed
     - call Shopping Planner
  -> Shopping Planner
     - run a second structured LLM pass
     - convert user text + BrainResponse + current state into strict DB commands
     - split multi-item requests into category-specific commands
  -> Shopping Handler / Repository
     - validate and execute Firestore operations
  -> Reply
     - generate persona reply from actual DB results after persistence succeeds
     - use deterministic fallback if reply generation fails validation
```

The core rule is: **the Brain routes; the Shopping Planner plans DB operations; the Repository persists.** Gemini-generated chat text must not be treated as the source of truth for Firestore writes.

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
│   │   └── shopping/         # Shopping module logic
│   │       ├── handler.ts
│   │       ├── planner.ts
│   │       ├── repository.ts
│   │       ├── types.ts
│   │       └── prompts/
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
   - _Storage:_ **Firestore** will be used to persist shopping lists, user preferences, and bot improvement feedback (bugs/suggestions).
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

### Phase 3: Shopping MVP Persistence (Firestore)

The goal of this phase is to finish the persistent shopping-list function end to end. Once shopping add/list/update/remove/clear works locally with Firestore and tests, the project should move directly to getting the bot online instead of waiting for audio, feedback, or broader language polish.

- **Step 3.1: Firebase Initialization**
  - Install `firebase-admin` and configure credentials.
  - Add a Firestore initialization module that can be mocked in tests.
  - _Verification:_ [x] `firebase-admin` initializes via explicit project/database config and repository tests use a mocked store.
- **Step 3.2: Shopping Domain Model & Firestore Repository**
  - Implement `src/modules/shopping/types.ts` with Zod schemas for shopping items, planner commands, and operation results.
  - Implement a Firestore repository that stores active shopping items under user-scoped category lists.
  - Recommended document shape:
    ```text
    users/{telegramUserId}/shoppingLists/{category}
    ```
  - Store `category`, an `items` array, and `lastUpdated` as epoch milliseconds; each item should contain `name` and `addedAt` as epoch milliseconds.
  - _Verification:_ [x] Repository unit tests cover add, list, update, remove, and clear using a mocked Firestore adapter.
- **Step 3.3.1: End-to-End `add_items` Slice**
  - Implement only enough planner, handler, dispatcher wiring, and reply formatting for `add_items`.
  - The full path must be: Telegram text -> core Brain routes `SHOPPING` -> shopping planner returns validated `add_items` -> handler groups items by normalized category -> repository writes the grouped batch -> Firestore is updated -> bot replies from the repository results.
  - Keep unsupported shopping actions explicitly unimplemented or safely rejected in this slice.
  - _Verification:_ [x] Unit tests cover `add_items` planner parsing, command execution, reply formatting, handler orchestration, and dispatcher routing.
  - _Verification:_ [ ] Manual Telegram test adds one item to the configured development Firestore database and the bot replies with the persisted item/category.
- **Step 3.3.1a: Replace Free-Form Store Text with List Category**
  - Free-form store/place text is not reliable enough as a grouping or display field. Replace it with a broader planner-controlled category before building more shopping slices.
  - Current categories: `grocery`, `pharmacy`, `pet_store`, `toy_shop`, `stationery`, `other`.
  - Do not persist free-form shop/place text.
  - _Verification:_ [x] Planner/schema/repository tests cover category normalization, `stationery`, and unknown category fallback to `other`.
- **Step 3.3.2: End-to-End `list_items` Slice**
  - Extend the planner, handler, dispatcher tests, and replies for `list_items`.
  - List replies must be built from Firestore results, not Gemini-generated prose.
  - _Verification:_ [x] Unit tests cover `list_items` planning/execution/reply behavior for empty and non-empty lists.
  - _Verification:_ [ ] Manual Telegram test lists the item added in Step 3.3.1 from the configured development Firestore database.
- **Step 3.3.3: End-to-End `remove_items` Slice**
  - Extend the same path for `remove_items`.
  - Missing-item behavior should be explicit and user-facing.
  - _Verification:_ [x] Repository unit tests cover successful removal and missing-item operation results.
  - _Verification:_ [ ] Handler/reply unit tests cover successful removal and missing-item user replies.
  - _Verification:_ [ ] Manual Telegram test removes an item from Firestore and confirms the reply reflects the repository result.
- **Step 3.3.4: End-to-End `update_items` Slice**
  - Extend the same path for `update_items`.
  - Use current list state as planner context when needed so rename/update requests target existing items safely.
  - _Verification:_ [x] Repository unit tests cover successful update and missing-item operation results.
  - _Verification:_ [ ] Handler/reply unit tests cover successful update and missing-item user replies.
  - _Verification:_ [ ] Manual Telegram test updates an item in Firestore and confirms the reply/list reflect the new item name.
- **Step 3.3.5: End-to-End `clear_list` Slice**
  - Extend the same path for `clear_list`.
  - Clearing should reply from the repository result and make the final stored list empty.
  - _Verification:_ [x] Repository unit tests cover clearing non-empty and already-empty lists.
  - _Verification:_ [ ] Handler/reply unit tests cover clearing non-empty and already-empty lists.
  - _Verification:_ [ ] Manual Telegram test clears the development Firestore list and confirms a follow-up list is empty.
- **Step 3.3.6: Full Shopping Smoke Test**
  - Add a manual smoke script or checklist that runs add, list, update, remove, and clear against configured `FIREBASE_PROJECT_ID` and `FIRESTORE_DATABASE_ID`.
  - _Verification:_ [ ] Smoke flow passes against the development Firestore database and leaves predictable test data.

### Phase 4: Online Launch

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
- **Step 4.4: Production Shopping Smoke Test**
  - Verify the deployed bot can add, list, update, remove, and clear shopping items against the intended Firestore project.
  - _Verification:_
    - [ ] Manual: Send shopping commands to the production Telegram bot and confirm Firestore state matches replies.

### Phase 5: Post-Launch Module Expansion

- **Step 5.1: Audio-to-Brain Link**
  - Implement `audio.ts` to fetch and pass audio bytes to Gemini.
  - _Verification:_ [ ] Send voice note and verify transcription and intent routing.
- **Step 5.2: Feedback & Bug Tracker**
  - Implement a `feedback` module.
  - Define a tool for Gemini: `record_bot_feedback(category, detail)`.
  - _Verification:_ [ ] Manual: Tell the bot "I found a bug: the audio is too slow", and verify it appears in the Firestore `feedback` collection.
- **Step 5.3: Multi-Language Loop**
  - Refine system prompts for language parity.
  - _Verification:_ [ ] Manual: Multi-language voice note testing (FR, EN, ES, etc.).

## 4. Multi-Language Strategy

- **Prompting:** The System Instruction will state: _"You are a helpful assistant. Detect the user's language. If a tool is called, process the data. Always output your final text response in the detected language."_
- **Voice:** **Gemini 3 Flash** natively understands audio in multiple languages with high fidelity.
