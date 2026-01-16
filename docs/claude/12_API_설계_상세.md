# 12. API 설계 상세

## 개요

K-ERP 시스템의 REST API 설계 상세 문서.
OpenAPI 3.0 스펙 기반으로 API 명세 정의.

### API 설계 원칙

| 원칙 | 설명 |
|------|------|
| RESTful | 자원 중심 설계, HTTP 메서드 활용 |
| Versioning | URL 경로 기반 버저닝 (`/api/v1/`) |
| Pagination | 커서 기반 페이지네이션 |
| Error Handling | RFC 7807 Problem Details 형식 |
| Security | JWT Bearer Token + RBAC |

---

## 1. OpenAPI 기본 정의

```yaml
# api/openapi.yaml
openapi: 3.0.3
info:
  title: K-ERP API
  description: |
    한국형 클라우드 ERP 시스템 API

    ## 인증
    모든 API는 Bearer Token 인증이 필요합니다.
    `/api/v1/auth/login` 엔드포인트에서 토큰을 발급받으세요.

    ## 멀티테넌시 (3중 계층 보안)
    API 요청 시 `X-Company-ID` 헤더로 회사를 지정합니다.
    토큰에 포함된 회사와 일치해야 합니다.

    **보안 계층:**
    1. **API 헤더**: X-Company-ID로 명시적 회사 지정
    2. **JWT 검증**: 미들웨어에서 토큰의 company_id와 헤더 일치 검증
    3. **DB RLS**: PostgreSQL Row Level Security로 최종 데이터 격리

    > 상세 구현: `08_멀티테넌트_구현.md` 참조

    ## Rate Limiting
    - 인증된 요청: 1000 requests/minute
    - 비인증 요청: 60 requests/minute
  version: 1.0.0
  contact:
    name: K-ERP Support
    email: support@kerp.example.com
  license:
    name: Proprietary
    url: https://kerp.example.com/license

servers:
  - url: https://api.kerp.example.com/api/v1
    description: Production
  - url: https://staging-api.kerp.example.com/api/v1
    description: Staging
  - url: http://localhost:8080/api/v1
    description: Local Development

tags:
  - name: Auth
    description: 인증/인가
  - name: Companies
    description: 회사 정보 관리
  - name: Accounts
    description: 계정과목 관리
  - name: Vouchers
    description: 전표 관리
  - name: Ledgers
    description: 원장 조회
  - name: Reports
    description: 재무제표
  - name: TaxInvoices
    description: 세금계산서
  - name: Employees
    description: 직원 관리
  - name: Payroll
    description: 급여 관리
  - name: Inventory
    description: 재고 관리

security:
  - BearerAuth: []

components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT 액세스 토큰

  parameters:
    CompanyIdHeader:
      name: X-Company-ID
      in: header
      required: true
      schema:
        type: string
        format: uuid
      description: 회사 ID

    PageCursor:
      name: cursor
      in: query
      schema:
        type: string
      description: 페이지네이션 커서

    PageSize:
      name: limit
      in: query
      schema:
        type: integer
        minimum: 1
        maximum: 100
        default: 20
      description: 페이지당 항목 수

    SortBy:
      name: sort
      in: query
      schema:
        type: string
      description: 정렬 필드 (예: created_at:desc)

  schemas:
    # ============ 공통 스키마 ============
    UUID:
      type: string
      format: uuid
      example: "550e8400-e29b-41d4-a716-446655440000"

    Timestamp:
      type: string
      format: date-time
      example: "2024-01-15T09:30:00Z"

    Money:
      type: object
      properties:
        amount:
          type: number
          format: double
          description: 금액
        currency:
          type: string
          default: KRW
          enum: [KRW, USD, EUR, JPY]
      required:
        - amount

    PaginationMeta:
      type: object
      properties:
        next_cursor:
          type: string
          nullable: true
          description: 다음 페이지 커서
        has_more:
          type: boolean
          description: 추가 데이터 존재 여부
        total_count:
          type: integer
          description: 전체 항목 수 (옵션)

    # ============ 에러 스키마 (RFC 7807) ============
    ProblemDetail:
      type: object
      properties:
        type:
          type: string
          format: uri
          description: 에러 타입 URI
          example: "https://api.kerp.example.com/errors/validation"
        title:
          type: string
          description: 에러 제목
          example: "Validation Error"
        status:
          type: integer
          description: HTTP 상태 코드
          example: 400
        detail:
          type: string
          description: 상세 에러 메시지
          example: "The request body contains invalid fields"
        instance:
          type: string
          format: uri
          description: 에러 발생 경로
          example: "/api/v1/vouchers"
        errors:
          type: array
          items:
            $ref: '#/components/schemas/FieldError'
          description: 필드별 에러 목록
        trace_id:
          type: string
          description: 추적 ID
          example: "abc123xyz"
      required:
        - type
        - title
        - status

    FieldError:
      type: object
      properties:
        field:
          type: string
          description: 필드명
          example: "debit_amount"
        message:
          type: string
          description: 에러 메시지
          example: "must be greater than 0"
        code:
          type: string
          description: 에러 코드
          example: "INVALID_VALUE"

  responses:
    BadRequest:
      description: 잘못된 요청
      content:
        application/problem+json:
          schema:
            $ref: '#/components/schemas/ProblemDetail'
          example:
            type: "https://api.kerp.example.com/errors/validation"
            title: "Validation Error"
            status: 400
            detail: "Request validation failed"
            errors:
              - field: "voucher_date"
                message: "required field"
                code: "REQUIRED"

    Unauthorized:
      description: 인증 필요
      content:
        application/problem+json:
          schema:
            $ref: '#/components/schemas/ProblemDetail'
          example:
            type: "https://api.kerp.example.com/errors/unauthorized"
            title: "Unauthorized"
            status: 401
            detail: "Invalid or expired token"

    Forbidden:
      description: 권한 없음
      content:
        application/problem+json:
          schema:
            $ref: '#/components/schemas/ProblemDetail'
          example:
            type: "https://api.kerp.example.com/errors/forbidden"
            title: "Forbidden"
            status: 403
            detail: "You don't have permission to access this resource"

    NotFound:
      description: 리소스를 찾을 수 없음
      content:
        application/problem+json:
          schema:
            $ref: '#/components/schemas/ProblemDetail'
          example:
            type: "https://api.kerp.example.com/errors/not-found"
            title: "Not Found"
            status: 404
            detail: "The requested resource was not found"

    TooManyRequests:
      description: 요청 제한 초과
      headers:
        X-RateLimit-Limit:
          schema:
            type: integer
          description: Rate limit 한도
        X-RateLimit-Remaining:
          schema:
            type: integer
          description: 남은 요청 수
        X-RateLimit-Reset:
          schema:
            type: integer
          description: 리셋 시간 (Unix timestamp)
        Retry-After:
          schema:
            type: integer
          description: 재시도 대기 시간 (초)
      content:
        application/problem+json:
          schema:
            $ref: '#/components/schemas/ProblemDetail'
          example:
            type: "https://api.kerp.example.com/errors/rate-limited"
            title: "Too Many Requests"
            status: 429
            detail: "Rate limit exceeded. Please retry after 60 seconds."
```

