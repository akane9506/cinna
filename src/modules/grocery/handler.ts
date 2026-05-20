import { Context } from "telegraf";
import { BrainResponse } from "../../core/types";
import { logger } from "../../core/logger";
import { GroceryRepository } from "./repository";
import {
  assertAddItemCommands,
  generateGroceryReply,
  GroceryPlannerInput,
  GroceryPlannerOutput,
  GroceryReplyInput,
  planGroceryCommands,
} from "./planner";

export const createGroceryHandler = (
  repository: GroceryRepository = new GroceryRepository(),
  planner: (
    input: GroceryPlannerInput,
  ) => Promise<GroceryPlannerOutput> = planGroceryCommands,
  replyGenerator: (
    input: GroceryReplyInput,
  ) => Promise<string> = generateGroceryReply,
) => {
  return async (
    ctx: Context,
    brainResponse: BrainResponse,
    userText: string,
  ): Promise<void> => {
    const userId = ctx.from?.id?.toString();
    if (!userId) {
      await ctx.reply("Sorry, I cannot update a grocery list without a user.");
      return;
    }

    let commands;
    try {
      const plan = await planner({ userText, brainResponse });
      commands = assertAddItemCommands(plan.commands);
    } catch (error) {
      logger.info(
        { error, action: brainResponse.action },
        "Unsupported grocery command",
      );
      await ctx.reply(
        brainResponse.language === "zh"
          ? "这个购物清单操作还没上线。"
          : "That grocery action is not supported yet.",
      );
      return;
    }

    const results = await Promise.all(
      commands.map((command) =>
        repository.addItem(userId, command.itemName, command.shopName),
      ),
    );
    await ctx.reply(
      await replyGenerator({
        userText,
        language: brainResponse.language,
        results,
      }),
    );
  };
};

export const handleGroceryIntent = createGroceryHandler();
