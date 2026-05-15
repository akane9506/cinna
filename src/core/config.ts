import { z } from "zod";

// GenAI
export const DEFAULT_GENAI_MODEL = "gemini-3.1-flash-lite";

// Environment variables
const configSchema = z.object({
  TELEGRAM_BOT_TOKEN: z.string(),
  GEMINI_API_KEY: z.string(),
  ALLOWED_USERS: z
    .string()
    .transform((val) => val.split(",").map((id) => parseInt(id.trim(), 10))),
  PORT: z.string().default("3000"),
  NODE_ENV: z
    .enum(["development", "production", "test"])
    .default("development"),
});

const parsedConfig = configSchema.safeParse(Bun.env);

let configData: z.infer<typeof configSchema>;

if (!parsedConfig.success) {
  const issues = parsedConfig.error.issues;
  const fieldErrors = issues.map((issue) => ({
    path: issue.path.join("."),
    message: issue.message,
  }));

  console.error(
    "❌ Invalid configuration:",
    JSON.stringify(fieldErrors, null, 2),
  );

  if (Bun.env.NODE_ENV === "test") {
    // Fallback for tests if env vars are missing
    configData = {
      TELEGRAM_BOT_TOKEN: "test_token",
      GEMINI_API_KEY: "test_key",
      ALLOWED_USERS: [12345], // Default test user ID
      PORT: "3000",
      NODE_ENV: "test",
    };
  } else {
    process.exit(1);
  }
} else {
  configData = parsedConfig.data;
}

export const config = configData;
