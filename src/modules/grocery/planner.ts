import { GoogleGenAI, Content } from "@google/genai";
import { z } from "zod";
import { config } from "../../core/config";
import { GeminiJsonSchema, toGeminiJsonSchema } from "../../core/jsonSchema";
import { logger } from "../../core/logger";
import { getPersona } from "../../core/persona";
import { BrainResponse } from "../../core/types";
import {
  GroceryOperationResult,
  GroceryPlannerCommand,
  GroceryPlannerCommandSchema,
} from "./types";

const GroceryPlannerOutputSchema = z.object({
  commands: z.array(GroceryPlannerCommandSchema).min(1),
});

const GroceryReplyOutputSchema = z.object({
  reply: z.string().trim().min(1),
});

export type GroceryPlannerOutput = z.infer<typeof GroceryPlannerOutputSchema>;

export type GroceryPlannerInput = {
  userText: string;
  brainResponse: BrainResponse;
};

export type GroceryReplyInput = {
  userText: string;
  language: string;
  results: GroceryOperationResult[];
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
      `src/modules/grocery/prompts/${fileName} not found, using fallback instruction.`,
    );
    promptCache.set(fileName, fallback);
    return fallback;
  }

  const prompt = await file.text();
  promptCache.set(fileName, prompt);
  return prompt;
};

export const getGroceryPlannerInstruction = (): Promise<string> =>
  loadPrompt(
    "planner.instruction.md",
    "Convert the user's grocery request into strict JSON commands. Split multiple grocery items into separate add_item commands.",
  );

const getGroceryReplyInstruction = (): Promise<string> =>
  loadPrompt(
    "reply.instruction.md",
    "You are writing the final Telegram confirmation after grocery persistence succeeded. Reply in the requested language. Mention the saved item names and shop briefly. Return only JSON that matches the schema.",
  );

const buildPlannerPrompt = ({
  userText,
  brainResponse,
}: GroceryPlannerInput): string =>
  JSON.stringify({
    userText,
    routedIntent: brainResponse.intent,
    routedAction: brainResponse.action,
    routedItem: brainResponse.item,
    routedShop: brainResponse.shop,
    language: brainResponse.language,
  });

export const createGroceryPlanner = (
  generateContent: GenerateContent = (request) =>
    client.models.generateContent(request),
) => {
  return async (input: GroceryPlannerInput): Promise<GroceryPlannerOutput> => {
    const groceryPlannerInstruction = await getGroceryPlannerInstruction();
    const response = await generateContent({
      model: config.GEMINI_MODEL,
      contents: [
        {
          role: "user",
          parts: [{ text: buildPlannerPrompt(input) }],
        },
      ],
      config: {
        systemInstruction: groceryPlannerInstruction,
        responseMimeType: "application/json",
        responseJsonSchema: toGeminiJsonSchema(GroceryPlannerOutputSchema),
      },
    });

    const responseText = response.text || "";
    try {
      return GroceryPlannerOutputSchema.parse(JSON.parse(responseText));
    } catch (error) {
      logger.error(
        { error, responseText },
        "Grocery planner schema validation failed",
      );
      throw new Error("Failed to plan grocery command from request", {
        cause: error,
      });
    }
  };
};

export const planGroceryCommands = createGroceryPlanner();

const buildReplyPrompt = ({
  userText,
  language,
  results,
}: GroceryReplyInput): string =>
  JSON.stringify({
    userText,
    language,
    persistedResults: results.map((result) => ({
      type: result.type,
      shopName: result.shopName,
      itemName: result.itemName,
      changed: result.changed,
    })),
  });

export const createGroceryReplyGenerator = (
  generateContent: GenerateContent = (request) =>
    client.models.generateContent(request),
) => {
  return async (input: GroceryReplyInput): Promise<string> => {
    const persona = await getPersona();
    const replyInstruction = await getGroceryReplyInstruction();
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
        responseJsonSchema: toGeminiJsonSchema(GroceryReplyOutputSchema),
      },
    });

    const responseText = response.text || "";
    try {
      return GroceryReplyOutputSchema.parse(JSON.parse(responseText)).reply;
    } catch (error) {
      logger.error(
        { error, responseText },
        "Grocery reply schema validation failed",
      );
      return formatGroceryReply(input.results, input.language);
    }
  };
};

export const generateGroceryReply = createGroceryReplyGenerator();

export const assertAddItemCommands = (
  commands: GroceryPlannerCommand[],
): Extract<GroceryPlannerCommand, { type: "add_item" }>[] => {
  const unsupportedCommand = commands.find(
    (command) => command.type !== "add_item",
  );
  if (unsupportedCommand) {
    throw new Error("Only grocery add_item is supported right now.");
  }
  return commands as Extract<GroceryPlannerCommand, { type: "add_item" }>[];
};

export const formatGroceryReply = (
  results: GroceryOperationResult[],
  language: string,
): string => {
  const unsupportedResult = results.find(
    (result) => result.type !== "add_item",
  );
  if (unsupportedResult) {
    return language === "zh"
      ? "这个购物清单操作还没上线。"
      : "That grocery action is not supported yet.";
  }

  const itemNames = results
    .map((result) => result.itemName)
    .filter((itemName): itemName is string => Boolean(itemName));
  const shopName = results[0]?.shopName ?? "default";

  if (language === "zh") {
    return `好哒～已经把 ${itemNames.join("、")} 加到 ${shopName} 的购物清单啦 (u･ω･u)`;
  }

  return `Done～ I added ${itemNames.join(", ")} to your ${shopName} grocery list. (u･ω･u)`;
};
