import { describe, expect, it } from "bun:test";
import {
  GroceryListStore,
  GroceryRepository,
  normalizeShopId,
} from "./repository";
import { GroceryListDoc } from "./types";

class InMemoryGroceryListStore implements GroceryListStore {
  readonly lists = new Map<string, GroceryListDoc>();

  async getList(
    userId: string,
    shopId: string,
  ): Promise<GroceryListDoc | null> {
    return this.lists.get(this.key(userId, shopId)) ?? null;
  }

  async saveList(
    userId: string,
    shopId: string,
    list: GroceryListDoc,
  ): Promise<void> {
    this.lists.set(this.key(userId, shopId), {
      ...list,
      items: list.items.map((item) => ({ ...item })),
    });
  }

  private key(userId: string, shopId: string): string {
    return `${userId}/${shopId}`;
  }
}

describe("normalizeShopId", () => {
  it("uses default when no shop is provided", () => {
    expect(normalizeShopId()).toBe("default");
    expect(normalizeShopId("  ")).toBe("default");
  });

  it("normalizes shop names for document ids", () => {
    expect(normalizeShopId("Trader Joe's")).toBe("trader-joe's");
    expect(normalizeShopId("Costco / Business")).toBe("costco---business");
  });
});

describe("GroceryRepository", () => {
  it("adds and lists grocery items under the requested shop", async () => {
    const store = new InMemoryGroceryListStore();
    const repository = new GroceryRepository(store);

    const addResult = await repository.addItem("user-1", "milk", "Costco");
    const listResult = await repository.listItems("user-1", "Costco");

    expect(addResult).toMatchObject({
      type: "add_item",
      shopName: "Costco",
      changed: true,
      itemName: "milk",
    });
    expect(listResult.items.map((item) => item.name)).toEqual(["milk"]);
    expect(Number.isInteger(listResult.items[0].addedAt)).toBe(true);
    expect(listResult.items[0].addedAt).toBeGreaterThan(0);
    expect(store.lists.get("user-1/costco")?.lastUpdated).toEqual(
      expect.any(Number),
    );
    expect(store.lists.has("user-1/costco")).toBe(true);
  });

  it("updates an existing grocery item", async () => {
    const store = new InMemoryGroceryListStore();
    const repository = new GroceryRepository(store);
    await repository.addItem("user-1", "milk");

    const result = await repository.updateItem("user-1", "MILK", "oat milk");

    expect(result).toMatchObject({
      type: "update_item",
      shopName: "default",
      changed: true,
      itemName: "oat milk",
      previousItemName: "MILK",
    });
    expect(result.items.map((item) => item.name)).toEqual(["oat milk"]);
  });

  it("does not write when updating a missing grocery item", async () => {
    const store = new InMemoryGroceryListStore();
    const repository = new GroceryRepository(store);
    await repository.addItem("user-1", "milk");
    const saveCountAfterAdd = store.lists.size;

    const result = await repository.updateItem("user-1", "eggs", "brown eggs");

    expect(result.changed).toBe(false);
    expect(result.items.map((item) => item.name)).toEqual(["milk"]);
    expect(store.lists.size).toBe(saveCountAfterAdd);
  });

  it("removes an existing grocery item", async () => {
    const store = new InMemoryGroceryListStore();
    const repository = new GroceryRepository(store);
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

  it("does not write when removing a missing grocery item", async () => {
    const store = new InMemoryGroceryListStore();
    const repository = new GroceryRepository(store);

    const result = await repository.removeItem("user-1", "milk");

    expect(result).toMatchObject({
      type: "remove_item",
      shopName: "default",
      changed: false,
      itemName: "milk",
    });
    expect(store.lists.size).toBe(0);
  });

  it("clears an existing grocery list", async () => {
    const store = new InMemoryGroceryListStore();
    const repository = new GroceryRepository(store);
    await repository.addItem("user-1", "milk", "Costco");

    const result = await repository.clearList("user-1", "Costco");

    expect(result).toMatchObject({
      type: "clear_list",
      shopName: "Costco",
      changed: true,
    });
    expect(result.items).toEqual([]);
    expect((await repository.listItems("user-1", "Costco")).items).toEqual([]);
  });
});
