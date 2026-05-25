import { beforeEach, describe, expect, it, mock } from "bun:test";

const mockHandleShoppingIntent = mock(async () => {});
const mockLoggerInfo = mock();
const mockLoggerError = mock();

mock.module("../modules/shopping/handler", () => ({
  handleShoppingIntent: mockHandleShoppingIntent,
}));

mock.module("./logger", () => ({
  logger: {
    info: mockLoggerInfo,
    error: mockLoggerError,
  },
}));

const { dispatchIntent } = await import("./dispatcher");

describe("dispatchIntent", () => {
  beforeEach(() => {
    mockHandleShoppingIntent.mockClear();
    mockLoggerInfo.mockClear();
    mockLoggerError.mockClear();
  });

  it("routes shopping responses to the shopping handler", async () => {
    const ctx = { reply: mock(async () => {}) } as any;
    const response = {
      intent: "SHOPPING",
      language: "en",
      reply: "AI reply should not be used.",
      action: "add",
      item: "milk",
    } as const;

    await dispatchIntent(ctx, response, "add milk at Costco");

    expect(mockHandleShoppingIntent).toHaveBeenCalledWith(
      ctx,
      response,
      "add milk at Costco",
    );
    expect(ctx.reply).not.toHaveBeenCalled();
  });

  it("routes shopping list requests to the shopping handler", async () => {
    const ctx = { reply: mock(async () => {}) } as any;
    const response = {
      intent: "SHOPPING",
      language: "en",
      reply: "AI reply should not be used.",
      action: "list",
    } as const;

    await dispatchIntent(ctx, response, "what is on my grocery list?");

    expect(mockHandleShoppingIntent).toHaveBeenCalledWith(
      ctx,
      response,
      "what is on my grocery list?",
    );
    expect(ctx.reply).not.toHaveBeenCalled();
  });

  it("continues to reply with brain text for other responses", async () => {
    const reply = mock(async () => {});
    const response = {
      intent: "OTHER",
      language: "en",
      reply: "Hello.",
    } as const;

    await dispatchIntent({ reply } as any, response);

    expect(reply).toHaveBeenCalledWith("Hello.");
    expect(mockHandleShoppingIntent).not.toHaveBeenCalled();
  });
});
