"use client";

import * as React from "react";
import { apiClient } from "@/lib/api";
import { useAuth } from "@/features/auth/hooks/use-auth";
import {
  saveCatalogToCache,
  getCatalogFromCache,
} from "@/lib/offline/db";
import type { Product, Category } from "../types";

function deriveCategories(rawProducts: Product[]): Category[] {
  const catMap = new Map<string, { id: string; name: string; count: number }>();
  rawProducts.forEach((p) => {
    const catName = p.category_name || "Tanpa Kategori";
    const catId = p.category_id || "uncategorized";
    const existing = catMap.get(catName);
    if (existing) {
      existing.count += 1;
    } else {
      catMap.set(catName, { id: catId, name: catName, count: 1 });
    }
  });

  return Array.from(catMap.values()).map((c) => ({
    id: c.id,
    name: c.name,
    product_count: c.count,
  }));
}

export function useCatalog() {
  const { user } = useAuth();
  const tenantSlug = user?.tenant_slug || "default";

  const [products, setProducts] = React.useState<Product[]>([]);
  const [categories, setCategories] = React.useState<Category[]>([]);
  const [selectedCategory, setSelectedCategory] = React.useState<string>("ALL");
  const [searchQuery, setSearchQuery] = React.useState<string>("");
  const [isLoading, setIsLoading] = React.useState<boolean>(true);
  const [isOfflineCache, setIsOfflineCache] = React.useState<boolean>(false);
  const [lastSyncAt, setLastSyncAt] = React.useState<string | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  // 1. Initial Fast Read from IndexedDB (< 15ms)
  React.useEffect(() => {
    let isMounted = true;

    async function loadCachedCatalog() {
      if (typeof window === "undefined" || !window.indexedDB) return;
      try {
        const cached = await getCatalogFromCache(tenantSlug);
        if (isMounted && cached.products.length > 0) {
          setProducts(cached.products);
          setCategories(deriveCategories(cached.products));
          setLastSyncAt(cached.lastSyncAt);
          setIsOfflineCache(true);
          setIsLoading(false);
        }
      } catch {
        // Storage unreadable or unavailable, proceed to network fetch
      }
    }

    loadCachedCatalog();

    return () => {
      isMounted = false;
    };
  }, [tenantSlug]);

  // 2. Network Fetch & Sync to IndexedDB
  const fetchCatalog = React.useCallback(async () => {
    try {
      setError(null);
      const res = await apiClient.get<Product[]>("/pos/products");
      if (res.success && res.data) {
        const rawProducts = Array.isArray(res.data) ? res.data : [];
        setProducts(rawProducts);
        setCategories(deriveCategories(rawProducts));
        setIsOfflineCache(false);
        const now = new Date().toISOString();
        setLastSyncAt(now);

        // Atomically sync to browser IndexedDB for offline access
        if (typeof window !== "undefined" && window.indexedDB) {
          saveCatalogToCache(tenantSlug, rawProducts).catch(() => {
            // Background cache write failure should not block UI
          });
        }
      } else {
        // If API fails but we already have cached products, keep serving cache
        if (products.length === 0) {
          // Try reading cache as final fallback
          const cached = await getCatalogFromCache(tenantSlug);
          if (cached.products.length > 0) {
            setProducts(cached.products);
            setCategories(deriveCategories(cached.products));
            setLastSyncAt(cached.lastSyncAt);
            setIsOfflineCache(true);
          } else {
            setError(res.error?.message || "Gagal memuat katalog produk");
          }
        } else {
          setIsOfflineCache(true);
        }
      }
    } catch (err: unknown) {
      // Network disconnected / server down
      const cached = await getCatalogFromCache(tenantSlug).catch(() => ({ products: [], lastSyncAt: null }));
      if (cached.products.length > 0) {
        setProducts(cached.products);
        setCategories(deriveCategories(cached.products));
        setLastSyncAt(cached.lastSyncAt);
        setIsOfflineCache(true);
      } else {
        const msg = err instanceof Error ? err.message : "Gagal menghubungi server katalog";
        setError(msg);
      }
    } finally {
      setIsLoading(false);
    }
  }, [tenantSlug, products.length]);

  React.useEffect(() => {
    fetchCatalog();
  }, [fetchCatalog]);

  // Client-side search & category filtering
  const filteredProducts = React.useMemo(() => {
    let result = products;

    if (selectedCategory !== "ALL") {
      result = result.filter(
        (p) => (p.category_name || "Tanpa Kategori") === selectedCategory
      );
    }

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase().trim();
      result = result.filter(
        (p) =>
          p.name.toLowerCase().includes(q) ||
          p.sku.toLowerCase().includes(q) ||
          p.barcode.toLowerCase().includes(q)
      );
    }

    return result;
  }, [products, selectedCategory, searchQuery]);

  return {
    products,
    filteredProducts,
    categories,
    selectedCategory,
    setSelectedCategory,
    searchQuery,
    setSearchQuery,
    isLoading,
    isOfflineCache,
    lastSyncAt,
    error,
    refetch: fetchCatalog,
  };
}
