import { Firestore } from "firebase-admin/firestore";
import { getFirestoreDb } from "../../core/firestore";
import {
  ShoppingItem,
  ShoppingListDoc,
  ShoppingListDocSchema,
  ShoppingOperationResult,
  ShoppingCategory,
  ShoppingCategorySchema,
} from "./types";

export interface ShoppingListStore {
  getList(
    userId: string,
    category: ShoppingCategory,
  ): Promise<ShoppingListDoc | null>;
  saveList(
    userId: string,
    category: ShoppingCategory,
    list: ShoppingListDoc,
  ): Promise<void>;
}

export const normalizeShoppingCategory = (
  category?: string,
): ShoppingCategory => {
  if (!category?.trim()) return "grocery";
  return ShoppingCategorySchema.parse(category);
};

export const createFirestoreShoppingListStore = (
  db: Firestore = getFirestoreDb(),
): ShoppingListStore => ({
  async getList(userId, category) {
    const snapshot = await db
      .doc(`users/${userId}/shoppingLists/${category}`)
      .get();
    if (!snapshot.exists) return null;
    return ShoppingListDocSchema.parse(snapshot.data());
  },

  async saveList(userId, category, list) {
    const parsedList = ShoppingListDocSchema.parse(list);
    await db.doc(`users/${userId}/shoppingLists/${category}`).set(parsedList, {
      merge: true,
    });
  },
});

export class ShoppingRepository {
  constructor(
    private readonly store: ShoppingListStore = createFirestoreShoppingListStore(),
  ) {}

  // The current strategy is that, with any updates, we overwrite the whole list,
  // rather than update part of the document.
  async addItem(
    userId: string,
    itemName: string,
    category?: string,
  ): Promise<ShoppingOperationResult> {
    const results = await this.addItems(userId, [itemName], category);
    return results[0];
  }

  async addItems(
    userId: string,
    itemNames: string[],
    category?: string,
  ): Promise<ShoppingOperationResult[]> {
    const list = await this.loadOrCreateList(userId, category);
    const now = Date.now();
    const itemsToAdd = itemNames.map((itemName) => ({
      name: itemName.trim(),
      addedAt: now,
    }));
    const updatedList = {
      ...list,
      items: [...list.items, ...itemsToAdd],
      lastUpdated: now,
    };
    await this.saveList(userId, updatedList.category, updatedList);
    return itemsToAdd.map((item) => ({
      type: "add_item",
      category: updatedList.category,
      items: updatedList.items,
      changed: true,
      itemName: item.name,
    }));
  }

  // Get the shopping list by controlled category. Unknown categories fall back to "other".
  async listItems(
    userId: string,
    category?: string,
  ): Promise<ShoppingOperationResult> {
    const list = await this.loadOrCreateList(userId, category);
    return {
      type: "list_items",
      category: list.category,
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
    category?: string,
  ): Promise<ShoppingOperationResult> {
    const list = await this.loadOrCreateList(userId, category);
    const index = this.findItemIndex(list.items, existingItemName);
    if (index === -1) {
      return {
        type: "update_item",
        category: list.category,
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
    await this.saveList(userId, updatedList.category, updatedList);
    return {
      type: "update_item",
      category: updatedList.category,
      items: updatedList.items,
      changed: true,
      itemName: newItemName.trim(),
      previousItemName: existingItemName.trim(),
    };
  }

  async removeItem(
    userId: string,
    itemName: string,
    category?: string,
  ): Promise<ShoppingOperationResult> {
    const list = await this.loadOrCreateList(userId, category);
    const index = this.findItemIndex(list.items, itemName);
    if (index === -1) {
      return {
        type: "remove_item",
        category: list.category,
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
    await this.saveList(userId, updatedList.category, updatedList);
    return {
      type: "remove_item",
      category: updatedList.category,
      items: updatedList.items,
      changed: true,
      itemName: itemName.trim(),
    };
  }

  async clearList(
    userId: string,
    category?: string,
  ): Promise<ShoppingOperationResult> {
    const list = await this.loadOrCreateList(userId, category);
    const now = Date.now();
    const updatedList = {
      ...list,
      items: [],
      lastUpdated: now,
    };
    await this.saveList(userId, updatedList.category, updatedList);
    return {
      type: "clear_list",
      category: updatedList.category,
      items: updatedList.items,
      changed: list.items.length > 0,
    };
  }

  private async loadOrCreateList(
    userId: string,
    category?: string,
  ): Promise<ShoppingListDoc> {
    const normalizedCategory = normalizeShoppingCategory(category);
    const list = await this.store.getList(userId, normalizedCategory);
    if (list) return list;
    const now = Date.now();
    return {
      category: normalizedCategory,
      items: [],
      lastUpdated: now,
    };
  }

  private async saveList(
    userId: string,
    category: ShoppingCategory,
    list: ShoppingListDoc,
  ): Promise<void> {
    await this.store.saveList(userId, category, list);
  }

  private findItemIndex(items: ShoppingItem[], itemName: string): number {
    const normalizedName = itemName.trim().toLowerCase();
    return items.findIndex(
      (item) => item.name.trim().toLowerCase() === normalizedName,
    );
  }
}
