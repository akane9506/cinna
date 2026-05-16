import { Context } from "telegraf";
import { BrainResponse } from "./types";
import { logger } from "./logger";

/**
 * Dispatcher for routing detected intents to their respective module handlers.
 */
export const dispatchIntent = async (
  ctx: Context,
  brainResponse: BrainResponse,
) => {
  const { intent, reply, item, action, shop, detail, category } = brainResponse;

  switch (intent) {
    case "GROCERY":
      // Phase 3.2: Implement Grocery module logic here
      if (item) {
        logger.info({ item, action, shop }, "Grocery intent detected");
      }
      break;

    case "FEEDBACK":
      // Phase 3.3: Implement Feedback module logic here
      if (detail) {
        logger.info({ detail, category }, "Feedback intent detected");
      }
      break;

    case "OTHER":
    default:
      // General conversation is handled by the Brain's reply
      break;
  }

  // Reply to the user with the AI-generated message
  if (reply) {
    try {
      await ctx.reply(reply);
    } catch (error) {
      logger.error({ error, reply }, "Failed to send reply via Telegram");
    }
  }
};