---

## 2. 인증 API

```yaml
# api/paths/auth.yaml
paths:
  /auth/login:
    post:
      tags:
        - Auth
      summary: 로그인
      description: 이메일/비밀번호로 로그인하여 JWT 토큰 발급
      operationId: login
      security: []  # 인증 불필요
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                email:
                  type: string
                  format: email
                  example: "user@example.com"
                password:
                  type: string
                  format: password
                  minLength: 8
                  example: "SecureP@ss123"
                company_id:
                  type: string
                  format: uuid
                  description: 로그인할 회사 ID
              required:
                - email
                - password
                - company_id
      responses:
        '200':
          description: 로그인 성공
          content:
            application/json:
              schema:
                type: object
                properties:
                  access_token:
                    type: string
                    description: JWT 액세스 토큰
                  refresh_token:
                    type: string
                    description: 리프레시 토큰
                  token_type:
                    type: string
                    default: Bearer
                  expires_in:
                    type: integer
                    description: 만료 시간 (초)
                    example: 1800
                  user:
                    $ref: '#/components/schemas/UserInfo'
        '401':
          $ref: '#/components/responses/Unauthorized'

  /auth/refresh:
    post:
      tags:
        - Auth
      summary: 토큰 갱신
      description: 리프레시 토큰으로 새 액세스 토큰 발급
      operationId: refreshToken
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                refresh_token:
                  type: string
              required:
                - refresh_token
      responses:
        '200':
          description: 토큰 갱신 성공
          content:
            application/json:
              schema:
                type: object
                properties:
                  access_token:
                    type: string
                  expires_in:
                    type: integer
        '401':
          $ref: '#/components/responses/Unauthorized'

  /auth/logout:
    post:
      tags:
        - Auth
      summary: 로그아웃
      description: 현재 토큰 무효화
      operationId: logout
      responses:
        '204':
          description: 로그아웃 성공

  /auth/me:
    get:
      tags:
        - Auth
      summary: 현재 사용자 정보
      description: 현재 로그인한 사용자 정보 조회
      operationId: getCurrentUser
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserInfo'

components:
  schemas:
    UserInfo:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        email:
          type: string
          format: email
        name:
          type: string
        company_id:
          $ref: '#/components/schemas/UUID'
        company_name:
          type: string
        roles:
          type: array
          items:
            type: string
          example: ["ADMIN", "ACCOUNTANT"]
        permissions:
          type: array
          items:
            type: string
          example: ["voucher:create", "voucher:approve", "report:read"]
```

---

## 3. 전표 API

