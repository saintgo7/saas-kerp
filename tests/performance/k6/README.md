# K6 Performance Tests

K-ERP SaaS Platform 성능 테스트 스위트입니다.

## Directory Structure

```
k6/
├── thresholds.js           # Shared thresholds and configuration
├── scenarios/
│   ├── auth-flow.js        # Authentication flow tests
│   ├── voucher-api.js      # Voucher CRUD and workflow tests
│   └── concurrent-posting.js # Concurrent ledger posting stress test
└── README.md
```

## Prerequisites

1. K6 설치
```bash
# macOS
brew install k6

# Ubuntu/Debian
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6
```

2. 테스트 환경 실행
```bash
# Development server
make dev-up
make run

# Or Docker Compose
docker compose -f deployments/docker/docker-compose.yml up -d
```

## Usage

### Run Individual Scenarios

```bash
# Auth Flow Test
k6 run tests/performance/k6/scenarios/auth-flow.js

# Voucher API Test
k6 run tests/performance/k6/scenarios/voucher-api.js

# Concurrent Posting Stress Test
k6 run tests/performance/k6/scenarios/concurrent-posting.js
```

### Run with Custom Environment

```bash
# Custom base URL
BASE_URL=https://staging.erp.example.com k6 run scenarios/auth-flow.js

# Custom VUs and duration
k6 run --vus 50 --duration 5m scenarios/voucher-api.js
```

### Run Smoke Test (Quick Validation)

```bash
k6 run --vus 5 --duration 30s scenarios/auth-flow.js
```

### Run with HTML Report

```bash
K6_WEB_DASHBOARD=true k6 run scenarios/voucher-api.js
```

### Run with InfluxDB (Grafana Dashboard)

```bash
k6 run --out influxdb=http://localhost:8086/k6 scenarios/auth-flow.js
```

## Test Scenarios

### 1. auth-flow.js

인증 플로우 성능 테스트:
- Login with credentials
- Get current user (authenticated)
- Token refresh
- Logout
- Unauthorized access validation

**Thresholds:**
- Login: p95 < 300ms
- Token Refresh: p95 < 150ms
- Get Me: p95 < 100ms
- Logout: p95 < 200ms

### 2. voucher-api.js

전표 CRUD 및 워크플로우 테스트:
- Create voucher with balanced entries
- Get voucher by ID
- Update voucher
- Approve voucher
- Post voucher (ledger update)
- List vouchers with pagination
- Search vouchers with filters

**Thresholds:**
- Create: p95 < 300ms
- Get: p95 < 150ms
- List: p95 < 250ms
- Update: p95 < 300ms
- Approve: p95 < 400ms
- Post: p95 < 500ms

### 3. concurrent-posting.js

동시성 스트레스 테스트:
- 100 concurrent users creating and posting vouchers
- Tests database transaction handling
- Validates deadlock prevention
- Checks data consistency

**Thresholds:**
- Posting p95 < 600ms
- Error rate < 2%
- Success rate > 95%
- Deadlock errors < 10

## Thresholds Configuration

```javascript
// From thresholds.js
export const thresholds = {
  http_req_duration: [
    'p(95)<200',  // 95th percentile under 200ms
    'p(99)<500',  // 99th percentile under 500ms
    'avg<100',    // Average under 100ms
    'max<2000',   // Max under 2 seconds
  ],
  http_req_failed: [
    'rate<0.01',  // Error rate under 1%
  ],
};
```

## Load Stages

### Standard Load Test
```
30s  -> 10 VUs (ramp up)
1m   -> 50 VUs (ramp up)
2m   -> 50 VUs (steady)
1m   -> 100 VUs (ramp up)
2m   -> 100 VUs (steady)
30s  -> 0 VUs (ramp down)
```

### Stress Test
```
1m   -> 50 VUs
2m   -> 100 VUs
2m   -> 150 VUs
2m   -> 200 VUs
1m   -> 0 VUs
```

### Spike Test
```
30s  -> 10 VUs
10s  -> 100 VUs (spike)
30s  -> 100 VUs
10s  -> 10 VUs (drop)
30s  -> 10 VUs
```

## CI/CD Integration

```yaml
# .github/workflows/performance.yml
- name: Run K6 Tests
  run: |
    k6 run --out json=results.json tests/performance/k6/scenarios/voucher-api.js

- name: Upload Results
  uses: actions/upload-artifact@v4
  with:
    name: k6-results
    path: results.json
```

## Custom Metrics

| Metric | Description |
|--------|-------------|
| `login_duration` | Login request duration |
| `refresh_duration` | Token refresh duration |
| `voucher_create_duration` | Voucher creation duration |
| `voucher_list_duration` | Voucher list duration |
| `posting_duration` | Voucher posting duration |
| `auth_errors` | Authentication error count |
| `voucher_errors` | Voucher operation error count |
| `posting_errors` | Posting error count |
| `deadlock_errors` | Database deadlock error count |
| `balance_errors` | Balance validation error count |

## Troubleshooting

### Common Issues

1. **Connection refused**
   - Ensure API server is running
   - Check BASE_URL environment variable

2. **401 Unauthorized**
   - Verify test user credentials exist
   - Check token expiration

3. **High error rate**
   - Check server logs for errors
   - Verify database connection pool size
   - Check for rate limiting

4. **Deadlock errors**
   - Review transaction isolation level
   - Check for missing indexes
   - Optimize concurrent update queries

## Best Practices

1. **Baseline First**: Run smoke tests before full load tests
2. **Gradual Increase**: Use ramping stages, don't spike immediately
3. **Monitor Resources**: Watch CPU, memory, and DB connections
4. **Clean Up**: Reset test data between runs if needed
5. **Realistic Data**: Use representative test data volumes
