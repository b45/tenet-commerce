import test from "node:test";
import assert from "node:assert/strict";
import {
  formatIDR,
  parseIDR,
  addMoney,
  subMoney,
  mulMoney,
  calculateTax,
  parseMoneyFromAPI,
  MAX_TENDER_AMOUNT,
} from "./money.ts";

test("formatIDR formats numbers into Indonesian currency strings", () => {
  assert.match(formatIDR(0), /Rp\s*0/);
  assert.match(formatIDR(45000), /Rp\s*45\.000/);
  assert.match(formatIDR(1500000), /Rp\s*1\.500\.000/);
  assert.match(formatIDR(45000.7), /Rp\s*45\.001/); // Rounds half up
});

test("parseIDR parses clean and formatted string inputs", () => {
  assert.equal(parseIDR("50000"), 50000);
  assert.equal(parseIDR("50.000"), 50000);
  assert.equal(parseIDR("Rp 50.000"), 50000);
  assert.equal(parseIDR("rp50000"), 50000);
  assert.equal(parseIDR("1.500.000"), 1500000);
  assert.equal(parseIDR("50.000,00"), 50000); // .00 cents tolerated
});

test("parseIDR rejects invalid strings, fractions, and out of bounds", () => {
  for (const input of ["1.5", "1.00", "50,00,12", "5e4", "-50.000", 100.5, -MAX_TENDER_AMOUNT - 1]) {
    assert.equal(parseIDR(input), null);
  }
  assert.equal(parseIDR("abc"), null);
  assert.equal(parseIDR("50.000,50"), null); // Non-zero cents rejected
  assert.equal(parseIDR("-50000"), null);     // Negative disallowed by default
  assert.equal(parseIDR("-50000", { allowNegative: true }), -50000);
  assert.equal(parseIDR(MAX_TENDER_AMOUNT + 1), null);
});

test("arithmetic helpers prevent precision loss and calculate taxes correctly", () => {
  assert.equal(addMoney(10000, 25000), 35000);
  assert.equal(subMoney(50000, 35000), 15000);
  assert.equal(mulMoney(10000, 1.11), 11100);
  assert.equal(calculateTax(100000, 11), 11000); // 11% PPN
  assert.equal(calculateTax(15250, 11), 1678);   // 15250 * 0.11 = 1677.5 -> 1678
});

test("parseMoneyFromAPI normalizes mixed backend types", () => {
  assert.equal(parseMoneyFromAPI(45000), 45000);
  assert.equal(parseMoneyFromAPI(45000.4), 45000);
  assert.equal(parseMoneyFromAPI(45000.6), 45001);
  assert.equal(parseMoneyFromAPI("45000"), 45000);
  assert.equal(parseMoneyFromAPI("45.000"), 45000);
  assert.equal(parseMoneyFromAPI(null), 0);
  assert.equal(parseMoneyFromAPI(undefined), 0);
});
