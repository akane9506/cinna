# Cinna Project: Status & Instructions

This file serves as the source of truth for Cinna's development progress and architectural standards. **Always check this file at the start of a session.**

## Project Overview

Cinna is a modular Telegram assistant using **Gemini 3 Flash** for intent routing and voice processing, with **Hono** for the webhook server.

## 🟢 Implementation Checklist

### Phase 1: Infrastructure & Basic Connectivity

- [x] **Step 1.1: Project Initialization** (bun, TS, bun:test, telegraf, hono)
  - [x] **Test 1.1:** `bun test` succeeds.
- [x] **Step 1.2: CI/CD Setup** (GitHub Actions with Bun)
  - [x] **Test 1.2:** Push to main triggers lint and test workflow.
- [x] **Step 1.3: Basic Bot "Heartbeat"** (Verify text reception)
  - [x] **Test 1.3:** Unit test for bot response logic; manual text verification.
- [x] **Step 1.4: Audio Reception Verification** (Verify voice detection)
  - [x] **Test 1.4:** Manual voice note verification (bot replies "Voice received").

### Phase 2: Core Brain (Gemini Integration)

- [x] **Step 2.1: Simple Gemini Text Completion** (Verify API connectivity)
  - [x] **Test 2.1:** Unit tests for `brain.ts` with mocked Gemini responses.
- [x] **Step 2.2: Text Intent Routing** (Grocery vs Other)
  - [x] **Test 2.2:** Unit tests for `brain.ts` with schema-driven structured output.

### Phase 3: Persistence (Firestore) & Modules

- [ ] **Step 3.1: Firebase Initialization** (Database setup)
  - [ ] **Test 3.1:** `firebase-admin` successfully initializes and connects.
- [ ] **Step 3.2: Persistent Grocery Module** (Grocery logic with Firestore)
  - [ ] **Test 3.2:** Items added via bot are saved in Firestore.
- [ ] **Step 3.3: Audio-to-Brain Link** (OGG download -> Gemini)
  - [ ] **Test 3.3:** Voice notes are transcribed and routed correctly.
- [ ] **Step 3.3: Feedback & Bug Tracker** (Meta-module for bot improvements)
  - [ ] **Test 3.3:** Verify Cinna can save bug reports and feature ideas via function calling.
- [ ] **Step 3.4: Multi-Language Loop** (Verify language detection & response)
  - [ ] **Test 3.4:** Unit tests for language detection prompts; manual multi-language voice tests.

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
6. **Modern Standards:** Always prioritize modern, non-legacy implementations and APIs.
7. **SDK Mandate:** ALWAYS use the new `@google/genai` library. NEVER use or refer to the deprecated `@google/generative-ai` package. Since AI models may have outdated internal knowledge of this new SDK, always refer to the latest documentation or existing codebase patterns for correct implementation.

## 📦 Core Stack

- **Runtime:** Bun
- **Framework:** Telegraf (Telegram), Hono (Web)
- **AI:** Google Generative AI (`@google/genai`, Gemini 3 Flash)
- **Database:** Firestore (Persistence)
- **Validation:** Zod (Config)
- **Testing:** bun:test
- **CI/CD:** GitHub Actions (Setup-Bun)
- **Deployment:** GCP Cloud Run (Docker)
