import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '5s', target: 15 },  // Ramp up to 15 concurrent cashiers
    { duration: '15s', target: 35 }, // 35 concurrent checkout terminals
    { duration: '10s', target: 50 }, // Peak flash-sale burst (50 concurrent cashiers)
    { duration: '5s', target: 0 },   // Ramp down
  ],
  thresholds: {
    // 95% of transactions should complete under 150ms
    http_req_duration: ['p(95)<150', 'p(99)<350'],
    // 0% server crashes (500s)
    'http_req_failed{status:500}': ['rate<0.001'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';

const SKUS = [
  'SKU-CAKE-BF20',
  'SKU-CAKE-RV18',
  'SKU-BREAD-BG01',
  'SKU-PASTRY-CA01',
];

export function setup() {
  // 1. Authenticate as Store Manager (has inventory:write)
  const managerLogin = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({
      tenant_slug: 'al-barakah-mart',
      email: 'manager1@albarakah.com',
      password: 'Password123!',
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(managerLogin, {
    'manager login successful': (r) => r.status === 200,
  });

  const managerToken = JSON.parse(managerLogin.body).data.access_token;

  // 2. Replenish test stock of products to 50,000 to sustain thousands of concurrent checkouts
  const products = [
    '10000000-0000-0000-0000-000000000011',
    '10000000-0000-0000-0000-000000000012',
    '10000000-0000-0000-0000-000000000013',
    '10000000-0000-0000-0000-000000000015',
  ];

  for (let i = 0; i < products.length; i++) {
    const replenishPayload = JSON.stringify({
      product_id: products[i],
      adjustment_type: 'SET',
      quantity: 50000,
      reason: 'RESTOCK',
      notes: 'Automated load-test inventory provisioning',
    });

    const res = http.post(`${BASE_URL}/api/v1/pos/inventory/adjust`, replenishPayload, {
      headers: {
        Authorization: `Bearer ${managerToken}`,
        'Content-Type': 'application/json',
        'Idempotency-Key': `replenish_${i}_${Date.now()}`,
      },
    });

    check(res, {
      'replenish status 200': (r) => r.status === 200,
    });
  }

  // 3. Authenticate as Cashier for executing checkout operations
  const cashierLogin = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({
      tenant_slug: 'al-barakah-mart',
      email: 'cashier1@albarakah.com',
      password: 'Password123!',
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(cashierLogin, {
    'cashier login successful': (r) => r.status === 200,
  });

  const cashierToken = JSON.parse(cashierLogin.body).data.access_token;

  return { token: cashierToken };
}

export default function (data) {
  const timestamp = Date.now();
  const idempotencyKey = `idemp_bench_${__VU}_${__ITER}_${timestamp}`;
  const selectedSku = SKUS[(__VU + __ITER) % SKUS.length];
  const isCash = __ITER % 2 === 0;

  const checkoutPayload = JSON.stringify({
    payment_method: isCash ? 'CASH' : 'QRIS',
    customer_name: `Pelanggan Terminal ${__VU}`,
    cash_tendered: isCash ? 200000.00 : 0,
    items: [
      {
        sku: selectedSku,
        quantity: 1,
      },
    ],
  });

  const params = {
    headers: {
      Authorization: `Bearer ${data.token}`,
      'Content-Type': 'application/json',
      'Idempotency-Key': idempotencyKey,
    },
  };

  const res = http.post(`${BASE_URL}/api/v1/pos/checkout`, checkoutPayload, params);

  check(res, {
    'checkout status is 201': (r) => r.status === 201,
    'no 500 server error': (r) => r.status < 500,
    'valid transaction number': (r) => {
      if (r.status === 201) {
        const json = JSON.parse(r.body);
        return json.data && json.data.transaction_number && json.data.transaction_number.startsWith('TXN-');
      }
      return false;
    },
    'valid change calculated': (r) => {
      if (r.status === 201 && isCash) {
        const json = JSON.parse(r.body);
        return json.data && json.data.change_amount >= 0;
      }
      return true;
    },
  });

  // 100ms pacing representing cashier scanning next barcode
  sleep(0.1);
}
