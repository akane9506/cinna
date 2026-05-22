import { Context } from "telegraf";
import { BrainResponse } from "../../core/types";
import { logger } from "../../core/logger";
import { normalizeShoppingCategory, ShoppingRepository } from "./repository";
import {
  assertAddItemCommands,
  generateShoppingReply,
  ShoppingPlannerInput,
  ShoppingPlannerOutput,
  ShoppingReplyInput,
  planShoppingCommands,
} from "./planner";

export const createShoppingHandler = (
  repository: ShoppingRepository = new ShoppingRepository(),
  planner: (
    input: ShoppingPlannerInput,
  ) => Promise<ShoppingPlannerOutput> = planShoppingCommands,
  replyGenerator: (
    input: ShoppingReplyInput,
  ) => Promise<string> = generateShoppingReply,
) => {
  return async (
    ctx: Context,
    brainResponse: BrainResponse,
    userText: string,
  ): Promise<void> => {
    const userId = ctx.from?.id?.toString();
    if (!userId) {
      await ctx.reply("Sorry, I cannot update a shopping list without a user.");
      return;
    }

    let commands;
    try {
      const plan = await planner({ userText, brainResponse });
      commands = assertAddItemCommands(plan.commands);
    } catch (error) {
      logger.info(
        { error, action: brainResponse.action },
        "Unsupported shopping command",
      );
      await ctx.reply(
        brainResponse.language === "zh"
          ? "这个购物清单操作还没上线。"
          : "That shopping action is not supported yet.",
      );
      return;
    }

    const itemNamesByCategory = new Map<string, string[]>();
    for (const command of commands) {
      const category = normalizeShoppingCategory(command.category);
      const itemNames = itemNamesByCategory.get(category) ?? [];
      itemNames.push(command.itemName);
      itemNamesByCategory.set(category, itemNames);
    }

    const results = (
      await Promise.all(
        [...itemNamesByCategory.entries()].map(([category, itemNames]) =>
          repository.addItems(userId, itemNames, category),
        ),
      )
    ).flat();
    await ctx.reply(
      await replyGenerator({
        userText,
        language: brainResponse.language,
        results,
      }),
    );
  };
};

export const handleShoppingIntent = createShoppingHandler();
