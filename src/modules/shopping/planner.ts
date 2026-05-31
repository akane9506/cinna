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

// Runtime schemas for LLM JSON responses.
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

type SupportedShoppingPlannerCommand = Extract<
  ShoppingPlannerCommand,
  { type: "add_items" | "list_items" }
>;

type ListItemsResult = Extract<ShoppingOperationResult, { type: "list_items" }>;

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

// Public planner API.
export async function getShoppingPlannerInstruction(): Promise<string> {
  return loadPrompt(
    "planner.instruction.md",
    "Convert the user's shopping request into strict JSON commands. Group shopping items by category into add_items commands. Use category as the only list grouping key.",
  );
}

export function createShoppingPlanner(generateContent?: GenerateContent) {
  return async (
    input: ShoppingPlannerInput,
  ): Promise<ShoppingPlannerOutput> => {
    const contentGenerator = generateContent ?? generateShoppingContent;
    const plannerInstruction = await getShoppingPlannerInstruction();

    const response = await contentGenerator({
      model: config.GEMINI_MODEL,
      contents: [
        {
          role: "user",
          parts: [{ text: buildPlannerPrompt(input) }],
        },
      ],
      config: {
        systemInstruction: plannerInstruction,
        responseMimeType: "application/json",
        responseJsonSchema: toGeminiJsonSchema(ShoppingPlannerOutputSchema),
      },
    });

    return parsePlannerResponse(response.text || "");
  };
}

export const planShoppingCommands = createShoppingPlanner();

// Public reply API.
export function createShoppingReplyGenerator(
  generateContent?: GenerateContent,
) {
  return async (input: ShoppingReplyInput): Promise<string> => {
    const contentGenerator = generateContent ?? generateShoppingContent;
    const persona = await getPersona();
    const replyInstruction = await getShoppingReplyInstruction();

    const response = await contentGenerator({
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

    return parseReplyResponse(response.text || "", input);
  };
}

export const generateShoppingReply = createShoppingReplyGenerator();

// Command validation helpers used by the shopping handler.
export function assertAddItemsCommands(
  commands: ShoppingPlannerCommand[],
): Extract<ShoppingPlannerCommand, { type: "add_items" }>[] {
  const unsupportedCommand = commands.find(
    (command) => command.type !== "add_items",
  );
  if (unsupportedCommand) {
    throw new Error("Only shopping add_items is supported right now.");
  }
  return commands as Extract<ShoppingPlannerCommand, { type: "add_items" }>[];
}

export function assertSupportedShoppingCommands(
  commands: ShoppingPlannerCommand[],
): SupportedShoppingPlannerCommand[] {
  const unsupportedCommand = commands.find(
    (command) => command.type !== "add_items" && command.type !== "list_items",
  );
  if (unsupportedCommand) {
    throw new Error(
      "Only shopping add_items and list_items are supported right now.",
    );
  }
  return commands as SupportedShoppingPlannerCommand[];
}

// Shared Gemini adapter.
const generateShoppingContent: GenerateContent = (request) => {
  return client.models.generateContent(request);
};

// Prompt loading.
async function getShoppingReplyInstruction(): Promise<string> {
  return loadPrompt(
    "reply.instruction.md",
    "You are writing the final Telegram confirmation after shopping-list persistence succeeded. Reply in the requested language. Mention the saved item names and category briefly. Return only JSON that matches the schema.",
  );
}

async function loadPrompt(fileName: string, fallback: string): Promise<string> {
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
}

// LLM prompt builders.
function buildPlannerPrompt({
  userText,
  brainResponse,
}: ShoppingPlannerInput): string {
  return JSON.stringify({
    userText,
    routedIntent: brainResponse.intent,
    routedAction: brainResponse.action,
    routedItem: brainResponse.item,
    language: brainResponse.language,
  });
}

function buildReplyPrompt({
  userText,
  language,
  results,
}: ShoppingReplyInput): string {
  return JSON.stringify({
    userText,
    language,
    persistedResults: results.map((result) => ({
      type: result.type,
      category: result.category,
      itemNames: "itemNames" in result ? result.itemNames : undefined,
      items:
        "items" in result
          ? result.items.map((item) => ({ name: item.name }))
          : undefined,
      staledItems:
        "staledItems" in result
          ? result.staledItems.map((item) => ({ name: item.name }))
          : undefined,
      changed: result.changed,
    })),
  });
}

// LLM response parsers.
function parsePlannerResponse(responseText: string): ShoppingPlannerOutput {
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
}

function parseReplyResponse(
  responseText: string,
  input: ShoppingReplyInput,
): string {
  try {
    return ShoppingReplyOutputSchema.parse(JSON.parse(responseText)).reply;
  } catch (error) {
    logger.error(
      { error, responseText },
      "Shopping reply schema validation failed",
    );
    return formatShoppingReply(input.results, input.language);
  }
}

// ========== Fallback reply formatters. ==========
// Deterministic fallback replies used when LLM reply generation fails.
export function formatShoppingReply(
  results: ShoppingOperationResult[],
  language: string,
): string {
  const unsupportedResult = results.find(
    (result) => result.type !== "add_items" && result.type !== "list_items",
  );
  if (unsupportedResult) {
    return unsupportedActionReply(language);
  }

  if (results.every((result) => result.type === "list_items")) {
    return formatListItemsReply(results as ListItemsResult[], language);
  }

  return formatAddItemsReply(results, language);
}

function formatListItemsReply(
  listResults: ListItemsResult[],
  language: string,
): string {
  const lines = listResults.flatMap((result) =>
    result.items.map((item) => `- ${item.name}`),
  );
  const categoryNames = [
    ...new Set(listResults.map((result) => result.category)),
  ];
  const listName = categoryNames.join(", ");

  if (lines.length === 0) {
    return language === "zh"
      ? `${listName} 的购物清单现在是空的 (u･ω･u)`
      : `Your ${listName} shopping list is empty. (u･ω･u)`;
  }

  return language === "zh"
    ? `${listName} 的购物清单有：\n${lines.join("\n")}`
    : `Your ${listName} shopping list has:\n${lines.join("\n")}`;
}

function formatAddItemsReply(
  results: ShoppingOperationResult[],
  language: string,
): string {
  const itemNames = results.flatMap((result) =>
    "itemNames" in result ? result.itemNames : [],
  );
  const firstResult = results[0];
  const listName = firstResult?.category ?? "grocery";

  if (language === "zh") {
    return `好哒～已经把 ${itemNames.join("、")} 加到 ${listName} 的购物清单啦 (u･ω･u)`;
  }

  return `Done～ I added ${itemNames.join(", ")} to your ${listName} shopping list. (u･ω･u)`;
}

function unsupportedActionReply(language: string): string {
  return language === "zh"
    ? "这个购物清单操作还没上线。"
    : "That shopping action is not supported yet.";
}
