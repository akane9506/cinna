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
- **`@google/generative-ai`**: Google's official SDK for Gemini 1.5.
- **`dotenv`**: Loads environment variables from `.env`.
- **`zod`**: TypeScript-first schema validation for environment variables.
- **`express`**: Lightweight web server for Webhook support on Cloud Run.
- **`axios`**: To download voice files from Telegram's file server.
- **`fluent-ffmpeg`**: (Optional) For audio manipulation if needed in future.

### Development Dependencies
- **`typescript`**: Static typing for the codebase.
- **`@types/node`**: Type definitions for Node.js.
- **`@types/express`**: Type definitions for Express.
- **`ts-node`**: To run TypeScript files directly in development.
- **`nodemon`**: Monitor for changes and restart the server automatically.
- **`vitest`**: Modern and fast unit testing framework.
- **`rimraf`**: To clean the `dist/` directory before builds.

---

## 3. Testing Strategy

### Unit Testing (Automated)
We will use **Vitest** for isolated logic testing.
- **Core Brain:** Mock Gemini API responses to verify that different text/voice inputs trigger the correct function calls (e.g., `manage_grocery_list`).
- **Grocery Module:** Test the `service.ts` logic for adding, removing, and listing items in memory to ensure data integrity.
- **Config Validation:** Verify that the Zod schema correctly identifies missing or invalid environment variables.

### Manual Testing (Integration)
Once the bot is live (locally via Long Polling), the following scenarios must be verified:
- **Text Entry:** 
    - Input: "Add 2 apples and milk" -> Expected: Grocery list updated, response in English.
    - Input: "Ajoute du pain" -> Expected: Grocery list updated, response in French.
- **Voice Entry:** 
    - Send voice note in English: "I need some eggs." -> Expected: Gemini transcribes and adds to list.
    - Send voice note in Cantonese: "買多支牛奶" -> Expected: Gemini transcribes and adds to list in Cantonese.
- **Intent Routing:** 
    - Input: "Hello, who are you?" -> Expected: General assistant response (no grocery tool triggered).
- **Edge Cases:** 
    - Empty voice message.
    - Large lists (10+ items at once).

---

## 4. Granular Implementation Roadmap

### Phase 1: Infrastructure & Basic Connectivity
*   **Step 1.1: Project Initialization**
    *   Initialize `npm`, install `telegraf`, `dotenv`, and `typescript`.
    *   *Verification:* Run `npx tsc` to ensure the environment is correctly set up.
*   **Step 1.2: Basic Bot "Heartbeat"**
    *   Create a minimal `src/index.ts` that listens for text and responds with "Received!".
    *   *Verification:* Send any text to the bot on Telegram and get the "Received!" reply.
*   **Step 1.3: Audio Reception Verification**
    *   Update the bot to detect voice messages and reply with "Voice message received!".
    *   *Verification:* Send a voice note; verify the bot recognizes it as audio.

### Phase 2: Core Brain (Gemini Integration)
*   **Step 2.1: Simple Gemini Text Completion**
    *   Connect the Gemini API. Send a hardcoded text prompt to Gemini and print the result to the console.
    *   *Verification:* Check console logs for a valid response from Gemini.
*   **Step 2.2: Text Intent Routing**
    *   Implement "The Brain" logic. Tell Gemini to detect if a text message is about "groceries" or "other".
    *   *Verification:* Send "I need milk" (Bot logs: `Intent: Grocery`) and "Who are you?" (Bot logs: `Intent: Other`).
*   **Step 2.3: Audio-to-Brain Link**
    *   Implement the `audio.ts` service to download the Telegram OGG file and send it to Gemini.
    *   *Verification:* Send a voice note "Buy bread". Verify Gemini returns the transcription and correctly identifies the "Grocery" intent in the logs.

### Phase 3: The Grocery Module
*   **Step 3.1: In-Memory List Logic**
    *   Create the `GroceryService` that can add/remove items from a simple array.
    *   *Verification:* Run a small Vitest test to add "Apples" and check if the list contains "Apples".
*   **Step 3.2: Tool Integration (Function Calling)**
    *   Connect Gemini's function call (`manage_grocery_list`) to the `GroceryService`.
    *   *Verification:* Send "Add eggs". Bot should reply with "Added eggs to your list. Current list: eggs."
*   **Step 3.3: Multi-Language Loop**
    *   Refine the system prompt to ensure the final response is in the user's input language.
    *   *Verification:* Send voice note in French "Ajoute du lait". Bot should reply in French confirming the addition.

### Phase 4: Deployment & Webhooks
*   **Step 4.1: Hono Webhook Server**
    *   Switch from Long Polling to a **Hono** server handling Telegram updates.
    *   *Verification:* Use `ngrok` (locally) to verify the bot still works via the webhook URL.
*   **Step 4.2: GCP Containerization**
    *   Create the `Dockerfile` and build the image locally.
    *   *Verification:* Run the container locally and verify it starts without errors.
*   **Step 4.3: Cloud Run Deployment**
    *   Deploy to GCP and set environment variables.
    *   *Verification:* Final test on the live Telegram bot.

## 4. Multi-Language Strategy
- **Prompting:** The System Instruction will state: *"You are a helpful assistant. Detect the user's language. If a tool is called, process the data. Always output your final text response in the detected language."*
- **Voice:** **Gemini 3 Flash** natively understands audio in multiple languages with high fidelity.
