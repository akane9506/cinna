import { z } from "zod";

export type GeminiJsonSchema = ReturnType<z.ZodType["toJSONSchema"]>;

export const toGeminiJsonSchema = <Schema extends z.ZodType>(
  schema: Schema,
): GeminiJsonSchema => schema.toJSONSchema();
