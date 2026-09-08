import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";
import ts from "typescript";
import { createCheckoutController } from "./checkout-controller.ts";
import { MAX_LINE_ITEM_QTY, MAX_TENDER_AMOUNT, MAX_TRANSACTION_AMOUNT } from "../../lib/money.ts";

// Transport fixtures are test-only. Execute the real API adapter without browser/logger side effects.
const apiSource = ts.transpileModule(readFileSync(new URL("../../lib/api.ts", import.meta.url), "utf8"), {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 },
}).outputText;
function apiWithFetch(fetch) {
  const testModule = { exports: {} };
  vm.runInNewContext(apiSource, {
    module: testModule, exports: testModule.exports, fetch, Headers, Response,
    require: (name) => {
      assert.equal(name, "./logger");
      return { logger: { renewTraceId: () => "trace-test", error() {}, debug() {} } };
    },
  });
  return testModule.exports.apiClient;
}
const cart = () => [{ product: { sku: "A" }, quantity: 1, subtotal: 10000 }];
const receipt = () => ({
  transaction_id: "transaction-test", transaction_number: "TXN-TEST", status: "COMPLETED",
  cashier_id: "cashier-test", subtotal_amount: 10000, tax_amount: 0, discount_amount: 0,
  total_amount: 10000, cash_tendered: 10000, change_amount: 0, created_at: "2026-09-05T00:00:00Z",
  items: [{ product_id: "product-test", sku: "A", quantity: 1, unit_price: 10000, subtotal_amount: 10000 }],
});
function setup(send, createKey) {
  let serial = 0;
  return createCheckoutController({
    send, createKey: createKey ?? (() => `key-${++serial}`),
    maxTotal: MAX_TRANSACTION_AMOUNT, maxTender: MAX_TENDER_AMOUNT, maxQuantity: MAX_LINE_ITEM_QTY,
  });
}

test("production API adapter network normalization reaches locked unknown, not rejected", async () => {
  const api = apiWithFetch(async () => { throw new Error("connection lost"); });
  const result = await api.post("/pos/checkout", {});
  assert.equal(result.error.status, 0);
  assert.equal(result.error.code, "NETWORK_ERROR");
  assert.equal(result.error.traceId, "trace-test");
  const c = setup(() => Promise.resolve(result));
  c.startReview(10000);
  await c.submitCheckout(cart(), 10000);
  assert.equal(c.getSnapshot().step, "unknown_error");
  assert.equal(c.closeReview(), false);
  assert.equal(c.startReview(10000), false);
  assert.equal(c.finishCompleted(() => assert.fail("Must not erase cart")), false);
  c.setCashTendered(20000);
  await c.submitCheckout(cart(), 10000);
  assert.equal(c.getSnapshot().cashTendered, 10000);
  assert.equal(c.getSnapshot().idempotencyKey, "key-1");
  assert.equal(c.getSnapshot().command.cash_tendered, 10000);
});

for (const [status, code, expected] of [
  [500, "CHECKOUT_FAILED", "unknown_error"], [502, "BAD_GATEWAY", "unknown_error"],
  [409, "CONCURRENT_MUTATION_IN_PROGRESS", "unknown_error"],
  [409, "IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_PAYLOAD", "unknown_error"],
  [429, "RATE_LIMITED", "unknown_error"], [401, "UNAUTHORIZED", "unknown_error"],
  [403, "FORBIDDEN", "unknown_error"], [409, "UNDOCUMENTED", "unknown_error"],
  [409, "INSUFFICIENT_STOCK", "rejected"], [404, "PRODUCT_NOT_FOUND", "rejected"],
  [400, "INSUFFICIENT_CASH_TENDERED", "rejected"],
  [400, "UNDOCUMENTED", "unknown_error"],
]) {
  test(`real adapter + controller classify ${status} ${code}`, async () => {
    const api = apiWithFetch(async () => Response.json({ success: false, error: { code, message: "test error" } }, {
      status, headers: { "X-Trace-ID": "server-trace" },
    }));
    const c = setup((body, key) => api.post("/pos/checkout", body, { headers: { "Idempotency-Key": key } }));
    c.startReview(10000);
    await c.submitCheckout(cart(), 10000);
    assert.equal(c.getSnapshot().step, expected);
    assert.equal(c.closeReview(), expected === "rejected");
  });
}

