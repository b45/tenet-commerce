import test from "node:test";
import assert from "node:assert/strict";

/**
 * Unit tests for POS domain cart logic, stock bounds clamping,
 * and financial change calculations.
 */

// Pure helper logic simulating useCart & useCheckout calculations
function calculateCartTotals(items) {
  const subtotal = items.reduce((sum, item) => sum + item.subtotal, 0);
  const tax = 0;
  const discount = 0;
  const total = Math.max(0, subtotal + tax - discount);
  const totalItems = items.reduce((sum, item) => sum + item.quantity, 0);

  return { subtotal, tax, discount, total, totalItems };
}

function addItemToCart(prevItems, product, quantity = 1) {
  if (product.stock_quantity <= 0) return prevItems;

  const existingIdx = prevItems.findIndex((i) => i.product.sku === product.sku);
  if (existingIdx >= 0) {
    const existing = prevItems[existingIdx];
    const newQty = Math.min(existing.quantity + quantity, product.stock_quantity);
    const next = [...prevItems];
    next[existingIdx] = {
      ...existing,
      quantity: newQty,
      subtotal: Math.round(newQty * product.unit_price),
    };
    return next;
  }

  const initialQty = Math.min(quantity, product.stock_quantity);
  return [
    ...prevItems,
    {
      product,
      quantity: initialQty,
      subtotal: Math.round(initialQty * product.unit_price),
    },
  ];
}

function calculateChange(cashTendered, totalAmount) {
  const change = Math.max(0, cashTendered - totalAmount);
  const shortage = Math.max(0, totalAmount - cashTendered);
  const isSufficient = cashTendered >= totalAmount;
  return { change, shortage, isSufficient };
}

test("Cart logic adds items and merges duplicate SKUs correctly", () => {
  const productA = {
    id: "p1",
    sku: "SKU-A",
    name: "Produk A",
    unit_price: 25000,
    stock_quantity: 10,
  };

  let cart = addItemToCart([], productA, 1);
  assert.equal(cart.length, 1);
  assert.equal(cart[0].quantity, 1);
  assert.equal(cart[0].subtotal, 25000);

  // Add same item again -> merges quantity to 2
  cart = addItemToCart(cart, productA, 2);
  assert.equal(cart.length, 1);
  assert.equal(cart[0].quantity, 3);
  assert.equal(cart[0].subtotal, 75000);
});

test("Cart logic strictly clamps quantity to stock_quantity", () => {
  const lowStockProduct = {
    id: "p2",
    sku: "SKU-LOW",
    name: "Produk Langka",
    unit_price: 50000,
    stock_quantity: 3,
  };

  let cart = addItemToCart([], lowStockProduct, 5); // Attempt to add 5
  assert.equal(cart[0].quantity, 3); // Clamped to 3
  assert.equal(cart[0].subtotal, 150000);

  // Attempt to add more
  cart = addItemToCart(cart, lowStockProduct, 1);
  assert.equal(cart[0].quantity, 3); // Still clamped to 3
});

test("Cart rejects products with zero stock", () => {
  const outOfStockProduct = {
    id: "p3",
    sku: "SKU-EMPTY",
    name: "Produk Habis",
    unit_price: 10000,
    stock_quantity: 0,
  };

  const cart = addItemToCart([], outOfStockProduct, 1);
  assert.equal(cart.length, 0);
});

test("Financial totals aggregate multiple items accurately", () => {
  const items = [
    { product: { sku: "A" }, quantity: 2, subtotal: 50000 },
    { product: { sku: "B" }, quantity: 1, subtotal: 35000 },
    { product: { sku: "C" }, quantity: 3, subtotal: 45000 },
  ];

  const totals = calculateCartTotals(items);
  assert.equal(totals.subtotal, 130000);
  assert.equal(totals.total, 130000);
  assert.equal(totals.totalItems, 6);
});

test("Cash tender change and shortage calculation", () => {
  const total = 75000;

  // Exact cash
  const exact = calculateChange(75000, total);
  assert.equal(exact.change, 0);
  assert.equal(exact.shortage, 0);
  assert.equal(exact.isSufficient, true);

  // Overpaid
  const over = calculateChange(100000, total);
  assert.equal(over.change, 25000);
  assert.equal(over.shortage, 0);
  assert.equal(over.isSufficient, true);

  // Shortage
  const short = calculateChange(50000, total);
  assert.equal(short.change, 0);
  assert.equal(short.shortage, 25000);
  assert.equal(short.isSufficient, false);
});
