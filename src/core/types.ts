import { z } from "zod";

export const BrainGroceryActionSchema = z.enum([
  "add",
  "remove",
  "clear",
  "list",
  "modify",
]);

export const BrainResponseSchema = z.object({
  intent: z.enum(["GROCERY", "FEEDBACK", "OTHER"]),
  language: z.string().describe("ISO 639-1 code"),
  reply: z.string().describe("Cute persona reply"),
  // Flattened params
  item: z
    .string()
    .optional()
    .describe("GROCERY: Item name in 'Original(English)' format"),
  action: BrainGroceryActionSchema.optional().describe("GROCERY: The action"),
  shop: z.string().optional().describe("GROCERY: Shop name"),
  detail: z.string().optional().describe("FEEDBACK: Content"),
  category: z
    .enum(["bug", "suggestion"])
    .optional()
    .describe("FEEDBACK: Category"),
});

export type BrainResponse = z.infer<typeof BrainResponseSchema>;
export type BrainGroceryAction = z.infer<typeof BrainGroceryActionSchema>;
// Re-export types for backward compatibility in tests
export type GroceryParams = {
  item: string;
  action: BrainGroceryAction;
  shop?: string;
};
export type FeedbackParams = { detail: string; category: string };
