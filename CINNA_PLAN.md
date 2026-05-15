# Cinna: Implementation Roadmap

Cinna is a modular, AI-powered Telegram assistant designed to handle voice and text inputs for various tasks, starting with grocery management. Powered by **Gemini 3 Flash**, it leverages frontier reasoning and agentic capabilities for seamless interaction.

## 1. Modular Architecture
The system follows a "Brain & Modules" pattern:
- **Core Brain (Gemini 3 Flash):** Acts as the intent router and transcription engine, utilizing its pro-grade reasoning to handle complex requests.
- **Feature Modules:** Independent logic for specific tasks (Grocery, Reminders, etc.).
- **Response Engine:** Ensures responses match the user's input language.

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
   - *Storage:* **Firestore** will be used to persist grocery lists, user preferences, and bot improvement feedback (bugs/suggestions).
   - Cinna will have a dedicated tool to `record_bot_feedback(category, detail)` where `category` is 'bug' or 'improvement'.

---

## 6. Granular Implementation Roadmap

### Phase 1: Infrastructure & Basic Connectivity
*   **Step 1.1: Project Initialization**
    *   Initialize `bun`, install `telegraf`, `hono`, `typescript`, `zod`.
    *   *Verification:* [x] `bun test` succeeds.
*   **Step 1.2: CI/CD Setup**
    *   Create `.github/workflows/ci.yml` using `oven-sh/setup-bun`.
    *   *Verification:* [x] Push to GitHub and see the green checkmark on the commit.
*   **Step 1.3: Basic Bot "Heartbeat"**
    *   Create `src/index.ts` with a simple Telegraf handler and Hono health check.
    *   *Verification:* 
        - [x] Manual: Send text to bot, get reply.
        - [x] Test: Unit test the message handler logic using `bun:test`.
*   **Step 1.4: Audio Reception Verification**
    *   Update bot to handle `voice` updates.
    *   *Verification:* 
        - [x] Manual: Send voice note, bot replies "Voice received".

### Phase 2: Core Brain (Gemini Integration)
*   **Step 2.1: Simple Gemini Text Completion**
    *   Integration with `@google/genai`.
    *   Wire up `src/core/bot.ts` to use Gemini for text replies.
    *   *Verification:* 
        - [ ] Test: Mock API call to Gemini and verify handling of the response.
        - [ ] Manual (CLI): Run a CLI test script (`scripts/test-gemini.ts`) that prints a Gemini completion to the console.
        - [ ] Manual (Telegram): Send "Hello" to the bot in Telegram and receive an AI-generated greeting.
*   **Step 2.2: Text Intent Routing & Memory**
    *   Implement `Brain` service for structured intent classification (JSON output).
    *   Integrate `systemInstruction` for personality.
    *   Implement basic in-memory session tracking for short-term context.
    *   *Verification:* 
        - [ ] Test: Unit tests with various inputs (Grocery vs Other) and mocked Gemini output.
        - [ ] Test: Verify that conversation history is correctly passed to the API.
        - [ ] Manual: Send "/test_route buy milk" followed by "what did I just say?" to verify memory.
*   **Step 2.3: Audio-to-Brain Link**
    *   Implement `audio.ts` to fetch and pass audio bytes to Gemini.
    *   *Verification:* 
        - [ ] Test: Integration test simulating audio download and Gemini processing.
        - [ ] Manual: Send a voice note "add eggs to my list" and verify the bot logs the correct transcription and intent.

### Phase 3: Modules & Persistence (Firestore)
*   **Step 3.1: Firestore Service Integration**
    *   Set up Firebase Admin SDK or Cloud Firestore SDK.
    *   Implement a base `FirestoreService` for standardized CRUD operations.
    *   *Verification:* [ ] Test: Integration test for basic read/write to a test collection.
*   **Step 3.2: Persistent Grocery Module**
    *   Refactor `GroceryService` to use Firestore.
    *   *Verification:* [ ] Test: Verify grocery list persists across bot restarts.
*   **Step 3.3: Feedback & Bug Tracker**
    *   Implement a `feedback` module.
    *   Define a tool for Gemini: `record_bot_feedback(category, detail)`.
    *   *Verification:* [ ] Manual: Tell the bot "I found a bug: the audio is too slow", and verify it appears in the Firestore `feedback` collection.
*   **Step 3.4: Multi-Language Loop**
    *   Refine system prompts for language parity.
    *   *Verification:* [ ] Manual: Multi-language voice note testing (FR, EN, ES, etc.).

### Phase 4: Deployment & Webhooks
*   **Step 4.1: Hono Webhook Server**
    *   Expose webhook endpoint using Hono.
    *   *Verification:* 
        - [ ] Test: Unit test for the Hono route handling Telegram updates.
*   **Step 4.2: GCP Containerization**
    *   Finalize `Dockerfile`.
    *   *Verification:* 
        - [ ] Manual: `docker build` succeeds; container starts locally.
*   **Step 4.3: Cloud Run Deployment**
    *   Setup `cd.yml` for automated deployment.
    *   *Verification:* 
        - [ ] Manual: Bot works in production via the Cloud Run URL.

## 4. Multi-Language Strategy
- **Prompting:** The System Instruction will state: *"You are a helpful assistant. Detect the user's language. If a tool is called, process the data. Always output your final text response in the detected language."*
- **Voice:** **Gemini 3 Flash** natively understands audio in multiple languages with high fidelity.
