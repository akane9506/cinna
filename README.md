# Cinna: Modular Telegram Assistant

Cinna is a modular Telegram assistant powered by Gemini 3 Flash and Hono.

## 🚀 Local Setup & Manual Testing

To verify **Phase 1 (Infrastructure & Heartbeat)**, follow these steps:

### 1. Create your Telegram Bot
1. Open Telegram and search for [@BotFather](https://t.me/botfather).
2. Use the `/newbot` command to create a new bot and get your **API Token**.
3. **Finding your bot:** BotFather will provide a link (e.g., `t.me/YourBotName_bot`). Click it or search for your bot's username in the Telegram search bar.
4. Click **Start** in the chat with your bot.

> **Note:** Since the bot is currently in development, it will only respond when you are running the project locally on your machine.

### 2. Configure Environment Variables
1. Create a `.env` file in the root directory (based on `.env.example`):
   ```bash
   cp .env.example .env
   ```
2. Open `.env` and fill in your credentials:
   - `TELEGRAM_BOT_TOKEN`: The token you got from BotFather.
   - `GEMINI_API_KEY`: Your Google AI Studio API Key.
   - `PORT`: 3000 (default).
   - `NODE_ENV`: development.

### 3. Install & Run
```bash
# Install dependencies
npm install

# Run in development mode (Vite)
npm run dev
```

### 4. Verify Phase 1
Once the console says `Bot is live!`, go to your Telegram bot and send:
- **Text Message:** Send "Hello". The bot should reply with **"Received!"**.
- **Voice Message:** Send a short voice note. The bot should reply with **"Voice received!"**.

---

## 🛠 Development Commands
- `npm run dev`: Start the bot with Vite (hot-reloading).
- `npm run build`: Compile TypeScript to JavaScript in `dist/`.
- `npm run test`: Run the Vitest suite.
- `npm run lint`: Run ESLint check.
