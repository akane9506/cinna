import { describe, it, expect, mock, beforeEach } from "bun:test";

// --- Mocks ---

const mockSendMessage = mock(async () => ({ text: "mock response" }));
const mockGetHistory = mock((): any[] => []);

const mockChatsCreate = mock((..._args: any[]): any => {
  return {
    sendMessage: mockSendMessage,
    getHistory: mockGetHistory,
  };
});

// Mock @google/genai BEFORE importing brain.ts
mock.module("@google/genai", () => {
  return {
    GoogleGenAI: class {
      chats = {
        create: mockChatsCreate,
      };
    },
    Chat: class {}, // Mock Chat class if used for typing/instanceof
  };
});

// Mock other dependencies
mock.module("./config", () => ({
  config: {
    GEMINI_API_KEY: "test-key",
    GEMINI_MODEL: "gemini-test",
    MAX_SESSIONS: 2,
    MAX_HISTORY_MESSAGES: 4,
  },
}));

mock.module("./logger", () => ({
  logger: { error: mock() },
}));

mock.module("./persona", () => ({
  getPersona: mock(async () => "test persona"),
}));

// Use dynamic import to ensure mocks are applied
const { generateCompletion } = await import("./brain");

describe("brain.ts", () => {
  beforeEach(() => {
    mockChatsCreate.mockClear();
    mockSendMessage.mockClear();
    mockGetHistory.mockClear();
  });

  it("should generate completion and create a new session if none exists", async () => {
    const response = await generateCompletion("hello", "chat-new");
    expect(response).toBe("mock response");
    expect(mockChatsCreate).toHaveBeenCalled();
  });

  it("should reuse an existing session", async () => {
    await generateCompletion("first", "chat-reuse");
    const countAfterFirst = mockChatsCreate.mock.calls.length;

    await generateCompletion("second", "chat-reuse");
    expect(mockChatsCreate.mock.calls.length).toBe(countAfterFirst);
    expect(mockSendMessage).toHaveBeenCalledTimes(2);
  });

  it("should evict old sessions when limit is reached", async () => {
    // MAX_SESSIONS is 2. Clear current sessions by filling them.
    await generateCompletion("msg", "s1");
    await generateCompletion("msg", "s2");
    const countBeforeEviction = mockChatsCreate.mock.calls.length;

    // This should evict s1 (the oldest)
    await generateCompletion("msg", "s3");
    expect(mockChatsCreate.mock.calls.length).toBe(countBeforeEviction + 1);

    // Calling s1 again should create a new session
    await generateCompletion("msg", "s1");
    expect(mockChatsCreate.mock.calls.length).toBe(countBeforeEviction + 2);
  });

  it("should truncate history when it exceeds the limit", async () => {
    // MAX_HISTORY_MESSAGES is 4
    mockGetHistory.mockReturnValue([
      { role: "user", parts: [{ text: "h1" }] },
      { role: "model", parts: [{ text: "r1" }] },
      { role: "user", parts: [{ text: "h2" }] },
      { role: "model", parts: [{ text: "r2" }] },
      { role: "user", parts: [{ text: "h3" }] },
    ]);

    const createCallCountBefore = mockChatsCreate.mock.calls.length;
    await generateCompletion("trigger truncation", "chat-trunc");

    // Should have re-created the chat session
    expect(mockChatsCreate.mock.calls.length).toBeGreaterThan(
      createCallCountBefore,
    );

    const truncationCall = mockChatsCreate.mock.calls.find(
      (call) => (call[0] as any).history !== undefined,
    );

    if (!truncationCall) {
      throw new Error("Truncation call not found");
    }
    expect(truncationCall.length).toBeGreaterThan(0);
    const truncationParams = truncationCall[0] as any;
    expect(truncationParams.history.length).toBeLessThanOrEqual(4);
    expect(truncationParams.history[0].role).toBe("user");
  });

  it("should handle Gemini API errors gracefully", async () => {
    mockSendMessage.mockImplementationOnce(async () => {
      throw new Error("API Error");
    });

    try {
      await generateCompletion("error prompt", "chat-err");
      // Should not reach here
      expect(true).toBe(false);
    } catch (error: any) {
      expect(error.message).toContain(
        "Failed to generate completion from Gemini",
      );
    }
  });
});
