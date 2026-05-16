import { describe, it, expect, mock, beforeEach } from "bun:test";

// --- Mocks ---

const mockReply = mock(async () => {});
const mockPersistentChatAction = mock(
  async (action: string, cb: () => Promise<void>) => {
    await cb();
  },
);
const mockBrainResponse = {
  intent: "OTHER",
  language: "en",
  reply: "mock response",
};
const mockGenerateCompletion = mock(async () => mockBrainResponse);
const mockDispatchIntent = mock(async (ctx: any, response: any) => {
  await ctx.reply(response.reply);
});

// Mock dependencies
mock.module("./brain", () => ({
  generateCompletion: mockGenerateCompletion,
}));

mock.module("./dispatcher", () => ({
  dispatchIntent: mockDispatchIntent,
}));

mock.module("./logger", () => ({
  logger: { error: mock() },
}));

mock.module("./config", () => ({
  config: {
    TELEGRAM_BOT_TOKEN: "test-token",
    ALLOWED_USERS: [123],
  },
}));

// Use dynamic import to ensure mocks are applied
const { handleTextMessage } = await import("./bot");

describe("bot.ts - handleTextMessage", () => {
  beforeEach(() => {
    mockReply.mockClear();
    mockPersistentChatAction.mockClear();
    mockGenerateCompletion.mockClear();
  });

  it("should use persistentChatAction and reply with generated completion", async () => {
    const mockCtx = {
      has: mock((filter: any) => {
        if (typeof filter === "function") {
          return filter({ message: { text: "hello" } });
        }
        return filter === "text";
      }),
      message: { text: "hello", from: { id: 123 } },
      chat: { id: 456 },
      persistentChatAction: mockPersistentChatAction,
      reply: mockReply,
    } as any;

    await handleTextMessage(mockCtx);

    expect(mockPersistentChatAction).toHaveBeenCalledWith(
      "typing",
      expect.any(Function),
    );
    expect(mockGenerateCompletion).toHaveBeenCalledWith("hello", "456");
    expect(mockReply).toHaveBeenCalledWith("mock response");
  });

  it("should handle errors and reply with error message", async () => {
    mockGenerateCompletion.mockImplementationOnce(async () => {
      throw new Error("Generation failed");
    });

    const mockCtx = {
      has: mock((filter: any) => {
        if (typeof filter === "function") {
          return filter({ message: { text: "hello" } });
        }
        return filter === "text";
      }),
      message: { text: "hello" },
      chat: { id: 456 },
      persistentChatAction: mockPersistentChatAction,
      reply: mockReply,
    } as any;

    await handleTextMessage(mockCtx);

    expect(mockReply).toHaveBeenCalledWith(
      "Sorry, I am having trouble thinking right now.",
    );
  });

  it("should do nothing if message is not text", async () => {
    const mockCtx = {
      has: (filter: string) => false,
    } as any;

    await handleTextMessage(mockCtx);

    expect(mockPersistentChatAction).not.toHaveBeenCalled();
    expect(mockReply).not.toHaveBeenCalled();
  });
});