```yaml
# api/paths/vouchers.yaml
paths:
  /vouchers:
    get:
      tags:
        - Vouchers
      summary: 전표 목록 조회
      description: 조건에 맞는 전표 목록을 조회합니다
      operationId: listVouchers
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - $ref: '#/components/parameters/PageCursor'
        - $ref: '#/components/parameters/PageSize'
        - name: start_date
          in: query
          schema:
            type: string
            format: date
          description: 시작일
        - name: end_date
          in: query
          schema:
            type: string
            format: date
          description: 종료일
        - name: status
          in: query
          schema:
            type: string
            enum: [DRAFT, PENDING, APPROVED, REJECTED, POSTED]
          description: 전표 상태
        - name: voucher_type
          in: query
          schema:
            type: string
            enum: [GENERAL, SALES, PURCHASE, RECEIPT, PAYMENT, ADJUSTMENT]
          description: 전표 유형
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/VoucherSummary'
                  meta:
                    $ref: '#/components/schemas/PaginationMeta'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'

    post:
      tags:
        - Vouchers
      summary: 전표 생성
      description: 새 전표를 생성합니다. 차변/대변 합계가 일치해야 합니다.
      operationId: createVoucher
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/VoucherCreateRequest'
      responses:
        '201':
          description: 전표 생성 성공
          headers:
            Location:
              schema:
                type: string
              description: 생성된 전표 URL
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Voucher'
        '400':
          $ref: '#/components/responses/BadRequest'
        '422':
          description: 차대변 불균형
          content:
            application/problem+json:
              schema:
                $ref: '#/components/schemas/ProblemDetail'
              example:
                type: "https://api.kerp.example.com/errors/unbalanced-voucher"
                title: "Unbalanced Voucher"
                status: 422
                detail: "Total debit (1,000,000) does not equal total credit (900,000)"

  /vouchers/{voucherId}:
    get:
      tags:
        - Vouchers
      summary: 전표 상세 조회
      operationId: getVoucher
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: voucherId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Voucher'
        '404':
          $ref: '#/components/responses/NotFound'

    put:
      tags:
        - Vouchers
      summary: 전표 수정
      description: 초안(DRAFT) 상태의 전표만 수정 가능
      operationId: updateVoucher
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: voucherId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/VoucherUpdateRequest'
      responses:
        '200':
          description: 수정 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Voucher'
        '409':
          description: 상태 충돌 (수정 불가 상태)
          content:
            application/problem+json:
              schema:
                $ref: '#/components/schemas/ProblemDetail'

    delete:
      tags:
        - Vouchers
      summary: 전표 삭제
      description: 초안(DRAFT) 상태의 전표만 삭제 가능
      operationId: deleteVoucher
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: voucherId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      responses:
        '204':
          description: 삭제 성공
        '409':
          description: 상태 충돌

  /vouchers/{voucherId}/approve:
    post:
      tags:
        - Vouchers
      summary: 전표 승인
      description: 대기(PENDING) 상태의 전표를 승인합니다
      operationId: approveVoucher
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: voucherId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                comment:
                  type: string
                  description: 승인 코멘트
      responses:
        '200':
          description: 승인 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Voucher'

  /vouchers/{voucherId}/reject:
    post:
      tags:
        - Vouchers
      summary: 전표 반려
      operationId: rejectVoucher
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: voucherId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                reason:
                  type: string
                  description: 반려 사유
              required:
                - reason
      responses:
        '200':
          description: 반려 성공

  /vouchers/{voucherId}/post:
    post:
      tags:
        - Vouchers
      summary: 전표 기장
      description: 승인된 전표를 원장에 기장합니다
      operationId: postVoucher
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: voucherId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      responses:
        '200':
          description: 기장 성공

components:
  schemas:
    VoucherStatus:
      type: string
      enum:
        - DRAFT      # 초안
        - PENDING    # 승인 대기
        - APPROVED   # 승인됨
        - REJECTED   # 반려됨
        - POSTED     # 기장됨

    VoucherType:
      type: string
      enum:
        - GENERAL    # 일반전표
        - SALES      # 매출전표
        - PURCHASE   # 매입전표
        - RECEIPT    # 수금전표
        - PAYMENT    # 지급전표
        - ADJUSTMENT # 조정전표

    VoucherSummary:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        voucher_number:
          type: string
          description: 전표번호
          example: "V2024-01-0001"
        voucher_date:
          type: string
          format: date
        voucher_type:
          $ref: '#/components/schemas/VoucherType'
        status:
          $ref: '#/components/schemas/VoucherStatus'
        description:
          type: string
        total_amount:
          type: number
          format: double
          description: 차변 합계 금액
        created_at:
          $ref: '#/components/schemas/Timestamp'

    Voucher:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        company_id:
          $ref: '#/components/schemas/UUID'
        voucher_number:
          type: string
        voucher_date:
          type: string
          format: date
        voucher_type:
          $ref: '#/components/schemas/VoucherType'
        status:
          $ref: '#/components/schemas/VoucherStatus'
        description:
          type: string
        lines:
          type: array
          items:
            $ref: '#/components/schemas/VoucherLine'
        attachments:
          type: array
          items:
            $ref: '#/components/schemas/Attachment'
        created_by:
          $ref: '#/components/schemas/UUID'
        approved_by:
          $ref: '#/components/schemas/UUID'
        approved_at:
          $ref: '#/components/schemas/Timestamp'
        posted_at:
          $ref: '#/components/schemas/Timestamp'
        created_at:
          $ref: '#/components/schemas/Timestamp'
        updated_at:
          $ref: '#/components/schemas/Timestamp'

    VoucherLine:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        line_number:
          type: integer
          minimum: 1
        account_code:
          type: string
          description: 계정과목 코드
          example: "1101"
        account_name:
          type: string
          description: 계정과목명
          example: "현금"
        debit_amount:
          type: number
          format: double
          minimum: 0
          description: 차변 금액
        credit_amount:
          type: number
          format: double
          minimum: 0
          description: 대변 금액
        description:
          type: string
          description: 적요
        partner_id:
          $ref: '#/components/schemas/UUID'
        partner_name:
          type: string
          description: 거래처명
        cost_center_id:
          $ref: '#/components/schemas/UUID'
        cost_center_name:
          type: string
          description: 코스트센터명

    VoucherCreateRequest:
      type: object
      properties:
        voucher_date:
          type: string
          format: date
        voucher_type:
          $ref: '#/components/schemas/VoucherType'
        description:
          type: string
          maxLength: 500
        lines:
          type: array
          minItems: 2
          items:
            $ref: '#/components/schemas/VoucherLineRequest'
      required:
        - voucher_date
        - voucher_type
        - lines

    VoucherLineRequest:
      type: object
      properties:
        account_code:
          type: string
          pattern: "^[0-9]{4}$"
        debit_amount:
          type: number
          format: double
          minimum: 0
        credit_amount:
          type: number
          format: double
          minimum: 0
        description:
          type: string
          maxLength: 200
        partner_id:
          $ref: '#/components/schemas/UUID'
        cost_center_id:
          $ref: '#/components/schemas/UUID'
      required:
        - account_code

    VoucherUpdateRequest:
      type: object
      properties:
        voucher_date:
          type: string
          format: date
        description:
          type: string
        lines:
          type: array
          minItems: 2
          items:
            $ref: '#/components/schemas/VoucherLineRequest'

    Attachment:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        filename:
          type: string
        file_size:
          type: integer
        mime_type:
          type: string
        url:
          type: string
          format: uri
```

