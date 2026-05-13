import { bot } from './core/bot.js';

console.log('Cinna is starting...');

// For production or traditional Node execution
if (process.env.NODE_ENV === 'production' || !process.env.VITE) {
  bot.launch()
    .then(() => {
      console.log('Bot is live (Long Polling)!');
    })
    .catch((err) => {
      console.error('Failed to launch bot:', err);
      process.exit(1);
    });
}

// For Vite development
export const viteNodeApp = bot;
