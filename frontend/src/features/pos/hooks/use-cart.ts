"use client";

import * as React from "react";
import { useAuth } from "@/features/auth/hooks/use-auth";
import { saveCartDraft, getCartDraft, clearCartDraft } from "@/lib/offline/db";
import type { Product, CartItem } from "../types";

export function useCart() {
  const { user } = useAuth();
  const tenantSlug = user?.tenant_slug || "default";

  const [items, setItems] = React.useState<CartItem[]>([]);
  const isInitialMount = React.useRef(true);

  // 1. Restore Cart Draft from IndexedDB on startup
  React.useEffect(() => {
    let isMounted = true;

    async function restoreDraft() {
      if (typeof window === "undefined" || !window.indexedDB) return;
      try {
        const draftItems = await getCartDraft(tenantSlug);
        if (isMounted && draftItems && draftItems.length > 0) {
          setItems(draftItems);
        }
      } catch {
        // Storage unreadable, keep empty cart
      } finally {
        isInitialMount.current = false;
      }
    }

    restoreDraft();

    return () => {
      isMounted = false;
    };
  }, [tenantSlug]);

  // 2. Persist Cart Draft to IndexedDB whenever items change (after initial mount)
  React.useEffect(() => {
    if (isInitialMount.current) return;
    if (typeof window === "undefined" || !window.indexedDB) return;

    if (items.length > 0) {
      saveCartDraft(tenantSlug, items).catch(() => {});
    } else {
      clearCartDraft(tenantSlug).catch(() => {});
    }
  }, [items, tenantSlug]);

  const addItem = React.useCallback((product: Product, quantity = 1) => {
    if (product.stock_quantity <= 0) return;

    setItems((prev) => {
      const existingIdx = prev.findIndex((item) => item.product.sku === product.sku);

      if (existingIdx >= 0) {
        const existing = prev[existingIdx];
        const newQty = Math.min(
          existing.quantity + quantity,
          product.stock_quantity
        );
        const updated = [...prev];
        updated[existingIdx] = {
          ...existing,
          quantity: newQty,
          subtotal: Math.round(newQty * product.unit_price),
        };
        return updated;
      }

      const initialQty = Math.min(quantity, product.stock_quantity);
      return [
        ...prev,
        {
          product,
          quantity: initialQty,
          subtotal: Math.round(initialQty * product.unit_price),
        },
      ];
    });
  }, []);

  const updateQuantity = React.useCallback((sku: string, qty: number) => {
    setItems((prev) => {
      return prev
        .map((item) => {
          if (item.product.sku !== sku) return item;
          const clampedQty = Math.max(1, Math.min(qty, item.product.stock_quantity));
          return {
            ...item,
            quantity: clampedQty,
            subtotal: Math.round(clampedQty * item.product.unit_price),
          };
        })
        .filter(Boolean);
    });
  }, []);

  const decrementItem = React.useCallback((sku: string) => {
    setItems((prev) => {
      const existing = prev.find((item) => item.product.sku === sku);
      if (!existing) return prev;

      if (existing.quantity > 1) {
        return prev.map((item) =>
          item.product.sku === sku
            ? {
                ...item,
                quantity: item.quantity - 1,
                subtotal: Math.round((item.quantity - 1) * item.product.unit_price),
              }
            : item
        );
      }

      return prev;
    });
  }, []);

  const removeItem = React.useCallback((sku: string) => {
    setItems((prev) => prev.filter((item) => item.product.sku !== sku));
  }, []);

  const clearCart = React.useCallback(() => {
    setItems([]);
  }, []);

  // Compute financial totals with integer precision
  const totals = React.useMemo(() => {
    const subtotal = items.reduce((sum, item) => sum + item.subtotal, 0);
    const tax = 0; // Tax is currently 0 in backend ledger
    const discount = 0;
    const total = Math.max(0, subtotal + tax - discount);
    const totalItems = items.reduce((sum, item) => sum + item.quantity, 0);

    return {
      subtotal,
      tax,
      discount,
      total,
      totalItems,
    };
  }, [items]);

  return {
    items,
    addItem,
    updateQuantity,
    decrementItem,
    removeItem,
    clearCart,
    totals,
  };
}
