# Cinna Project: Status & Instructions

This file serves as the source of truth for Cinna's development progress and architectural standards. **Always check this file at the start of a session.**

## Project Overview
Cinna is a modular Telegram assistant using **Gemini 3 Flash** for intent routing and voice processing, with **Hono** for the webhook server.

## 🟢 Implementation Checklist

### Phase 1: Infrastructure & Basic Connectivity
- [ ] **Step 1.1: Project Initialization** (npm, TS, telegraf, hono)
- [ ] **Step 1.2: Basic Bot "Heartbeat"** (Verify text reception)
- [ ] **Step 1.3: Audio Reception Verification** (Verify voice detection)

### Phase 2: Core Brain (Gemini Integration)
- [ ] **Step 2.1: Simple Gemini Text Completion** (Verify API connectivity)
- [ ] **Step 2.2: Text Intent Routing** (Grocery vs Other)
- [ ] **Step 2.3: Audio-to-Brain Link** (OGG download -> Gemini)

### Phase 3: The Grocery Module
- [ ] **Step 3.1: In-Memory List Logic** (Unit tested service)
- [ ] **Step 3.2: Tool Integration** (Gemini Function Calling)
- [ ] **Step 3.3: Multi-Language Loop** (Verify language detection & response)

### Phase 4: Deployment & Webhooks
- [ ] **Step 4.1: Hono Webhook Server** (Local verification via ngrok)
- [ ] **Step 4.2: GCP Containerization** (Dockerfile verification)
- [ ] **Step 4.3: Cloud Run Deployment** (Final live test)

---

## 🛠 Architectural Mandates
1. **Modular First:** All features MUST live in `src/modules/{feature_name}`.
2. **Brain-Routed:** All user inputs MUST go through `src/core/brain.ts` for intent classification.
3. **Multi-Language:** The `Brain` MUST detect the input language and the bot MUST respond in that same language.
4. **Verified Steps:** Never move to the next checkbox until the current one is verified as per `CINNA_PLAN.md`.

## 📦 Core Stack
- **Framework:** Telegraf (Telegram), Hono (Web)
- **AI:** Google Generative AI (Gemini 3 Flash)
- **Validation:** Zod (Config)
- **Testing:** Vitest
- **Deployment:** GCP Cloud Run (Docker)