---

## 4. 세금계산서 API

```yaml
# api/paths/tax-invoices.yaml
paths:
  /tax-invoices:
    get:
      tags:
        - TaxInvoices
      summary: 세금계산서 목록 조회
      operationId: listTaxInvoices
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - $ref: '#/components/parameters/PageCursor'
        - $ref: '#/components/parameters/PageSize'
        - name: issue_type
          in: query
          schema:
            type: string
            enum: [ISSUED, RECEIVED]
          description: 발행/수취 구분
        - name: status
          in: query
          schema:
            type: string
            enum: [DRAFT, ISSUED, SENT, ACCEPTED, REFUSED, CANCELLED]
        - name: start_date
          in: query
          schema:
            type: string
            format: date
        - name: end_date
          in: query
          schema:
            type: string
            format: date
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/TaxInvoiceSummary'
                  meta:
                    $ref: '#/components/schemas/PaginationMeta'

    post:
      tags:
        - TaxInvoices
      summary: 세금계산서 생성
      operationId: createTaxInvoice
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TaxInvoiceCreateRequest'
      responses:
        '201':
          description: 생성 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TaxInvoice'

  /tax-invoices/{invoiceId}:
    get:
      tags:
        - TaxInvoices
      summary: 세금계산서 상세 조회
      operationId: getTaxInvoice
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: invoiceId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TaxInvoice'

  /tax-invoices/{invoiceId}/issue:
    post:
      tags:
        - TaxInvoices
      summary: 세금계산서 발행
      description: 세금계산서를 국세청에 발행합니다 (Popbill 연동)
      operationId: issueTaxInvoice
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: invoiceId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                send_email:
                  type: boolean
                  default: true
                  description: 거래처에 이메일 전송 여부
      responses:
        '200':
          description: 발행 성공
          content:
            application/json:
              schema:
                type: object
                properties:
                  nts_confirm_num:
                    type: string
                    description: 국세청 승인번호
                  issued_at:
                    $ref: '#/components/schemas/Timestamp'
        '422':
          description: 발행 실패
          content:
            application/problem+json:
              schema:
                $ref: '#/components/schemas/ProblemDetail'
              example:
                type: "https://api.kerp.example.com/errors/tax-invoice-issue-failed"
                title: "Tax Invoice Issue Failed"
                status: 422
                detail: "Failed to issue tax invoice: Invalid business number"

  /tax-invoices/{invoiceId}/cancel:
    post:
      tags:
        - TaxInvoices
      summary: 세금계산서 취소
      operationId: cancelTaxInvoice
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: invoiceId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                reason:
                  type: string
                  description: 취소 사유
              required:
                - reason
      responses:
        '200':
          description: 취소 성공

  /tax-invoices/bulk-issue:
    post:
      tags:
        - TaxInvoices
      summary: 세금계산서 일괄 발행
      operationId: bulkIssueTaxInvoices
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                invoice_ids:
                  type: array
                  items:
                    $ref: '#/components/schemas/UUID'
                  minItems: 1
                  maxItems: 100
              required:
                - invoice_ids
      responses:
        '200':
          description: 일괄 발행 결과
          content:
            application/json:
              schema:
                type: object
                properties:
                  success_count:
                    type: integer
                  failure_count:
                    type: integer
                  results:
                    type: array
                    items:
                      type: object
                      properties:
                        invoice_id:
                          $ref: '#/components/schemas/UUID'
                        success:
                          type: boolean
                        nts_confirm_num:
                          type: string
                        error:
                          type: string

components:
  schemas:
    TaxInvoiceType:
      type: string
      enum:
        - TAX        # 세금계산서
        - ZERO_TAX   # 영세율 세금계산서
        - RECEIPT    # 계산서 (면세)

    TaxInvoiceStatus:
      type: string
      enum:
        - DRAFT      # 임시저장
        - ISSUED     # 발행됨
        - SENT       # 전송됨
        - ACCEPTED   # 승인됨
        - REFUSED    # 거부됨
        - CANCELLED  # 취소됨

    TaxInvoiceSummary:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        mgt_key:
          type: string
          description: 관리번호 (문서번호)
        issue_type:
          type: string
          enum: [ISSUED, RECEIVED]
        invoice_type:
          $ref: '#/components/schemas/TaxInvoiceType'
        status:
          $ref: '#/components/schemas/TaxInvoiceStatus'
        invoice_date:
          type: string
          format: date
        supply_value:
          type: number
          description: 공급가액
        tax_amount:
          type: number
          description: 세액
        total_amount:
          type: number
          description: 합계금액
        partner_name:
          type: string
          description: 거래처명
        nts_confirm_num:
          type: string
          description: 국세청 승인번호
        created_at:
          $ref: '#/components/schemas/Timestamp'

    TaxInvoice:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        company_id:
          $ref: '#/components/schemas/UUID'
        mgt_key:
          type: string
          description: 관리번호
        issue_type:
          type: string
          enum: [ISSUED, RECEIVED]
        invoice_type:
          $ref: '#/components/schemas/TaxInvoiceType'
        status:
          $ref: '#/components/schemas/TaxInvoiceStatus'
        invoice_date:
          type: string
          format: date
        # 공급자 정보
        supplier:
          $ref: '#/components/schemas/TaxInvoiceParty'
        # 공급받는자 정보
        buyer:
          $ref: '#/components/schemas/TaxInvoiceParty'
        # 품목 정보
        items:
          type: array
          items:
            $ref: '#/components/schemas/TaxInvoiceItem'
        # 금액 정보
        supply_value:
          type: number
        tax_amount:
          type: number
        total_amount:
          type: number
        # 국세청 정보
        nts_confirm_num:
          type: string
        nts_result_code:
          type: string
        # 이력
        issued_at:
          $ref: '#/components/schemas/Timestamp'
        sent_at:
          $ref: '#/components/schemas/Timestamp'
        created_at:
          $ref: '#/components/schemas/Timestamp'
        updated_at:
          $ref: '#/components/schemas/Timestamp'

    TaxInvoiceParty:
      type: object
      properties:
        corp_num:
          type: string
          pattern: "^[0-9]{10}$"
          description: 사업자등록번호 (하이픈 없이)
        corp_name:
          type: string
          description: 상호
        ceo_name:
          type: string
          description: 대표자명
        address:
          type: string
          description: 주소
        biz_type:
          type: string
          description: 업태
        biz_class:
          type: string
          description: 종목
        email:
          type: string
          format: email
        contact_name:
          type: string
          description: 담당자명
        tel:
          type: string
          description: 전화번호

    TaxInvoiceItem:
      type: object
      properties:
        serial_num:
          type: integer
          minimum: 1
          maximum: 99
          description: 일련번호
        trade_date:
          type: string
          format: date
          description: 거래일자
        item_name:
          type: string
          description: 품목명
        spec:
          type: string
          description: 규격
        quantity:
          type: number
          description: 수량
        unit_price:
          type: number
          description: 단가
        supply_value:
          type: number
          description: 공급가액
        tax_amount:
          type: number
          description: 세액
        remark:
          type: string
          description: 비고

    TaxInvoiceCreateRequest:
      type: object
      properties:
        invoice_type:
          $ref: '#/components/schemas/TaxInvoiceType'
        invoice_date:
          type: string
          format: date
        buyer:
          $ref: '#/components/schemas/TaxInvoicePartyRequest'
        items:
          type: array
          minItems: 1
          maxItems: 99
          items:
            $ref: '#/components/schemas/TaxInvoiceItemRequest'
        remark:
          type: string
          description: 비고
      required:
        - invoice_type
        - invoice_date
        - buyer
        - items

    TaxInvoicePartyRequest:
      type: object
      properties:
        corp_num:
          type: string
          pattern: "^[0-9]{10}$"
        corp_name:
          type: string
        ceo_name:
          type: string
        address:
          type: string
        email:
          type: string
          format: email
      required:
        - corp_num
        - corp_name

    TaxInvoiceItemRequest:
      type: object
      properties:
        trade_date:
          type: string
          format: date
        item_name:
          type: string
        spec:
          type: string
        quantity:
          type: number
        unit_price:
          type: number
        supply_value:
          type: number
        tax_amount:
          type: number
      required:
        - item_name
        - supply_value
```

