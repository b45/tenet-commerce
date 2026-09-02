import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '5s', target: 20 },  // Ramp-up to 20 users
    { duration: '15s', target: 50 }, // Sustained heavy read load (50 users)
    { duration: '10s', target: 100 },// Peak stress burst (100 users)
    { duration: '5s', target: 0 },   // Cool down
  ],
  thresholds: {
    http_req_duration: ['p(95)<50', 'p(99)<100'], // 95% of requests under 50ms, 99% under 100ms
    http_req_failed: ['rate<0.01'],               // Less than 1% errors
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';

export function setup() {
  // 1. Authenticate as cashier
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

  // 1. Browse Catalog
  const prodRes = http.get(`${BASE_URL}/api/v1/pos/products`, params);
  check(prodRes, {
    'products status 200': (r) => r.status === 200,
    'products returned': (r) => {
      const json = JSON.parse(r.body);
      return json.data && json.data.length > 0;
    },
  });

  // 2. Browse Categories
  const catRes = http.get(`${BASE_URL}/api/v1/pos/categories`, params);
  check(catRes, {
    'categories status 200': (r) => r.status === 200,
  });

  // 3. Check Low Stock Alerts
  const lowStockRes = http.get(`${BASE_URL}/api/v1/pos/inventory/low-stock`, params);
  check(lowStockRes, {
    'low stock status 200': (r) => r.status === 200,
  });

  sleep(0.1); // 100ms pacing between cashier queries
}
