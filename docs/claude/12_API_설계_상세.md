# K-ERP v0.2 - API 설계 상세

**문서 버전**: 0.2.0
**작성일**: 2026-01-16
**상태**: 검토 대기

---

## 목차

1. [API 설계 원칙](#1-api-설계-원칙)
2. [인증/인가 API](#2-인증인가-api)
3. [회계 모듈 API](#3-회계-모듈-api)
4. [세금계산서 API](#4-세금계산서-api)
5. [인사/급여 API](#5-인사급여-api)
6. [공통 응답 형식](#6-공통-응답-형식)

---

## 1. API 설계 원칙

### 1.1 RESTful 원칙

| 원칙 | 설명 | 예시 |
|------|------|------|
| 리소스 기반 | URL은 명사로 표현 | /api/v1/invoices |
| HTTP Method | CRUD에 맞는 메서드 | GET, POST, PUT, DELETE |
| 상태 코드 | 적절한 HTTP 상태 코드 | 200, 201, 400, 404, 500 |
| 버전 관리 | URL에 버전 포함 | /api/v1/, /api/v2/ |
| 일관성 | 네이밍 컨벤션 통일 | snake_case |

### 1.2 URL 구조

```
https://api.kerp.io/api/v1/{resource}/{id}/{sub-resource}

예시:
GET    /api/v1/companies/{company_id}/invoices
POST   /api/v1/companies/{company_id}/invoices
GET    /api/v1/companies/{company_id}/invoices/{invoice_id}
PUT    /api/v1/companies/{company_id}/invoices/{invoice_id}
DELETE /api/v1/companies/{company_id}/invoices/{invoice_id}
POST   /api/v1/companies/{company_id}/invoices/{invoice_id}/issue
```

### 1.3 공통 헤더

```
# Request Headers
Authorization: Bearer {jwt_token}
Content-Type: application/json
Accept: application/json
X-Request-ID: {uuid}
X-Tenant-ID: {company_id}  # 멀티테넌시

# Response Headers
X-Request-ID: {uuid}
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640000000
```

---

## 2. 인증/인가 API

### 2.1 로그인

```yaml
POST /api/v1/auth/login

Request:
  Content-Type: application/json
  Body:
    email: string (required, email format)
    password: string (required, min 8 chars)

Response 200:
  {
    "success": true,
    "data": {
      "access_token": "eyJhbGciOiJSUzI1NiIs...",
      "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
      "token_type": "Bearer",
      "expires_in": 3600,
      "user": {
        "id": "uuid",
        "email": "user@example.com",
        "name": "홍길동",
        "company_id": "uuid",
        "company_name": "테스트 주식회사",
        "roles": ["admin", "accountant"]
      }
    }
  }

Response 401:
  {
    "success": false,
    "error": {
      "code": "AUTH_INVALID_CREDENTIALS",
      "message": "이메일 또는 비밀번호가 올바르지 않습니다."
    }
  }
```

### 2.2 토큰 갱신

```yaml
POST /api/v1/auth/refresh

Request:
  Content-Type: application/json
  Body:
    refresh_token: string (required)

Response 200:
  {
    "success": true,
    "data": {
      "access_token": "eyJhbGciOiJSUzI1NiIs...",
      "expires_in": 3600
    }
  }
```

### 2.3 로그아웃

```yaml
POST /api/v1/auth/logout

Request:
  Authorization: Bearer {access_token}

Response 200:
  {
    "success": true,
    "data": {
      "message": "로그아웃되었습니다."
    }
  }
```

---

## 3. 회계 모듈 API

### 3.1 계정과목

```yaml
# 계정과목 목록 조회
GET /api/v1/accounts

Query Parameters:
  type: string (asset, liability, equity, revenue, expense)
  level: integer (1-5)
  is_active: boolean
  search: string
  page: integer (default: 1)
  limit: integer (default: 20, max: 100)

Response 200:
  {
    "success": true,
    "data": {
      "items": [
        {
          "id": "uuid",
          "code": "101",
          "name": "현금",
          "type": "asset",
          "level": 1,
          "parent_id": null,
          "is_active": true,
          "created_at": "2026-01-01T00:00:00Z"
        }
      ],
      "pagination": {
        "page": 1,
        "limit": 20,
        "total_items": 150,
        "total_pages": 8
      }
    }
  }

# 계정과목 생성
POST /api/v1/accounts

Request:
  {
    "code": "10101",
    "name": "보통예금",
    "type": "asset",
    "parent_id": "uuid",
    "description": "은행 보통예금 계정"
  }

Response 201:
  {
    "success": true,
    "data": {
      "id": "uuid",
      "code": "10101",
      "name": "보통예금",
      ...
    }
  }
```

### 3.2 전표 관리

```yaml
# 전표 목록 조회
GET /api/v1/vouchers

Query Parameters:
  start_date: date (required, YYYY-MM-DD)
  end_date: date (required, YYYY-MM-DD)
  voucher_type: string (general, sales, purchase, payment, receipt)
  status: string (draft, approved, posted, cancelled)
  account_id: uuid
  search: string
  page: integer
  limit: integer

Response 200:
  {
    "success": true,
    "data": {
      "items": [
        {
          "id": "uuid",
          "voucher_no": "GV-2026-0001",
          "voucher_date": "2026-01-15",
          "voucher_type": "general",
          "status": "posted",
          "description": "사무용품 구입",
          "total_debit": 110000,
          "total_credit": 110000,
          "entries": [
            {
              "id": "uuid",
              "line_no": 1,
              "account_id": "uuid",
              "account_code": "803",
              "account_name": "소모품비",
              "debit_amount": 100000,
              "credit_amount": 0,
              "description": "사무용품"
            },
            {
              "id": "uuid",
              "line_no": 2,
              "account_id": "uuid",
              "account_code": "135",
              "account_name": "부가세대급금",
              "debit_amount": 10000,
              "credit_amount": 0,
              "description": "부가세"
            },
            {
              "id": "uuid",
              "line_no": 3,
              "account_id": "uuid",
              "account_code": "253",
              "account_name": "미지급금",
              "debit_amount": 0,
              "credit_amount": 110000,
              "description": ""
            }
          ],
          "created_by": "uuid",
          "created_at": "2026-01-15T10:30:00Z"
        }
      ],
      "pagination": {...}
    }
  }

# 전표 생성
POST /api/v1/vouchers

Request:
  {
    "voucher_date": "2026-01-15",
    "voucher_type": "general",
    "description": "사무용품 구입",
    "entries": [
      {
        "account_id": "uuid",
        "debit_amount": 100000,
        "credit_amount": 0,
        "description": "사무용품"
      },
      {
        "account_id": "uuid",
        "debit_amount": 10000,
        "credit_amount": 0,
        "description": "부가세"
      },
      {
        "account_id": "uuid",
        "debit_amount": 0,
        "credit_amount": 110000,
        "description": ""
      }
    ]
  }

Response 201:
  {
    "success": true,
    "data": {
      "id": "uuid",
      "voucher_no": "GV-2026-0002",
      ...
    }
  }

Response 400 (차대변 불균형):
  {
    "success": false,
    "error": {
      "code": "VOUCHER_UNBALANCED",
      "message": "차변과 대변의 합계가 일치하지 않습니다.",
      "details": {
        "total_debit": 110000,
        "total_credit": 100000,
        "difference": 10000
      }
    }
  }
```

### 3.3 재무제표

```yaml
# 재무상태표 조회
GET /api/v1/reports/balance-sheet

Query Parameters:
  as_of_date: date (required, YYYY-MM-DD)
  compare_date: date (optional, 비교 기간)

Response 200:
  {
    "success": true,
    "data": {
      "as_of_date": "2026-01-31",
      "currency": "KRW",
      "assets": {
        "current_assets": {
          "total": 150000000,
          "items": [
            {"code": "101", "name": "현금", "amount": 50000000},
            {"code": "103", "name": "보통예금", "amount": 80000000},
            {"code": "108", "name": "매출채권", "amount": 20000000}
          ]
        },
        "non_current_assets": {
          "total": 200000000,
          "items": [...]
        },
        "total": 350000000
      },
      "liabilities": {
        "current_liabilities": {...},
        "non_current_liabilities": {...},
        "total": 100000000
      },
      "equity": {
        "capital": 200000000,
        "retained_earnings": 50000000,
        "total": 250000000
      },
      "total_liabilities_and_equity": 350000000
    }
  }

# 손익계산서 조회
GET /api/v1/reports/income-statement

Query Parameters:
  start_date: date (required)
  end_date: date (required)
  compare_start_date: date (optional)
  compare_end_date: date (optional)

Response 200:
  {
    "success": true,
    "data": {
      "period": {
        "start_date": "2026-01-01",
        "end_date": "2026-01-31"
      },
      "revenue": {
        "total": 100000000,
        "items": [
          {"code": "401", "name": "매출", "amount": 100000000}
        ]
      },
      "cost_of_sales": {
        "total": 60000000,
        "items": [...]
      },
      "gross_profit": 40000000,
      "operating_expenses": {
        "total": 25000000,
        "items": [...]
      },
      "operating_income": 15000000,
      "non_operating_income": 1000000,
      "non_operating_expenses": 500000,
      "income_before_tax": 15500000,
      "income_tax": 3410000,
      "net_income": 12090000
    }
  }
```

---

## 4. 세금계산서 API

### 4.1 세금계산서 목록

```yaml
GET /api/v1/invoices

Query Parameters:
  type: string (sales, purchase)
  status: string (draft, issued, sent, cancelled)
  start_date: date
  end_date: date
  partner_id: uuid
  nts_confirm_num: string
  page: integer
  limit: integer

Response 200:
  {
    "success": true,
    "data": {
      "items": [
        {
          "id": "uuid",
          "invoice_no": "INV-2026-0001",
          "type": "sales",
          "status": "issued",
          "issue_date": "2026-01-15",
          "supply_amount": 1000000,
          "tax_amount": 100000,
          "total_amount": 1100000,
          "nts_confirm_num": "20260115-41234567-12345678",
          "supplier": {
            "business_number": "123-45-67890",
            "company_name": "테스트 주식회사",
            "representative": "홍길동"
          },
          "buyer": {
            "business_number": "234-56-78901",
            "company_name": "고객사 주식회사",
            "representative": "김철수"
          },
          "items": [
            {
              "date": "2026-01-15",
              "description": "소프트웨어 개발",
              "quantity": 1,
              "unit_price": 1000000,
              "supply_amount": 1000000,
              "tax_amount": 100000
            }
          ],
          "provider_used": "scraper",
          "created_at": "2026-01-15T09:00:00Z"
        }
      ],
      "pagination": {...},
      "summary": {
        "total_supply_amount": 50000000,
        "total_tax_amount": 5000000,
        "count_by_status": {
          "draft": 5,
          "issued": 45,
          "sent": 40,
          "cancelled": 2
        }
      }
    }
  }
```

### 4.2 세금계산서 발행

```yaml
POST /api/v1/invoices

Request:
  {
    "type": "sales",
    "issue_date": "2026-01-15",
    "buyer": {
      "business_number": "234-56-78901",
      "company_name": "고객사 주식회사",
      "representative": "김철수",
      "address": "서울시 강남구 테헤란로 123",
      "business_type": "서비스",
      "business_item": "소프트웨어 개발",
      "email": "buyer@customer.com"
    },
    "items": [
      {
        "date": "2026-01-15",
        "description": "소프트웨어 개발",
        "quantity": 1,
        "unit_price": 1000000
      }
    ],
    "notes": "1월 개발 대금",
    "auto_issue": true
  }

Response 202 (비동기 발행):
  {
    "success": true,
    "data": {
      "id": "uuid",
      "invoice_no": "INV-2026-0002",
      "status": "pending",
      "message": "세금계산서 발행이 요청되었습니다. 처리 완료 시 알림이 발송됩니다.",
      "estimated_completion": "2026-01-15T09:05:00Z"
    }
  }

Response 201 (동기 발행 완료):
  {
    "success": true,
    "data": {
      "id": "uuid",
      "invoice_no": "INV-2026-0002",
      "status": "issued",
      "nts_confirm_num": "20260115-41234567-12345679",
      "provider_used": "popbill",
      ...
    }
  }
```

### 4.3 세금계산서 수집 (홈택스 스크래핑)

```yaml
POST /api/v1/invoices/scrape

Request:
  {
    "type": "purchase",  // sales or purchase
    "start_date": "2026-01-01",
    "end_date": "2026-01-31",
    "cert_path": "/path/to/cert",  # 또는 cert_id로 저장된 인증서 참조
    "cert_password": "encrypted_password"
  }

Response 202:
  {
    "success": true,
    "data": {
      "job_id": "uuid",
      "status": "processing",
      "message": "세금계산서 수집이 시작되었습니다.",
      "estimated_completion": "2026-01-15T09:10:00Z"
    }
  }

# 수집 상태 조회
GET /api/v1/invoices/scrape/{job_id}

Response 200:
  {
    "success": true,
    "data": {
      "job_id": "uuid",
      "status": "completed",
      "result": {
        "total_found": 25,
        "new_imported": 20,
        "already_exists": 5,
        "failed": 0
      },
      "completed_at": "2026-01-15T09:08:30Z"
    }
  }
```

---

## 5. 인사/급여 API

### 5.1 사원 관리

```yaml
# 사원 목록
GET /api/v1/employees

Query Parameters:
  department_id: uuid
  status: string (active, resigned, on_leave)
  search: string
  page: integer
  limit: integer

Response 200:
  {
    "success": true,
    "data": {
      "items": [
        {
          "id": "uuid",
          "employee_no": "EMP-001",
          "name": "홍길동",
          "email": "hong@company.com",
          "department": {
            "id": "uuid",
            "name": "개발팀"
          },
          "position": {
            "id": "uuid",
            "name": "과장"
          },
          "hire_date": "2020-03-01",
          "status": "active",
          "insurance_info": {
            "nps_acquired": true,
            "nhis_acquired": true,
            "ei_acquired": true,
            "wci_acquired": true
          }
        }
      ],
      "pagination": {...}
    }
  }

# 사원 등록
POST /api/v1/employees

Request:
  {
    "name": "김철수",
    "resident_number": "encrypted_rrn",  # 암호화된 주민등록번호
    "email": "kim@company.com",
    "phone": "010-1234-5678",
    "address": "서울시 강남구",
    "department_id": "uuid",
    "position_id": "uuid",
    "hire_date": "2026-01-15",
    "salary_info": {
      "base_salary": 4000000,
      "payment_type": "monthly",
      "bank_code": "004",
      "account_number": "encrypted_account"
    },
    "insurance_acquire": {
      "nps": true,
      "nhis": true,
      "ei": true,
      "wci": true
    }
  }

Response 201:
  {
    "success": true,
    "data": {
      "id": "uuid",
      "employee_no": "EMP-002",
      ...
    }
  }
```

### 5.2 급여 계산

```yaml
# 급여 명세서 목록
GET /api/v1/payrolls

Query Parameters:
  year: integer (required)
  month: integer (required)
  department_id: uuid
  status: string (draft, calculated, confirmed, paid)

Response 200:
  {
    "success": true,
    "data": {
      "period": {
        "year": 2026,
        "month": 1
      },
      "items": [
        {
          "id": "uuid",
          "employee_id": "uuid",
          "employee_name": "홍길동",
          "employee_no": "EMP-001",
          "status": "calculated",
          "earnings": {
            "base_salary": 4000000,
            "overtime_pay": 200000,
            "bonus": 0,
            "allowances": 100000,
            "total": 4300000
          },
          "deductions": {
            "income_tax": 187000,
            "local_income_tax": 18700,
            "nps": 193500,
            "nhis": 153510,
            "ei": 38700,
            "other": 0,
            "total": 591410
          },
          "net_pay": 3708590
        }
      ],
      "summary": {
        "total_employees": 50,
        "total_earnings": 215000000,
        "total_deductions": 29570500,
        "total_net_pay": 185429500
      }
    }
  }

# 급여 계산 실행
POST /api/v1/payrolls/calculate

Request:
  {
    "year": 2026,
    "month": 1,
    "employee_ids": ["uuid1", "uuid2"],  # null이면 전체 사원
    "include_overtime": true
  }

Response 202:
  {
    "success": true,
    "data": {
      "job_id": "uuid",
      "status": "processing",
      "message": "급여 계산이 시작되었습니다."
    }
  }
```

### 5.3 4대보험 신고

```yaml
# 4대보험 신고 요청
POST /api/v1/insurance/reports

Request:
  {
    "report_type": "acquisition",  # acquisition, loss, change
    "agency": "all",  # all, nps, nhis, ei, wci
    "employee_ids": ["uuid1", "uuid2"],
    "effective_date": "2026-01-15"
  }

Response 202:
  {
    "success": true,
    "data": {
      "job_id": "uuid",
      "status": "processing",
      "reports": [
        {"agency": "nps", "status": "pending"},
        {"agency": "nhis", "status": "pending"},
        {"agency": "ei", "status": "pending"},
        {"agency": "wci", "status": "pending"}
      ]
    }
  }

# 신고 결과 조회
GET /api/v1/insurance/reports/{job_id}

Response 200:
  {
    "success": true,
    "data": {
      "job_id": "uuid",
      "status": "completed",
      "reports": [
        {
          "agency": "nps",
          "status": "success",
          "receipt_number": "NPS-2026-0001",
          "submitted_at": "2026-01-15T10:30:00Z"
        },
        {
          "agency": "nhis",
          "status": "success",
          "receipt_number": "NHIS-2026-0001",
          "submitted_at": "2026-01-15T10:31:00Z"
        },
        ...
      ]
    }
  }
```

---

## 6. 공통 응답 형식

### 6.1 성공 응답

```json
{
  "success": true,
  "data": {
    // 실제 데이터
  },
  "meta": {
    "request_id": "uuid",
    "timestamp": "2026-01-15T10:30:00Z"
  }
}
```

### 6.2 에러 응답

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "사용자에게 보여줄 메시지",
    "details": {
      // 추가 정보 (개발용)
    },
    "field_errors": [
      {
        "field": "email",
        "message": "올바른 이메일 형식이 아닙니다."
      }
    ]
  },
  "meta": {
    "request_id": "uuid",
    "timestamp": "2026-01-15T10:30:00Z"
  }
}
```

### 6.3 에러 코드 체계

| 코드 | HTTP Status | 설명 |
|------|-------------|------|
| AUTH_* | 401, 403 | 인증/인가 에러 |
| VALIDATION_* | 400 | 입력값 검증 에러 |
| RESOURCE_NOT_FOUND | 404 | 리소스 없음 |
| RESOURCE_CONFLICT | 409 | 리소스 충돌 |
| VOUCHER_* | 400 | 전표 관련 에러 |
| INVOICE_* | 400, 500 | 세금계산서 에러 |
| INSURANCE_* | 400, 500 | 4대보험 에러 |
| EXTERNAL_* | 502, 503 | 외부 연동 에러 |
| INTERNAL_ERROR | 500 | 내부 서버 에러 |

### 6.4 페이지네이션

```json
{
  "pagination": {
    "page": 1,
    "limit": 20,
    "total_items": 150,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  }
}
```

---

**다음 문서**: [13_DB_스키마_상세.md](./13_DB_스키마_상세.md)
