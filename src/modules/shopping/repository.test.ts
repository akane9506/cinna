import { describe, expect, it } from "bun:test";
import {
  ShoppingListStore,
  ShoppingRepository,
  normalizeShoppingCategory,
} from "./repository";
import { ShoppingListDoc, ShoppingCategory } from "./types";

class InMemoryShoppingListStore implements ShoppingListStore {
  readonly lists = new Map<string, ShoppingListDoc>();
  saveCount = 0;

  async getList(
    userId: string,
    category: ShoppingCategory,
  ): Promise<ShoppingListDoc | null> {
    return this.lists.get(this.key(userId, category)) ?? null;
  }

  async saveList(
    userId: string,
    category: ShoppingCategory,
    list: ShoppingListDoc,
  ): Promise<void> {
    this.saveCount += 1;
    this.lists.set(this.key(userId, category), {
      ...list,
      items: list.items.map((item) => ({ ...item })),
    });
  }

  private key(userId: string, category: ShoppingCategory): string {
    return `${userId}/${category}`;
  }
}

describe("normalizeShoppingCategory", () => {
  it("uses grocery when no category is provided", () => {
    expect(normalizeShoppingCategory()).toBe("grocery");
  });

  it("preserves known categories and falls unknown categories back to other", () => {
    expect(normalizeShoppingCategory("pharmacy")).toBe("pharmacy");
    expect(normalizeShoppingCategory("stationery")).toBe("stationery");
    expect(normalizeShoppingCategory("hardware")).toBe("other");
  });
});

