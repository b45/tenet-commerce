/**
 * Tenet Commerce — Offline POS Native IndexedDB Storage Engine
 * W3C Standard IndexedDB Promise Wrapper without external bloatware.
 * Enforces strict tenant data isolation in browser client storage.
 */

import type { Product, CartItem } from "@/features/pos/types";

const DB_NAME = "tenet_pos_offline_db";
const DB_VERSION = 1;

export interface CachedProductRecord {
  sku: string;
  tenant_slug: string;
  product: Product;
  cached_at: string;
}

export interface CartDraftRecord {
  tenant_slug: string;
  items: CartItem[];
  updated_at: string;
}

export interface SyncMetaRecord {
  key: string;
  tenant_slug: string;
  last_sync_at: string;
  total_items: number;
}

let dbInstance: IDBDatabase | null = null;

export function openPOSDatabase(): Promise<IDBDatabase> {
  if (typeof window === "undefined" || !window.indexedDB) {
    return Promise.reject(new Error("IndexedDB is not available in this environment"));
  }

  if (dbInstance) {
    return Promise.resolve(dbInstance);
  }

  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result;

      // 1. Catalog Products Store: primary key [tenant_slug+sku] or compound key
      if (!db.objectStoreNames.contains("catalog_products")) {
        const productStore = db.createObjectStore("catalog_products", {
          keyPath: ["tenant_slug", "sku"],
        });
        productStore.createIndex("tenant_slug", "tenant_slug", { unique: false });
        productStore.createIndex("category_id", "product.category_id", { unique: false });
      }

      // 2. Cart Drafts Store: keyed by tenant_slug
      if (!db.objectStoreNames.contains("cart_drafts")) {
        db.createObjectStore("cart_drafts", { keyPath: "tenant_slug" });
      }

      // 3. Sync Meta Store: keyed by [tenant_slug, key]
      if (!db.objectStoreNames.contains("sync_meta")) {
        db.createObjectStore("sync_meta", { keyPath: ["tenant_slug", "key"] });
      }
    };

    request.onsuccess = (event) => {
      dbInstance = (event.target as IDBOpenDBRequest).result;
      dbInstance.onversionchange = () => {
        dbInstance?.close();
        dbInstance = null;
      };
      resolve(dbInstance);
    };

    request.onerror = (event) => {
      reject((event.target as IDBOpenDBRequest).error);
    };
  });
}

/**
 * Save product catalog snapshot to IndexedDB under the active tenant
 */
export async function saveCatalogToCache(
  tenantSlug: string,
  products: Product[]
): Promise<void> {
  if (!tenantSlug || !Array.isArray(products)) return;

  const db = await openPOSDatabase();
  const tx = db.transaction(["catalog_products", "sync_meta"], "readwrite");
  const productStore = tx.objectStore("catalog_products");
  const metaStore = tx.objectStore("sync_meta");

  const now = new Date().toISOString();

  // 1. Clear previous catalog for this tenant to remove deleted products
  const index = productStore.index("tenant_slug");
  const keyRange = IDBKeyRange.only(tenantSlug);
  const cursorRequest = index.openKeyCursor(keyRange);

  await new Promise<void>((resolve, reject) => {
    cursorRequest.onsuccess = (event) => {
      const cursor = (event.target as IDBRequest<IDBCursor>).result;
      if (cursor) {
        productStore.delete(cursor.primaryKey);
        cursor.continue();
      } else {
        resolve();
      }
    };
    cursorRequest.onerror = () => reject(cursorRequest.error);
  });

  // 2. Insert new products
  for (const product of products) {
    const record: CachedProductRecord = {
      sku: product.sku,
      tenant_slug: tenantSlug,
      product,
      cached_at: now,
    };
    productStore.put(record);
  }

  // 3. Record sync metadata
  const meta: SyncMetaRecord = {
    key: "catalog",
    tenant_slug: tenantSlug,
    last_sync_at: now,
    total_items: products.length,
  };
  metaStore.put(meta);

  return new Promise<void>((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
    tx.onabort = () => reject(tx.error);
  });
}

