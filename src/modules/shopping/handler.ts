import { Context } from "telegraf";
import { BrainResponse } from "../../core/types";
import { logger } from "../../core/logger";
import { normalizeShoppingCategory, ShoppingRepository } from "./repository";
import {
  assertSupportedShoppingCommands,
  generateShoppingReply,
  ShoppingPlannerInput,
  ShoppingPlannerOutput,
  ShoppingReplyInput,
  planShoppingCommands,
} from "./planner";
import {
  SHOPPING_CATEGORIES,
  ShoppingOperationResult,
  ShoppingPlannerCommand,
} from "./types";

type AddItemsCommand = Extract<ShoppingPlannerCommand, { type: "add_items" }>;
type ListItemsCommand = Extract<ShoppingPlannerCommand, { type: "list_items" }>;

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

    let existingItemsByCategory;
    if (brainResponse.action === "add") {
      try {
        existingItemsByCategory = await loadExistingItemsByCategory(
          repository,
          userId,
        );
      } catch (error) {
        logger.error(
          { error, userId },
          "Failed to load shopping items for planning",
        );
        await ctx.reply(
          brainResponse.language === "zh"
            ? "读取购物清单时出错了，请稍后再试。"
            : "Sorry, I encountered an error while reading your shopping list.",
        );
        return;
      }
    }

    let commands;
    try {
      const plan = await planner({
        userText,
        brainResponse,
        existingItemsByCategory,
      });
      commands = assertSupportedShoppingCommands(plan.commands);
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

    const addCommands = commands.filter(
      (command): command is AddItemsCommand => command.type === "add_items",
    );
    const listCommands = commands.filter(
      (command): command is ListItemsCommand => command.type === "list_items",
    );

    const results: ShoppingOperationResult[] = [];

    if (addCommands.length > 0) {
      try {
        results.push(
          ...(await executeAddItems(repository, userId, addCommands)),
        );
      } catch (error) {
        logger.error({ error, userId }, "Failed to persist shopping items");
        await ctx.reply(
          brainResponse.language === "zh"
            ? "保存购物清单时出错了，请稍后再试。"
            : "Sorry, I encountered an error while saving your shopping list.",
        );
        return;
      }
    }

    if (listCommands.length > 0) {
      try {
        results.push(
          ...(await executeListItems(repository, userId, listCommands)),
        );
      } catch (error) {
        logger.error({ error, userId }, "Failed to list shopping items");
        await ctx.reply(
          brainResponse.language === "zh"
            ? "读取购物清单时出错了，请稍后再试。"
            : "Sorry, I encountered an error while reading your shopping list.",
        );
        return;
      }
    }

    try {
      await ctx.reply(
        await replyGenerator({
          userText,
          language: brainResponse.language,
          results,
        }),
      );
    } catch (error) {
      logger.error({ error, userId }, "Failed to generate shopping reply");
      await ctx.reply(
        brainResponse.language === "zh"
          ? "机器人出错了，请稍后再试。"
          : "Bot error. Please try again later.",
      );
    }
  };
};

const loadExistingItemsByCategory = async (
  repository: ShoppingRepository,
  userId: string,
) => {
  const listResults = await Promise.all(
    SHOPPING_CATEGORIES.map((category) =>
      repository.listItems(userId, category),
    ),
  );

  return Object.fromEntries(
    listResults.map((result) => [
      result.category,
      result.items.map((item) => item.name),
    ]),
  );
};

const executeAddItems = async (
  repository: ShoppingRepository,
  userId: string,
  commands: AddItemsCommand[],
): Promise<ShoppingOperationResult[]> => {
  const itemNamesByCategory = new Map<string, string[]>();

  for (const command of commands) {
    const category = normalizeShoppingCategory(command.category);
    const itemNames = itemNamesByCategory.get(category) ?? [];
    itemNames.push(...command.itemNames);
    itemNamesByCategory.set(category, itemNames);
  }

  return Promise.all(
    [...itemNamesByCategory.entries()].map(([category, itemNames]) =>
      repository.addItems(userId, itemNames, category),
    ),
  );
};

const executeListItems = async (
  repository: ShoppingRepository,
  userId: string,
  commands: ListItemsCommand[],
): Promise<ShoppingOperationResult[]> => {
  return Promise.all(
    commands.map((command) =>
      repository.listItems(userId, normalizeShoppingCategory(command.category)),
    ),
  );
};

export const handleShoppingIntent = createShoppingHandler();
