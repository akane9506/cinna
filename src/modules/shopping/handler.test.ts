import { describe, expect, it, mock } from "bun:test";
import { createShoppingHandler } from "./handler";

describe("createShoppingHandler", () => {
  it("plans and adds multiple items before replying with persisted details", async () => {
    const addItems = mock(async () => [
      {
        type: "add_item" as const,
        category: "grocery" as const,
        items: [
          { name: "milk", addedAt: 1 },
          { name: "eggs", addedAt: 1 },
        ],
        changed: true,
        itemName: "milk",
      },
      {
        type: "add_item" as const,
        category: "grocery" as const,
        items: [
          { name: "milk", addedAt: 1 },
          { name: "eggs", addedAt: 1 },
        ],
        changed: true,
        itemName: "eggs",
      },
    ]);
    const planner = mock(async () => ({
      commands: [
        {
          type: "add_item" as const,
          itemName: "milk",
          category: "grocery" as const,
        },
        {
          type: "add_item" as const,
          itemName: "eggs",
          category: "grocery" as const,
        },
      ],
    }));
    const replyGenerator = mock(async () => "Saved with persona.");
    const reply = mock(async () => {});
    const handler = createShoppingHandler(
      { addItems } as any,
      planner,
      replyGenerator,
    );

    await handler(
      {
        from: { id: 123 },
        reply,
      } as any,
      {
        intent: "SHOPPING",
        language: "en",
        reply: "AI reply should not be used.",
        action: "add",
      },
      "add milk and eggs at Costco",
    );

    expect(planner).toHaveBeenCalledWith({
      userText: "add milk and eggs at Costco",
      brainResponse: {
        intent: "SHOPPING",
        language: "en",
        reply: "AI reply should not be used.",
        action: "add",
      },
    });
    expect(addItems).toHaveBeenCalledTimes(1);
    expect(addItems).toHaveBeenCalledWith("123", ["milk", "eggs"], "grocery");
    expect(replyGenerator).toHaveBeenCalledWith({
      userText: "add milk and eggs at Costco",
      language: "en",
      results: [
        {
          type: "add_item",
          category: "grocery",
          items: [
            { name: "milk", addedAt: 1 },
            { name: "eggs", addedAt: 1 },
          ],
          changed: true,
          itemName: "milk",
        },
        {
          type: "add_item",
          category: "grocery",
          items: [
            { name: "milk", addedAt: 1 },
            { name: "eggs", addedAt: 1 },
          ],
          changed: true,
          itemName: "eggs",
        },
      ],
    });
    expect(reply).toHaveBeenCalledWith("Saved with persona.");
  });

  it("safely rejects unsupported actions", async () => {
    const addItems = mock(async () => {
      throw new Error("should not be called");
    });
    const planner = mock(async () => ({
      commands: [{ type: "list_items" as const, category: "grocery" as const }],
    }));
    const replyGenerator = mock(async () => "should not be called");
    const reply = mock(async () => {});
    const handler = createShoppingHandler(
      { addItems } as any,
      planner,
      replyGenerator,
    );

    await handler(
      {
        from: { id: 123 },
        reply,
      } as any,
      {
        intent: "SHOPPING",
        language: "en",
        reply: "List groceries.",
        action: "list",
      },
      "what is on my Costco list?",
    );

    expect(addItems).not.toHaveBeenCalled();
    expect(replyGenerator).not.toHaveBeenCalled();
    expect(reply).toHaveBeenCalledWith(
      "That shopping action is not supported yet.",
    );
  });

  it("replies with a save error when persistence fails", async () => {
    const addItems = mock(async () => {
      throw new Error("database unavailable");
    });
    const planner = mock(async () => ({
      commands: [
        {
          type: "add_item" as const,
          itemName: "milk",
          category: "grocery" as const,
        },
      ],
    }));
    const replyGenerator = mock(async () => "should not be called");
    const reply = mock(async () => {});
    const handler = createShoppingHandler(
      { addItems } as any,
      planner,
      replyGenerator,
    );

    await handler(
      {
        from: { id: 123 },
        reply,
      } as any,
      {
        intent: "SHOPPING",
        language: "en",
        reply: "AI reply should not be used.",
        action: "add",
      },
      "add milk",
    );

    expect(addItems).toHaveBeenCalledWith("123", ["milk"], "grocery");
    expect(replyGenerator).not.toHaveBeenCalled();
    expect(reply).toHaveBeenCalledWith(
      "Sorry, I encountered an error while saving your shopping list.",
    );
  });
});
