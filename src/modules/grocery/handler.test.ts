import { describe, expect, it, mock } from "bun:test";
import { createGroceryHandler } from "./handler";

describe("createGroceryHandler", () => {
  it("plans and adds multiple items before replying with persisted details", async () => {
    const addedItemNames = ["milk", "eggs"];
    const addItem = mock(async () => ({
      type: "add_item" as const,
      shopName: "Costco",
      items: [{ name: "milk", addedAt: 1 }],
      changed: true,
      itemName: addedItemNames.shift(),
    }));
    const planner = mock(async () => ({
      commands: [
        { type: "add_item" as const, itemName: "milk", shopName: "Costco" },
        { type: "add_item" as const, itemName: "eggs", shopName: "Costco" },
      ],
    }));
    const replyGenerator = mock(async () => "Saved with persona.");
    const reply = mock(async () => {});
    const handler = createGroceryHandler(
      { addItem } as any,
      planner,
      replyGenerator,
    );

    await handler(
      {
        from: { id: 123 },
        reply,
      } as any,
      {
        intent: "GROCERY",
        language: "en",
        reply: "AI reply should not be used.",
        action: "add",
        shop: "Costco",
      },
      "add milk and eggs at Costco",
    );

    expect(planner).toHaveBeenCalledWith({
      userText: "add milk and eggs at Costco",
      brainResponse: {
        intent: "GROCERY",
        language: "en",
        reply: "AI reply should not be used.",
        action: "add",
        shop: "Costco",
      },
    });
    expect(addItem).toHaveBeenCalledWith("123", "milk", "Costco");
    expect(addItem).toHaveBeenCalledWith("123", "eggs", "Costco");
    expect(replyGenerator).toHaveBeenCalledWith({
      userText: "add milk and eggs at Costco",
      language: "en",
      results: [
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
    });
    expect(reply).toHaveBeenCalledWith("Saved with persona.");
  });

  it("safely rejects unsupported actions", async () => {
    const addItem = mock(async () => {
      throw new Error("should not be called");
    });
    const planner = mock(async () => ({
      commands: [{ type: "list_items" as const, shopName: "Costco" }],
    }));
    const replyGenerator = mock(async () => "should not be called");
    const reply = mock(async () => {});
    const handler = createGroceryHandler(
      { addItem } as any,
      planner,
      replyGenerator,
    );

    await handler(
      {
        from: { id: 123 },
        reply,
      } as any,
      {
        intent: "GROCERY",
        language: "en",
        reply: "List groceries.",
        action: "list",
      },
      "what is on my Costco list?",
    );

    expect(addItem).not.toHaveBeenCalled();
    expect(replyGenerator).not.toHaveBeenCalled();
    expect(reply).toHaveBeenCalledWith(
      "That grocery action is not supported yet.",
    );
  });
});
