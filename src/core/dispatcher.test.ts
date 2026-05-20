import { beforeEach, describe, expect, it, mock } from "bun:test";

const mockHandleGroceryIntent = mock(async () => {});
const mockLoggerInfo = mock();
const mockLoggerError = mock();

mock.module("../modules/grocery/handler", () => ({
  handleGroceryIntent: mockHandleGroceryIntent,
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
    mockHandleGroceryIntent.mockClear();
    mockLoggerInfo.mockClear();
    mockLoggerError.mockClear();
  });

  it("routes grocery responses to the grocery handler", async () => {
    const ctx = { reply: mock(async () => {}) } as any;
    const response = {
      intent: "GROCERY",
      language: "en",
      reply: "AI reply should not be used.",
      action: "add",
      item: "milk",
      shop: "Costco",
    } as const;

    await dispatchIntent(ctx, response, "add milk at Costco");

    expect(mockHandleGroceryIntent).toHaveBeenCalledWith(
      ctx,
      response,
      "add milk at Costco",
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
    expect(mockHandleGroceryIntent).not.toHaveBeenCalled();
  });
});
