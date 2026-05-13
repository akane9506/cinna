import { Hono } from 'hono';
import { bot } from './core/bot';
import { config } from './core/config';

const app = new Hono();

app.get('/', (c) => c.text('Cinna is alive!'));

console.log('Cinna is starting...');

// Launch bot (Long Polling)
bot.launch()
  .then(() => {
    console.log('Bot is live (Long Polling)!');
  })
  .catch((err) => {
    console.error('Failed to launch bot:', err);
  });

export default {
  port: config.PORT || 3000,
  fetch: app.fetch,
};
