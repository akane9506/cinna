import { generateCompletion } from "../src/core/brain";

/**
 * Manual verification script for Step 2.1.
 * Run with: bun scripts/test-gemini.ts
 */

const prompt = "Say 'Gemini is online and ready to assist Cinna!'";
console.log(`🚀 Testing Gemini API connection...`);
console.log(`📝 Prompt: "${prompt}"`);

try {
  const response = await generateCompletion(prompt);
  console.log(`✅ Success!`);
  console.log(`🤖 Response: ${response.trim()}`);
} catch (error) {
  console.error(`❌ Gemini API test failed.`);
  console.error(error);
  process.exit(1);
}
