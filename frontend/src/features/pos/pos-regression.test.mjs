import test from "node:test";
import assert from "node:assert/strict";
import { createCheckoutController, isDefiniteRejection } from "./checkout-controller.ts";
import { MAX_LINE_ITEM_QTY, MAX_TENDER_AMOUNT, MAX_TRANSACTION_AMOUNT } from "../../lib/money.ts";

/**
 * TPC-027 Regression Suite:
 * Verifies that the existing POS client behavior remains invariant after backend
 * hardening (ADR 002 integer money precision, tenant search_path isolation, and Redis idempotency).
 */

const makeCart = () => [
  { product: { sku: "SKU-BEV-01", name: "Kopi Arabika", stock_quantity: 10, unit_price: 25000 }, quantity: 2, subtotal: 50000 },
  { product: { sku: "SKU-SNK-02", name: "Keripik Singkong", stock_quantity: 5, unit_price: 15000 }, quantity: 1, subtotal: 15000 },
];

const makeReceipt = (overrides = {}) => ({
  transaction_id: "txn-uuid-101",
  transaction_number: "TXN-20260907-0001",
  status: "COMPLETED",
  cashier_id: "cashier-001",
  subtotal_amount: 65000,
  tax_amount: 0,
  discount_amount: 0,
  total_amount: 65000,
  cash_tendered: 100000,
  change_amount: 35000,
  created_at: new Date().toISOString(),
  items: [
    { product_id: "prod-1", sku: "SKU-BEV-01", name: "Kopi Arabika", quantity: 2, unit_price: 25000, subtotal_amount: 50000 },
    { product_id: "prod-2", sku: "SKU-SNK-02", name: "Keripik Singkong", quantity: 1, unit_price: 15000, subtotal_amount: 15000 },
  ],
  ...overrides,
});

test("TPC-027: Cash checkout transition to review, submission, and receipt confirmation", async () => {
  let recordedBody = null;
  let recordedKey = null;

  const controller = createCheckoutController({
    createKey: () => "idem-test-key-abc",
    maxTotal: MAX_TRANSACTION_AMOUNT,
    maxTender: MAX_TENDER_AMOUNT,
    maxQuantity: MAX_LINE_ITEM_QTY,
    send: async (body, key) => {
      recordedBody = body;
      recordedKey = key;
      return {
        success: true,
        data: makeReceipt({
          cash_tendered: body.cash_tendered,
          change_amount: body.cash_tendered - 65000,
        }),
      };
    },
  });

  // 1. Initial State
  assert.equal(controller.getSnapshot().step, "idle");

  // 2. Open Tender Review
  const started = controller.startReview(65000);
  assert.equal(started, true);
  assert.equal(controller.getSnapshot().step, "review");
  assert.equal(controller.getSnapshot().cashTendered, 65000);

  // 3. Adjust Cash Tendered to 100,000 IDR
  controller.setCashTendered(100000);
  assert.equal(controller.getSnapshot().cashTendered, 100000);

  // 4. Submit Checkout
  const cart = makeCart();
  await controller.submitCheckout(cart, 65000);

  assert.equal(recordedKey, "idem-test-key-abc");
  assert.equal(recordedBody.payment_method, "CASH");
  assert.equal(recordedBody.cash_tendered, 100000);
  assert.equal(recordedBody.items.length, 2);

  // 5. Verify Completed Status and Authoritative Server Receipt
  const snapshot = controller.getSnapshot();
  assert.equal(snapshot.step, "completed");
  assert.notEqual(snapshot.receipt, null);
  assert.equal(snapshot.receipt.transaction_number, "TXN-20260907-0001");
  assert.equal(snapshot.receipt.change_amount, 35000);

  // 6. Dismiss Receipt on New Transaction clears cart and returns to idle
  let cartCleared = false;
  const finished = controller.finishCompleted(() => { cartCleared = true; });
  assert.equal(finished, true);
  assert.equal(cartCleared, true);
  assert.equal(controller.getSnapshot().step, "idle");
});

test("TPC-027: Duplicate click / concurrent submission guard preserves single command identity", async () => {
  let callCount = 0;
  let delayedResolve;

  const controller = createCheckoutController({
    createKey: () => "idem-duplicate-prevent-key",
    maxTotal: MAX_TRANSACTION_AMOUNT,
    maxTender: MAX_TENDER_AMOUNT,
    maxQuantity: MAX_LINE_ITEM_QTY,
    send: async () => {
      callCount++;
      return new Promise((resolve) => { delayedResolve = resolve; });
    },
  });

  controller.startReview(50000);
  controller.setCashTendered(50000);

  const cart = [makeCart()[0]];
  // Trigger two rapid submissions before network responds
  const promise1 = controller.submitCheckout(cart, 50000);
  const promise2 = controller.submitCheckout(cart, 50000);

  assert.equal(controller.getSnapshot().step, "submitting");
  assert.equal(callCount, 1, "send must be invoked exactly once on concurrent double-click");

  delayedResolve({
    success: true,
    data: makeReceipt({
      total_amount: 50000,
      subtotal_amount: 50000,
      cash_tendered: 50000,
      change_amount: 0,
      items: [{ product_id: "p1", sku: "SKU-BEV-01", quantity: 2, unit_price: 25000, subtotal_amount: 50000 }],
    }),
  });

  await Promise.all([promise1, promise2]);
  assert.equal(controller.getSnapshot().step, "completed");
});

test("TPC-027: Definite stock depletion rejection allows operator to edit cart without locking UI", async () => {
  const controller = createCheckoutController({
    createKey: () => "idem-stock-rejection-key",
    maxTotal: MAX_TRANSACTION_AMOUNT,
    maxTender: MAX_TENDER_AMOUNT,
    maxQuantity: MAX_LINE_ITEM_QTY,
    send: async () => ({
      success: false,
      data: null,
      error: {
        status: 409,
        code: "INSUFFICIENT_STOCK",
        message: "Stok tidak mencukupi untuk SKU-BEV-01",
      },
    }),
  });

  controller.startReview(50000);
  await controller.submitCheckout([makeCart()[0]], 50000);

  const snapshot = controller.getSnapshot();
  assert.equal(snapshot.step, "rejected");
  assert.equal(isDefiniteRejection({ status: 409, code: "INSUFFICIENT_STOCK", message: "" }), true);

  // Review can be safely closed so cashier can adjust quantity
  const closed = controller.closeReview();
  assert.equal(closed, true);
  assert.equal(controller.getSnapshot().step, "idle");
});

test("TPC-027: Transient timeout / unknown network failure hard-locks review to prevent double-charging", async () => {
  const controller = createCheckoutController({
    createKey: () => "idem-timeout-lock-key",
    maxTotal: MAX_TRANSACTION_AMOUNT,
    maxTender: MAX_TENDER_AMOUNT,
    maxQuantity: MAX_LINE_ITEM_QTY,
    send: async () => ({
      success: false,
      data: null,
      error: {
        status: 504,
        code: "GATEWAY_TIMEOUT",
        message: "Upstream timeout",
      },
    }),
  });

  controller.startReview(50000);
  await controller.submitCheckout([makeCart()[0]], 50000);

  const snapshot = controller.getSnapshot();
  assert.equal(snapshot.step, "unknown_error");

  // In unknown state, review CANNOT be dismissed or retried blindly
  assert.equal(controller.closeReview(), false);
  assert.equal(controller.startReview(50000), false);
  assert.equal(controller.finishCompleted(() => assert.fail("Must not clear cart")), false);
  assert.equal(snapshot.idempotencyKey, "idem-timeout-lock-key");
});