/**
 * Retrieve cached products for a specific tenant from IndexedDB
 */
export async function getCatalogFromCache(
  tenantSlug: string
): Promise<{ products: Product[]; lastSyncAt: string | null }> {
  if (!tenantSlug) return { products: [], lastSyncAt: null };

  const db = await openPOSDatabase();
  const tx = db.transaction(["catalog_products", "sync_meta"], "readonly");
  const productStore = tx.objectStore("catalog_products");
  const metaStore = tx.objectStore("sync_meta");

  const index = productStore.index("tenant_slug");
  const keyRange = IDBKeyRange.only(tenantSlug);
  const getProductsPromise = new Promise<Product[]>((resolve, reject) => {
    const products: Product[] = [];
    const request = index.openCursor(keyRange);

    request.onsuccess = (event) => {
      const cursor = (event.target as IDBRequest<IDBCursorWithValue>).result;
      if (cursor) {
        const record = cursor.value as CachedProductRecord;
        if (record && record.product) {
          products.push(record.product);
        }
        cursor.continue();
      } else {
        resolve(products);
      }
    };

    request.onerror = () => reject(request.error);
  });

  const getMetaPromise = new Promise<string | null>((resolve) => {
    const request = metaStore.get([tenantSlug, "catalog"]);
    request.onsuccess = () => {
      const record = request.result as SyncMetaRecord | undefined;
      resolve(record?.last_sync_at || null);
    };
    request.onerror = () => resolve(null);
  });

  const [products, lastSyncAt] = await Promise.all([getProductsPromise, getMetaPromise]);
  return { products, lastSyncAt };
}

/**
 * Save current cart draft to IndexedDB to survive page refresh / power cuts
 */
export async function saveCartDraft(
  tenantSlug: string,
  items: CartItem[]
): Promise<void> {
  if (!tenantSlug) return;

  const db = await openPOSDatabase();
  const tx = db.transaction("cart_drafts", "readwrite");
  const store = tx.objectStore("cart_drafts");

  const record: CartDraftRecord = {
    tenant_slug: tenantSlug,
    items,
    updated_at: new Date().toISOString(),
  };

  store.put(record);

  return new Promise<void>((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

/**
 * Retrieve saved cart draft for a tenant
 */
export async function getCartDraft(tenantSlug: string): Promise<CartItem[]> {
  if (!tenantSlug) return [];

  const db = await openPOSDatabase();
  const tx = db.transaction("cart_drafts", "readonly");
  const store = tx.objectStore("cart_drafts");

  return new Promise((resolve, reject) => {
    const request = store.get(tenantSlug);
    request.onsuccess = () => {
      const record = request.result as CartDraftRecord | undefined;
      resolve(record?.items || []);
    };
    request.onerror = () => reject(request.error);
  });
}

/**
 * Clear cart draft upon checkout completion or manual cart reset
 */
export async function clearCartDraft(tenantSlug: string): Promise<void> {
  if (!tenantSlug) return;

  const db = await openPOSDatabase();
  const tx = db.transaction("cart_drafts", "readwrite");
  const store = tx.objectStore("cart_drafts");

  store.delete(tenantSlug);

  return new Promise<void>((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
  });
}

/**
 * Remove every offline record when the authenticated session ends.
 * Keeping tenant-scoped records across users could expose the previous
 * user's catalog or draft cart after logout or session expiry.
 */
export async function clearAllOfflineState(): Promise<void> {
  if (typeof window === "undefined" || !window.indexedDB) return;

  const db = await openPOSDatabase();
  const tx = db.transaction(["catalog_products", "cart_drafts", "sync_meta"], "readwrite");
  tx.objectStore("catalog_products").clear();
  tx.objectStore("cart_drafts").clear();
  tx.objectStore("sync_meta").clear();

  return new Promise<void>((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
    tx.onabort = () => reject(tx.error);
  });
}
