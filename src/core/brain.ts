import { Chat, GoogleGenAI } from "@google/genai";
import { config } from "./config";
import { logger } from "./logger";
import { getPersona } from "./persona";

const {
  GEMINI_API_KEY,
  GEMINI_MODEL,
  MAX_SESSIONS,
  MAX_HISTORY_MESSAGES,
} = config;

const client = new GoogleGenAI({
  apiKey: GEMINI_API_KEY,
});

/**
 * In-memory storage for chat sessions to maintain context.
 * Key: chatId (string)
 * Value: Single Chat session instance
 *
 * We use a Map to store sessions. In JavaScript, Map preserves insertion order,
 * which allows us to implement a simple LRU (Least Recently Used) eviction policy.
 */
const activeSessions = new Map<string, Chat>();

/**
 * Retrieves a session and marks it as recently used.
 */
const getSession = (chatId: string): Chat | undefined => {
  const session = activeSessions.get(chatId);
  if (session) {
    // Re-insert to move it to the end of the insertion order (most recently used)
    activeSessions.delete(chatId);
    activeSessions.set(chatId, session);
  }
  return session;
};

/**
 * Stores a session and enforces the maximum session limit.
 */
const setSession = (chatId: string, session: Chat): void => {
  if (activeSessions.has(chatId)) {
    activeSessions.delete(chatId);
  } else if (activeSessions.size >= MAX_SESSIONS) {
    // Evict the oldest session (first item in the Map)
    const oldestKey = activeSessions.keys().next().value;
    if (oldestKey !== undefined) {
      activeSessions.delete(oldestKey);
    }
  }
  activeSessions.set(chatId, session);
};

/**
 * Generates a text completion using Gemini's stateful chat session.
 * Maintains context by reusing active sessions and managing history growth.
 */
export const generateCompletion = async (
  prompt: string,
  chatId: string = "default",
  model?: string,
): Promise<string> => {
  try {
    const selectedModel = model || GEMINI_MODEL;

    // 1. Try to find an existing session for this user/chat
    let chat = getSession(chatId);

    // 2. If no session exists, create a new one with the system persona
    if (!chat) {
      const persona = await getPersona();
      chat = client.chats.create({
        model: selectedModel,
        config: {
          systemInstruction: persona,
        },
      });
      setSession(chatId, chat);
    }

    // 3. Send the prompt to the stateful session
    const result = await chat.sendMessage({ message: prompt });
    const responseText = result.text;

    // 4. Manage history length to prevent context window bloat and excessive token usage
    // The history is updated automatically by the SDK after sendMessage.
    const history = chat.getHistory();
    if (history.length > MAX_HISTORY_MESSAGES) {
      // Keep only the most recent messages
      let truncatedHistory = history.slice(-MAX_HISTORY_MESSAGES);

      // Ensure the history starts with a 'user' message (SDK requirement)
      while (
        truncatedHistory.length > 0 &&
        truncatedHistory[0].role !== "user"
      ) {
        truncatedHistory.shift();
      }

      if (truncatedHistory.length > 0) {
        const persona = await getPersona();
        // Re-create the chat session with the truncated history to effectively "trim" it
        const newChat = client.chats.create({
          model: selectedModel,
          config: {
            systemInstruction: persona,
          },
          history: truncatedHistory,
        });
        setSession(chatId, newChat);
      }
    }

    return (
      responseText ||
      "I'm sorry, Cinna seems to be offline. Please contact the administrator to check my status."
    );
  } catch (error) {
    logger.error({ error, prompt, chatId }, "Gemini API chat failure");
    throw new Error("Failed to generate completion from Gemini", {
      cause: error,
    });
  }
};