---

## 5. 재무제표 API

```yaml
# api/paths/reports.yaml
paths:
  /reports/balance-sheet:
    get:
      tags:
        - Reports
      summary: 재무상태표 조회
      description: 특정 기준일의 재무상태표를 조회합니다
      operationId: getBalanceSheet
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: as_of_date
          in: query
          required: true
          schema:
            type: string
            format: date
          description: 기준일
        - name: compare_date
          in: query
          schema:
            type: string
            format: date
          description: 비교 기준일 (전년 동기)
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/BalanceSheet'

  /reports/income-statement:
    get:
      tags:
        - Reports
      summary: 손익계산서 조회
      operationId: getIncomeStatement
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: start_date
          in: query
          required: true
          schema:
            type: string
            format: date
        - name: end_date
          in: query
          required: true
          schema:
            type: string
            format: date
        - name: compare_start_date
          in: query
          schema:
            type: string
            format: date
        - name: compare_end_date
          in: query
          schema:
            type: string
            format: date
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/IncomeStatement'

  /reports/trial-balance:
    get:
      tags:
        - Reports
      summary: 시산표 조회
      operationId: getTrialBalance
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: start_date
          in: query
          required: true
          schema:
            type: string
            format: date
        - name: end_date
          in: query
          required: true
          schema:
            type: string
            format: date
        - name: level
          in: query
          schema:
            type: integer
            minimum: 1
            maximum: 4
            default: 2
          description: 계정과목 표시 수준
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TrialBalance'

  /reports/general-ledger:
    get:
      tags:
        - Reports
      summary: 총계정원장 조회
      operationId: getGeneralLedger
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - $ref: '#/components/parameters/PageCursor'
        - $ref: '#/components/parameters/PageSize'
        - name: account_code
          in: query
          required: true
          schema:
            type: string
          description: 계정과목 코드
        - name: start_date
          in: query
          required: true
          schema:
            type: string
            format: date
        - name: end_date
          in: query
          required: true
          schema:
            type: string
            format: date
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                type: object
                properties:
                  account:
                    $ref: '#/components/schemas/AccountInfo'
                  opening_balance:
                    type: number
                  closing_balance:
                    type: number
                  entries:
                    type: array
                    items:
                      $ref: '#/components/schemas/LedgerEntry'
                  meta:
                    $ref: '#/components/schemas/PaginationMeta'

components:
  schemas:
    BalanceSheet:
      type: object
      properties:
        as_of_date:
          type: string
          format: date
        compare_date:
          type: string
          format: date
        assets:
          $ref: '#/components/schemas/FinancialSection'
        liabilities:
          $ref: '#/components/schemas/FinancialSection'
        equity:
          $ref: '#/components/schemas/FinancialSection'
        total_assets:
          type: number
        total_liabilities_equity:
          type: number

    IncomeStatement:
      type: object
      properties:
        period:
          type: object
          properties:
            start_date:
              type: string
              format: date
            end_date:
              type: string
              format: date
        revenue:
          $ref: '#/components/schemas/FinancialSection'
        cost_of_sales:
          $ref: '#/components/schemas/FinancialSection'
        gross_profit:
          type: number
        operating_expenses:
          $ref: '#/components/schemas/FinancialSection'
        operating_income:
          type: number
        non_operating:
          $ref: '#/components/schemas/FinancialSection'
        income_before_tax:
          type: number
        income_tax:
          type: number
        net_income:
          type: number

    FinancialSection:
      type: object
      properties:
        items:
          type: array
          items:
            $ref: '#/components/schemas/FinancialLineItem'
        subtotal:
          type: number
        compare_subtotal:
          type: number
        change_amount:
          type: number
        change_percent:
          type: number

    FinancialLineItem:
      type: object
      properties:
        account_code:
          type: string
        account_name:
          type: string
        level:
          type: integer
        amount:
          type: number
        compare_amount:
          type: number
        change_amount:
          type: number
        change_percent:
          type: number
        is_subtotal:
          type: boolean

    TrialBalance:
      type: object
      properties:
        period:
          type: object
          properties:
            start_date:
              type: string
              format: date
            end_date:
              type: string
              format: date
        accounts:
          type: array
          items:
            $ref: '#/components/schemas/TrialBalanceAccount'
        totals:
          type: object
          properties:
            opening_debit:
              type: number
            opening_credit:
              type: number
            period_debit:
              type: number
            period_credit:
              type: number
            closing_debit:
              type: number
            closing_credit:
              type: number

    TrialBalanceAccount:
      type: object
      properties:
        account_code:
          type: string
        account_name:
          type: string
        level:
          type: integer
        opening_debit:
          type: number
        opening_credit:
          type: number
        period_debit:
          type: number
        period_credit:
          type: number
        closing_debit:
          type: number
        closing_credit:
          type: number

    LedgerEntry:
      type: object
      properties:
        voucher_id:
          $ref: '#/components/schemas/UUID'
        voucher_number:
          type: string
        voucher_date:
          type: string
          format: date
        description:
          type: string
        debit_amount:
          type: number
        credit_amount:
          type: number
        balance:
          type: number

    AccountInfo:
      type: object
      properties:
        code:
          type: string
        name:
          type: string
        type:
          type: string
          enum: [ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE]
```

