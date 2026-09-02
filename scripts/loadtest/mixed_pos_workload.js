import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '5s', target: 20 },  // Morning opening ramp-up
    { duration: '20s', target: 60 }, // Peak lunch/rush hour traffic (60 concurrent users)
    { duration: '10s', target: 100 },// High-intensity flash promotion (100 concurrent users)
    { duration: '5s', target: 0 },   // Evening closing ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<150', 'p(99)<400'],
    http_req_failed: ['rate<0.01'],
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
  // 1. Authenticate as Cashier
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({
      tenant_slug: 'al-barakah-mart',
      email: 'cashier1@albarakah.com',
      password: 'Password123!',
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(loginRes, {
    'login successful': (r) => r.status === 200,
  });

  const body = JSON.parse(loginRes.body);
  return { token: body.data.access_token };
}

export default function (data) {
  const params = {
    headers: {
      Authorization: `Bearer ${data.token}`,
      'Content-Type': 'application/json',
    },
  };

  const rand = Math.random();

  if (rand < 0.65) {
    // 65% Catalog & Product Browsing
    const res = http.get(`${BASE_URL}/api/v1/pos/products`, params);
    check(res, {
      'catalog status 200': (r) => r.status === 200,
    });
  } else if (rand < 0.85) {
    // 20% Concurrent Atomic Checkouts
    const timestamp = Date.now();
    const idempotencyKey = `mixed_idem_${__VU}_${__ITER}_${timestamp}`;
    const selectedSku = SKUS[(__VU + __ITER) % SKUS.length];
    const isCash = __ITER % 2 === 0;

    const checkoutPayload = JSON.stringify({
      payment_method: isCash ? 'CASH' : 'QRIS',
      customer_name: `Customer ${__VU}`,
      cash_tendered: isCash ? 200000.00 : 0,
      items: [
        {
          sku: selectedSku,
          quantity: 1,
        },
      ],
    });

    const checkoutParams = {
      headers: {
        Authorization: `Bearer ${data.token}`,
        'Content-Type': 'application/json',
        'Idempotency-Key': idempotencyKey,
      },
    };

    const res = http.post(`${BASE_URL}/api/v1/pos/checkout`, checkoutPayload, checkoutParams);
    check(res, {
      'checkout success': (r) => r.status === 201,
    });
  } else if (rand < 0.93) {
    // 8% Order History Inquiries
    const res = http.get(`${BASE_URL}/api/v1/pos/orders?limit=10&page=1`, params);
    check(res, {
      'order history status 200': (r) => r.status === 200,
    });
  } else {
    // 7% Inventory & Category Health Checks
    const res = http.get(`${BASE_URL}/api/v1/pos/inventory/low-stock`, params);
    check(res, {
      'low stock status 200': (r) => r.status === 200,
    });
  }

  sleep(0.08); // 80ms pacing
}
