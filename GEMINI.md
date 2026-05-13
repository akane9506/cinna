# Cinna Project: Status & Instructions

This file serves as the source of truth for Cinna's development progress and architectural standards. **Always check this file at the start of a session.**

## Project Overview
Cinna is a modular Telegram assistant using **Gemini 3 Flash** for intent routing and voice processing, with **Hono** for the webhook server.

## 🟢 Implementation Checklist

### Phase 1: Infrastructure & Basic Connectivity
- [x] **Step 1.1: Project Initialization** (npm, TS, vitest, telegraf, hono)
    - [x] **Test 1.1:** `npm run build` succeeds; `vitest` runs with 0 tests.
- [x] **Step 1.2: CI/CD Setup** (GitHub Actions)
    - [x] **Test 1.2:** Push to main triggers lint, build, and test workflow.
- [ ] **Step 1.3: Basic Bot "Heartbeat"** (Verify text reception)
    - [ ] **Test 1.3:** Unit test for bot response logic; manual text verification.
- [ ] **Step 1.4: Audio Reception Verification** (Verify voice detection)
    - [ ] **Test 1.4:** Manual voice note verification (bot replies "Voice received").

### Phase 2: Core Brain (Gemini Integration)
- [ ] **Step 2.1: Simple Gemini Text Completion** (Verify API connectivity)
    - [ ] **Test 2.1:** Integration test for Gemini API connectivity.
- [ ] **Step 2.2: Text Intent Routing** (Grocery vs Other)
    - [ ] **Test 2.2:** Unit tests for `brain.ts` with mocked Gemini responses for various intents.
- [ ] **Step 2.3: Audio-to-Brain Link** (OGG download -> Gemini)
    - [ ] **Test 2.3:** Integration test for audio processing pipeline.

### Phase 3: The Grocery Module
- [ ] **Step 3.1: In-Memory List Logic** (Unit tested service)
    - [ ] **Test 3.1:** 100% coverage for `GroceryService` unit tests.
- [ ] **Step 3.2: Tool Integration** (Gemini Function Calling)
    - [ ] **Test 3.2:** Integration test for Gemini -> GroceryService link.
- [ ] **Step 3.3: Multi-Language Loop** (Verify language detection & response)
    - [ ] **Test 3.3:** Unit tests for language detection prompts; manual multi-language voice tests.

### Phase 4: Deployment & Webhooks
- [ ] **Step 4.1: Hono Webhook Server** (Local verification via ngrok)
    - [ ] **Test 4.1:** Integration test for Hono webhook endpoint.
- [ ] **Step 4.2: GCP Containerization** (Dockerfile verification)
    - [ ] **Test 4.2:** Local docker build and run succeeds.
- [ ] **Step 4.3: Cloud Run Deployment** (Final live test)
    - [ ] **Test 4.3:** GitHub Action auto-deploys to Cloud Run on tag/release.

---

## 🛠 Architectural Mandates
1. **Modular First:** All features MUST live in `src/modules/{feature_name}`.
2. **Brain-Routed:** All user inputs MUST go through `src/core/brain.ts` for intent classification.
3. **Multi-Language:** The `Brain` MUST detect the input language and the bot MUST respond in that same language.
4. **Verified Steps:** Never move to the next checkbox until the current one is verified as per `CINNA_PLAN.md`.
5. **CI/CD Driven:** No code is merged without passing the GitHub Actions CI pipeline.

## 📦 Core Stack
- **Framework:** Telegraf (Telegram), Hono (Web)
- **AI:** Google Generative AI (Gemini 3 Flash)
- **Validation:** Zod (Config)
- **Testing:** Vitest
- **CI/CD:** GitHub Actions
- **Deployment:** GCP Cloud Run (Docker)
