import { describe, expect, it } from "bun:test";
import { BrainResponse } from "../../core/types";
import {
  assertAddItemsCommands,
  assertSupportedShoppingCommands,
  createShoppingPlanner,
  createShoppingReplyGenerator,
  formatShoppingReply,
} from "./planner";

describe("createShoppingPlanner", () => {
  it("plans an add_items command from structured LLM output", async () => {
    const response: BrainResponse = {
      intent: "SHOPPING",
      language: "en",
      reply: "Added milk.",
      action: "add",
    };
    const planner = createShoppingPlanner(async () => ({
      text: JSON.stringify({
        commands: [
          {
            type: "add_items",
            itemNames: ["milk", "eggs"],
            category: "grocery",
          },
        ],
      }),
    }));

    await expect(
      planner({
        userText: "add milk and eggs at Costco",
        brainResponse: response,
      }),
    ).resolves.toEqual({
      commands: [
        {
          type: "add_items",
          itemNames: ["milk", "eggs"],
          category: "grocery",
        },
      ],
    });
  });

  it("plans a list_items command from structured LLM output", async () => {
    const planner = createShoppingPlanner(async () => ({
      text: JSON.stringify({
        commands: [
          {
            type: "list_items",
            category: "grocery",
          },
        ],
      }),
    }));

    await expect(
      planner({
        userText: "what is on my grocery list?",
        brainResponse: {
          intent: "SHOPPING",
          language: "en",
          reply: "Here is your grocery list.",
          action: "list",
        },
      }),
    ).resolves.toEqual({
      commands: [
        {
          type: "list_items",
          category: "grocery",
        },
      ],
    });
  });

  it("defaults list_items with an unknown category to other", async () => {
    const planner = createShoppingPlanner(async () => ({
      text: JSON.stringify({
        commands: [
          {
            type: "list_items",
            category: "hardware",
          },
        ],
      }),
    }));

    await expect(
      planner({
        userText: "show my hardware list",
        brainResponse: {
          intent: "SHOPPING",
          language: "en",
          reply: "Here is your list.",
          action: "list",
        },
      }),
    ).resolves.toEqual({
      commands: [
        {
          type: "list_items",
          category: "other",
        },
      ],
    });
  });

  it("falls unknown categories back to other", async () => {
    const planner = createShoppingPlanner(async () => ({
      text: JSON.stringify({
        commands: [
          {
            type: "add_items",
            itemNames: ["screws"],
            category: "hardware",
          },
        ],
      }),
    }));

    await expect(
      planner({
        userText: "add screws from Ace Hardware",
        brainResponse: {
          intent: "SHOPPING",
          language: "en",
          reply: "Added screws.",
          action: "add",
        },
      }),
    ).resolves.toEqual({
      commands: [
        {
          type: "add_items",
          itemNames: ["screws"],
          category: "other",
        },
      ],
    });
  });

  it("preserves stationery categories", async () => {
    const planner = createShoppingPlanner(async () => ({
      text: JSON.stringify({
        commands: [
          {
            type: "add_items",
            itemNames: ["notebooks", "shipping labels"],
            category: "stationery",
          },
        ],
      }),
    }));

    await expect(
      planner({
        userText: "add notebooks and shipping labels",
        brainResponse: {
          intent: "SHOPPING",
          language: "en",
          reply: "Added notebooks and shipping labels.",
          action: "add",
        },
      }),
    ).resolves.toEqual({
      commands: [
        {
          type: "add_items",
          itemNames: ["notebooks", "shipping labels"],
          category: "stationery",
        },
      ],
    });
  });

  it("rejects invalid planner output", async () => {
    const planner = createShoppingPlanner(async () => ({ text: "{}" }));

    await expect(
      planner({
        userText: "add milk",
        brainResponse: {
          intent: "SHOPPING",
          language: "en",
          reply: "Added milk.",
          action: "add",
        },
      }),
    ).rejects.toThrow("Failed to plan shopping command from request");
  });
});

describe("shopping planner instruction", () => {
  it("guards against adding recipe or dish placeholders as shopping items", async () => {
    const prompt = await Bun.file(
      `${import.meta.dir}/prompts/planner.instruction.md`,
    ).text();

    expect(prompt).toContain("只能把具体可购买商品写入 `itemNames`");
    expect(prompt).toContain("不要把菜名、料理名、任务名或概括词当成商品写入清单");
    expect(prompt).toContain("如果用户是在问某道菜需要哪些食材、怎么做、要准备什么");
  });
});

