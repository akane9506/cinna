import { describe, it, expect, mock, beforeEach } from "bun:test";

// --- Mocks ---
const mockReply = mock(async () => {});
const mockNext = mock(async () => {});

mock.module("./config", () => ({
  config: {
    ALLOWED_USERS: [123, 456],
  },
}));

// Use dynamic import to ensure mocks are applied
const { whitelistMiddleware } = await import("./middleware");

describe("middleware.ts - whitelistMiddleware", () => {
  beforeEach(() => {
    mockReply.mockClear();
    mockNext.mockClear();
  });

  it("should call next() if user is in the whitelist", async () => {
    const mockCtx = {
      from: { id: 123 },
      chat: { id: 999 },
      reply: mockReply,
    } as any;

    await whitelistMiddleware(mockCtx, mockNext);

    expect(mockNext).toHaveBeenCalled();
    expect(mockReply).not.toHaveBeenCalled();
  });

  it("should reply with rejection and NOT call next() if user is NOT in the whitelist", async () => {
    const mockCtx = {
      from: { id: 789 }, // Not in [123, 456]
      chat: { id: 999 },
      reply: mockReply,
    } as any;

    await whitelistMiddleware(mockCtx, mockNext);

    expect(mockNext).not.toHaveBeenCalled();
    expect(mockReply).toHaveBeenCalledWith(
      "Sorry, I am only a personal agent that is not publicly available."
    );
  });

  it("should reply with rejection if userId is missing", async () => {
    const mockCtx = {
      from: {}, // No id
      chat: { id: 999 },
      reply: mockReply,
    } as any;

    await whitelistMiddleware(mockCtx, mockNext);

    expect(mockNext).not.toHaveBeenCalled();
    expect(mockReply).toHaveBeenCalled();
  });

  it("should not reply if it is not a chat interaction (safety check)", async () => {
    const mockCtx = {
      from: { id: 789 },
      // chat is missing
      reply: mockReply,
    } as any;

    await whitelistMiddleware(mockCtx, mockNext);

    expect(mockNext).not.toHaveBeenCalled();
    expect(mockReply).not.toHaveBeenCalled();
  });
});
