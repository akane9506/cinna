import { describe, expect, it, mock } from "bun:test";
import { createShoppingHandler } from "./handler";

describe("createShoppingHandler", () => {
  it("plans and adds multiple items before replying with persisted details", async () => {
    const addItems = mock(async () => ({
      type: "add_items" as const,
      category: "grocery" as const,
      changed: true,
      itemNames: ["milk", "eggs"],
    }));
    const planner = mock(async () => ({
      commands: [
        {
          type: "add_items" as const,
          itemNames: ["milk", "eggs"],
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
          type: "add_items",
          category: "grocery",
          changed: true,
          itemNames: ["milk", "eggs"],
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
      commands: [
        {
          type: "remove_items" as const,
          itemNames: ["milk"],
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
        reply: "List groceries.",
        action: "list",
      },
      "remove milk",
    );

    expect(addItems).not.toHaveBeenCalled();
    expect(replyGenerator).not.toHaveBeenCalled();
    expect(reply).toHaveBeenCalledWith(
      "That shopping action is not supported yet.",
    );
  });

  it("plans and lists items before replying with Firestore results", async () => {
    const listItems = mock(async () => ({
      type: "list_items" as const,
      category: "grocery" as const,
      changed: false,
      items: [
        { name: "milk", addedAt: 1 },
        { name: "eggs", addedAt: 2 },
      ],
    }));
    const planner = mock(async () => ({
      commands: [{ type: "list_items" as const, category: "grocery" as const }],
    }));
    const replyGenerator = mock(async () => "Your grocery shopping list has.");
    const reply = mock(async () => {});
    const handler = createShoppingHandler(
      { listItems } as any,
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
        action: "list",
      },
      "what is on my grocery list?",
    );

    expect(listItems).toHaveBeenCalledWith("123", "grocery");
    expect(replyGenerator).toHaveBeenCalledWith({
      userText: "what is on my grocery list?",
      language: "en",
      results: [
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
    });
    expect(reply).toHaveBeenCalledWith("Your grocery shopping list has.");
  });

  it("replies with an empty list result", async () => {
    const listItems = mock(async () => ({
      type: "list_items" as const,
      category: "pharmacy" as const,
      changed: false,
      items: [],
    }));
    const planner = mock(async () => ({
      commands: [{ type: "list_items" as const, category: "pharmacy" as const }],
    }));
    const replyGenerator = mock(async () => "Your pharmacy list is empty.");
    const reply = mock(async () => {});
    const handler = createShoppingHandler(
      { listItems } as any,
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
        action: "list",
      },
      "what is on my pharmacy list?",
    );

    expect(listItems).toHaveBeenCalledWith("123", "pharmacy");
    expect(reply).toHaveBeenCalledWith("Your pharmacy list is empty.");
  });

  it("replies with a read error when listing fails", async () => {
    const listItems = mock(async () => {
      throw new Error("database unavailable");
    });
    const planner = mock(async () => ({
      commands: [{ type: "list_items" as const, category: "grocery" as const }],
    }));
    const replyGenerator = mock(async () => "should not be called");
    const reply = mock(async () => {});
    const handler = createShoppingHandler(
      { listItems } as any,
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
        action: "list",
      },
      "what is on my grocery list?",
    );

    expect(listItems).toHaveBeenCalledWith("123", "grocery");
    expect(replyGenerator).not.toHaveBeenCalled();
    expect(reply).toHaveBeenCalledWith(
      "Sorry, I encountered an error while reading your shopping list.",
    );
  });

  it("replies with a save error when persistence fails", async () => {
    const addItems = mock(async () => {
      throw new Error("database unavailable");
    });
    const planner = mock(async () => ({
      commands: [
        {
          type: "add_items" as const,
          itemNames: ["milk"],
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
