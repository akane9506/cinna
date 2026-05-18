import { Firestore } from "firebase-admin/firestore";
import { getFirestoreDb } from "../../core/firestore";
import {
  GroceryItem,
  GroceryListDoc,
  GroceryListDocSchema,
  GroceryOperationResult,
} from "./types";

export interface GroceryListStore {
  getList(userId: string, shopId: string): Promise<GroceryListDoc | null>;
  saveList(userId: string, shopId: string, list: GroceryListDoc): Promise<void>;
}

export const normalizeShopId = (shopName?: string): string => {
  if (!shopName?.trim()) return "default";
  return shopName
    .trim()
    .toLowerCase()
    .replaceAll("/", "-")
    .replace(/\s+/g, "-");
};

const getDisplayShopName = (shopName?: string): string =>
  shopName?.trim() || "default";

export const createFirestoreGroceryListStore = (
  db: Firestore = getFirestoreDb(),
): GroceryListStore => ({
  async getList(userId, shopId) {
    const snapshot = await db
      .doc(`users/${userId}/groceryLists/${shopId}`)
      .get();
    if (!snapshot.exists) return null;
    return GroceryListDocSchema.parse(snapshot.data());
  },

  async saveList(userId, shopId, list) {
    const parsedList = GroceryListDocSchema.parse(list);
    await db.doc(`users/${userId}/groceryLists/${shopId}`).set(parsedList, {
      merge: true,
    });
  },
});

export class GroceryRepository {
  constructor(
    private readonly store: GroceryListStore = createFirestoreGroceryListStore(),
  ) {}

  // The current strategy is that, with any updates, we overwrite the whole list,
  // rather than update part of the document.
  async addItem(
    userId: string,
    itemName: string,
    shopName?: string,
  ): Promise<GroceryOperationResult> {
    const list = await this.loadOrCreateList(userId, shopName);
    const now = Date.now();
    const item: GroceryItem = { name: itemName.trim(), addedAt: now };
    const updatedList = {
      ...list,
      items: [...list.items, item],
      lastUpdated: now,
    };
    await this.saveList(userId, shopName, updatedList);
    return {
      type: "add_item",
      shopName: updatedList.shopName,
      items: updatedList.items,
      changed: true,
      itemName: item.name,
    };
  }

  // get the grocery list by shopName. If name not provided, just return "default" list
  async listItems(
    userId: string,
    shopName?: string,
  ): Promise<GroceryOperationResult> {
    const list = await this.loadOrCreateList(userId, shopName);
    return {
      type: "list_items",
      shopName: list.shopName,
      items: list.items,
      changed: false,
    };
  }

  // update item name. the name can also include quantity or other related information.
  // this is by design to make the name include as much information as possible
  async updateItem(
    userId: string,
    existingItemName: string,
    newItemName: string,
    shopName?: string,
  ): Promise<GroceryOperationResult> {
    const list = await this.loadOrCreateList(userId, shopName);
    const index = this.findItemIndex(list.items, existingItemName);
    if (index === -1) {
      return {
        type: "update_item",
        shopName: list.shopName,
        items: list.items,
        changed: false,
        itemName: newItemName.trim(),
        previousItemName: existingItemName.trim(),
      };
    }
    // we only update the name but not change it's added date
    const now = Date.now();
    const updatedItems = [...list.items];
    updatedItems[index] = {
      ...updatedItems[index],
      name: newItemName.trim(),
    };
    const updatedList = {
      ...list,
      items: updatedItems,
      lastUpdated: now,
    };
    await this.saveList(userId, shopName, updatedList);
    return {
      type: "update_item",
      shopName: updatedList.shopName,
      items: updatedList.items,
      changed: true,
      itemName: newItemName.trim(),
      previousItemName: existingItemName.trim(),
    };
  }

  async removeItem(
    userId: string,
    itemName: string,
    shopName?: string,
  ): Promise<GroceryOperationResult> {
    const list = await this.loadOrCreateList(userId, shopName);
    const index = this.findItemIndex(list.items, itemName);
    if (index === -1) {
      return {
        type: "remove_item",
        shopName: list.shopName,
        items: list.items,
        changed: false,
        itemName: itemName.trim(),
      };
    }
    const now = Date.now();
    const updatedItems = list.items.filter(
      (_, itemIndex) => itemIndex !== index,
    );
    const updatedList = {
      ...list,
      items: updatedItems,
      lastUpdated: now,
    };
    await this.saveList(userId, shopName, updatedList);
    return {
      type: "remove_item",
      shopName: updatedList.shopName,
      items: updatedList.items,
      changed: true,
      itemName: itemName.trim(),
    };
  }

  async clearList(
    userId: string,
    shopName?: string,
  ): Promise<GroceryOperationResult> {
    const list = await this.loadOrCreateList(userId, shopName);
    const now = Date.now();
    const updatedList = {
      ...list,
      items: [],
      lastUpdated: now,
    };
    await this.saveList(userId, shopName, updatedList);
    return {
      type: "clear_list",
      shopName: updatedList.shopName,
      items: updatedList.items,
      changed: list.items.length > 0,
    };
  }

  private async loadOrCreateList(
    userId: string,
    shopName?: string,
  ): Promise<GroceryListDoc> {
    const shopId = normalizeShopId(shopName);
    const list = await this.store.getList(userId, shopId);
    if (list) return list;
    const now = Date.now();
    return {
      shopName: getDisplayShopName(shopName),
      items: [],
      lastUpdated: now,
    };
  }

  private async saveList(
    userId: string,
    shopName: string | undefined,
    list: GroceryListDoc,
  ): Promise<void> {
    await this.store.saveList(userId, normalizeShopId(shopName), list);
  }

  private findItemIndex(items: GroceryItem[], itemName: string): number {
    const normalizedName = itemName.trim().toLowerCase();
    return items.findIndex(
      (item) => item.name.trim().toLowerCase() === normalizedName,
    );
  }
}
