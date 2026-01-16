import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import { concurrentThresholds, env } from '../thresholds.js';

// Custom metrics
const postingDuration = new Trend('posting_duration');
const postingErrors = new Counter('posting_errors');
const postingSuccessRate = new Rate('posting_success_rate');
const concurrentPostings = new Gauge('concurrent_postings');
const deadlockErrors = new Counter('deadlock_errors');
const balanceErrors = new Counter('balance_errors');

// Test configuration - high concurrency stress test
export const options = {
  scenarios: {
    concurrent_posting: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 20 },  // Ramp up to 20 VUs
        { duration: '1m', target: 50 },   // Ramp up to 50 VUs
        { duration: '2m', target: 50 },   // Stay at 50 VUs
        { duration: '30s', target: 100 }, // Spike to 100 VUs
        { duration: '1m', target: 100 },  // Stay at 100 VUs
        { duration: '30s', target: 0 },   // Ramp down
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    ...concurrentThresholds,
    http_req_failed: ['rate<0.02'], // 2% error rate allowed
    http_req_duration: ['p(95)<600'],
    posting_success_rate: ['rate>0.95'], // 95% success rate required
    deadlock_errors: ['count<10'], // Max 10 deadlocks allowed
  },
  tags: {
    scenario: 'concurrent-posting',
  },
};

// Shared test data - accounts
const accounts = new SharedArray('accounts', function () {
  return [
    { code: '101', name: 'Cash', type: 'asset' },
    { code: '102', name: 'Bank', type: 'asset' },
    { code: '103', name: 'Receivables', type: 'asset' },
    { code: '201', name: 'Payables', type: 'liability' },
    { code: '301', name: 'Capital', type: 'equity' },
    { code: '401', name: 'Sales', type: 'revenue' },
    { code: '501', name: 'COGS', type: 'expense' },
    { code: '601', name: 'Expenses', type: 'expense' },
  ];
});

// Test credentials
const testUser = {
  email: 'admin@test.com',
  password: 'Test123!@#',
};

// Helper functions
function getHeaders(token) {
  return {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
    'Authorization': `Bearer ${token}`,
  };
}

function getRandomAccounts() {
  const debitIdx = Math.floor(Math.random() * accounts.length);
  let creditIdx;
  do {
    creditIdx = Math.floor(Math.random() * accounts.length);
  } while (creditIdx === debitIdx);

  return {
    debit: accounts[debitIdx],
    credit: accounts[creditIdx],
  };
}

function getRandomAmount() {
  return Math.floor(Math.random() * 500000 + 50000); // 50,000 ~ 550,000
}

function generateVoucherNumber() {
  const date = new Date();
  const dateStr = date.toISOString().slice(0, 10).replace(/-/g, '');
  const vuId = __VU.toString().padStart(3, '0');
  const iter = __ITER.toString().padStart(5, '0');
  return `V${dateStr}-${vuId}-${iter}`;
}

function createVoucherPayload() {
  const accts = getRandomAccounts();
  const amount = getRandomAmount();

  return {
    voucherNumber: generateVoucherNumber(),
    voucherDate: new Date().toISOString().slice(0, 10),
    description: `Concurrent test voucher - VU${__VU} ITER${__ITER}`,
    entries: [
      {
        accountCode: accts.debit.code,
        accountName: accts.debit.name,
        debit: amount,
        credit: 0,
        description: `Debit - ${accts.debit.name}`,
      },
      {
        accountCode: accts.credit.code,
        accountName: accts.credit.name,
        debit: 0,
        credit: amount,
        description: `Credit - ${accts.credit.name}`,
      },
    ],
    status: 'draft',
  };
}

