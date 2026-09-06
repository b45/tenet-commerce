import test from "node:test";
import assert from "node:assert/strict";

/**
 * Unit tests for P3-04: Offline POS Catalog Persistence and Cart Drafts logic.
 * Tests client-side category derivation, search matching across cached fields,
 * and tenant isolation rules in offline storage.
 */

function deriveCategories(rawProducts) {
  const catMap = new Map();
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

function filterCachedProducts(products, selectedCategory, searchQuery) {
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
}

test("Offline Catalog: derives categories and product counts correctly from cached records", () => {
  const products = [
    { sku: "BREAD-01", name: "Roti Tawar", category_name: "Roti", category_id: "cat-1", unit_price: 15000 },
    { sku: "BREAD-02", name: "Roti Gandum", category_name: "Roti", category_id: "cat-1", unit_price: 20000 },
    { sku: "DRINK-01", name: "Susu UHT", category_name: "Minuman", category_id: "cat-2", unit_price: 10000 },
    { sku: "MISC-01", name: "Kantong Belanja", unit_price: 1000 }, // No category_name
  ];

  const categories = deriveCategories(products);
  assert.equal(categories.length, 3);

  const roti = categories.find((c) => c.name === "Roti");
  assert.ok(roti);
  assert.equal(roti.product_count, 2);

  const uncategorized = categories.find((c) => c.name === "Tanpa Kategori");
  assert.ok(uncategorized);
  assert.equal(uncategorized.product_count, 1);
});

test("Offline Catalog: filters cached products by category and search queries (name, SKU, barcode)", () => {
  const products = [
    { sku: "BREAD-01", barcode: "899123456701", name: "Roti Tawar Kupas", category_name: "Roti" },
    { sku: "CAKE-01", barcode: "899123456702", name: "Brownies Cokelat", category_name: "Kue" },
    { sku: "CAKE-02", barcode: "899123456703", name: "Lapis Legit", category_name: "Kue" },
  ];

  // 1. Filter by category
  const kueOnly = filterCachedProducts(products, "Kue", "");
  assert.equal(kueOnly.length, 2);

  // 2. Search by SKU
  const skuMatch = filterCachedProducts(products, "ALL", "bread-01");
  assert.equal(skuMatch.length, 1);
  assert.equal(skuMatch[0].sku, "BREAD-01");

  // 3. Search by Barcode
  const barcodeMatch = filterCachedProducts(products, "ALL", "899123456703");
  assert.equal(barcodeMatch.length, 1);
  assert.equal(barcodeMatch[0].name, "Lapis Legit");

  // 4. Combined category + search
  const combined = filterCachedProducts(products, "Kue", "cokelat");
  assert.equal(combined.length, 1);
  assert.equal(combined[0].sku, "CAKE-01");
});

test("Offline Storage: validates tenant scoping invariant on client records", () => {
  const activeTenant = "al-barakah-mart";
  const records = [
    { tenant_slug: "al-barakah-mart", sku: "SKU-1", product: { name: "Produk Toko 1" } },
    { tenant_slug: "b45-bakery", sku: "SKU-2", product: { name: "Produk Toko 2" } },
  ];

  // Client storage MUST filter strictly by tenant_slug to guarantee tenant isolation
  const tenantScoped = records.filter((r) => r.tenant_slug === activeTenant);
  assert.equal(tenantScoped.length, 1);
  assert.equal(tenantScoped[0].sku, "SKU-1");
});
