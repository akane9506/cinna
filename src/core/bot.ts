import { Context, Telegraf } from "telegraf";
import { message } from "telegraf/filters";
import { config } from "./config";
import { generateCompletion } from "./brain";
import { logger } from "./logger";
import { whitelistMiddleware } from "./middleware";

export const bot = new Telegraf(config.TELEGRAM_BOT_TOKEN);

// Apply whitelist middleware globally
bot.use(whitelistMiddleware);

export const handleTextMessage = async (ctx: Context) => {
  if (ctx.has(message("text"))) {
    try {
      const chatId = ctx.chat?.id.toString() || "unknown";
      await ctx.persistentChatAction("typing", async () => {
        const response = await generateCompletion(ctx.message.text, chatId);
        await ctx.reply(response);
      });
    } catch (error) {
      logger.error({ error, text: ctx.message.text }, "Bot Error (Text)");
      await ctx.reply("Sorry, I am having trouble thinking right now.");
    }
  }
};

export const handleVoiceMessage = async (ctx: Context) => {
  if (ctx.has(message("voice"))) {
    await ctx.reply("Voice received!");
  }
};

bot.on(message("text"), handleTextMessage);
bot.on(message("voice"), handleVoiceMessage);