test("same-tick double submit sends once and snapshots mutable cart/tender", async () => {
  let resolve;
  let calls = 0;
  let sent;
  const c = setup((body, key) => {
    calls++;
    sent = { body, key };
    return new Promise(done => { resolve = done; });
  });
  c.startReview(10000);
  const items = cart();
  const pending = c.submitCheckout(items, 10000);
  await c.submitCheckout(items, 10000);
  assert.equal(calls, 1);
  assert.equal(c.closeReview(), false);
  assert.equal(c.startReview(10000), false);
  items[0].quantity = 9;
  c.setCashTendered(20000);
  assert.equal(sent.body.items[0].quantity, 1);
  assert.equal(sent.body.cash_tendered, 10000);
  assert.ok(Object.isFrozen(sent.body.items[0]));
  assert.ok(Object.isFrozen(sent.body));
  resolve({ success: true, data: receipt() });
  await pending;
  assert.equal(c.getSnapshot().step, "completed");
  await c.submitCheckout(items, 10000);
  assert.equal(calls, 1);
});

test("all confirmed dismissal paths clear paid cart once, preserve receipt, and do not send", async () => {
  let calls = 0;
  let items = cart();
  const c = setup(async () => { calls++; return { success: true, data: receipt() }; });
  c.startReview(10000);
  await c.submitCheckout(items, 10000);
  assert.equal(c.closeReview(), false);
  let clears = 0;
  const clear = () => { clears++; items = []; };
  assert.equal(c.finishCompleted(clear), true);
  assert.equal(c.finishCompleted(clear), false);
  assert.equal(clears, 1);
  assert.equal(items.length, 0);
  assert.equal(c.getSnapshot().receipt.transaction_number, "TXN-TEST");
  assert.equal(calls, 1);
  c.startReview(0);
  await c.submitCheckout(items, 0);
  assert.equal(calls, 1);
});

test("definite rejection requires closing and reviewing before a fresh key", async () => {
  const keys = [];
  const c = setup(async (_body, key) => {
    keys.push(key);
    return { success: false, data: null, error: { status: 409, code: "INSUFFICIENT_STOCK" } };
  });
  c.startReview(10000);
  await c.submitCheckout(cart(), 10000);
  await c.submitCheckout(cart(), 10000);
  assert.deepEqual(keys, ["key-1"]);
  c.closeReview();
  c.startReview(10000);
  await c.submitCheckout(cart(), 10000);
  assert.deepEqual(keys, ["key-1", "key-2"]);
});

test("malformed/non-final receipt and thrown transport never unlock checkout", async () => {
  for (const data of [null, {}, { ...receipt(), status: "PENDING" }, { ...receipt(), change_amount: 99 }]) {
    const c = setup(async () => ({ success: true, data }));
    c.startReview(10000);
    await c.submitCheckout(cart(), 10000);
    assert.equal(c.getSnapshot().step, "unknown_error");
  }
  const c = setup(async () => { throw new Error("unexpected transport failure"); });
  c.startReview(10000);
  await c.submitCheckout(cart(), 10000);
  assert.equal(c.getSnapshot().step, "unknown_error");
});

test("invalid money and missing secure key do not reach transport", async () => {
  const c = setup(() => assert.fail("Must not send"));
  c.startReview(10000);
  for (const amount of [NaN, Infinity, -1, 9999, 10000.5, MAX_TENDER_AMOUNT + 1]) {
    c.setCashTendered(amount);
    await c.submitCheckout(cart(), 10000);
    assert.equal(c.getSnapshot().step, "review");
    assert.ok(c.getSnapshot().errorMessage);
  }
  const noKey = setup(() => assert.fail("Must not send"), () => { throw new Error("no crypto"); });
  noKey.startReview(10000);
  await noKey.submitCheckout(cart(), 10000);
  assert.equal(noKey.getSnapshot().step, "review");
  assert.equal(noKey.getSnapshot().idempotencyKey, "");
});
