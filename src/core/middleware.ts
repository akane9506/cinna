import { Context, MiddlewareFn } from "telegraf";
import { config } from "./config";

/**
 * Middleware to restrict access to the bot to a specific list of user IDs.
 * If the user is not in the ALLOWED_USERS list, it replies with a standard rejection message.
 */
export const whitelistMiddleware: MiddlewareFn<Context> = async (ctx, next) => {
  const userId = ctx.from?.id;

  if (!userId || !config.ALLOWED_USERS.includes(userId)) {
    // We only reply if it's an interaction we can reply to
    if (ctx.chat) {
      await ctx.reply(
        "Sorry, I am only a personal agent that is not publicly available.",
      );
    }
    return; // Stop the middleware chain
  }

  return next(); // Proceed to the next handler
};
