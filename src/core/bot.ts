import { Context, Telegraf } from 'telegraf';
import { message } from 'telegraf/filters';
import { config } from './config';

export const bot = new Telegraf(config.TELEGRAM_BOT_TOKEN);

export const handleTextMessage = async (ctx: Context) => {
  if (ctx.has(message('text'))) {
    await ctx.reply('Received!');
  }
};

export const handleVoiceMessage = async (ctx: Context) => {
  if (ctx.has(message('voice'))) {
    await ctx.reply('Voice received!');
  }
};

bot.on(message('text'), handleTextMessage);
bot.on(message('voice'), handleVoiceMessage);

process.once('SIGINT', () => bot.stop('SIGINT'));
process.once('SIGTERM', () => bot.stop('SIGTERM'));
