import { Context } from "telegraf";
import { BrainResponse } from "../../core/types";
import { logger } from "../../core/logger";
import { normalizeShoppingCategory } from "./repository";
import {
  assertSupportedShoppingCommands,
  ShoppingPlannerInput,
  ShoppingPlannerOutput,
  ShoppingReplyInput,
} from "./planner";
import { ShoppingOperationResult, ShoppingPlannerCommand } from "./types";

type AddItemsCommand = Extract<ShoppingPlannerCommand, { type: "add_items" }>;
type ListItemsCommand = Extract<ShoppingPlannerCommand, { type: "list_items" }>;

type SupportedShoppingCommand = AddItemsCommand | ListItemsCommand;

type ShoppingRepositoryPort = {
  addItems(
    userId: string,
    itemNames: string[],
    category?: string,
  ): Promise<Extract<ShoppingOperationResult, { type: "add_items" }>>;
  listItems(
    userId: string,
    category?: string,
  ): Promise<Extract<ShoppingOperationResult, { type: "list_items" }>>;
};

type ShoppingPlanner = (
  input: ShoppingPlannerInput,
) => Promise<ShoppingPlannerOutput>;

type ShoppingReplyGenerator = (input: ShoppingReplyInput) => Promise<string>;

type ShoppingServiceDependencies = {
  repository: ShoppingRepositoryPort;
  planner: ShoppingPlanner;
  replyGenerator: ShoppingReplyGenerator;
};

type ShoppingServiceInput = {
  userId?: string;
  brainResponse: BrainResponse;
  userText: string;
};

type ShoppingServiceResult = {
  replies: string[];
};

type ShoppingHandlerDependencies = {
  service: (input: ShoppingServiceInput) => Promise<ShoppingServiceResult>;
};

type ShoppingStepResult<T> =
  | {
      ok: true;
      value: T;
    }
  | {
      ok: false;
      result: ShoppingServiceResult;
    };

export const createShoppingService = ({
  repository,
  planner,
  replyGenerator,
}: ShoppingServiceDependencies) => {
  return async ({
    userId,
    brainResponse,
    userText,
  }: ShoppingServiceInput): Promise<ShoppingServiceResult> => {
    // Validate the Telegram user context before planning any side effects.
    if (!userId) {
      return {
        replies: ["Sorry, I cannot update a shopping list without a user."],
      };
    }

    // Convert user text into validated shopping commands.
    const commandPlan = await planSupportedShoppingCommands({
      planner,
      userText,
      brainResponse,
    });
    if (!commandPlan.ok) return commandPlan.result;

    // Execute supported commands against the repository.
    const operationResults = await executeShoppingCommands({
      repository,
      userId,
      brainResponse,
      commands: commandPlan.value,
    });
    if (!operationResults.ok) return operationResults.result;

    // Generate the final user-facing reply from persisted operation results.
    return generateShoppingServiceReply({
      replyGenerator,
      userId,
      userText,
      brainResponse,
      results: operationResults.value,
    });
  };
};

const planSupportedShoppingCommands = async ({
  planner,
  userText,
  brainResponse,
}: {
  planner: ShoppingPlanner;
  userText: string;
  brainResponse: BrainResponse;
}): Promise<ShoppingStepResult<SupportedShoppingCommand[]>> => {
  try {
    const plan = await planner({ userText, brainResponse });
    return {
      ok: true,
      value: assertSupportedShoppingCommands(plan.commands),
    };
  } catch (error) {
    logger.info(
      { error, action: brainResponse.action },
      "Unsupported shopping command",
    );
    return {
      ok: false,
      result: {
        replies: [unsupportedActionReply(brainResponse.language)],
      },
    };
  }
};

const executeShoppingCommands = async ({
  repository,
  userId,
  brainResponse,
  commands,
}: {
  repository: ShoppingRepositoryPort;
  userId: string;
  brainResponse: BrainResponse;
  commands: SupportedShoppingCommand[];
}): Promise<ShoppingStepResult<ShoppingOperationResult[]>> => {
  const addCommands = commands.filter(
    (command): command is AddItemsCommand => command.type === "add_items",
  );
  const listCommands = commands.filter(
    (command): command is ListItemsCommand => command.type === "list_items",
  );

  const results: ShoppingOperationResult[] = [];

  if (addCommands.length > 0) {
    try {
      results.push(...(await executeAddItems(repository, userId, addCommands)));
    } catch (error) {
      logger.error({ error, userId }, "Failed to persist shopping items");
      return {
        ok: false,
        result: {
          replies: [saveErrorReply(brainResponse.language)],
        },
      };
    }
  }

  if (listCommands.length > 0) {
    try {
      results.push(
        ...(await executeListItems(repository, userId, listCommands)),
      );
    } catch (error) {
      logger.error({ error, userId }, "Failed to list shopping items");
      return {
        ok: false,
        result: {
          replies: [readErrorReply(brainResponse.language)],
        },
      };
    }
  }

  return {
    ok: true,
    value: results,
  };
};

const generateShoppingServiceReply = async ({
  replyGenerator,
  userId,
  userText,
  brainResponse,
  results,
}: {
  replyGenerator: ShoppingReplyGenerator;
  userId: string;
  userText: string;
  brainResponse: BrainResponse;
  results: ShoppingOperationResult[];
}): Promise<ShoppingServiceResult> => {
  try {
    return {
      replies: [
        await replyGenerator({
          userText,
          language: brainResponse.language,
          results,
        }),
      ],
    };
  } catch (error) {
    logger.error({ error, userId }, "Failed to generate shopping reply");
    return {
      replies: [botErrorReply(brainResponse.language)],
    };
  }
};

export const createShoppingHandler = ({
  service,
}: ShoppingHandlerDependencies) => {
  return async (
    ctx: Context,
    brainResponse: BrainResponse,
    userText: string,
  ): Promise<void> => {
    const result = await service({
      userId: ctx.from?.id?.toString(),
      brainResponse,
      userText,
    });

    for (const reply of result.replies) {
      await ctx.reply(reply);
    }
  };
};

const executeAddItems = async (
  repository: Pick<ShoppingRepositoryPort, "addItems">,
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
  repository: Pick<ShoppingRepositoryPort, "listItems">,
  userId: string,
  commands: ListItemsCommand[],
): Promise<ShoppingOperationResult[]> => {
  return Promise.all(
    commands.map((command) =>
      repository.listItems(userId, normalizeShoppingCategory(command.category)),
    ),
  );
};

const unsupportedActionReply = (language: string): string =>
  language === "zh"
    ? "这个购物清单操作还没上线。"
    : "That shopping action is not supported yet.";

const readErrorReply = (language: string): string =>
  language === "zh"
    ? "读取购物清单时出错了，请稍后再试。"
    : "Sorry, I encountered an error while reading your shopping list.";

const saveErrorReply = (language: string): string =>
  language === "zh"
    ? "保存购物清单时出错了，请稍后再试。"
    : "Sorry, I encountered an error while saving your shopping list.";

const botErrorReply = (language: string): string =>
  language === "zh"
    ? "机器人出错了，请稍后再试。"
    : "Bot error. Please try again later.";