---

## 6. 급여 API

```yaml
# api/paths/payroll.yaml
paths:
  /payroll/periods:
    get:
      tags:
        - Payroll
      summary: 급여 기간 목록
      operationId: listPayrollPeriods
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: year
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/PayrollPeriod'

    post:
      tags:
        - Payroll
      summary: 급여 기간 생성
      operationId: createPayrollPeriod
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                year:
                  type: integer
                month:
                  type: integer
                  minimum: 1
                  maximum: 12
                pay_date:
                  type: string
                  format: date
              required:
                - year
                - month
                - pay_date
      responses:
        '201':
          description: 생성 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/PayrollPeriod'

  /payroll/periods/{periodId}/calculate:
    post:
      tags:
        - Payroll
      summary: 급여 일괄 계산
      description: 해당 기간의 모든 직원 급여를 계산합니다
      operationId: calculatePayroll
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: periodId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      responses:
        '202':
          description: 계산 시작됨
          content:
            application/json:
              schema:
                type: object
                properties:
                  job_id:
                    type: string
                    description: 백그라운드 작업 ID
                  status:
                    type: string
                    enum: [PROCESSING]

  /payroll/periods/{periodId}/payslips:
    get:
      tags:
        - Payroll
      summary: 급여 명세서 목록
      operationId: listPayslips
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - $ref: '#/components/parameters/PageCursor'
        - $ref: '#/components/parameters/PageSize'
        - name: periodId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/PayslipSummary'
                  meta:
                    $ref: '#/components/schemas/PaginationMeta'

  /payroll/payslips/{payslipId}:
    get:
      tags:
        - Payroll
      summary: 급여 명세서 상세
      operationId: getPayslip
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: payslipId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      responses:
        '200':
          description: 성공
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Payslip'

  /payroll/periods/{periodId}/confirm:
    post:
      tags:
        - Payroll
      summary: 급여 확정
      description: 급여를 확정하고 전표를 생성합니다
      operationId: confirmPayroll
      parameters:
        - $ref: '#/components/parameters/CompanyIdHeader'
        - name: periodId
          in: path
          required: true
          schema:
            $ref: '#/components/schemas/UUID'
      responses:
        '200':
          description: 확정 성공
          content:
            application/json:
              schema:
                type: object
                properties:
                  voucher_id:
                    $ref: '#/components/schemas/UUID'
                  total_gross:
                    type: number
                  total_deductions:
                    type: number
                  total_net:
                    type: number

components:
  schemas:
    PayrollPeriod:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        year:
          type: integer
        month:
          type: integer
        pay_date:
          type: string
          format: date
        status:
          type: string
          enum: [OPEN, CALCULATED, CONFIRMED, PAID]
        employee_count:
          type: integer
        total_gross:
          type: number
        total_net:
          type: number
        calculated_at:
          $ref: '#/components/schemas/Timestamp'
        confirmed_at:
          $ref: '#/components/schemas/Timestamp'

    PayslipSummary:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        employee_id:
          $ref: '#/components/schemas/UUID'
        employee_name:
          type: string
        employee_number:
          type: string
        department:
          type: string
        gross_salary:
          type: number
        total_deductions:
          type: number
        net_salary:
          type: number

    Payslip:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        period_id:
          $ref: '#/components/schemas/UUID'
        employee:
          $ref: '#/components/schemas/EmployeeInfo'
        # 지급 항목
        earnings:
          type: array
          items:
            $ref: '#/components/schemas/PayItem'
        # 공제 항목
        deductions:
          type: array
          items:
            $ref: '#/components/schemas/PayItem'
        # 합계
        gross_salary:
          type: number
          description: 지급액 합계
        total_deductions:
          type: number
          description: 공제액 합계
        net_salary:
          type: number
          description: 실지급액
        # 4대보험 상세
        insurance:
          $ref: '#/components/schemas/InsuranceDetail'
        # 소득세 상세
        income_tax:
          $ref: '#/components/schemas/IncomeTaxDetail'

    PayItem:
      type: object
      properties:
        code:
          type: string
        name:
          type: string
        amount:
          type: number
        taxable:
          type: boolean
          description: 과세 여부

    InsuranceDetail:
      type: object
      properties:
        national_pension:
          type: object
          properties:
            employee:
              type: number
            employer:
              type: number
        health_insurance:
          type: object
          properties:
            employee:
              type: number
            employer:
              type: number
        long_term_care:
          type: object
          properties:
            employee:
              type: number
            employer:
              type: number
        employment_insurance:
          type: object
          properties:
            employee:
              type: number
            employer:
              type: number
        industrial_accident:
          type: object
          properties:
            employer:
              type: number

    IncomeTaxDetail:
      type: object
      properties:
        taxable_income:
          type: number
          description: 과세표준
        income_tax:
          type: number
          description: 소득세
        local_income_tax:
          type: number
          description: 지방소득세
        total:
          type: number
          description: 소득세 합계

    EmployeeInfo:
      type: object
      properties:
        id:
          $ref: '#/components/schemas/UUID'
        employee_number:
          type: string
        name:
          type: string
        department:
          type: string
        position:
          type: string
        join_date:
          type: string
          format: date
```

