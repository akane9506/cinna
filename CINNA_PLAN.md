# Cinna: Implementation Roadmap

Cinna is a modular, AI-powered Telegram assistant designed to handle voice and text inputs for various tasks, starting with grocery management. Powered by **Gemini 3 Flash**, it leverages frontier reasoning and agentic capabilities for seamless interaction.

## 1. Modular Architecture
The system follows a "Brain & Modules" pattern:
- **Core Brain (Gemini 3 Flash):** Acts as the intent router and transcription engine, utilizing its pro-grade reasoning to handle complex requests.
- **Feature Modules:** Independent logic for specific tasks (Grocery, Reminders, etc.).
- **Response Engine:** Ensures responses match the user's input language.

### Project Structure
```text
omnibot/
├── .github/workflows/
│   ├── ci.yml                # CI: Lint, Build, Test
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
│   ├── index.ts              # Entry point
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
- **`telegraf`**: Modern Telegram Bot Framework for Node.js.
- **`@google/generative-ai`**: Google's official SDK for Gemini.
- **`dotenv`**: Loads environment variables from `.env`.
- **`zod`**: TypeScript-first schema validation for environment variables.
- **`hono`**: Ultrafast web framework for the edge and Cloud Run.
- **`@hono/node-server`**: Adapter for Hono on Node.js.
- **`axios`**: To download voice files from Telegram's file server.

### Development Dependencies
- **`typescript`**: Static typing for the codebase.
- **`@types/node`**: Type definitions for Node.js.
- **`ts-node`**: To run TypeScript files directly in development.
- **`nodemon`**: Monitor for changes and restart the server automatically.
- **`vitest`**: Modern and fast unit testing framework.
- **`rimraf`**: To clean the `dist/` directory before builds.
- **`eslint`**: Linter to maintain code quality.

---

## 3. CI/CD Strategy (GitHub Actions)

### CI (Continuous Integration)
Triggered on every pull request and push to `main`.
- **Lint:** Run `eslint` to ensure style consistency.
- **Build:** Run `npm run build` to verify TypeScript compilation.
- **Test:** Run `npm run test` (Vitest) to ensure no regressions.

### CD (Continuous Deployment)
Triggered on merge to `main` or specific tags.
- **Build & Push:** Build Docker image and push to Google Artifact Registry.
- **Deploy:** Deploy the new image to GCP Cloud Run.

---

## 4. Granular Implementation Roadmap

### Phase 1: Infrastructure & Basic Connectivity
*   **Step 1.1: Project Initialization**
    *   Initialize `npm`, install `telegraf`, `hono`, `dotenv`, `typescript`, `vitest`.
    *   *Verification:* `npm run build` succeeds; `vitest` runs successfully.
*   **Step 1.2: CI/CD Setup**
    *   Create `.github/workflows/ci.yml` for linting, building, and testing.
    *   *Verification:* Push to GitHub and see the green checkmark on the commit.
*   **Step 1.3: Basic Bot "Heartbeat"**
    *   Create `src/index.ts` with a simple Telegraf handler.
    *   *Verification:* 
        - [ ] Manual: Send text to bot, get reply.
        - [ ] Test: Unit test the message handler logic.
*   **Step 1.4: Audio Reception Verification**
    *   Update bot to handle `voice` updates.
    *   *Verification:* 
        - [ ] Manual: Send voice note, bot replies "Voice received".

### Phase 2: Core Brain (Gemini Integration)
*   **Step 2.1: Simple Gemini Text Completion**
    *   Integration with `@google/generative-ai`.
    *   *Verification:* 
        - [ ] Test: Mock API call to Gemini and verify handling of the response.
*   **Step 2.2: Text Intent Routing**
    *   Implement `Brain` service for intent classification.
    *   *Verification:* 
        - [ ] Test: Unit tests with various inputs (Grocery vs Other) and mocked Gemini output.
*   **Step 2.3: Audio-to-Brain Link**
    *   Implement `audio.ts` to fetch and pass audio bytes to Gemini.
    *   *Verification:* 
        - [ ] Test: Integration test simulating audio download and Gemini processing.

### Phase 3: The Grocery Module
*   **Step 3.1: In-Memory List Logic**
    *   `GroceryService` for state management.
    *   *Verification:* 
        - [ ] Test: 100% unit test coverage for `add`, `remove`, `list` methods.
*   **Step 3.2: Tool Integration (Function Calling)**
    *   Configure Gemini tools for grocery management.
    *   *Verification:* 
        - [ ] Test: End-to-end integration test (Mock Gemini -> Tool Call -> Service Update).
*   **Step 3.3: Multi-Language Loop**
    *   Refine system prompts for language parity.
    *   *Verification:* 
        - [ ] Manual: Multi-language voice note testing (FR, EN, ES, etc.).

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
