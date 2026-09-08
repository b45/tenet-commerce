import test from "node:test";
import assert from "node:assert/strict";

function calculateSummaryMetrics(summary) {
  const netSales = summary.gross_sales - summary.discounts;
  const grossProfit = netSales - summary.total_cogs;
  const marginPercent =
    netSales > 0 ? Number(((grossProfit / netSales) * 100).toFixed(1)) : 0.0;

  return {
    netSales,
    grossProfit,
    marginPercent,
  };
}

function aggregatePaymentBreakdown(breakdown) {
  let totalCount = 0;
  let totalAmount = 0;

  for (const method of Object.values(breakdown)) {
    totalCount += method.count;
    totalAmount += method.total_amount;
  }

  return {
    totalCount,
    totalAmount,
  };
}

test("DailySummary: correctly computes net sales, gross profit, and margin percentage", () => {
  const mockSummary = {
    date: "2026-09-06",
    total_orders: 12,
    completed_orders: 9,
    voided_orders: 3,
    gross_sales: 1014000,
    discounts: 30000,
    net_sales: 984000,
    total_cogs: 810000,
    gross_profit: 174000,
    payment_breakdown: {
      CASH: { count: 6, total_amount: 870000 },
      QRIS: { count: 3, total_amount: 114000 },
    },
  };

  const metrics = calculateSummaryMetrics(mockSummary);
  assert.equal(metrics.netSales, 984000);
  assert.equal(metrics.grossProfit, 174000);
  // (174000 / 984000) * 100 = 17.6829... -> 17.7%
  assert.equal(metrics.marginPercent, 17.7);
});

test("DailySummary: handles zero sales and empty payment breakdown safely", () => {
  const emptySummary = {
    date: "2026-09-06",
    total_orders: 0,
    completed_orders: 0,
    voided_orders: 0,
    gross_sales: 0,
    discounts: 0,
    net_sales: 0,
    total_cogs: 0,
    gross_profit: 0,
    payment_breakdown: {},
  };

  const metrics = calculateSummaryMetrics(emptySummary);
  assert.equal(metrics.netSales, 0);
  assert.equal(metrics.grossProfit, 0);
  assert.equal(metrics.marginPercent, 0.0);

  const aggregated = aggregatePaymentBreakdown(emptySummary.payment_breakdown);
  assert.equal(aggregated.totalCount, 0);
  assert.equal(aggregated.totalAmount, 0);
});

test("DailySummary: verifies payment method sum matches net sales for completed transactions", () => {
  const mockSummary = {
    payment_breakdown: {
      CASH: { count: 6, total_amount: 870000 },
      QRIS: { count: 3, total_amount: 114000 },
    },
  };

  const aggregated = aggregatePaymentBreakdown(mockSummary.payment_breakdown);
  assert.equal(aggregated.totalCount, 9);
  assert.equal(aggregated.totalAmount, 984000);
});
