import { describe, expect, it } from "bun:test";
import { BrainResponse } from "../../core/types";
import {
  assertAddItemCommands,
  createGroceryPlanner,
  createGroceryReplyGenerator,
  formatGroceryReply,
} from "./planner";

describe("createGroceryPlanner", () => {
  it("plans multiple add_item commands from structured LLM output", async () => {
    const response: BrainResponse = {
      intent: "GROCERY",
      language: "en",
      reply: "Added milk.",
      action: "add",
      shop: "Costco",
    };
    const planner = createGroceryPlanner(async () => ({
      text: JSON.stringify({
        commands: [
          { type: "add_item", itemName: "milk", shopName: "Costco" },
          { type: "add_item", itemName: "eggs", shopName: "Costco" },
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
        { type: "add_item", itemName: "milk", shopName: "Costco" },
        { type: "add_item", itemName: "eggs", shopName: "Costco" },
      ],
    });
  });

  it("rejects invalid planner output", async () => {
    const planner = createGroceryPlanner(async () => ({ text: "{}" }));

    await expect(
      planner({
        userText: "add milk",
        brainResponse: {
          intent: "GROCERY",
          language: "en",
          reply: "Added milk.",
          action: "add",
        },
      }),
    ).rejects.toThrow("Failed to plan grocery command from request");
  });
});

describe("assertAddItemCommands", () => {
  it("allows add_item commands", () => {
    expect(
      assertAddItemCommands([
        { type: "add_item", itemName: "milk", shopName: "Costco" },
      ]),
    ).toEqual([{ type: "add_item", itemName: "milk", shopName: "Costco" }]);
  });

  it("rejects unsupported grocery actions for the first slice", () => {
    expect(() =>
      assertAddItemCommands([{ type: "list_items", shopName: "Costco" }]),
    ).toThrow("Only grocery add_item is supported right now.");
  });

  it("rejects empty command lists before execution", async () => {
    const planner = createGroceryPlanner(async () => ({
      text: JSON.stringify({ commands: [] }),
    }));

    await expect(
      planner({
        userText: "add milk",
        brainResponse: {
          intent: "GROCERY",
          language: "en",
          reply: "Added milk.",
          action: "add",
        },
      }),
    ).rejects.toThrow("Failed to plan grocery command from request");
  });
});

describe("formatGroceryReply", () => {
  it("formats add_item replies from repository results after persistence", () => {
    expect(
      formatGroceryReply(
        [
          {
            type: "add_item",
            shopName: "Costco",
            items: [{ name: "milk", addedAt: 1 }],
            changed: true,
            itemName: "milk",
          },
          {
            type: "add_item",
            shopName: "Costco",
            items: [{ name: "milk", addedAt: 1 }],
            changed: true,
            itemName: "eggs",
          },
        ],
        "en",
      ),
    ).toBe("Done～ I added milk, eggs to your Costco grocery list. (u･ω･u)");
  });

  it("formats Chinese add_item replies", () => {
    expect(
      formatGroceryReply(
        [
          {
            type: "add_item",
            shopName: "Costco",
            items: [{ name: "两箱全脂牛奶(milk)", addedAt: 1 }],
            changed: true,
            itemName: "两箱全脂牛奶(milk)",
          },
        ],
        "zh",
      ),
    ).toBe("好哒～已经把 两箱全脂牛奶(milk) 加到 Costco 的购物清单啦 (u･ω･u)");
  });
});

describe("createGroceryReplyGenerator", () => {
  it("generates a final persona reply from persisted results", async () => {
    const replyGenerator = createGroceryReplyGenerator(async () => ({
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
            shopName: "Costco",
            items: [{ name: "milk", addedAt: 1 }],
            changed: true,
            itemName: "milk",
          },
        ],
      }),
    ).resolves.toBe("好哒～牛奶和鸡蛋都存好啦 (u･ω･u)");
  });

  it("falls back to deterministic formatting when final reply output is invalid", async () => {
    const replyGenerator = createGroceryReplyGenerator(async () => ({
      text: "{}",
    }));

    await expect(
      replyGenerator({
        userText: "add milk",
        language: "en",
        results: [
          {
            type: "add_item",
            shopName: "Costco",
            items: [{ name: "milk", addedAt: 1 }],
            changed: true,
            itemName: "milk",
          },
        ],
      }),
    ).resolves.toBe("Done～ I added milk to your Costco grocery list. (u･ω･u)");
  });
});
