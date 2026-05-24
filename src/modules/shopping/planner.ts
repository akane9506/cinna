import { GoogleGenAI, Content } from "@google/genai";
import { z } from "zod";
import { config } from "../../core/config";
import { GeminiJsonSchema, toGeminiJsonSchema } from "../../core/jsonSchema";
import { logger } from "../../core/logger";
import { getPersona } from "../../core/persona";
import { BrainResponse } from "../../core/types";
import {
  ShoppingOperationResult,
  ShoppingPlannerCommand,
  ShoppingPlannerCommandSchema,
} from "./types";

const ShoppingPlannerOutputSchema = z.object({
  commands: z.array(ShoppingPlannerCommandSchema).min(1),
});

const ShoppingReplyOutputSchema = z.object({
  reply: z.string().trim().min(1),
});

export type ShoppingPlannerOutput = z.infer<typeof ShoppingPlannerOutputSchema>;

export type ShoppingPlannerInput = {
  userText: string;
  brainResponse: BrainResponse;
};

export type ShoppingReplyInput = {
  userText: string;
  language: string;
  results: ShoppingOperationResult[];
};

type GenerateContent = (request: {
  model: string;
  contents: Content[];
  config: {
    systemInstruction: string;
    responseMimeType: "application/json";
    responseJsonSchema: GeminiJsonSchema;
  };
}) => Promise<{ text?: string }>;

const client = new GoogleGenAI({
  apiKey: config.GEMINI_API_KEY,
});

const promptCache = new Map<string, string>();

const loadPrompt = async (
  fileName: string,
  fallback: string,
): Promise<string> => {
  const cachedPrompt = promptCache.get(fileName);
  if (cachedPrompt) return cachedPrompt;

  const file = Bun.file(`${import.meta.dir}/prompts/${fileName}`);

  if (!(await file.exists())) {
    logger.warn(
      `src/modules/shopping/prompts/${fileName} not found, using fallback instruction.`,
    );
    promptCache.set(fileName, fallback);
    return fallback;
  }

  const prompt = await file.text();
  promptCache.set(fileName, prompt);
  return prompt;
};

export const getShoppingPlannerInstruction = (): Promise<string> =>
  loadPrompt(
    "planner.instruction.md",
    "Convert the user's shopping request into strict JSON commands. Group shopping items by category into add_items commands. Use category as the only list grouping key.",
  );

const getShoppingReplyInstruction = (): Promise<string> =>
  loadPrompt(
    "reply.instruction.md",
    "You are writing the final Telegram confirmation after shopping-list persistence succeeded. Reply in the requested language. Mention the saved item names and category briefly. Return only JSON that matches the schema.",
  );

const buildPlannerPrompt = ({
  userText,
  brainResponse,
}: ShoppingPlannerInput): string =>
  JSON.stringify({
    userText,
    routedIntent: brainResponse.intent,
    routedAction: brainResponse.action,
    routedItem: brainResponse.item,
    language: brainResponse.language,
  });

export const createShoppingPlanner = (
  generateContent: GenerateContent = (request) =>
    client.models.generateContent(request),
) => {
  return async (
    input: ShoppingPlannerInput,
  ): Promise<ShoppingPlannerOutput> => {
    const shoppingPlannerInstruction = await getShoppingPlannerInstruction();
    const response = await generateContent({
      model: config.GEMINI_MODEL,
      contents: [
        {
          role: "user",
          parts: [{ text: buildPlannerPrompt(input) }],
        },
      ],
      config: {
        systemInstruction: shoppingPlannerInstruction,
        responseMimeType: "application/json",
        responseJsonSchema: toGeminiJsonSchema(ShoppingPlannerOutputSchema),
      },
    });

    const responseText = response.text || "";
    try {
      return ShoppingPlannerOutputSchema.parse(JSON.parse(responseText));
    } catch (error) {
      logger.error(
        { error, responseText },
        "Shopping planner schema validation failed",
      );
      throw new Error("Failed to plan shopping command from request", {
        cause: error,
      });
    }
  };
};

export const planShoppingCommands = createShoppingPlanner();

const buildReplyPrompt = ({
  userText,
  language,
  results,
}: ShoppingReplyInput): string =>
  JSON.stringify({
    userText,
    language,
    persistedResults: results.map((result) => ({
      type: result.type,
      category: result.category,
      itemNames: "itemNames" in result ? result.itemNames : undefined,
      changed: result.changed,
    })),
  });

export const createShoppingReplyGenerator = (
  generateContent: GenerateContent = (request) =>
    client.models.generateContent(request),
) => {
  return async (input: ShoppingReplyInput): Promise<string> => {
    const persona = await getPersona();
    const replyInstruction = await getShoppingReplyInstruction();
    const response = await generateContent({
      model: config.GEMINI_MODEL,
      contents: [
        {
          role: "user",
          parts: [{ text: buildReplyPrompt(input) }],
        },
      ],
      config: {
        systemInstruction: `${persona}\n\n${replyInstruction}`,
        responseMimeType: "application/json",
        responseJsonSchema: toGeminiJsonSchema(ShoppingReplyOutputSchema),
      },
    });

    const responseText = response.text || "";
    try {
      return ShoppingReplyOutputSchema.parse(JSON.parse(responseText)).reply;
    } catch (error) {
      logger.error(
        { error, responseText },
        "Shopping reply schema validation failed",
      );
      return formatShoppingReply(input.results, input.language);
    }
  };
};

export const generateShoppingReply = createShoppingReplyGenerator();

export const assertAddItemsCommands = (
  commands: ShoppingPlannerCommand[],
): Extract<ShoppingPlannerCommand, { type: "add_items" }>[] => {
  const unsupportedCommand = commands.find(
    (command) => command.type !== "add_items",
  );
  if (unsupportedCommand) {
    throw new Error("Only shopping add_items is supported right now.");
  }
  return commands as Extract<ShoppingPlannerCommand, { type: "add_items" }>[];
};

export const formatShoppingReply = (
  results: ShoppingOperationResult[],
  language: string,
): string => {
  const unsupportedResult = results.find(
    (result) => result.type !== "add_items",
  );
  if (unsupportedResult) {
    return language === "zh"
      ? "这个购物清单操作还没上线。"
      : "That shopping action is not supported yet.";
  }

  const itemNames = results.flatMap((result) =>
    "itemNames" in result ? result.itemNames : [],
  );
  const firstResult = results[0];
  const listName = firstResult?.category ?? "grocery";

  if (language === "zh") {
    return `好哒～已经把 ${itemNames.join("、")} 加到 ${listName} 的购物清单啦 (u･ω･u)`;
  }

  return `Done～ I added ${itemNames.join(", ")} to your ${listName} shopping list. (u･ω･u)`;
};
