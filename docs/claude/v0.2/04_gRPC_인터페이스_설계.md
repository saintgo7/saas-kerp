# K-ERP v0.2 - gRPC 인터페이스 설계

**문서 버전**: 0.2.0
**작성일**: 2026-01-16
**상태**: 검토 대기

---

## 목차

1. [개요](#1-개요)
2. [Proto 파일 구조](#2-proto-파일-구조)
3. [서비스 정의](#3-서비스-정의)
4. [에러 처리](#4-에러-처리)
5. [인증 및 보안](#5-인증-및-보안)

---

## 1. 개요

### 1.1 gRPC 선택 이유

| 항목 | REST API | gRPC | 선택 |
|------|----------|------|------|
| 성능 | JSON 직렬화 | Protocol Buffers | gRPC |
| 타입 안전성 | 스키마 없음 | 강타입 | gRPC |
| 스트리밍 | 제한적 | 양방향 지원 | gRPC |
| 코드 생성 | 수동 | 자동 | gRPC |
| 서비스간 통신 | HTTP/1.1 | HTTP/2 | gRPC |

### 1.2 서비스 포트 매핑

| 서비스 | 포트 | 설명 |
|--------|------|------|
| Tax Scraper | 50051 | 세금계산서 스크래핑 |
| Insurance EDI | 50052 | 4대보험 EDI |

### 1.3 Proto 파일 위치

```
api/
└── proto/
    ├── common/
    │   └── types.proto      # 공통 타입 정의
    ├── scraper/
    │   └── scraper.proto    # 세금계산서 스크래퍼
    └── insurance/
        └── insurance.proto  # 4대보험 EDI
```

---

## 2. Proto 파일 구조

### 2.1 공통 타입 정의

```protobuf
// api/proto/common/types.proto
syntax = "proto3";

package kerp.common.v1;

option go_package = "github.com/kerp/api/proto/common";

// 에러 상세 정보
message ErrorDetail {
    string code = 1;          // 에러 코드
    string message = 2;       // 에러 메시지
    string field = 3;         // 관련 필드 (validation error)
    map<string, string> metadata = 4; // 추가 메타데이터
}

// 페이지네이션 요청
message PaginationRequest {
    int32 page = 1;           // 페이지 번호 (1부터 시작)
    int32 page_size = 2;      // 페이지 크기 (기본 20, 최대 100)
}

// 페이지네이션 응답
message PaginationResponse {
    int32 page = 1;
    int32 page_size = 2;
    int32 total_count = 3;
    int32 total_pages = 4;
    bool has_next = 5;
    bool has_prev = 6;
}

// 기간 범위
message DateRange {
    string start_date = 1;    // YYYY-MM-DD
    string end_date = 2;      // YYYY-MM-DD
}

// 금액
message Money {
    int64 amount = 1;         // 금액 (원 단위)
    string currency = 2;      // 통화 코드 (KRW)
}

// 헬스 체크
message HealthRequest {}

message HealthResponse {
    enum Status {
        UNKNOWN = 0;
        SERVING = 1;
        NOT_SERVING = 2;
        SERVICE_UNKNOWN = 3;
    }
    Status status = 1;
    string message = 2;
    map<string, string> details = 3;
}
```

### 2.2 세금계산서 스크래퍼 Proto

```protobuf
// api/proto/scraper/scraper.proto
syntax = "proto3";

package kerp.scraper.v1;

option go_package = "github.com/kerp/api/proto/scraper";

import "common/types.proto";

// 세금계산서 스크래핑 서비스
service ScraperService {
    // 세금계산서 발행
    rpc IssueInvoice(IssueInvoiceRequest) returns (IssueInvoiceResponse);

    // 세금계산서 목록 조회
    rpc ListInvoices(ListInvoicesRequest) returns (ListInvoicesResponse);

    // 세금계산서 상세 조회
    rpc GetInvoice(GetInvoiceRequest) returns (GetInvoiceResponse);

    // 세금계산서 취소
    rpc CancelInvoice(CancelInvoiceRequest) returns (CancelInvoiceResponse);

    // 세금계산서 동기화 (홈택스 → DB)
    rpc SyncInvoices(SyncInvoicesRequest) returns (SyncInvoicesResponse);

    // 헬스 체크
    rpc HealthCheck(kerp.common.v1.HealthRequest) returns (kerp.common.v1.HealthResponse);
}

// ========== 인증서 정보 ==========
message CertificateInfo {
    string cert_der_base64 = 1;    // 인증서 DER (Base64)
    string key_encrypted_base64 = 2; // 암호화된 개인키 (Base64)
    string password = 3;            // 비밀번호
    string business_number = 4;     // 사업자번호
}

// ========== 거래처 정보 ==========
message PartyInfo {
    string business_number = 1;     // 사업자등록번호 (10자리)
    string corp_number = 2;         // 법인등록번호 (선택)
    string name = 3;                // 상호
    string ceo_name = 4;            // 대표자명
    string address = 5;             // 주소
    string business_type = 6;       // 업태
    string business_item = 7;       // 종목
    string email = 8;               // 이메일
    string phone = 9;               // 전화번호
}

// ========== 품목 정보 ==========
message InvoiceItem {
    int32 serial_number = 1;        // 일련번호
    string item_date = 2;           // 거래일자 (YYYY-MM-DD)
    string name = 3;                // 품목명
    string specification = 4;       // 규격
    double quantity = 5;            // 수량
    int64 unit_price = 6;           // 단가
    int64 supply_amount = 7;        // 공급가액
    int64 tax_amount = 8;           // 세액
    string remarks = 9;             // 비고
}

// ========== 세금계산서 ==========
message Invoice {
    string id = 1;                  // 내부 ID
    string nts_confirm_number = 2;  // 국세청 승인번호
    string issue_date = 3;          // 발행일자

    enum Type {
        TYPE_UNSPECIFIED = 0;
        SALES = 1;                  // 매출
        PURCHASE = 2;               // 매입
    }
    Type type = 4;

    enum IssueType {
        ISSUE_TYPE_UNSPECIFIED = 0;
        NORMAL = 1;                 // 정발행
        REVERSE = 2;                // 역발행
    }
    IssueType issue_type = 5;

    PartyInfo supplier = 6;         // 공급자
    PartyInfo buyer = 7;            // 공급받는자

    int64 supply_amount = 8;        // 공급가액
    int64 tax_amount = 9;           // 세액
    int64 total_amount = 10;        // 합계

    repeated InvoiceItem items = 11; // 품목

    enum Status {
        STATUS_UNSPECIFIED = 0;
        ISSUED = 1;                 // 발행됨
        CANCELLED = 2;              // 취소됨
        MODIFIED = 3;               // 수정됨
    }
    Status status = 12;

    string remarks = 13;            // 비고
    string created_at = 14;         // 생성일시
    string updated_at = 15;         // 수정일시
}

// ========== 발행 요청/응답 ==========
message IssueInvoiceRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;

    string issue_date = 3;          // YYYY-MM-DD
    Invoice.IssueType issue_type = 4;

    PartyInfo supplier = 5;
    PartyInfo buyer = 6;

    int64 supply_amount = 7;
    int64 tax_amount = 8;
    int64 total_amount = 9;

    repeated InvoiceItem items = 10;
    string remarks = 11;

    // 재시도 옵션
    bool allow_retry = 12;          // 실패시 재시도 허용
    int32 max_retries = 13;         // 최대 재시도 횟수
}

message IssueInvoiceResponse {
    bool success = 1;
    Invoice invoice = 2;

    // 에러 정보
    string error_code = 3;
    string error_message = 4;
    repeated kerp.common.v1.ErrorDetail errors = 5;
}

// ========== 목록 조회 ==========
message ListInvoicesRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;

    kerp.common.v1.DateRange date_range = 3;
    Invoice.Type type = 4;          // 매출/매입 필터

    kerp.common.v1.PaginationRequest pagination = 5;
}

message ListInvoicesResponse {
    repeated Invoice invoices = 1;
    kerp.common.v1.PaginationResponse pagination = 2;
}

// ========== 상세 조회 ==========
message GetInvoiceRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;
    string nts_confirm_number = 3;
}

message GetInvoiceResponse {
    bool success = 1;
    Invoice invoice = 2;
    string error_code = 3;
    string error_message = 4;
}

// ========== 취소 ==========
message CancelInvoiceRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;
    string nts_confirm_number = 3;
    string reason = 4;
}

message CancelInvoiceResponse {
    bool success = 1;
    string cancelled_at = 2;
    string error_code = 3;
    string error_message = 4;
}

// ========== 동기화 ==========
message SyncInvoicesRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;
    kerp.common.v1.DateRange date_range = 3;
    Invoice.Type type = 4;
}

message SyncInvoicesResponse {
    bool success = 1;
    int32 synced_count = 2;
    int32 new_count = 3;
    int32 updated_count = 4;
    repeated Invoice invoices = 5;
    string error_code = 6;
    string error_message = 7;
}
```

### 2.3 4대보험 EDI Proto

```protobuf
// api/proto/insurance/insurance.proto
syntax = "proto3";

package kerp.insurance.v1;

option go_package = "github.com/kerp/api/proto/insurance";

import "common/types.proto";

// 4대보험 EDI 서비스
service InsuranceEDIService {
    // 자격취득 신고
    rpc ReportAcquisition(AcquisitionRequest) returns (EDIResponse);

    // 자격상실 신고
    rpc ReportLoss(LossRequest) returns (EDIResponse);

    // 보수총액 신고
    rpc ReportTotalSalary(TotalSalaryRequest) returns (EDIResponse);

    // 이직확인서 발급 (고용보험)
    rpc IssueSeparationCertificate(SeparationRequest) returns (SeparationResponse);

    // 보험료 조회
    rpc GetPremium(PremiumRequest) returns (PremiumResponse);

    // 신고 내역 조회
    rpc GetReportHistory(ReportHistoryRequest) returns (ReportHistoryResponse);

    // 헬스 체크
    rpc HealthCheck(kerp.common.v1.HealthRequest) returns (kerp.common.v1.HealthResponse);
}

// ========== 공통 ==========

// 보험 종류
enum InsuranceType {
    INSURANCE_UNSPECIFIED = 0;
    NATIONAL_PENSION = 1;       // 국민연금
    HEALTH_INSURANCE = 2;       // 건강보험
    EMPLOYMENT_INSURANCE = 3;   // 고용보험
    INDUSTRIAL_ACCIDENT = 4;    // 산재보험
    ALL_INSURANCE = 5;          // 4대보험 전체
}

// 인증서 정보
message CertificateInfo {
    string cert_der_base64 = 1;
    string key_encrypted_base64 = 2;
    string password = 3;
    string business_number = 4;
}

// 사업장 정보
message BusinessInfo {
    string business_number = 1;     // 사업자등록번호
    string workplace_code = 2;      // 사업장관리번호
    string name = 3;                // 사업장명
    string ceo_name = 4;            // 대표자명
    string address = 5;             // 주소
    string phone = 6;               // 전화번호
}

// 직원 정보
message EmployeeInfo {
    string employee_id = 1;         // 내부 직원 ID
    string resident_number = 2;     // 주민등록번호 (암호화)
    string name = 3;                // 성명
    string birth_date = 4;          // 생년월일 (YYYYMMDD)

    enum Gender {
        GENDER_UNSPECIFIED = 0;
        MALE = 1;
        FEMALE = 2;
    }
    Gender gender = 5;

    string nationality = 6;         // 국적 코드
    string address = 7;             // 주소
    string phone = 8;               // 전화번호
}

// EDI 응답
message EDIResponse {
    bool success = 1;
    string transaction_id = 2;      // EDI 처리번호
    string processed_at = 3;        // 처리일시

    string result_code = 4;         // 결과 코드
    string result_message = 5;      // 결과 메시지

    // 보험별 결과
    repeated InsuranceResult insurance_results = 6;

    string error_code = 7;
    string error_message = 8;
}

message InsuranceResult {
    InsuranceType insurance_type = 1;
    bool success = 2;
    string result_code = 3;
    string result_message = 4;
}

// ========== 취득신고 ==========
message AcquisitionRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;

    repeated InsuranceType insurance_types = 3;

    BusinessInfo business = 4;
    EmployeeInfo employee = 5;

    string acquisition_date = 6;    // 취득일자 (YYYYMMDD)

    enum AcquisitionReason {
        REASON_UNSPECIFIED = 0;
        NEW_HIRE = 1;               // 신규입사
        TRANSFER_IN = 2;            // 전입
        REINSTATEMENT = 3;          // 복직
        OTHER = 99;
    }
    AcquisitionReason reason = 7;

    int64 monthly_salary = 8;       // 월 보수액
    string job_code = 9;            // 직종코드

    bool weekly_under_15_hours = 10; // 주 15시간 미만 여부
}

// ========== 상실신고 ==========
message LossRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;

    repeated InsuranceType insurance_types = 3;

    BusinessInfo business = 4;
    EmployeeInfo employee = 5;

    string loss_date = 6;           // 상실일자 (YYYYMMDD)

    enum LossReason {
        REASON_UNSPECIFIED = 0;
        RESIGNATION = 1;            // 자진퇴사
        DISMISSAL = 2;              // 해고
        CONTRACT_END = 3;           // 계약만료
        RETIREMENT = 4;             // 정년퇴직
        DEATH = 5;                  // 사망
        TRANSFER_OUT = 6;           // 전출
        OTHER = 99;
    }
    LossReason reason = 7;

    int64 total_salary = 8;         // 해당연도 보수총액
    int32 working_days = 9;         // 근무일수
}

// ========== 보수총액 신고 ==========
message TotalSalaryRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;

    repeated InsuranceType insurance_types = 3;

    BusinessInfo business = 4;
    string report_year = 5;         // 신고연도 (YYYY)

    repeated EmployeeSalaryInfo employees = 6;
}

message EmployeeSalaryInfo {
    EmployeeInfo employee = 1;
    int64 total_salary = 2;         // 연간 보수총액
    int32 working_months = 3;       // 근무 개월수
    int64 bonus_amount = 4;         // 상여금
}

// ========== 이직확인서 ==========
message SeparationRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;

    BusinessInfo business = 3;
    EmployeeInfo employee = 4;

    string separation_date = 5;     // 이직일자

    enum SeparationReason {
        REASON_UNSPECIFIED = 0;
        VOLUNTARY = 1;              // 자발적 이직
        INVOLUNTARY = 2;            // 비자발적 이직
        CONTRACT_END = 3;           // 계약만료
    }
    SeparationReason reason = 6;
    string reason_detail = 7;       // 상세 사유

    // 급여 내역 (최근 18개월)
    repeated MonthlySalary salary_records = 8;
}

message MonthlySalary {
    string year_month = 1;          // YYYYMM
    int64 total_salary = 2;         // 총 급여
    int32 working_days = 3;         // 근무일수
    int32 paid_days = 4;            // 유급일수
    int64 bonus = 5;                // 상여금
}

message SeparationResponse {
    bool success = 1;
    string certificate_number = 2;  // 이직확인서 번호
    string issued_at = 3;           // 발급일시
    bytes pdf_content = 4;          // PDF 파일 (Base64)
    string error_code = 5;
    string error_message = 6;
}

// ========== 보험료 조회 ==========
message PremiumRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;

    InsuranceType insurance_type = 3;
    BusinessInfo business = 4;
    string year_month = 5;          // YYYYMM
}

message PremiumResponse {
    bool success = 1;
    InsuranceType insurance_type = 2;
    string year_month = 3;

    int64 company_premium = 4;      // 사업주 부담금
    int64 employee_premium = 5;     // 근로자 부담금
    int64 total_premium = 6;        // 합계

    string due_date = 7;            // 납부기한 (YYYYMMDD)
    string payment_status = 8;      // 납부 상태

    string error_code = 9;
    string error_message = 10;
}

// ========== 신고 내역 조회 ==========
message ReportHistoryRequest {
    string company_id = 1;
    CertificateInfo certificate = 2;

    InsuranceType insurance_type = 3;
    kerp.common.v1.DateRange date_range = 4;
    kerp.common.v1.PaginationRequest pagination = 5;
}

message ReportHistoryResponse {
    repeated ReportRecord records = 1;
    kerp.common.v1.PaginationResponse pagination = 2;
}

message ReportRecord {
    string transaction_id = 1;
    InsuranceType insurance_type = 2;

    enum ReportType {
        TYPE_UNSPECIFIED = 0;
        ACQUISITION = 1;            // 취득
        LOSS = 2;                   // 상실
        TOTAL_SALARY = 3;           // 보수총액
        CHANGE = 4;                 // 내용변경
    }
    ReportType report_type = 3;

    string employee_name = 4;
    string reported_at = 5;
    string result_code = 6;
    string result_message = 7;
}
```

---

## 3. 서비스 정의

### 3.1 코드 생성 스크립트

```bash
#!/bin/bash
# scripts/gen-proto.sh

PROTO_DIR=./api/proto
GO_OUT=./api/proto
PYTHON_OUT=./python-services/shared/proto

# Go 코드 생성
protoc \
    --proto_path=${PROTO_DIR} \
    --go_out=${GO_OUT} \
    --go_opt=paths=source_relative \
    --go-grpc_out=${GO_OUT} \
    --go-grpc_opt=paths=source_relative \
    ${PROTO_DIR}/common/types.proto \
    ${PROTO_DIR}/scraper/scraper.proto \
    ${PROTO_DIR}/insurance/insurance.proto

# Python 코드 생성
python -m grpc_tools.protoc \
    --proto_path=${PROTO_DIR} \
    --python_out=${PYTHON_OUT} \
    --grpc_python_out=${PYTHON_OUT} \
    --pyi_out=${PYTHON_OUT} \
    ${PROTO_DIR}/common/types.proto \
    ${PROTO_DIR}/scraper/scraper.proto \
    ${PROTO_DIR}/insurance/insurance.proto

echo "Proto generation complete!"
```

---

## 4. 에러 처리

### 4.1 gRPC 상태 코드 매핑

| gRPC Status | HTTP | 사용 상황 |
|-------------|------|----------|
| OK | 200 | 성공 |
| INVALID_ARGUMENT | 400 | 잘못된 요청 파라미터 |
| UNAUTHENTICATED | 401 | 인증서 오류 |
| PERMISSION_DENIED | 403 | 권한 없음 |
| NOT_FOUND | 404 | 리소스 없음 |
| ALREADY_EXISTS | 409 | 중복 |
| RESOURCE_EXHAUSTED | 429 | Rate limit |
| INTERNAL | 500 | 서버 오류 |
| UNAVAILABLE | 503 | 서비스 이용 불가 |

### 4.2 에러 코드 정의

```go
// pkg/errors/grpc_errors.go
package errors

// 세금계산서 에러 코드
const (
    ErrCodeInvoiceNotFound     = "INVOICE_NOT_FOUND"
    ErrCodeInvoiceAlreadyIssued = "INVOICE_ALREADY_ISSUED"
    ErrCodeInvoiceIssueFailed  = "INVOICE_ISSUE_FAILED"
    ErrCodeInvoiceCancelFailed = "INVOICE_CANCEL_FAILED"
    ErrCodeCertificateInvalid  = "CERTIFICATE_INVALID"
    ErrCodeCertificateExpired  = "CERTIFICATE_EXPIRED"
    ErrCodeHometaxUnavailable  = "HOMETAX_UNAVAILABLE"
)

// 4대보험 에러 코드
const (
    ErrCodeEDIFailed           = "EDI_FAILED"
    ErrCodeEDITimeout          = "EDI_TIMEOUT"
    ErrCodeInvalidEmployee     = "INVALID_EMPLOYEE"
    ErrCodeDuplicateReport     = "DUPLICATE_REPORT"
    ErrCodeAgencyUnavailable   = "AGENCY_UNAVAILABLE"
)
```

---

## 5. 인증 및 보안

### 5.1 mTLS 설정

```yaml
# deployments/k8s/grpc-mtls-config.yaml
apiVersion: v1
kind: Secret
metadata:
  name: grpc-mtls-certs
type: kubernetes.io/tls
data:
  ca.crt: <BASE64_CA_CERT>
  tls.crt: <BASE64_SERVER_CERT>
  tls.key: <BASE64_SERVER_KEY>
```

### 5.2 메타데이터 인증

```go
// pkg/grpc/auth_interceptor.go
func AuthInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "missing metadata")
    }

    // 서비스 토큰 검증
    tokens := md.Get("authorization")
    if len(tokens) == 0 {
        return nil, status.Error(codes.Unauthenticated, "missing auth token")
    }

    if !validateServiceToken(tokens[0]) {
        return nil, status.Error(codes.Unauthenticated, "invalid token")
    }

    return handler(ctx, req)
}
```

---

**다음 문서**: [05_Python_서비스_설계.md](./05_Python_서비스_설계.md)
