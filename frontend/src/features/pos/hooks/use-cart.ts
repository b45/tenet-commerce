"use client";

import * as React from "react";
import type { Product, CartItem } from "../types";

export function useCart() {
  const [items, setItems] = React.useState<CartItem[]>([]);

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