describe("ShoppingRepository", () => {
  it("adds and lists shopping items under the requested category", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);

    const addResult = await repository.addItems("user-1", ["milk"], "grocery");
    const listResult = await repository.listItems("user-1", "grocery");

    expect(addResult).toMatchObject({
      type: "add_items",
      category: "grocery",
      changed: true,
      itemNames: ["milk"],
    });
    expect(listResult.items.map((item) => item.name)).toEqual(["milk"]);
    expect(listResult.staledItems).toEqual([]);
    expect(Number.isInteger(listResult.items[0].addedAt)).toBe(true);
    expect(listResult.items[0].addedAt).toBeGreaterThan(0);
    expect(store.lists.get("user-1/grocery")?.lastUpdated).toEqual(
      expect.any(Number),
    );
    expect(store.lists.has("user-1/grocery")).toBe(true);
  });

  it("adds multiple shopping items to one category with a single save", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    await repository.addItems("user-1", ["bread"], "grocery");
    const saveCountAfterFirstAdd = store.saveCount;

    const result = await repository.addItems("user-1", [
      "milk",
      "eggs",
      "butter",
    ]);

    expect(result.itemNames).toEqual(["milk", "eggs", "butter"]);
    expect(result.category).toBe("grocery");
    expect(store.saveCount).toBe(saveCountAfterFirstAdd + 1);
    expect(
      (await repository.listItems("user-1", "grocery")).items.map(
        (item) => item.name,
      ),
    ).toEqual(["bread", "milk", "eggs", "butter"]);
  });

  it("stores items older than two weeks in the staled items field when listing", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    const now = Date.now();
    const fifteenDaysAgo = now - 15 * 24 * 60 * 60 * 1000;
    const thirteenDaysAgo = now - 13 * 24 * 60 * 60 * 1000;

    store.lists.set("user-1/grocery", {
      category: "grocery",
      lastUpdated: now,
      items: [
        { name: "old milk", addedAt: fifteenDaysAgo },
        { name: "fresh eggs", addedAt: now },
        { name: "recent bread", addedAt: thirteenDaysAgo },
      ],
    });

    const result = await repository.listItems("user-1", "grocery");

    expect(result.items.map((item) => item.name)).toEqual([
      "old milk",
      "fresh eggs",
      "recent bread",
    ]);
    expect(result.staledItems).toEqual([
      { name: "old milk", addedAt: fifteenDaysAgo },
    ]);
  });

  it("falls unknown categories back to other", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);

    const result = await repository.addItems("user-1", ["screws"], "hardware");

    expect(result).toMatchObject({
      type: "add_items",
      category: "other",
      changed: true,
      itemNames: ["screws"],
    });
    expect(store.lists.has("user-1/other")).toBe(true);
  });

  it("does not write when there are no item names to add", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);

    const result = await repository.addItems("user-1", [" ", ""], "grocery");

    expect(result).toMatchObject({
      type: "add_items",
      category: "grocery",
      changed: false,
      itemNames: [],
    });
    expect(store.saveCount).toBe(0);
  });

  it("updates existing shopping items in a batch", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    await repository.addItems("user-1", ["milk", "eggs"]);

    const result = await repository.updateItems("user-1", [
      { existingItemName: "MILK", newItemName: "oat milk" },
      { existingItemName: "eggs", newItemName: "brown eggs" },
    ]);

    expect(result).toMatchObject({
      type: "update_items",
      category: "grocery",
      changed: true,
      itemNames: ["oat milk", "brown eggs"],
      previousItemNames: ["MILK", "eggs"],
    });
    expect(
      (await repository.listItems("user-1", "grocery")).items.map(
        (item) => item.name,
      ),
    ).toEqual(["oat milk", "brown eggs"]);
  });

  it("updates every matching item while preserving addedAt", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    await repository.addItems("user-1", ["milk", "milk", "eggs"]);
    const originalItems = (await repository.listItems("user-1")).items;

    const result = await repository.updateItems("user-1", [
      { existingItemName: "milk", newItemName: "oat milk" },
    ]);

    expect(result).toMatchObject({
      type: "update_items",
      changed: true,
      itemNames: ["oat milk", "oat milk"],
      previousItemNames: ["milk", "milk"],
    });
    expect((await repository.listItems("user-1")).items).toEqual([
      { name: "oat milk", addedAt: originalItems[0].addedAt },
      { name: "oat milk", addedAt: originalItems[1].addedAt },
      { name: "eggs", addedAt: originalItems[2].addedAt },
    ]);
  });

  it("does not write when updating a missing shopping item", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    await repository.addItems("user-1", ["milk"]);
    const saveCountAfterAdd = store.lists.size;

    const result = await repository.updateItems("user-1", [
      { existingItemName: "eggs", newItemName: "brown eggs" },
    ]);

    expect(result.changed).toBe(false);
    expect(result.itemNames).toEqual(["brown eggs"]);
    expect(result.previousItemNames).toEqual(["eggs"]);
    expect(
      (await repository.listItems("user-1", "grocery")).items.map(
        (item) => item.name,
      ),
    ).toEqual(["milk"]);
    expect(store.lists.size).toBe(saveCountAfterAdd);
  });

  it("removes existing shopping items in a batch", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    await repository.addItems("user-1", ["milk", "eggs", "butter"]);

    const result = await repository.removeItems("user-1", ["milk", "EGGS"]);

    expect(result).toMatchObject({
      type: "remove_items",
      changed: true,
      itemNames: ["milk", "eggs"],
    });
    expect(
      (await repository.listItems("user-1", "grocery")).items.map(
        (item) => item.name,
      ),
    ).toEqual(["butter"]);
  });

  it("does not write when removing a missing shopping item", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);

    const result = await repository.removeItems("user-1", ["milk"]);

    expect(result).toMatchObject({
      type: "remove_items",
      category: "grocery",
      changed: false,
      itemNames: ["milk"],
    });
    expect(store.lists.size).toBe(0);
  });

  it("clears an existing shopping list", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    await repository.addItems("user-1", ["milk"], "grocery");

    const result = await repository.clearList("user-1", "grocery");

    expect(result).toMatchObject({
      type: "clear_list",
      category: "grocery",
      changed: true,
    });
    expect((await repository.listItems("user-1", "grocery")).items).toEqual([]);
  });
});
