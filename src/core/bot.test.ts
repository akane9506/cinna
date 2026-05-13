import { describe, it, expect, vi } from 'vitest';
import { Context } from 'telegraf';

vi.mock('./config.js', () => ({
  config: {
    TELEGRAM_BOT_TOKEN: 'dummy_token',
    GEMINI_API_KEY: 'dummy_key',
    PORT: '3000',
    NODE_ENV: 'test',
  },
}));

// Import bot handler AFTER mocking config
const { handleTextMessage, handleVoiceMessage } = await import('./bot.js');

describe('Bot Handlers', () => {
  it('should reply with "Received!" when a text message is received', async () => {
    const ctx = {
      has: vi.fn().mockReturnValue(true),
      reply: vi.fn().mockResolvedValue({}),
    } as unknown as Context;

    await handleTextMessage(ctx);

    expect(ctx.has).toHaveBeenCalled();
    expect(ctx.reply).toHaveBeenCalledWith('Received!');
  });

  it('should reply with "Voice received!" when a voice message is received', async () => {
    const ctx = {
      has: vi.fn().mockReturnValue(true),
      reply: vi.fn().mockResolvedValue({}),
    } as unknown as Context;

    await handleVoiceMessage(ctx);

    expect(ctx.has).toHaveBeenCalled();
    expect(ctx.reply).toHaveBeenCalledWith('Voice received!');
  });
});
