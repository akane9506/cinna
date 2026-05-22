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
    expect(normalizeShoppingCategory("hardware")).toBe("other");
  });
});

describe("ShoppingRepository", () => {
  it("adds and lists shopping items under the requested category", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);

    const addResult = await repository.addItem("user-1", "milk", "grocery");
    const listResult = await repository.listItems("user-1", "grocery");

    expect(addResult).toMatchObject({
      type: "add_item",
      category: "grocery",
      changed: true,
      itemName: "milk",
    });
    expect(listResult.items.map((item) => item.name)).toEqual(["milk"]);
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
    await repository.addItem("user-1", "bread", "grocery");
    const saveCountAfterFirstAdd = store.saveCount;

    const results = await repository.addItems("user-1", [
      "milk",
      "eggs",
      "butter",
    ]);

    expect(results.map((result) => result.itemName)).toEqual([
      "milk",
      "eggs",
      "butter",
    ]);
    expect(results.every((result) => result.category === "grocery")).toBe(true);
    expect(results[0].items.map((item) => item.name)).toEqual([
      "bread",
      "milk",
      "eggs",
      "butter",
    ]);
    expect(store.saveCount).toBe(saveCountAfterFirstAdd + 1);
    expect(
      (await repository.listItems("user-1", "grocery")).items.map(
        (item) => item.name,
      ),
    ).toEqual(["bread", "milk", "eggs", "butter"]);
  });

  it("falls unknown categories back to other", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);

    const result = await repository.addItem("user-1", "screws", "hardware");

    expect(result).toMatchObject({
      type: "add_item",
      category: "other",
      changed: true,
      itemName: "screws",
    });
    expect(store.lists.has("user-1/other")).toBe(true);
  });

  it("updates an existing shopping item", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    await repository.addItem("user-1", "milk");

    const result = await repository.updateItem("user-1", "MILK", "oat milk");

    expect(result).toMatchObject({
      type: "update_item",
      category: "grocery",
      changed: true,
      itemName: "oat milk",
      previousItemName: "MILK",
    });
    expect(result.items.map((item) => item.name)).toEqual(["oat milk"]);
  });

  it("does not write when updating a missing shopping item", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    await repository.addItem("user-1", "milk");
    const saveCountAfterAdd = store.lists.size;

    const result = await repository.updateItem("user-1", "eggs", "brown eggs");

    expect(result.changed).toBe(false);
    expect(result.items.map((item) => item.name)).toEqual(["milk"]);
    expect(store.lists.size).toBe(saveCountAfterAdd);
  });

  it("removes an existing shopping item", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    await repository.addItem("user-1", "milk");
    await repository.addItem("user-1", "eggs");

    const result = await repository.removeItem("user-1", "milk");

    expect(result).toMatchObject({
      type: "remove_item",
      changed: true,
      itemName: "milk",
    });
    expect(result.items.map((item) => item.name)).toEqual(["eggs"]);
  });

  it("does not write when removing a missing shopping item", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);

    const result = await repository.removeItem("user-1", "milk");

    expect(result).toMatchObject({
      type: "remove_item",
      category: "grocery",
      changed: false,
      itemName: "milk",
    });
    expect(store.lists.size).toBe(0);
  });

  it("clears an existing shopping list", async () => {
    const store = new InMemoryShoppingListStore();
    const repository = new ShoppingRepository(store);
    await repository.addItem("user-1", "milk", "grocery");

    const result = await repository.clearList("user-1", "grocery");

    expect(result).toMatchObject({
      type: "clear_list",
      category: "grocery",
      changed: true,
    });
    expect(result.items).toEqual([]);
    expect((await repository.listItems("user-1", "grocery")).items).toEqual([]);
  });
});
