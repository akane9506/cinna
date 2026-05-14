import { describe, it, expect, mock, beforeEach } from "bun:test";

/**
 * 1. Setup Mocks
 */
const mockGenerateContent = mock();
const mockGetGenerativeModel = mock(() => ({
  generateContent: mockGenerateContent,
}));

mock.module("@google/generative-ai", () => ({
  GoogleGenerativeAI: class {
    getGenerativeModel = mockGetGenerativeModel;
  },
}));

const mockLoggerError = mock();
mock.module("./logger", () => ({
  logger: {
    error: mockLoggerError,
  },
}));

mock.module("./config", () => ({
  config: {
    GEMINI_API_KEY: "test-api-key",
  },
}));

/**
 * 2. Import SUT
 */
const { generateCompletion } = await import("./brain");
const { logger } = await import("./logger");

describe("Brain Service", () => {
  beforeEach(() => {
    mockGenerateContent.mockClear();
    mockLoggerError.mockClear();
  });

  it("should return a completion from Gemini", async () => {
    mockGenerateContent.mockResolvedValue({
      response: { text: () => "Mocked Gemini Response" },
    });
    const response = await generateCompletion("Hello");
    expect(response).toBe("Mocked Gemini Response");
    expect(mockGetGenerativeModel).toHaveBeenCalledWith({
      model: "gemini-1.5-flash",
    });
  });

  it("should throw a wrapped error if the Gemini API fails", async () => {
    const testError = new Error("API Failure");
    mockGenerateContent.mockRejectedValue(testError);
    let caughtError: Error | undefined = undefined;
    try {
      await generateCompletion("Hello");
    } catch (e) {
      if (e instanceof Error) {
        caughtError = e;
        expect(caughtError.message).toBe(
          "Failed to generate completion from Gemini",
        );
        expect(caughtError.cause).toBe(testError);
      }
    }
    expect(caughtError).toBeDefined();
    expect(logger.error).toHaveBeenCalled();
  });
});
