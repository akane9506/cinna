import { z } from "zod";

export const ShoppingTimestampMillisSchema = z.number().int().nonnegative();

export const ShoppingItemSchema = z.object({
  name: z.string().trim().min(1),
  addedAt: ShoppingTimestampMillisSchema,
});

export const ShoppingCategorySchema = z
  .enum(["grocery", "pharmacy", "pet_store", "toy_shop", "stationery", "other"])
  .catch("other");

export const ShoppingListDocSchema = z.object({
  category: ShoppingCategorySchema,
  items: z.array(ShoppingItemSchema),
  lastUpdated: ShoppingTimestampMillisSchema,
});

export const ShoppingPlannerCommandSchema = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("add_items"),
    itemNames: z.array(z.string().trim().min(1)).min(1),
    category: ShoppingCategorySchema.optional(),
  }),
  z.object({
    type: z.literal("remove_items"),
    itemNames: z.array(z.string().trim().min(1)).min(1),
    category: ShoppingCategorySchema.optional(),
  }),
  z.object({
    type: z.literal("update_items"),
    itemUpdates: z
      .array(
        z.object({
          existingItemName: z.string().trim().min(1),
          newItemName: z.string().trim().min(1),
        }),
      )
      .min(1),
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

const ShoppingOperationResultBaseSchema = z.object({
  category: ShoppingCategorySchema,
  changed: z.boolean(),
});

export const ShoppingOperationResultSchema = z.discriminatedUnion("type", [
  ShoppingOperationResultBaseSchema.extend({
    type: z.literal("add_items"),
    itemNames: z.array(z.string()),
  }),
  ShoppingOperationResultBaseSchema.extend({
    type: z.literal("remove_items"),
    itemNames: z.array(z.string()),
  }),
  ShoppingOperationResultBaseSchema.extend({
    type: z.literal("update_items"),
    itemNames: z.array(z.string()),
    previousItemNames: z.array(z.string()),
  }),
  ShoppingOperationResultBaseSchema.extend({
    type: z.literal("list_items"),
    items: z.array(ShoppingItemSchema),
    staledItems: z.array(ShoppingItemSchema),
  }),
  ShoppingOperationResultBaseSchema.extend({
    type: z.literal("clear_list"),
  }),
]);

export type ShoppingItem = z.infer<typeof ShoppingItemSchema>;
export type ShoppingListDoc = z.infer<typeof ShoppingListDocSchema>;
export type ShoppingPlannerCommand = z.infer<
  typeof ShoppingPlannerCommandSchema
>;
export type ShoppingCategory = z.infer<typeof ShoppingCategorySchema>;
export type ShoppingOperationResult = z.infer<
  typeof ShoppingOperationResultSchema
>;
