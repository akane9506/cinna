import { z } from "zod";

export const BrainShoppingActionSchema = z.enum([
  "add",
  "remove",
  "clear",
  "list",
  "modify",
]);

export const BrainResponseSchema = z.object({
  intent: z.enum(["SHOPPING", "FEEDBACK", "OTHER"]),
  language: z.string().describe("ISO 639-1 code"),
  reply: z.string().describe("Cute persona reply"),
  // Flattened params
  item: z
    .string()
    .optional()
    .describe("SHOPPING: Item name in 'Original(English)' format"),
  action: BrainShoppingActionSchema.optional().describe("SHOPPING: The action"),
  detail: z.string().optional().describe("FEEDBACK: Content"),
  category: z
    .enum(["bug", "suggestion"])
    .optional()
    .describe("FEEDBACK: Category"),
});

export type BrainResponse = z.infer<typeof BrainResponseSchema>;
export type BrainShoppingAction = z.infer<typeof BrainShoppingActionSchema>;
// Re-export types for backward compatibility in tests
export type ShoppingParams = {
  item: string;
  action: BrainShoppingAction;
};
export type FeedbackParams = { detail: string; category: string };
