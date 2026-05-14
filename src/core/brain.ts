import { GoogleGenerativeAI } from "@google/generative-ai";
import { config } from "./config";
import { logger } from "./logger";

const genAI = new GoogleGenerativeAI(config.GEMINI_API_KEY);
const model = genAI.getGenerativeModel({ model: "gemini-1.5-flash" });

/**
 * Generates a simple text completion using Gemini 1.5 Flash.
 */
export const generateCompletion = async (prompt: string): Promise<string> => {
  try {
    const result = await model.generateContent(prompt);
    const response = result.response;
    return response.text();
  } catch (error) {
    logger.error({ error, prompt }, "Gemini API completion failure");
    throw new Error("Failed to generate completion from Gemini", {
      cause: error,
    });
  }
};
