import { Context, Telegraf } from 'telegraf';
import { message } from 'telegraf/filters';
import { config } from './config';
import { generateCompletion } from './brain';
import { logger } from './logger';
import { whitelistMiddleware } from './middleware';

export const bot = new Telegraf(config.TELEGRAM_BOT_TOKEN);

// Apply whitelist middleware globally
bot.use(whitelistMiddleware);

export const handleTextMessage = async (ctx: Context) => {
  if (ctx.has(message('text'))) {
    try {
      const response = await generateCompletion(ctx.message.text);
      await ctx.reply(response);
    } catch (error) {
      logger.error({ error, text: ctx.message.text }, 'Bot Error (Text)');
      await ctx.reply('Sorry, I am having trouble thinking right now.');
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

process.once("SIGINT", () => bot.stop("SIGINT"));
process.once("SIGTERM", () => bot.stop("SIGTERM"));
