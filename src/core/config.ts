import { z } from "zod";

// Environment variables
const configSchema = z.object({
  TELEGRAM_BOT_TOKEN: z.string(),
  GEMINI_API_KEY: z.string(),
  GEMINI_MODEL: z.string().default("gemini-3.1-flash-lite"),
  FIREBASE_PROJECT_ID: z.string().min(1),
  FIREBASE_SERVICE_ACCOUNT_PATH: z.string().min(1).optional(),
  FIRESTORE_DATABASE_ID: z.string().default("(default)"),
  ALLOWED_USERS: z
    .string()
    .transform((val) => val.split(",").map((id) => parseInt(id.trim(), 10))),
  PORT: z.string().default("3000"),
  NODE_ENV: z
    .enum(["development", "production", "test"])
    .default("development"),
  MAX_SESSIONS: z.coerce.number().default(10),
  MAX_HISTORY_MESSAGES: z.coerce.number().default(100),
});

const parsedConfig = configSchema.safeParse(Bun.env);

let configData: z.infer<typeof configSchema>;

if (!parsedConfig.success) {
  if (Bun.env.NODE_ENV === "test") {
    // Fallback for tests if env vars are missing
    configData = {
      TELEGRAM_BOT_TOKEN: "test_token",
      GEMINI_API_KEY: "test_key",
      GEMINI_MODEL: "gemini-3.1-flash-lite",
      FIREBASE_PROJECT_ID: "test-project",
      FIREBASE_SERVICE_ACCOUNT_PATH: undefined,
      FIRESTORE_DATABASE_ID: "(default)",
      ALLOWED_USERS: [12345], // Default test user ID
      PORT: "3000",
      NODE_ENV: "test",
      MAX_SESSIONS: 1000,
      MAX_HISTORY_MESSAGES: 20,
    };
  } else {
    const issues = parsedConfig.error.issues;
    const fieldErrors = issues.map((issue) => ({
      path: issue.path.join("."),
      message: issue.message,
    }));

    console.error(
      "❌ Invalid configuration:",
      JSON.stringify(fieldErrors, null, 2),
    );

    process.exit(1);
  }
} else {
  configData = parsedConfig.data;
}

export const config = configData;
