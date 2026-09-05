"use client";

import * as React from "react";
import { apiClient } from "@/lib/api";
import { logger } from "@/lib/logger";
import type {
  InventoryProduct,
  Category,
  StockAdjustmentPayload,
  StockAdjustmentResponse,
  CreateProductPayload,
  UpdateProductPayload,
  InventoryFilter,
  StockStatusFilter,
} from "../types";

export function useInventory() {
  const [products, setProducts] = React.useState<InventoryProduct[]>([]);
  const [categories, setCategories] = React.useState<Category[]>([]);
  const [lowStockItems, setLowStockItems] = React.useState<InventoryProduct[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // Filters state
  const [filters, setFilters] = React.useState<InventoryFilter>({
    search: "",
    category_id: "",
    stock_status: "all",
  });

  // Fetch all initial data
  const fetchData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [prodRes, catRes, lowRes] = await Promise.all([
        apiClient.get<InventoryProduct[]>("/pos/products"),
        apiClient.get<Category[]>("/pos/categories"),
        apiClient.get<InventoryProduct[]>("/pos/inventory/low-stock"),
      ]);

      if (prodRes.success && prodRes.data) {
        setProducts(prodRes.data);
      } else if (prodRes.error) {
        setError(prodRes.error.message);
      }

      if (catRes.success && catRes.data) {
        setCategories(catRes.data);
      }

      if (lowRes.success && lowRes.data) {
        setLowStockItems(lowRes.data);
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to load inventory data";
      setError(msg);
      logger.error("Failed to load inventory", { error: msg });
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Filter setters
  const setSearch = React.useCallback((search: string) => {
    setFilters((prev) => ({ ...prev, search }));
  }, []);

  const setCategory = React.useCallback((category_id: string) => {
    setFilters((prev) => ({ ...prev, category_id }));
  }, []);

  const setStockStatus = React.useCallback((stock_status: StockStatusFilter) => {
    setFilters((prev) => ({ ...prev, stock_status }));
  }, []);

  // Filtered products list
  const filteredProducts = React.useMemo(() => {
    return products.filter((item) => {
      // 1. Search keyword (name, sku, barcode)
      if (filters.search.trim()) {
        const query = filters.search.toLowerCase().trim();
        const matchName = item.name.toLowerCase().includes(query);
        const matchSku = item.sku.toLowerCase().includes(query);
        const matchBarcode = item.barcode ? item.barcode.toLowerCase().includes(query) : false;
        if (!matchName && !matchSku && !matchBarcode) return false;
      }

      // 2. Category filter
      if (filters.category_id && item.category_id !== filters.category_id) {
        return false;
      }

      // 3. Stock status filter
      const threshold = item.reorder_threshold ?? 5;
      if (filters.stock_status === "low_stock") {
        if (item.stock_quantity <= 0 || item.stock_quantity > threshold) return false;
      } else if (filters.stock_status === "out_of_stock") {
        if (item.stock_quantity > 0) return false;
      }

      return true;
    });
  }, [products, filters]);

  // Mutation: Stock Adjustment (Opname)
  const adjustStock = React.useCallback(
    async (payload: StockAdjustmentPayload): Promise<{ success: boolean; data?: StockAdjustmentResponse; error?: string }> => {
      const idempotencyKey = `adj_${crypto.randomUUID()}`;
      try {
        const res = await apiClient.post<StockAdjustmentResponse>("/pos/inventory/adjust", payload, {
          headers: {
            "Idempotency-Key": idempotencyKey,
          },
        });

        if (res.success && res.data) {
          logger.audit("Stock adjusted successfully", {
            product_id: payload.product_id,
            delta: res.data.quantity_delta,
            new_qty: res.data.new_quantity,
          });
          await fetchData();
          return { success: true, data: res.data };
        }
        return { success: false, error: res.error?.message || "Stock adjustment failed" };
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : "Stock adjustment failed";
        return { success: false, error: msg };
      }
    },
    [fetchData]
  );

  // Mutation: Create Product
  const createProduct = React.useCallback(
    async (payload: CreateProductPayload): Promise<{ success: boolean; data?: InventoryProduct; error?: string }> => {
      try {
        const res = await apiClient.post<InventoryProduct>("/pos/products", payload);
        if (res.success && res.data) {
          logger.audit("Product created", { sku: payload.sku, name: payload.name });
          await fetchData();
          return { success: true, data: res.data };
        }
        return { success: false, error: res.error?.message || "Failed to create product" };
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : "Failed to create product";
        return { success: false, error: msg };
      }
    },
    [fetchData]
  );

  // Mutation: Update Product
  const updateProduct = React.useCallback(
    async (id: string, payload: UpdateProductPayload): Promise<{ success: boolean; data?: InventoryProduct; error?: string }> => {
      try {
        const res = await apiClient.put<InventoryProduct>(`/pos/products/${id}`, payload);
        if (res.success && res.data) {
          logger.audit("Product updated", { id, name: payload.name });
          await fetchData();
          return { success: true, data: res.data };
        }
        return { success: false, error: res.error?.message || "Failed to update product" };
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : "Failed to update product";
        return { success: false, error: msg };
      }
    },
    [fetchData]
  );

  // Mutation: Delete Product (Soft delete / deactivate)
  const deleteProduct = React.useCallback(
    async (id: string): Promise<{ success: boolean; error?: string }> => {
      try {
        const res = await apiClient.delete(`/pos/products/${id}`);
        if (res.success) {
          logger.audit("Product deactivated", { id });
          await fetchData();
          return { success: true };
        }
        return { success: false, error: res.error?.message || "Failed to delete product" };
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : "Failed to delete product";
        return { success: false, error: msg };
      }
    },
    [fetchData]
  );

  return {
    products,
    filteredProducts,
    categories,
    lowStockItems,
    loading,
    error,
    filters,
    setSearch,
    setCategory,
    setStockStatus,
    refresh: fetchData,
    adjustStock,
    createProduct,
    updateProduct,
    deleteProduct,
  };
}
