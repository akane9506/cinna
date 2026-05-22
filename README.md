# Cinna: Modular Telegram Assistant

Cinna is a modular Telegram assistant powered by Gemini 3 Flash and Hono.

## How Shopping Requests Flow

Shopping commands use a staged flow: the core Brain first classifies the Telegram
message, the shopping planner runs a second structured LLM pass to create one or
more DB commands, the repository saves them to Firestore, then Cinna generates a
persona reply from the persisted result.

## Local Prompt Files

Prompt files are local and git-ignored. Add or edit them here:

- `src/core/persona.md`: Cinna's main persona.
- `src/modules/shopping/prompts/planner.instruction.md`: shopping command planner.
- `src/modules/shopping/prompts/reply.instruction.md`: shopping success reply.

If a prompt file is missing, the app uses a small fallback instruction.

## 🚀 Local Setup

### 1. Create your Telegram Bot

1. Open Telegram and search for [@BotFather](https://t.me/botfather).
2. Use the `/newbot` command to create a new bot and get your **API Token**.
3. Search for your bot's username and click **Start**.

### 2. Configure Environment Variables

1. Create a `.env` file based on `.env.example`:
   ```bash
   cp .env.example .env
   ```
2. Fill in your credentials:
   - `TELEGRAM_BOT_TOKEN`: Your BotFather token.
   - `GEMINI_API_KEY`: Your Google AI Studio API Key.
   - `FIREBASE_PROJECT_ID`: Firebase/GCP project ID.
   - `FIRESTORE_DATABASE_ID`: Firestore database ID, such as `development` locally or `(default)` in production.
   - `FIREBASE_SERVICE_ACCOUNT_PATH`: Optional local path to a Firebase service account JSON file, such as `.firebase/firebase-service-account-dev.json`.
   - `ALLOWED_USERS`: Comma-separated Telegram User IDs allowed to use the bot.
   - `PORT`: 3000 (default).
   - `NODE_ENV`: development.

   Keep service account JSON files out of git. The repo ignores `.firebase/` for local credentials.

### 3. Install & Run

```bash
# Install dependencies
bun install

# Run in development mode (hot-reloading)
bun dev
```

---

## 🛠 Development Commands

- `bun dev`: Start the bot in development mode with hot-reloading.
- `bun start`: Start the bot in production mode.
- `bun test:isolate`: Run the test suite with isolation.
- `bun test:coverage`: Run tests and generate coverage report.
- `bun lint`: Run ESLint check.
