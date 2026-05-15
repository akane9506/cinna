import { Hono } from "hono";
import { bot } from "./core/bot";
import { config } from "./core/config";

import { logger } from "./core/logger";

const app = new Hono();

app.get("/", (c) => c.text("Cinna is alive!"));

console.log("Cinna is starting...");

// Launch bot (Long Polling)
bot
  .launch()
  .then(() => {
    console.log("Bot is live (Long Polling)!");
  })
  .catch((err) => {
    console.error("Failed to launch bot:", err);
  });

const shutdown = (signal: string) => {
  logger.info(`Shutting down via ${signal}...`);
  bot.stop(signal);
};

process.once("SIGINT", () => shutdown("SIGINT"));
process.once("SIGTERM", () => shutdown("SIGTERM"));

export default {
  port: config.PORT || 3000,
  fetch: app.fetch,
};