describe("assertAddItemsCommands", () => {
  it("allows add_items commands", () => {
    expect(
      assertAddItemsCommands([
        {
          type: "add_items",
          itemNames: ["milk"],
          category: "grocery",
        },
      ]),
    ).toEqual([
      {
        type: "add_items",
        itemNames: ["milk"],
        category: "grocery",
      },
    ]);
  });

  it("rejects unsupported shopping actions for the first slice", () => {
    expect(() =>
      assertAddItemsCommands([{ type: "list_items", category: "grocery" }]),
    ).toThrow("Only shopping add_items is supported right now.");
  });

  it("rejects empty command lists before execution", async () => {
    const planner = createShoppingPlanner(async () => ({
      text: JSON.stringify({ commands: [] }),
    }));

    await expect(
      planner({
        userText: "add milk",
        brainResponse: {
          intent: "SHOPPING",
          language: "en",
          reply: "Added milk.",
          action: "add",
        },
      }),
    ).rejects.toThrow("Failed to plan shopping command from request");
  });
});

describe("assertSupportedShoppingCommands", () => {
  it("allows add_items and list_items commands", () => {
    expect(
      assertSupportedShoppingCommands([
        {
          type: "add_items",
          itemNames: ["milk"],
          category: "grocery",
        },
        { type: "list_items", category: "grocery" },
      ]),
    ).toEqual([
      {
        type: "add_items",
        itemNames: ["milk"],
        category: "grocery",
      },
      { type: "list_items", category: "grocery" },
    ]);
  });

  it("rejects shopping actions outside the active slices", () => {
    expect(() =>
      assertSupportedShoppingCommands([
        { type: "remove_items", itemNames: ["milk"], category: "grocery" },
      ]),
    ).toThrow("Only shopping add_items and list_items are supported right now.");
  });
});

describe("formatShoppingReply", () => {
  it("formats add_items replies from repository results after persistence", () => {
    expect(
      formatShoppingReply(
        [
          {
            type: "add_items",
            category: "grocery",
            changed: true,
            itemNames: ["milk", "eggs"],
          },
        ],
        "en",
      ),
    ).toBe("Done～ I added milk, eggs to your grocery shopping list. (u･ω･u)");
  });

  it("formats Chinese add_items replies with category when shop is absent", () => {
    expect(
      formatShoppingReply(
        [
          {
            type: "add_items",
            category: "grocery",
            changed: true,
            itemNames: ["两箱全脂牛奶(milk)"],
          },
        ],
        "zh",
      ),
    ).toBe("好哒～已经把 两箱全脂牛奶(milk) 加到 grocery 的购物清单啦 (u･ω･u)");
  });

  it("formats a non-empty list_items reply from repository results", () => {
    expect(
      formatShoppingReply(
        [
          {
            type: "list_items",
            category: "grocery",
            changed: false,
            items: [
              { name: "milk", addedAt: 1 },
              { name: "eggs", addedAt: 2 },
            ],
          },
        ],
        "en",
      ),
    ).toBe("Your grocery shopping list has:\n- milk\n- eggs");
  });

  it("formats an empty list_items reply from repository results", () => {
    expect(
      formatShoppingReply(
        [
          {
            type: "list_items",
            category: "pharmacy",
            changed: false,
            items: [],
          },
        ],
        "en",
      ),
    ).toBe("Your pharmacy shopping list is empty. (u･ω･u)");
  });
});

describe("createShoppingReplyGenerator", () => {
  it("generates a final persona reply from persisted results", async () => {
    const replyGenerator = createShoppingReplyGenerator(async () => ({
      text: JSON.stringify({
        reply: "好哒～牛奶和鸡蛋都存好啦 (u･ω･u)",
      }),
    }));

    await expect(
      replyGenerator({
        userText: "add milk and eggs",
        language: "zh",
        results: [
          {
            type: "add_items",
            category: "grocery",
            changed: true,
            itemNames: ["milk"],
          },
        ],
      }),
    ).resolves.toBe("好哒～牛奶和鸡蛋都存好啦 (u･ω･u)");
  });

  it("formats list replies without using generated prose", async () => {
    const replyGenerator = createShoppingReplyGenerator(async () => {
      throw new Error("LLM should not be called for list replies");
    });

    await expect(
      replyGenerator({
        userText: "show my grocery list",
        language: "en",
        results: [
          {
            type: "list_items",
            category: "grocery",
            changed: false,
            items: [{ name: "milk", addedAt: 1 }],
          },
        ],
      }),
    ).resolves.toBe("Your grocery shopping list has:\n- milk");
  });

  it("falls back to deterministic formatting when final reply output is invalid", async () => {
    const replyGenerator = createShoppingReplyGenerator(async () => ({
      text: "{}",
    }));

    await expect(
      replyGenerator({
        userText: "add milk",
        language: "en",
        results: [
          {
            type: "add_items",
            category: "grocery",
            changed: true,
            itemNames: ["milk"],
          },
        ],
      }),
    ).resolves.toBe(
      "Done～ I added milk to your grocery shopping list. (u･ω･u)",
    );
  });
});
