// K6 Performance Test Thresholds Configuration
// K-ERP SaaS Platform Performance Standards

export const thresholds = {
  // HTTP Request Duration Thresholds
  http_req_duration: [
    'p(95)<200', // 95th percentile under 200ms
    'p(99)<500', // 99th percentile under 500ms
    'avg<100',   // Average under 100ms
    'max<2000',  // Max under 2 seconds
  ],

  // HTTP Request Failed Rate
  http_req_failed: [
    'rate<0.01', // Error rate under 1%
  ],

  // HTTP Request Waiting (TTFB)
  http_req_waiting: [
    'p(95)<150', // 95th percentile TTFB under 150ms
  ],

  // HTTP Request Receiving
  http_req_receiving: [
    'p(95)<50', // 95th percentile receive time under 50ms
  ],

  // Virtual Users
  vus: [
    'value<=100', // Max 100 concurrent users
  ],

  // Iteration Duration
  iteration_duration: [
    'p(95)<3000', // 95th percentile iteration under 3 seconds
  ],
};

// Scenario-specific thresholds
export const voucherThresholds = {
  'http_req_duration{name:create_voucher}': ['p(95)<300'],
  'http_req_duration{name:get_voucher}': ['p(95)<150'],
  'http_req_duration{name:list_vouchers}': ['p(95)<250'],
  'http_req_duration{name:update_voucher}': ['p(95)<300'],
  'http_req_duration{name:approve_voucher}': ['p(95)<400'],
  'http_req_duration{name:post_voucher}': ['p(95)<500'],
};

export const authThresholds = {
  'http_req_duration{name:login}': ['p(95)<300'],
  'http_req_duration{name:refresh_token}': ['p(95)<150'],
  'http_req_duration{name:get_me}': ['p(95)<100'],
  'http_req_duration{name:logout}': ['p(95)<200'],
};

export const concurrentThresholds = {
  'http_req_duration{name:concurrent_post}': ['p(95)<600'],
  'http_req_failed{name:concurrent_post}': ['rate<0.02'], // 2% error rate allowed for concurrent
};

// Load test stages configuration
export const stages = {
  // Ramp-up pattern for standard load test
  standard: [
    { duration: '30s', target: 10 },  // Ramp up to 10 users
    { duration: '1m', target: 50 },   // Ramp up to 50 users
    { duration: '2m', target: 50 },   // Stay at 50 users
    { duration: '1m', target: 100 },  // Ramp up to 100 users
    { duration: '2m', target: 100 },  // Stay at 100 users
    { duration: '30s', target: 0 },   // Ramp down to 0
  ],

  // Smoke test pattern
  smoke: [
    { duration: '30s', target: 5 },   // 5 users for 30 seconds
  ],

  // Stress test pattern
  stress: [
    { duration: '1m', target: 50 },
    { duration: '2m', target: 100 },
    { duration: '2m', target: 150 },
    { duration: '2m', target: 200 },
    { duration: '1m', target: 0 },
  ],

  // Spike test pattern
  spike: [
    { duration: '30s', target: 10 },
    { duration: '10s', target: 100 }, // Sudden spike
    { duration: '30s', target: 100 },
    { duration: '10s', target: 10 },  // Sudden drop
    { duration: '30s', target: 10 },
  ],
};

// Environment configuration
export const env = {
  BASE_URL: __ENV.BASE_URL || 'http://localhost:8080',
  API_VERSION: '/api',
};

// Test data generation helpers
export function generateCompanyId() {
  return `company-${Math.random().toString(36).substring(7)}`;
}

export function generateUserId() {
  return `user-${Math.random().toString(36).substring(7)}`;
}

export function generateVoucherId() {
  return `voucher-${Date.now()}-${Math.random().toString(36).substring(7)}`;
}
