import { z } from "zod";

export const GroceryTimestampMillisSchema = z.number().int().nonnegative();

export const GroceryItemSchema = z.object({
  name: z.string().trim().min(1),
  addedAt: GroceryTimestampMillisSchema,
});

export const GroceryListDocSchema = z.object({
  shopName: z.string().trim().min(1),
  items: z.array(GroceryItemSchema),
  lastUpdated: GroceryTimestampMillisSchema,
});

export const GroceryPlannerCommandSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("add_item"),
    itemName: z.string().trim().min(1),
    shopName: z.string().trim().min(1).optional(),
  }),
  z.object({
    type: z.literal("remove_item"),
    itemName: z.string().trim().min(1),
    shopName: z.string().trim().min(1).optional(),
  }),
  z.object({
    type: z.literal("update_item"),
    existingItemName: z.string().trim().min(1),
    newItemName: z.string().trim().min(1),
    shopName: z.string().trim().min(1).optional(),
  }),
  z.object({
    type: z.literal("list_items"),
    shopName: z.string().trim().min(1).optional(),
  }),
  z.object({
    type: z.literal("clear_list"),
    shopName: z.string().trim().min(1).optional(),
  }),
]);

export const GroceryOperationResultSchema = z.object({
  type: z.enum([
    "add_item",
    "remove_item",
    "update_item",
    "list_items",
    "clear_list",
  ]),
  shopName: z.string().trim().min(1),
  items: z.array(GroceryItemSchema),
  changed: z.boolean(),
  itemName: z.string().optional(),
  previousItemName: z.string().optional(),
});

export type GroceryItem = z.infer<typeof GroceryItemSchema>;
export type GroceryListDoc = z.infer<typeof GroceryListDocSchema>;
export type GroceryPlannerCommand = z.infer<typeof GroceryPlannerCommandSchema>;
export type GroceryOperationResult = z.infer<
  typeof GroceryOperationResultSchema
>;
