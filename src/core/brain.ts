import { Chat, GoogleGenAI } from "@google/genai";
import { config } from "./config";
import { logger } from "./logger";
import { getPersona } from "./persona";

// Default model to use if not specified
export const DEFAULT_MODEL = "gemini-3.1-flash-lite";

const client = new GoogleGenAI({
  apiKey: config.GEMINI_API_KEY,
});

/**
 * In-memory storage for chat sessions to maintain context.
 * Key: chatId (string)
 * Value: Single Chat session instance
 */
const activeSessions = new Map<string, Chat>();

/**
 * Generates a text completion using Gemini's stateful chat session.
 * Maintains context by reusing active sessions.
 */
export const generateCompletion = async (
  prompt: string,
  chatId: string = "default",
  model?: string,
): Promise<string> => {
  try {
    // 1. Try to find an existing session for this user/chat
    let chat = activeSessions.get(chatId);

    // 2. If no session exists, create a new one with the system persona
    if (!chat) {
      const persona = await getPersona();
      chat = client.chats.create({
        model: model || DEFAULT_MODEL,
        config: {
          systemInstruction: persona,
        },
        // Note: History is handled automatically by the stateful Chat object after creation
      });
      activeSessions.set(chatId, chat);
    }

    // 3. Send the prompt to the stateful session
    const result = await chat.sendMessage({ message: prompt });
    const responseText = result.text;

    // Log the current session history for debugging/tracing
    // const histories = chat.getHistory();

    return (
      responseText ||
      "对不起呢...Cinna 好像离线了，您可以给管理员发个消息，问问他Cinna现在的状况吗？～"
    );
  } catch (error) {
    logger.error({ error, prompt, chatId }, "Gemini API chat failure");
    throw new Error("Failed to generate completion from Gemini", {
      cause: error,
    });
  }
};
