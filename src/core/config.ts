import { z } from 'zod';
import dotenv from 'dotenv';

dotenv.config();

const configSchema = z.object({
  TELEGRAM_BOT_TOKEN: z.string(),
  GEMINI_API_KEY: z.string(),
  PORT: z.string().default('3000'),
  NODE_ENV: z.enum(['development', 'production', 'test']).default('development'),
});

const parsedConfig = configSchema.safeParse(process.env);

let configData: z.infer<typeof configSchema>;

if (!parsedConfig.success) {
  const issues = parsedConfig.error.issues;
  const fieldErrors = issues.map(issue => ({
    path: issue.path.join('.'),
    message: issue.message
  }));
  
  console.error('❌ Invalid configuration:', JSON.stringify(fieldErrors, null, 2));
  
  if (process.env.NODE_ENV === 'test') {
    // Fallback for tests if env vars are missing
    configData = {
      TELEGRAM_BOT_TOKEN: 'test_token',
      GEMINI_API_KEY: 'test_key',
      PORT: '3000',
      NODE_ENV: 'test',
    };
  } else {
    process.exit(1);
  }
} else {
  configData = parsedConfig.data;
}

export const config = configData;
