import { createShoppingHandler, createShoppingService } from "./handler";
import { generateShoppingReply, planShoppingCommands } from "./planner";
import { ShoppingRepository } from "./repository";

export const createProductionShoppingHandler = () => {
  const service = createShoppingService({
    repository: new ShoppingRepository(),
    planner: planShoppingCommands,
    replyGenerator: generateShoppingReply,
  });

  return createShoppingHandler({ service });
};

type ShoppingHandler = ReturnType<typeof createProductionShoppingHandler>;

let productionShoppingHandler: ShoppingHandler | undefined;

const getProductionShoppingHandler = (): ShoppingHandler => {
  productionShoppingHandler ??= createProductionShoppingHandler();
  return productionShoppingHandler;
};

export const handleShoppingIntent: ShoppingHandler = (...args) =>
  getProductionShoppingHandler()(...args);
