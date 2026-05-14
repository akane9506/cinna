import { describe, it, expect, mock, beforeEach } from "bun:test";
import { Context } from "telegraf";

/**
 * 1. Setup Mocks
 */
const mockGenerateCompletion = mock();

mock.module("./config", () => ({
  config: {
    TELEGRAM_BOT_TOKEN: "dummy_token",
    ALLOWED_USERS: [12345],
  },
}));

mock.module("./brain", () => ({
  generateCompletion: mockGenerateCompletion,
}));

/**
 * 2. Import SUT
 */
const { handleTextMessage, handleVoiceMessage } = await import("./bot");
const { whitelistMiddleware } = await import("./middleware");

describe("Bot Core Logic", () => {
  beforeEach(() => {
    mockGenerateCompletion.mockClear();
  });

  describe("whitelistMiddleware", () => {
    it("should call next() for authorized users", async () => {
      const next = mock();
      const ctx = {
        from: { id: 12345 },
        chat: {},
        reply: mock(),
      } as unknown as Context;
      await whitelistMiddleware(ctx, next);
      expect(next).toHaveBeenCalled();
      expect(ctx.reply).not.toHaveBeenCalled();
    });

    it("should reject unauthorized users and not call next()", async () => {
      const next = mock();
      const ctx = {
        from: { id: 99999 },
        chat: {},
        reply: mock(),
      } as unknown as Context;
      await whitelistMiddleware(ctx, next);
      expect(next).not.toHaveBeenCalled();
      expect(ctx.reply).toHaveBeenCalledWith(
        "Sorry, I am only a personal agent that is not publicly available.",
      );
    });
  });

  describe("Handlers (assuming whitelist passed)", () => {
    it("should reply with a Gemini completion for text", async () => {
      mockGenerateCompletion.mockResolvedValue("Mocked Response");
      const ctx = {
        has: mock(() => true),
        reply: mock(),
        message: { text: "Hello" },
      } as unknown as Context;
      await handleTextMessage(ctx);
      expect(ctx.reply).toHaveBeenCalledWith("Mocked Response");
    });

    it("should handle voice messages", async () => {
      const ctx = {
        has: mock(() => true),
        reply: mock(),
      } as unknown as Context;
      await handleVoiceMessage(ctx);
      expect(ctx.reply).toHaveBeenCalledWith("Voice received!");
    });
  });
});