---

## 7. 에러 코드 정의

```yaml
# api/errors.yaml
x-error-codes:
  # 인증 관련 (AUTH_xxx)
  AUTH_001:
    title: Invalid Credentials
    detail: The provided email or password is incorrect
    status: 401

  AUTH_002:
    title: Token Expired
    detail: The access token has expired
    status: 401

  AUTH_003:
    title: Token Invalid
    detail: The provided token is invalid
    status: 401

  AUTH_004:
    title: Refresh Token Invalid
    detail: The refresh token is invalid or expired
    status: 401

  AUTH_005:
    title: Account Locked
    detail: Account is locked due to too many failed login attempts
    status: 403

  # 권한 관련 (AUTHZ_xxx)
  AUTHZ_001:
    title: Permission Denied
    detail: You don't have permission to perform this action
    status: 403

  AUTHZ_002:
    title: Company Access Denied
    detail: You don't have access to this company
    status: 403

  # 검증 관련 (VAL_xxx)
  VAL_001:
    title: Validation Error
    detail: Request validation failed
    status: 400

  VAL_002:
    title: Invalid Date Range
    detail: Start date must be before end date
    status: 400

  VAL_003:
    title: Invalid Account Code
    detail: The specified account code does not exist
    status: 400

  # 전표 관련 (VOUCHER_xxx)
  VOUCHER_001:
    title: Unbalanced Voucher
    detail: Total debit must equal total credit
    status: 422

  VOUCHER_002:
    title: Invalid Voucher Status
    detail: Cannot perform this operation on voucher with current status
    status: 409

  VOUCHER_003:
    title: Duplicate Voucher Number
    detail: A voucher with this number already exists
    status: 409

  VOUCHER_004:
    title: Closed Period
    detail: Cannot modify voucher in a closed accounting period
    status: 409

  # 세금계산서 관련 (TAX_xxx)
  TAX_001:
    title: Invalid Business Number
    detail: The business registration number is invalid
    status: 400

  TAX_002:
    title: Issue Failed
    detail: Failed to issue tax invoice to NTS
    status: 422

  TAX_003:
    title: Already Issued
    detail: Tax invoice has already been issued
    status: 409

  TAX_004:
    title: Popbill API Error
    detail: External API error occurred
    status: 502

  # 급여 관련 (PAYROLL_xxx)
  PAYROLL_001:
    title: Period Not Calculated
    detail: Payroll period must be calculated before confirmation
    status: 409

  PAYROLL_002:
    title: Period Already Confirmed
    detail: Payroll period has already been confirmed
    status: 409

  PAYROLL_003:
    title: Employee Not Found
    detail: Employee record not found for payroll calculation
    status: 404

  # 재고 관련 (INV_xxx)
  INV_001:
    title: Insufficient Stock
    detail: Not enough stock available
    status: 422

  INV_002:
    title: Negative Stock
    detail: Stock quantity cannot be negative
    status: 422

  # 시스템 관련 (SYS_xxx)
  SYS_001:
    title: Internal Server Error
    detail: An unexpected error occurred
    status: 500

  SYS_002:
    title: Service Unavailable
    detail: The service is temporarily unavailable
    status: 503

  SYS_003:
    title: Rate Limit Exceeded
    detail: Too many requests
    status: 429
```

