import { describe, it, expect, mock, beforeEach } from "bun:test";

// --- Mocks ---

const mockJsonResponse = JSON.stringify({
  intent: "OTHER",
  language: "en",
  reply: "mock response",
});

const mockGenerateContent = mock(async () => ({ text: mockJsonResponse }));

// Mock dependencies BEFORE any imports
mock.module("@google/genai", () => {
  return {
    GoogleGenAI: class {
      models = {
        generateContent: mockGenerateContent,
      };
    },
  };
});

mock.module("./config", () => ({
  config: {
    GEMINI_API_KEY: "test-key",
    GEMINI_MODEL: "gemini-test",
    MAX_SESSIONS: 2,
    MAX_HISTORY_MESSAGES: 4,
  },
}));

mock.module("./logger", () => ({
  logger: { error: mock(), info: mock() },
}));

mock.module("./persona", () => ({
  getPersona: mock(async () => "test persona"),
}));

import { GroceryParams } from "./types";
// Use dynamic import to ensure mocks are applied
const { generateCompletion } = await import("./brain");

describe("brain.ts", () => {
  beforeEach(() => {
    mockGenerateContent.mockClear();
  });

  it("should generate completion", async () => {
    const response = await generateCompletion("hello", "chat-new");
    expect(response.reply).toBe("mock response");
    expect(response.intent).toBe("OTHER");
    expect(mockGenerateContent).toHaveBeenCalled();
  });

  it("should correctly parse structured intent routing", async () => {
    const groceryJson = JSON.stringify({
      intent: "GROCERY",
      language: "zh",
      reply: "好哒～已经帮你在 Costco 的清单里加上两箱全脂牛奶啦！(u･ω･u)",
      item: "两箱全脂牛奶(milk)",
      action: "add",
      shop: "Costco",
    });
    mockGenerateContent.mockImplementationOnce(async () => ({
      text: groceryJson,
    }));

    const response = await generateCompletion(
      "买两箱全脂牛奶 at Costco",
      "chat-grocery",
    );
    expect(response.intent).toBe("GROCERY");
    expect(response.item).toBe("两箱全脂牛奶(milk)");
    expect(response.shop).toBe("Costco");
    expect(response.action).toBe("add");
  });

  it("should correctly parse grocery 'modify' action", async () => {
    const modifyJson = JSON.stringify({
      intent: "GROCERY",
      language: "zh",
      reply: "好哒～已经帮你把牛奶改成了两箱全脂牛奶啦！(u･ω･u)",
      item: "两箱全脂牛奶(milk)",
      action: "modify",
    });
    mockGenerateContent.mockImplementationOnce(async () => ({
      text: modifyJson,
    }));

    const response = await generateCompletion(
      "把牛奶改成两箱全脂牛奶",
      "chat-modify",
    );
    expect(response.intent).toBe("GROCERY");
    expect(response.item).toBe("两箱全脂牛奶(milk)");
    expect(response.action).toBe("modify");
  });

  it("should handle correctly formatted JSON", async () => {
    const cleanJson = JSON.stringify({
      intent: "OTHER",
      language: "en",
      reply: "Clean response",
    });
    mockGenerateContent.mockImplementationOnce(async () => ({
      text: cleanJson,
    }));

    const response = await generateCompletion("hello", "chat-clean");
    expect(response.intent).toBe("OTHER");
    expect(response.reply).toBe("Clean response");
  });

  it("should maintain history in subsequent calls", async () => {
    await generateCompletion("first message", "chat-history");
    const jsonReply = JSON.stringify({
      intent: "OTHER",
      language: "en",
      reply: "reply 1",
    });
    mockGenerateContent.mockImplementationOnce(async () => ({
      text: jsonReply,
    }));

    await generateCompletion("second message", "chat-history");

    expect(mockGenerateContent).toHaveBeenCalledTimes(2);
    const secondCall = (mockGenerateContent.mock.calls as any)[1][0];
    expect(secondCall.contents.length).toBe(3); // [first msg, first reply, second msg]
    expect(secondCall.contents[0].parts[0].text).toBe("first message");
  });

  it("should handle Gemini API errors gracefully", async () => {
    mockGenerateContent.mockImplementationOnce(async () => {
      throw new Error("API Error");
    });

    try {
      await generateCompletion("error prompt", "chat-err");
      expect(true).toBe(false);
    } catch (error: any) {
      expect(error.message).toContain(
        "Failed to generate completion from Gemini",
      );
    }
  });
});
