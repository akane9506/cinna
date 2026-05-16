import { GoogleGenAI, Content } from "@google/genai";
import { zodToJsonSchema } from "zod-to-json-schema";
import { config } from "./config";
import { logger } from "./logger";
import { getPersona } from "./persona";
import { BrainResponse, BrainResponseSchema } from "./types";

const { GEMINI_API_KEY, GEMINI_MODEL, MAX_SESSIONS, MAX_HISTORY_MESSAGES } =
  config;

const client = new GoogleGenAI({
  apiKey: GEMINI_API_KEY,
});

/**
 * In-memory storage for conversation history.
 * Key: chatId (string)
 * Value: Array of Content objects
 */
const historyStore = new Map<string, Content[]>();

/**
 * Retrieves history and manages LRU eviction for the store.
 */
const getHistory = (chatId: string): Content[] => {
  const history = historyStore.get(chatId) || [];
  if (historyStore.has(chatId)) {
    historyStore.delete(chatId);
    historyStore.set(chatId, history);
  }
  return history;
};

/**
 * Saves history and enforces session limits.
 */
const setHistory = (chatId: string, history: Content[]): void => {
  if (historyStore.has(chatId)) {
    historyStore.delete(chatId);
  } else if (historyStore.size >= MAX_SESSIONS) {
    const oldestKey = historyStore.keys().next().value;
    if (oldestKey !== undefined) {
      historyStore.delete(oldestKey);
    }
  }
  historyStore.set(chatId, history);
};

/**
 * Truncates history to stay within token/message limits.
 */
const truncateHistory = (history: Content[]): Content[] => {
  if (history.length <= MAX_HISTORY_MESSAGES) return history;
  
  let truncated = history.slice(-MAX_HISTORY_MESSAGES);
  // Ensure history starts with a user message
  while (truncated.length > 0 && truncated[0].role !== "user") {
    truncated.shift();
  }
  return truncated;
};

/**
 * Generates a structured response using Gemini's generateContent API.
 * Manages conversation history manually to ensure strict schema adherence.
 */
export const generateCompletion = async (
  prompt: string,
  chatId: string = "default",
  model?: string,
): Promise<BrainResponse> => {
  try {
    const selectedModel = model || GEMINI_MODEL;
    const persona = await getPersona();
    const history = getHistory(chatId);

    // Prepare contents: existing history + the new user prompt
    const contents: Content[] = [
      ...history,
      { role: "user", parts: [{ text: prompt }] },
    ];

    const response = await client.models.generateContent({
      model: selectedModel,
      contents,
      config: {
        systemInstruction: persona,
        responseMimeType: "application/json",
        responseJsonSchema: zodToJsonSchema(BrainResponseSchema as any),
      },
    });

    const responseText = response.text || "";
    
    let brainResponse: BrainResponse;
    try {
      brainResponse = BrainResponseSchema.parse(JSON.parse(responseText));
      
      // Update history with the model's response if parsing succeeded
      const updatedHistory = [
        ...contents,
        { role: "model", parts: [{ text: responseText }] },
      ];
      setHistory(chatId, truncateHistory(updatedHistory));

    } catch (parseError) {
      logger.error({ responseText, parseError }, "Gemini Schema Validation Failed");
      
      // Fallback response
      brainResponse = {
        intent: "OTHER",
        language: "unknown",
        reply: responseText || "I'm sorry, I'm having a little trouble understanding. Could you say that again? (u･ω･u)",
      };
    }

    return brainResponse;

  } catch (error) {
    logger.error({ error, prompt, chatId }, "Gemini API failure");
    throw new Error("Failed to generate completion from Gemini", {
      cause: error,
    });
  }
};
