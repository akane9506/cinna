import { bot } from './core/bot.js';

console.log('Cinna is starting (Long Polling)...');

bot.launch()
  .then(() => {
    console.log('Bot is live!');
  })
  .catch((err) => {
    console.error('Failed to launch bot:', err);
    process.exit(1);
  });
