import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import { voucherThresholds, stages, env } from '../thresholds.js';

// Custom metrics
const voucherCreateDuration = new Trend('voucher_create_duration');
const voucherListDuration = new Trend('voucher_list_duration');
const voucherErrors = new Counter('voucher_errors');
const voucherSuccessRate = new Rate('voucher_success_rate');
const balanceValidationErrors = new Counter('balance_validation_errors');

// Test configuration
export const options = {
  stages: stages.standard,
  thresholds: {
    ...voucherThresholds,
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<300'],
  },
  tags: {
    scenario: 'voucher-api',
  },
};

// Shared test data
const accounts = new SharedArray('accounts', function () {
  return [
    { code: '101', name: 'Cash', type: 'asset' },
    { code: '102', name: 'Bank Deposit', type: 'asset' },
    { code: '201', name: 'Accounts Payable', type: 'liability' },
    { code: '301', name: 'Capital', type: 'equity' },
    { code: '401', name: 'Sales Revenue', type: 'revenue' },
    { code: '501', name: 'Cost of Goods Sold', type: 'expense' },
    { code: '601', name: 'Office Supplies', type: 'expense' },
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

function getRandomAccount(exclude = null) {
  let account;
  do {
    account = accounts[Math.floor(Math.random() * accounts.length)];
  } while (exclude && account.code === exclude.code);
  return account;
}

function getRandomAmount() {
  return Math.floor(Math.random() * 900000 + 100000); // 100,000 ~ 1,000,000
}

function generateVoucherNumber() {
  const date = new Date();
  const dateStr = date.toISOString().slice(0, 10).replace(/-/g, '');
  return `V${dateStr}-${Math.random().toString(36).substring(2, 8).toUpperCase()}`;
}

function createBalancedEntries() {
  const debitAccount = getRandomAccount();
  const creditAccount = getRandomAccount(debitAccount);
  const amount = getRandomAmount();

  return [
    {
      accountCode: debitAccount.code,
      accountName: debitAccount.name,
      debit: amount,
      credit: 0,
      description: `Debit entry for ${debitAccount.name}`,
    },
    {
      accountCode: creditAccount.code,
      accountName: creditAccount.name,
      debit: 0,
      credit: amount,
      description: `Credit entry for ${creditAccount.name}`,
    },
  ];
}

function createVoucherPayload() {
  const entries = createBalancedEntries();
  return {
    voucherNumber: generateVoucherNumber(),
    voucherDate: new Date().toISOString().slice(0, 10),
    description: `Performance test voucher - ${Date.now()}`,
    entries: entries,
    status: 'draft',
  };
}

// Main test function
export default function (data) {
  const token = data.accessToken;
  let createdVoucherId = null;

  // ==========================================================================
  // Group: Create Voucher
  // ==========================================================================
  group('Create Voucher', function () {
    const payload = JSON.stringify(createVoucherPayload());

    const createStart = Date.now();
    const createRes = http.post(
      `${env.BASE_URL}${env.API_VERSION}/vouchers`,
      payload,
      {
        headers: getHeaders(token),
        tags: { name: 'create_voucher' },
      }
    );
    const createEnd = Date.now();
    voucherCreateDuration.add(createEnd - createStart);

    const createSuccess = check(createRes, {
      'create voucher status is 201': (r) => r.status === 201,
      'create response has voucher id': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.id;
        } catch {
          return false;
        }
      },
      'voucher is balanced': (r) => {
        try {
          const body = JSON.parse(r.body);
          if (!body.data) return false;
          // Check that total debit equals total credit
          const entries = body.data.entries || [];
          const totalDebit = entries.reduce((sum, e) => sum + (e.debit || 0), 0);
          const totalCredit = entries.reduce((sum, e) => sum + (e.credit || 0), 0);
          return totalDebit === totalCredit;
        } catch {
          return false;
        }
      },
    });

    if (createSuccess) {
      voucherSuccessRate.add(1);
      try {
        const body = JSON.parse(createRes.body);
        createdVoucherId = body.data.id;
      } catch (e) {
        voucherErrors.add(1);
      }
    } else {
      voucherSuccessRate.add(0);
      voucherErrors.add(1);

      // Check if balance validation error
      if (createRes.body && createRes.body.includes('balance')) {
        balanceValidationErrors.add(1);
      }
    }

    sleep(0.3);
  });

  // ==========================================================================
  // Group: Get Voucher
  // ==========================================================================
  if (createdVoucherId) {
    group('Get Voucher', function () {
      const getRes = http.get(
        `${env.BASE_URL}${env.API_VERSION}/vouchers/${createdVoucherId}`,
        {
          headers: getHeaders(token),
          tags: { name: 'get_voucher' },
        }
      );

      check(getRes, {
        'get voucher status is 200': (r) => r.status === 200,
        'get response has correct id': (r) => {
          try {
            const body = JSON.parse(r.body);
            return body.data && body.data.id === createdVoucherId;
          } catch {
            return false;
          }
        },
      });

      sleep(0.2);
    });

    // ==========================================================================
    // Group: Update Voucher
    // ==========================================================================
    group('Update Voucher', function () {
      const updatePayload = JSON.stringify({
        description: `Updated voucher - ${Date.now()}`,
      });

      const updateRes = http.patch(
        `${env.BASE_URL}${env.API_VERSION}/vouchers/${createdVoucherId}`,
        updatePayload,
        {
          headers: getHeaders(token),
          tags: { name: 'update_voucher' },
        }
      );

      check(updateRes, {
        'update voucher status is 200': (r) => r.status === 200,
      });

      sleep(0.2);
    });

    // ==========================================================================
    // Group: Approve Voucher
    // ==========================================================================
    group('Approve Voucher', function () {
      const approveRes = http.post(
        `${env.BASE_URL}${env.API_VERSION}/vouchers/${createdVoucherId}/approve`,
        null,
        {
          headers: getHeaders(token),
          tags: { name: 'approve_voucher' },
        }
      );

      check(approveRes, {
        'approve voucher status is 200': (r) => r.status === 200,
        'voucher status is approved': (r) => {
          try {
            const body = JSON.parse(r.body);
            return body.data && body.data.status === 'approved';
          } catch {
            return false;
          }
        },
      });

      sleep(0.2);
    });

    // ==========================================================================
    // Group: Post Voucher
    // ==========================================================================
    group('Post Voucher', function () {
      const postRes = http.post(
        `${env.BASE_URL}${env.API_VERSION}/vouchers/${createdVoucherId}/post`,
        null,
        {
          headers: getHeaders(token),
          tags: { name: 'post_voucher' },
        }
      );

      check(postRes, {
        'post voucher status is 200': (r) => r.status === 200,
        'voucher status is posted': (r) => {
          try {
            const body = JSON.parse(r.body);
            return body.data && body.data.status === 'posted';
          } catch {
            return false;
          }
        },
      });

      sleep(0.2);
    });
  }

  // ==========================================================================
  // Group: List Vouchers
  // ==========================================================================
  group('List Vouchers', function () {
    const listStart = Date.now();
    const listRes = http.get(
      `${env.BASE_URL}${env.API_VERSION}/vouchers?page=1&limit=20`,
      {
        headers: getHeaders(token),
        tags: { name: 'list_vouchers' },
      }
    );
    const listEnd = Date.now();
    voucherListDuration.add(listEnd - listStart);

    check(listRes, {
      'list vouchers status is 200': (r) => r.status === 200,
      'list response has items': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && Array.isArray(body.data.items);
        } catch {
          return false;
        }
      },
      'list response has pagination': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && typeof body.data.total !== 'undefined';
        } catch {
          return false;
        }
      },
    });

    sleep(0.3);
  });

  // ==========================================================================
  // Group: Search Vouchers
  // ==========================================================================
  group('Search Vouchers', function () {
    const today = new Date().toISOString().slice(0, 10);
    const searchRes = http.get(
      `${env.BASE_URL}${env.API_VERSION}/vouchers?startDate=${today}&endDate=${today}&status=draft`,
      {
        headers: getHeaders(token),
        tags: { name: 'search_vouchers' },
      }
    );

    check(searchRes, {
      'search vouchers status is 200': (r) => r.status === 200,
    });

    sleep(0.2);
  });

  // Wait between iterations
  sleep(1);
}

// Setup function - authenticate and prepare test data
export function setup() {
  console.log('Voucher API Performance Test Starting...');
  console.log(`Target: ${env.BASE_URL}`);

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

// Teardown function
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`Voucher API Performance Test Completed in ${duration}s`);
}