// Main test function - Concurrent Posting Workflow
export default function (data) {
  const token = data.accessToken;
  let voucherId = null;

  concurrentPostings.add(1);

  // ==========================================================================
  // Step 1: Create Voucher
  // ==========================================================================
  group('Create Voucher', function () {
    const payload = JSON.stringify(createVoucherPayload());

    const createRes = http.post(
      `${env.BASE_URL}${env.API_VERSION}/vouchers`,
      payload,
      {
        headers: getHeaders(token),
        tags: { name: 'concurrent_create' },
      }
    );

    const createSuccess = check(createRes, {
      'create status is 201': (r) => r.status === 201,
    });

    if (createSuccess) {
      try {
        const body = JSON.parse(createRes.body);
        voucherId = body.data.id;
      } catch (e) {
        postingErrors.add(1);
      }
    } else {
      postingErrors.add(1);
      if (createRes.body && createRes.body.includes('balance')) {
        balanceErrors.add(1);
      }
    }

    sleep(0.1);
  });

  if (!voucherId) {
    concurrentPostings.add(-1);
    return;
  }

  // ==========================================================================
  // Step 2: Quick Approve (skip for concurrent stress)
  // ==========================================================================
  group('Quick Approve', function () {
    const approveRes = http.post(
      `${env.BASE_URL}${env.API_VERSION}/vouchers/${voucherId}/approve`,
      null,
      {
        headers: getHeaders(token),
        tags: { name: 'concurrent_approve' },
      }
    );

    check(approveRes, {
      'approve status is 200': (r) => r.status === 200,
    });

    sleep(0.05);
  });

  // ==========================================================================
  // Step 3: Post Voucher (CRITICAL - concurrent ledger updates)
  // ==========================================================================
  group('Post Voucher', function () {
    const postStart = Date.now();
    const postRes = http.post(
      `${env.BASE_URL}${env.API_VERSION}/vouchers/${voucherId}/post`,
      null,
      {
        headers: getHeaders(token),
        tags: { name: 'concurrent_post' },
        timeout: '10s', // Allow longer timeout for posting
      }
    );
    const postEnd = Date.now();
    postingDuration.add(postEnd - postStart);

    const postSuccess = check(postRes, {
      'post status is 200': (r) => r.status === 200,
      'voucher posted': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.status === 'posted';
        } catch {
          return false;
        }
      },
    });

    if (postSuccess) {
      postingSuccessRate.add(1);
    } else {
      postingSuccessRate.add(0);
      postingErrors.add(1);

      // Check for specific error types
      if (postRes.body) {
        const bodyStr = postRes.body.toLowerCase();
        if (bodyStr.includes('deadlock') || bodyStr.includes('lock timeout')) {
          deadlockErrors.add(1);
        }
        if (bodyStr.includes('balance') || bodyStr.includes('unbalanced')) {
          balanceErrors.add(1);
        }
      }
    }
  });

  concurrentPostings.add(-1);

  // Minimal sleep for high concurrency
  sleep(0.2);
}

// Setup function
export function setup() {
  console.log('Concurrent Posting Stress Test Starting...');
  console.log(`Target: ${env.BASE_URL}`);
  console.log('This test validates database transaction handling under concurrent load.');

  // Login to get access token
  const loginRes = http.post(
    `${env.BASE_URL}${env.API_VERSION}/auth/login`,
    JSON.stringify({
      email: testUser.email,
      password: testUser.password,
    }),
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  );

  if (loginRes.status !== 200) {
    console.error('Setup login failed!');
    return { accessToken: null };
  }

  let accessToken = null;
  try {
    const body = JSON.parse(loginRes.body);
    accessToken = body.data.tokens.accessToken;
  } catch (e) {
    console.error('Failed to parse login response');
  }

  return {
    accessToken: accessToken,
    startTime: Date.now(),
  };
}

// Teardown function with summary
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log('==========================================');
  console.log('Concurrent Posting Stress Test Summary');
  console.log('==========================================');
  console.log(`Duration: ${duration}s`);
  console.log('Check console output for detailed metrics.');
  console.log('==========================================');
}

// Handle summary for custom reporting
export function handleSummary(data) {
  const summary = {
    timestamp: new Date().toISOString(),
    scenario: 'concurrent-posting',
    metrics: {
      totalRequests: data.metrics.http_reqs ? data.metrics.http_reqs.count : 0,
      failedRequests: data.metrics.http_req_failed ? data.metrics.http_req_failed.rate : 0,
      avgDuration: data.metrics.http_req_duration ? data.metrics.http_req_duration.avg : 0,
      p95Duration: data.metrics.http_req_duration ? data.metrics.http_req_duration['p(95)'] : 0,
      postingSuccessRate: data.metrics.posting_success_rate ? data.metrics.posting_success_rate.rate : 0,
    },
  };

  return {
    'summary.json': JSON.stringify(summary, null, 2),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

// Text summary helper
function textSummary(data, options) {
  const indent = options.indent || '';
  let output = '';

  output += `\n${indent}Concurrent Posting Test Results\n`;
  output += `${indent}================================\n`;

  if (data.metrics.http_req_duration) {
    output += `${indent}Request Duration:\n`;
    output += `${indent}  avg: ${data.metrics.http_req_duration.avg.toFixed(2)}ms\n`;
    output += `${indent}  p95: ${data.metrics.http_req_duration['p(95)'].toFixed(2)}ms\n`;
    output += `${indent}  max: ${data.metrics.http_req_duration.max.toFixed(2)}ms\n`;
  }

  if (data.metrics.http_req_failed) {
    output += `${indent}Error Rate: ${(data.metrics.http_req_failed.rate * 100).toFixed(2)}%\n`;
  }

  if (data.metrics.posting_success_rate) {
    output += `${indent}Posting Success Rate: ${(data.metrics.posting_success_rate.rate * 100).toFixed(2)}%\n`;
  }

  return output;
}
