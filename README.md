# Cinna: Modular Telegram Assistant

Cinna is a modular Telegram assistant powered by Gemini 3 Flash and Hono.

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
   - `ALLOWED_USERS`: Comma-separated Telegram User IDs allowed to use the bot.
   - `PORT`: 3000 (default).
   - `NODE_ENV`: development.

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
