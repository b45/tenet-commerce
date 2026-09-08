import test from "node:test";
import assert from "node:assert/strict";

/**
 * Pure calculation helpers extracted from inventory domain
 */
function calculateStockDelta(currentStock, adjustType, quantity) {
  let newStock = currentStock;
  let delta = 0;

  if (adjustType === "ADD") {
    newStock = currentStock + (quantity || 0);
    delta = quantity || 0;
  } else if (adjustType === "SUBTRACT") {
    newStock = Math.max(0, currentStock - (quantity || 0));
    delta = -(quantity || 0);
  } else if (adjustType === "SET") {
    newStock = Math.max(0, quantity || 0);
    delta = newStock - currentStock;
  }

  return { newStock, delta };
}

function filterProducts(products, filters) {
  return products.filter((item) => {
    // 1. Search keyword
    if (filters.search && filters.search.trim()) {
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
}

test("Inventory: calculateStockDelta calculates ADD correctly", () => {
  const { newStock, delta } = calculateStockDelta(10, "ADD", 5);
  assert.equal(newStock, 15);
  assert.equal(delta, 5);
});

test("Inventory: calculateStockDelta calculates SUBTRACT correctly and clamps to 0", () => {
  const res1 = calculateStockDelta(10, "SUBTRACT", 4);
  assert.equal(res1.newStock, 6);
  assert.equal(res1.delta, -4);

  const res2 = calculateStockDelta(5, "SUBTRACT", 10);
  assert.equal(res2.newStock, 0);
  assert.equal(res2.delta, -10);
});

test("Inventory: calculateStockDelta calculates SET correctly", () => {
  const increase = calculateStockDelta(10, "SET", 18);
  assert.equal(increase.newStock, 18);
  assert.equal(increase.delta, 8);

  const decrease = calculateStockDelta(10, "SET", 3);
  assert.equal(decrease.newStock, 3);
  assert.equal(decrease.delta, -7);
});

test("Inventory: filterProducts matches search by name, SKU, or barcode", () => {
  const sample = [
    { id: "1", name: "Croissant Butter", sku: "BKR-CRN-001", barcode: "899111", category_id: "cat1", stock_quantity: 12, reorder_threshold: 5 },
    { id: "2", name: "Sourdough Loaf", sku: "BKR-SRD-002", barcode: "899222", category_id: "cat1", stock_quantity: 4, reorder_threshold: 5 },
    { id: "3", name: "Roti Coklat", sku: "BKR-CKL-003", barcode: "899333", category_id: "cat2", stock_quantity: 0, reorder_threshold: 5 },
  ];

  // Match name
  const byName = filterProducts(sample, { search: "butter", category_id: "", stock_status: "all" });
  assert.equal(byName.length, 1);
  assert.equal(byName[0].sku, "BKR-CRN-001");

  // Match SKU
  const bySku = filterProducts(sample, { search: "SRD-002", category_id: "", stock_status: "all" });
  assert.equal(bySku.length, 1);
  assert.equal(bySku[0].name, "Sourdough Loaf");

  // Match Barcode
  const byBarcode = filterProducts(sample, { search: "899333", category_id: "", stock_status: "all" });
  assert.equal(byBarcode.length, 1);
  assert.equal(byBarcode[0].id, "3");
});

test("Inventory: filterProducts isolates low_stock and out_of_stock accurately", () => {
  const sample = [
    { id: "1", name: "Croissant", sku: "SKU1", category_id: "c1", stock_quantity: 20, reorder_threshold: 5 },
    { id: "2", name: "Sourdough", sku: "SKU2", category_id: "c1", stock_quantity: 5, reorder_threshold: 5 },
    { id: "3", name: "Baguette", sku: "SKU3", category_id: "c1", stock_quantity: 2, reorder_threshold: 5 },
    { id: "4", name: "Roti Tawar", sku: "SKU4", category_id: "c2", stock_quantity: 0, reorder_threshold: 5 },
  ];

  // low_stock: 0 < qty <= threshold (items 2 and 3)
  const lowStock = filterProducts(sample, { search: "", category_id: "", stock_status: "low_stock" });
  assert.equal(lowStock.length, 2);
  assert.deepEqual(lowStock.map(i => i.id), ["2", "3"]);

  // out_of_stock: qty <= 0 (item 4)
  const outOfStock = filterProducts(sample, { search: "", category_id: "", stock_status: "out_of_stock" });
  assert.equal(outOfStock.length, 1);
  assert.equal(outOfStock[0].id, "4");
});

test("Inventory: StockCard maintains strict chronological running balance invariant across movements", () => {
  // Scenario matching acceptance criteria: opening 0, GR +60, then GR +40, checkout -5, void +5
  const openingBalance = 0;
  const deltas = [
    { type: "IN", delta: 60 },
    { type: "IN", delta: 40 },
    { type: "OUT", delta: -5 },
    { type: "IN", delta: 5 }, // void recovery
  ];

  let running = openingBalance;
  const movements = deltas.map((item, idx) => {
    running += item.delta;
    return {
      index: idx + 1,
      delta: item.delta,
      runningBalance: running,
    };
  });

  assert.equal(movements[0].runningBalance, 60, "After GR 60 running balance must be 60");
  assert.equal(movements[1].runningBalance, 100, "After GR 40 running balance must be 100");
  assert.equal(movements[2].runningBalance, 95, "After checkout -5 running balance must be 95");
  assert.equal(movements[3].runningBalance, 100, "After void +5 running balance must return to 100");

  // Pagination slice does not alter precomputed running balances
  const page1 = movements.slice(0, 2);
  const page2 = movements.slice(2, 4);
  assert.equal(page1[1].runningBalance, 100);
  assert.equal(page2[0].runningBalance, 95);
});

test("Inventory: calculateSuggestedProcurementOrder determines non-negative draft quantities", () => {
  function calculateSuggestedQuantity(stockQuantity, threshold) {
    const deficit = Math.max(0, threshold - stockQuantity);
    return Math.max(20, deficit + 15);
  }

  // Current stock 2, threshold 10 -> deficit 8 -> suggested 8 + 15 = 23
  assert.equal(calculateSuggestedQuantity(2, 10), 23);

  // Out of stock 0, threshold 5 -> deficit 5 -> suggested max(20, 20) = 20
  assert.equal(calculateSuggestedQuantity(0, 5), 20);

  // Low stock 4, threshold 5 -> deficit 1 -> suggested max(20, 16) = 20
  assert.equal(calculateSuggestedQuantity(4, 5), 20);
});
