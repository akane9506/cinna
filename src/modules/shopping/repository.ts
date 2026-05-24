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

type ShoppingItemUpdate = {
  existingItemName: string;
  newItemName: string;
};

type ShoppingAddItemsResult = Extract<
  ShoppingOperationResult,
  { type: "add_item" }
>;
type ShoppingListItemsResult = Extract<
  ShoppingOperationResult,
  { type: "list_items" }
>;
type ShoppingUpdateItemsResult = Extract<
  ShoppingOperationResult,
  { type: "update_item" }
>;
type ShoppingRemoveItemsResult = Extract<
  ShoppingOperationResult,
  { type: "remove_item" }
>;
type ShoppingClearListResult = Extract<
  ShoppingOperationResult,
  { type: "clear_list" }
>;

export class ShoppingRepository {
  constructor(
    private readonly store: ShoppingListStore = createFirestoreShoppingListStore(),
  ) {}

  // The current strategy is that, with any updates, we overwrite the whole list,
  // rather than update part of the document.
  async addItems(
    userId: string,
    itemNames: string[],
    category?: string,
  ): Promise<ShoppingAddItemsResult> {
    const list = await this.loadOrCreateList(userId, category);
    const normalizedItemNames = itemNames
      .map((itemName) => itemName.trim())
      .filter(Boolean);

    if (normalizedItemNames.length === 0) {
      return {
        type: "add_item",
        category: list.category,
        changed: false,
        itemNames: [],
      };
    }

    const now = Date.now();
    const itemsToAdd = normalizedItemNames.map((itemName) => ({
      name: itemName,
      addedAt: now,
    }));
    const updatedList = {
      ...list,
      items: [...list.items, ...itemsToAdd],
      lastUpdated: now,
    };
    await this.saveList(userId, updatedList.category, updatedList);
    return {
      type: "add_item",
      category: updatedList.category,
      changed: true,
      itemNames: itemsToAdd.map((item) => item.name),
    };
  }

  // Get the shopping list by controlled category. Unknown categories fall back to "other".
  async listItems(
    userId: string,
    category?: string,
  ): Promise<ShoppingListItemsResult> {
    const list = await this.loadOrCreateList(userId, category);
    return {
      type: "list_items",
      category: list.category,
      items: list.items,
      changed: false,
    };
  }

  async updateItems(
    userId: string,
    itemUpdates: ShoppingItemUpdate[],
    category?: string,
  ): Promise<ShoppingUpdateItemsResult> {
    const list = await this.loadOrCreateList(userId, category);
    const normalizedUpdates = itemUpdates
      .map((itemUpdate) => ({
        existingItemName: itemUpdate.existingItemName.trim(),
        newItemName: itemUpdate.newItemName.trim(),
      }))
      .filter(
        (itemUpdate) =>
          itemUpdate.existingItemName.length > 0 &&
          itemUpdate.newItemName.length > 0,
      );

    if (normalizedUpdates.length === 0) {
      return {
        type: "update_item",
        category: list.category,
        changed: false,
        itemNames: [],
        previousItemNames: [],
      };
    }

    const updatedItems = [...list.items];
    const changedUpdates: ShoppingItemUpdate[] = [];

    for (const itemUpdate of normalizedUpdates) {
      const index = this.findItemIndex(
        updatedItems,
        itemUpdate.existingItemName,
      );
      if (index === -1) continue;

      updatedItems[index] = {
        name: itemUpdate.newItemName,
        addedAt: Date.now(),
      };
      changedUpdates.push(itemUpdate);
    }

    if (changedUpdates.length === 0) {
      return {
        type: "update_item",
        category: list.category,
        changed: false,
        itemNames: normalizedUpdates.map(
          (itemUpdate) => itemUpdate.newItemName,
        ),
        previousItemNames: normalizedUpdates.map(
          (itemUpdate) => itemUpdate.existingItemName,
        ),
      };
    }

    const updatedList = {
      ...list,
      items: updatedItems,
      lastUpdated: Date.now(),
    };
    await this.saveList(userId, updatedList.category, updatedList);
    return {
      type: "update_item",
      category: updatedList.category,
      changed: true,
      itemNames: changedUpdates.map((itemUpdate) => itemUpdate.newItemName),
      previousItemNames: changedUpdates.map(
        (itemUpdate) => itemUpdate.existingItemName,
      ),
    };
  }

  async removeItems(
    userId: string,
    itemNames: string[],
    category?: string,
  ): Promise<ShoppingRemoveItemsResult> {
    const list = await this.loadOrCreateList(userId, category);
    const normalizedItemNames = itemNames
      .map((itemName) => itemName.trim())
      .filter(Boolean);

    if (normalizedItemNames.length === 0) {
      return {
        type: "remove_item",
        category: list.category,
        changed: false,
        itemNames: [],
      };
    }

    const requestedNames = new Set(
      normalizedItemNames.map((itemName) => itemName.toLowerCase()),
    );
    const removedItems: ShoppingItem[] = [];
    const updatedItems = list.items.filter((item) => {
      if (!requestedNames.has(item.name.trim().toLowerCase())) return true;
      removedItems.push(item);
      return false;
    });

    if (removedItems.length === 0) {
      return {
        type: "remove_item",
        category: list.category,
        changed: false,
        itemNames: normalizedItemNames,
      };
    }

    const now = Date.now();
    const updatedList = {
      ...list,
      items: updatedItems,
      lastUpdated: now,
    };
    await this.saveList(userId, updatedList.category, updatedList);
    return {
      type: "remove_item",
      category: updatedList.category,
      changed: true,
      itemNames: removedItems.map((item) => item.name),
    };
  }

  async clearList(
    userId: string,
    category?: string,
  ): Promise<ShoppingClearListResult> {
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
