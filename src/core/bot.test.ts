import { describe, it, expect, mock } from "bun:test";
import { Context } from "telegraf";

mock.module("./config", () => ({
  config: {
    TELEGRAM_BOT_TOKEN: "dummy_token",
    GEMINI_API_KEY: "dummy_key",
    PORT: "3000",
    NODE_ENV: "test",
  },
}));

// Import bot handler AFTER mocking config
const { handleTextMessage, handleVoiceMessage } = await import("./bot");

describe("Bot Handlers", () => {
  it('should reply with "Received!" when a text message is received', async () => {
    const ctx = {
      has: mock(() => true),
      reply: mock(async () => ({})),
    } as unknown as Context;

    await handleTextMessage(ctx);

    expect(ctx.has).toHaveBeenCalled();
    expect(ctx.reply).toHaveBeenCalledWith("Received!");
  });

  it('should reply with "Voice received!" when a voice message is received', async () => {
    const ctx = {
      has: mock(() => true),
      reply: mock(async () => ({})),
    } as unknown as Context;

    await handleVoiceMessage(ctx);

    expect(ctx.has).toHaveBeenCalled();
    expect(ctx.reply).toHaveBeenCalledWith("Voice received!");
  });
});
