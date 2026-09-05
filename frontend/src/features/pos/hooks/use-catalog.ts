"use client";

import * as React from "react";
import { apiClient } from "@/lib/api";
import type { Product, Category } from "../types";

export function useCatalog() {
  const [products, setProducts] = React.useState<Product[]>([]);
  const [categories, setCategories] = React.useState<Category[]>([]);
  const [selectedCategory, setSelectedCategory] = React.useState<string>("ALL");
  const [searchQuery, setSearchQuery] = React.useState<string>("");
  const [isLoading, setIsLoading] = React.useState<boolean>(true);
  const [error, setError] = React.useState<string | null>(null);

  const fetchCatalog = React.useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const res = await apiClient.get<Product[]>("/pos/products");
      if (res.success && res.data) {
        const rawProducts = Array.isArray(res.data) ? res.data : [];
        setProducts(rawProducts);

        // Derive categories from product data
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

        const derivedCategories: Category[] = Array.from(catMap.values()).map((c) => ({
          id: c.id,
          name: c.name,
          product_count: c.count,
        }));

        setCategories(derivedCategories);
      } else {
        setError(res.error?.message || "Gagal memuat katalog produk");
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal menghubungi server katalog";
      setError(msg);
    } finally {
      setIsLoading(false);
    }
  }, []);

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
    error,
    refetch: fetchCatalog,
  };
}
