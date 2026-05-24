import { describe, expect, it } from "bun:test";
import { BrainResponse } from "../../core/types";
import {
  assertAddItemCommands,
  createShoppingPlanner,
  createShoppingReplyGenerator,
  formatShoppingReply,
} from "./planner";

describe("createShoppingPlanner", () => {
  it("plans multiple add_item commands from structured LLM output", async () => {
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
            type: "add_item",
            itemName: "milk",
            category: "grocery",
          },
          {
            type: "add_item",
            itemName: "eggs",
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
          type: "add_item",
          itemName: "milk",
          category: "grocery",
        },
        {
          type: "add_item",
          itemName: "eggs",
          category: "grocery",
        },
      ],
    });
  });

  it("falls unknown categories back to other", async () => {
    const planner = createShoppingPlanner(async () => ({
      text: JSON.stringify({
        commands: [
          {
            type: "add_item",
            itemName: "screws",
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
          type: "add_item",
          itemName: "screws",
          category: "other",
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

describe("assertAddItemCommands", () => {
  it("allows add_item commands", () => {
    expect(
      assertAddItemCommands([
        {
          type: "add_item",
          itemName: "milk",
          category: "grocery",
        },
      ]),
    ).toEqual([
      {
        type: "add_item",
        itemName: "milk",
        category: "grocery",
      },
    ]);
  });

  it("rejects unsupported shopping actions for the first slice", () => {
    expect(() =>
      assertAddItemCommands([{ type: "list_items", category: "grocery" }]),
    ).toThrow("Only shopping add_item is supported right now.");
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

describe("formatShoppingReply", () => {
  it("formats add_item replies from repository results after persistence", () => {
    expect(
      formatShoppingReply(
        [
          {
            type: "add_item",
            category: "grocery",
            changed: true,
            itemNames: ["milk", "eggs"],
          },
        ],
        "en",
      ),
    ).toBe("Done～ I added milk, eggs to your grocery shopping list. (u･ω･u)");
  });

  it("formats Chinese add_item replies with category when shop is absent", () => {
    expect(
      formatShoppingReply(
        [
          {
            type: "add_item",
            category: "grocery",
            changed: true,
            itemNames: ["两箱全脂牛奶(milk)"],
          },
        ],
        "zh",
      ),
    ).toBe("好哒～已经把 两箱全脂牛奶(milk) 加到 grocery 的购物清单啦 (u･ω･u)");
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
            type: "add_item",
            category: "grocery",
            changed: true,
            itemNames: ["milk"],
          },
        ],
      }),
    ).resolves.toBe("好哒～牛奶和鸡蛋都存好啦 (u･ω･u)");
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
            type: "add_item",
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
