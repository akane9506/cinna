import { z } from "zod";

export const ShoppingTimestampMillisSchema = z.number().int().nonnegative();

export const ShoppingItemSchema = z.object({
  name: z.string().trim().min(1),
  addedAt: ShoppingTimestampMillisSchema,
});

export const ShoppingCategorySchema = z
  .enum(["grocery", "pharmacy", "pet_store", "toy_shop", "other"])
  .catch("other");

export const ShoppingListDocSchema = z.object({
  category: ShoppingCategorySchema,
  items: z.array(ShoppingItemSchema),
  lastUpdated: ShoppingTimestampMillisSchema,
});

export const ShoppingPlannerCommandSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("add_item"),
    itemName: z.string().trim().min(1),
    category: ShoppingCategorySchema.optional(),
  }),
  z.object({
    type: z.literal("remove_item"),
    itemName: z.string().trim().min(1),
    category: ShoppingCategorySchema.optional(),
  }),
  z.object({
    type: z.literal("update_item"),
    existingItemName: z.string().trim().min(1),
    newItemName: z.string().trim().min(1),
    category: ShoppingCategorySchema.optional(),
  }),
  z.object({
    type: z.literal("list_items"),
    category: ShoppingCategorySchema.optional(),
  }),
  z.object({
    type: z.literal("clear_list"),
    category: ShoppingCategorySchema.optional(),
  }),
]);

export const ShoppingOperationResultSchema = z.object({
  type: z.enum([
    "add_item",
    "remove_item",
    "update_item",
    "list_items",
    "clear_list",
  ]),
  category: ShoppingCategorySchema,
  items: z.array(ShoppingItemSchema),
  changed: z.boolean(),
  itemName: z.string().optional(),
  previousItemName: z.string().optional(),
});

export type ShoppingItem = z.infer<typeof ShoppingItemSchema>;
export type ShoppingListDoc = z.infer<typeof ShoppingListDocSchema>;
export type ShoppingPlannerCommand = z.infer<
  typeof ShoppingPlannerCommandSchema
>;
export type ShoppingCategory = z.infer<typeof ShoppingCategorySchema>;
export type ShoppingOperationResult = z.infer<
  typeof ShoppingOperationResultSchema
>;