---

## 8. Go Swagger 애노테이션

```go
// internal/handler/voucher_handler.go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

// @Summary 전표 목록 조회
// @Description 조건에 맞는 전표 목록을 조회합니다
// @Tags Vouchers
// @Accept json
// @Produce json
// @Param X-Company-ID header string true "회사 ID" format(uuid)
// @Param cursor query string false "페이지네이션 커서"
// @Param limit query int false "페이지당 항목 수" default(20) minimum(1) maximum(100)
// @Param start_date query string false "시작일" format(date)
// @Param end_date query string false "종료일" format(date)
// @Param status query string false "전표 상태" Enums(DRAFT, PENDING, APPROVED, REJECTED, POSTED)
// @Success 200 {object} ListVouchersResponse
// @Failure 400 {object} ProblemDetail
// @Failure 401 {object} ProblemDetail
// @Router /vouchers [get]
func (h *VoucherHandler) ListVouchers(c *gin.Context) {
    // 구현
}

// @Summary 전표 생성
// @Description 새 전표를 생성합니다. 차변/대변 합계가 일치해야 합니다.
// @Tags Vouchers
// @Accept json
// @Produce json
// @Param X-Company-ID header string true "회사 ID" format(uuid)
// @Param request body VoucherCreateRequest true "전표 생성 요청"
// @Success 201 {object} Voucher
// @Failure 400 {object} ProblemDetail
// @Failure 422 {object} ProblemDetail "차대변 불균형"
// @Router /vouchers [post]
func (h *VoucherHandler) CreateVoucher(c *gin.Context) {
    // 구현
}

// @Summary 전표 상세 조회
// @Tags Vouchers
// @Accept json
// @Produce json
// @Param X-Company-ID header string true "회사 ID" format(uuid)
// @Param voucherId path string true "전표 ID" format(uuid)
// @Success 200 {object} Voucher
// @Failure 404 {object} ProblemDetail
// @Router /vouchers/{voucherId} [get]
func (h *VoucherHandler) GetVoucher(c *gin.Context) {
    // 구현
}

// @Summary 전표 승인
// @Description 대기(PENDING) 상태의 전표를 승인합니다
// @Tags Vouchers
// @Accept json
// @Produce json
// @Param X-Company-ID header string true "회사 ID" format(uuid)
// @Param voucherId path string true "전표 ID" format(uuid)
// @Param request body ApproveVoucherRequest false "승인 요청"
// @Success 200 {object} Voucher
// @Failure 409 {object} ProblemDetail "상태 충돌"
// @Router /vouchers/{voucherId}/approve [post]
func (h *VoucherHandler) ApproveVoucher(c *gin.Context) {
    // 구현
}
```

### Swagger 생성 설정

```go
// cmd/api/main.go

// @title K-ERP API
// @version 1.0
// @description 한국형 클라우드 ERP 시스템 API
// @termsOfService https://kerp.example.com/terms

// @contact.name K-ERP Support
// @contact.email support@kerp.example.com

// @license.name Proprietary
// @license.url https://kerp.example.com/license

// @host api.kerp.example.com
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Bearer Token

func main() {
    // ...
}
```

### Makefile 타겟

```makefile
# Makefile

.PHONY: swagger
swagger:
	@echo "Generating Swagger documentation..."
	swag init -g cmd/api/main.go -o api/swagger --parseDependency --parseInternal
	@echo "Swagger docs generated at api/swagger/"

.PHONY: swagger-serve
swagger-serve:
	@echo "Starting Swagger UI at http://localhost:8081"
	docker run -p 8081:8080 -e SWAGGER_JSON=/api/swagger.json \
		-v $(PWD)/api/swagger:/api swaggerapi/swagger-ui
```

---

## 9. API 버저닝 전략

### URL 경로 기반 버저닝

```
/api/v1/vouchers      ← 현재 버전
/api/v2/vouchers      ← 새 버전 (Breaking changes)
```

### 버전 미들웨어

```go
// internal/middleware/version.go
package middleware

import (
    "regexp"
    "strings"

    "github.com/gin-gonic/gin"
)

var versionRegex = regexp.MustCompile(`^/api/v(\d+)/`)

func APIVersion() gin.HandlerFunc {
    return func(c *gin.Context) {
        matches := versionRegex.FindStringSubmatch(c.Request.URL.Path)
        if len(matches) > 1 {
            c.Set("api_version", matches[1])
        } else {
            c.Set("api_version", "1") // 기본값
        }
        c.Next()
    }
}

// GetAPIVersion 현재 API 버전 조회
func GetAPIVersion(c *gin.Context) string {
    if v, exists := c.Get("api_version"); exists {
        return v.(string)
    }
    return "1"
}
```

---

## 10. 체크리스트

### API 설계 체크리스트

- [ ] OpenAPI 3.0 스펙 정의
- [ ] 인증/인가 API (JWT)
- [ ] 전표 CRUD API
- [ ] 세금계산서 API (Popbill 연동)
- [ ] 재무제표 API
- [ ] 급여 API
- [ ] 에러 코드 표준화 (RFC 7807)
- [ ] Swagger 애노테이션
- [ ] API 버저닝 전략
- [ ] Rate Limiting 헤더

### API 응답 시간 목표

| 엔드포인트 | 목표 (p95) | 비고 |
|-----------|-----------|------|
| GET /vouchers | < 100ms | 목록 조회 |
| POST /vouchers | < 200ms | 전표 생성 |
| GET /reports/* | < 500ms | 재무제표 |
| POST /tax-invoices/*/issue | < 3s | 외부 API 연동 |
