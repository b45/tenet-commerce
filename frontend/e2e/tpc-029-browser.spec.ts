import { expect, request, test } from "@playwright/test";

test.describe("TPC-029 browser language and responsive smoke", () => {
  test("login is usable by keyboard", async ({ page }) => {
    await page.goto("/login");
    await expect(page).toHaveTitle(/Tenet Commerce/i);
    await expect(page.getByRole("main")).toBeVisible();
    const firstControl = page.locator("main button, main input, main select").first();
    await firstControl.focus();
    await expect(firstControl).toBeFocused();
  });

  test("locale selector changes document direction to RTL for Arabic", async ({ page }) => {
    await page.goto("/login");
    const selector = page.getByRole("combobox");
    await expect(selector).toBeVisible();
    await selector.selectOption("ar");
    await expect(page.locator("html")).toHaveAttribute("dir", "rtl");
    await expect(page.locator("html")).toHaveAttribute("lang", "ar");
    await expect(page.getByRole("main")).toBeVisible();
  });
});

test.describe("TPC-029 real backend authentication", () => {
  test("logs into the seeded demo tenant through the real API", async ({ page }) => {
    await page.goto("/login");
    await page.locator("#tenant_slug").fill("al-barakah-mart");
    await page.locator("#email").fill("cashier1@albarakah.com");
    await page.locator("#password").fill("Password123!");
    await page.getByRole("button", { name: /login|masuk|تسجيل/i }).click();
    await expect(page).toHaveURL(/\/pos/);
    await expect(page.getByRole("main")).toBeVisible();
  });

  test("runs the seeded purchase, receipt, sale, void and ledger journey", async ({ page }) => {
    const apiURL = process.env.TPC_E2E_API_URL ?? "http://127.0.0.1:8081";
    const api = await request.newContext({ baseURL: apiURL });
    const key = `tpc029-${Date.now()}-${test.info().project.name}`;
    const login = await api.post("/api/v1/auth/login", {
      data: { tenant_slug: "al-barakah-mart", email: "superadmin@albarakah.com", password: "Password123!" },
    });
    expect(login.ok()).toBeTruthy();
    const loginBody = await login.json();
    const token = loginBody.data.access_token;
    const headers = { Authorization: `Bearer ${token}`, "X-Tenant-Slug": "al-barakah-mart" };
    const product = { id: "10000000-0000-0000-0000-000000000099" };
    const supplier = { id: "a1000000-0000-0000-0000-000000000001" };
    const certificate = { id: "c1000000-0000-0000-0000-000000000001" };

    const po = await api.post("/api/v1/supply-chain/purchase-orders", {
      headers: { ...headers, "Idempotency-Key": `${key}-po` },
      data: { supplier_id: supplier.id, compliance_cert_id: certificate.id, items: [{ product_id: product.id, quantity: 100, unit_cost: 10000 }] },
    });
    const poBody = await po.json();
    expect(po.status(), JSON.stringify(poBody)).toBe(201);
    const poID = poBody.data.id;
    for (const [quantity, suffix] of [[60, "gr60"], [40, "gr40"]] as const) {
      const receipt = await api.post("/api/v1/supply-chain/goods-receipts", {
        headers: { ...headers, "Idempotency-Key": `${key}-${suffix}` },
        data: { purchase_order_id: poID, items: [{ product_id: product.id, delivered_quantity: quantity, accepted_quantity: quantity, rejected_quantity: 0 }] },
      });
      expect(receipt.status()).toBe(201);
    }

    await page.goto("/pos");
    await expect(page.getByRole("main")).toBeVisible();
    const checkout = await api.post("/api/v1/pos/checkout", {
      headers: { ...headers, "Idempotency-Key": `${key}-sale` },
      data: { items: [{ sku: "SKU-DEMO-01", quantity: 1 }], payment_method: "CASH", cash_tendered: 15000 },
    });
    expect(checkout.status()).toBe(201);
    const transactionID = (await checkout.json()).data.transaction_id;
    const voided = await api.post(`/api/v1/pos/orders/${transactionID}/void`, {
      headers: { ...headers, "Idempotency-Key": `${key}-void` },
      data: { reason: "TPC-029 browser golden journey reversal" },
    });
    expect([200, 201]).toContain(voided.status());

    for (const route of ["/inventory", "/pos/orders", "/ledger/entries"]) {
      await page.goto(route);
      await expect(page.getByRole("main")).toBeVisible();
    }
    const journal = await api.get(`/api/v1/ledger/entries?source_document_type=POS_VOID`, { headers });
    expect(journal.ok()).toBeTruthy();
    await api.dispose();
  });
});
