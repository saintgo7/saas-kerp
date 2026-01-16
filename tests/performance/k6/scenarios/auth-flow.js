import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { authThresholds, stages, env } from '../thresholds.js';

// Custom metrics
const loginDuration = new Trend('login_duration');
const refreshDuration = new Trend('refresh_duration');
const authErrors = new Counter('auth_errors');
const authSuccessRate = new Rate('auth_success_rate');

// Test configuration
export const options = {
  stages: stages.standard,
  thresholds: {
    ...authThresholds,
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<300'],
  },
  tags: {
    scenario: 'auth-flow',
  },
};

// Test data
const testUsers = [
  { email: 'admin@test.com', password: 'Test123!@#', role: 'admin' },
  { email: 'accountant@test.com', password: 'Test123!@#', role: 'accountant' },
  { email: 'viewer@test.com', password: 'Test123!@#', role: 'viewer' },
];

// Helper functions
function getRandomUser() {
  return testUsers[Math.floor(Math.random() * testUsers.length)];
}

function getHeaders(token = null) {
  const headers = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

// Main test function
export default function () {
  const user = getRandomUser();
  let accessToken = null;
  let refreshToken = null;

  // ==========================================================================
  // Group: Login Flow
  // ==========================================================================
  group('Login Flow', function () {
    const loginPayload = JSON.stringify({
      email: user.email,
      password: user.password,
    });

    const loginStart = Date.now();
    const loginRes = http.post(
      `${env.BASE_URL}${env.API_VERSION}/auth/login`,
      loginPayload,
      {
        headers: getHeaders(),
        tags: { name: 'login' },
      }
    );
    const loginEnd = Date.now();
    loginDuration.add(loginEnd - loginStart);

    const loginSuccess = check(loginRes, {
      'login status is 200': (r) => r.status === 200,
      'login response has tokens': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.tokens && body.data.tokens.accessToken;
        } catch {
          return false;
        }
      },
      'login response has user': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.user && body.data.user.id;
        } catch {
          return false;
        }
      },
    });

    if (loginSuccess) {
      authSuccessRate.add(1);
      try {
        const body = JSON.parse(loginRes.body);
        accessToken = body.data.tokens.accessToken;
        refreshToken = body.data.tokens.refreshToken;
      } catch (e) {
        authErrors.add(1);
      }
    } else {
      authSuccessRate.add(0);
      authErrors.add(1);
    }

    sleep(0.5);
  });

  // ==========================================================================
  // Group: Authenticated Request
  // ==========================================================================
  if (accessToken) {
    group('Get Current User', function () {
      const meRes = http.get(
        `${env.BASE_URL}${env.API_VERSION}/auth/me`,
        {
          headers: getHeaders(accessToken),
          tags: { name: 'get_me' },
        }
      );

      check(meRes, {
        'get me status is 200': (r) => r.status === 200,
        'get me response has user data': (r) => {
          try {
            const body = JSON.parse(r.body);
            return body.data && body.data.email === user.email;
          } catch {
            return false;
          }
        },
      });

      sleep(0.3);
    });

    // ==========================================================================
    // Group: Token Refresh
    // ==========================================================================
    group('Token Refresh', function () {
      const refreshPayload = JSON.stringify({
        refreshToken: refreshToken,
      });

      const refreshStart = Date.now();
      const refreshRes = http.post(
        `${env.BASE_URL}${env.API_VERSION}/auth/refresh`,
        refreshPayload,
        {
          headers: getHeaders(),
          tags: { name: 'refresh_token' },
        }
      );
      const refreshEnd = Date.now();
      refreshDuration.add(refreshEnd - refreshStart);

      const refreshSuccess = check(refreshRes, {
        'refresh status is 200': (r) => r.status === 200,
        'refresh response has new tokens': (r) => {
          try {
            const body = JSON.parse(r.body);
            return body.data && body.data.accessToken;
          } catch {
            return false;
          }
        },
      });

      if (refreshSuccess) {
        try {
          const body = JSON.parse(refreshRes.body);
          accessToken = body.data.accessToken;
          if (body.data.refreshToken) {
            refreshToken = body.data.refreshToken;
          }
        } catch (e) {
          // Continue with existing token
        }
      }

      sleep(0.3);
    });

    // ==========================================================================
    // Group: Logout
    // ==========================================================================
    group('Logout', function () {
      const logoutRes = http.post(
        `${env.BASE_URL}${env.API_VERSION}/auth/logout`,
        null,
        {
          headers: getHeaders(accessToken),
          tags: { name: 'logout' },
        }
      );

      check(logoutRes, {
        'logout status is 200 or 204': (r) => r.status === 200 || r.status === 204,
      });

      sleep(0.2);
    });
  }

  // ==========================================================================
  // Group: Unauthorized Access
  // ==========================================================================
  group('Unauthorized Access', function () {
    // Test access without token
    const unauthorizedRes = http.get(
      `${env.BASE_URL}${env.API_VERSION}/auth/me`,
      {
        headers: getHeaders(),
        tags: { name: 'unauthorized' },
      }
    );

    check(unauthorizedRes, {
      'unauthorized status is 401': (r) => r.status === 401,
    });

    // Test access with invalid token
    const invalidTokenRes = http.get(
      `${env.BASE_URL}${env.API_VERSION}/auth/me`,
      {
        headers: getHeaders('invalid-token'),
        tags: { name: 'invalid_token' },
      }
    );

    check(invalidTokenRes, {
      'invalid token status is 401': (r) => r.status === 401,
    });

    sleep(0.2);
  });

  // Wait between iterations
  sleep(1);
}

// Setup function - runs once before test
export function setup() {
  console.log('Auth Flow Performance Test Starting...');
  console.log(`Target: ${env.BASE_URL}`);

  // Health check
  const healthRes = http.get(`${env.BASE_URL}/health`);
  if (healthRes.status !== 200) {
    console.error('Health check failed!');
  }

  return { startTime: Date.now() };
}

// Teardown function - runs once after test
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`Auth Flow Performance Test Completed in ${duration}s`);
}
