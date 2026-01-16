"""
Hometax constants and configuration values.
"""

# Hometax URLs
HOMETAX_BASE_URL = "https://www.hometax.go.kr"
HOMETAX_MAIN_URL = f"{HOMETAX_BASE_URL}/websquare/websquare.wq"
HOMETAX_LOGIN_URL = f"{HOMETAX_BASE_URL}/wqAction.do"

# Menu IDs
MENU_TAX_INVOICE_SALES = "UTXPPBAA01"  # 세금계산서 발급
MENU_TAX_INVOICE_SEARCH = "UTECBAA01"  # 세금계산서 조회
MENU_TAX_INVOICE_PURCHASE = "UTECBAB01"  # 매입 세금계산서

# Status codes
STATUS_DRAFT = "01"
STATUS_ISSUED = "02"
STATUS_TRANSMITTED = "03"
STATUS_CONFIRMED = "04"
STATUS_CANCELLED = "05"
STATUS_REJECTED = "06"

STATUS_MAP = {
    STATUS_DRAFT: "draft",
    STATUS_ISSUED: "issued",
    STATUS_TRANSMITTED: "transmitted",
    STATUS_CONFIRMED: "confirmed",
    STATUS_CANCELLED: "cancelled",
    STATUS_REJECTED: "rejected",
}

# Tax types
TAX_TYPE_TAXABLE = "01"  # 과세
TAX_TYPE_ZERO = "02"  # 영세
TAX_TYPE_EXEMPT = "03"  # 면세

TAX_TYPE_MAP = {
    TAX_TYPE_TAXABLE: "taxable",
    TAX_TYPE_ZERO: "zero_rate",
    TAX_TYPE_EXEMPT: "exempt",
}

# Invoice types
INVOICE_TYPE_NORMAL = "01"  # 일반
INVOICE_TYPE_MODIFIED = "02"  # 수정
INVOICE_TYPE_CANCEL = "03"  # 취소

# Authentication types
AUTH_CERT = "certificate"
AUTH_SIMPLE = "simple_auth"
AUTH_ID_PW = "id_password"

# Selectors for Playwright
SELECTORS = {
    # Login page
    "login_cert_btn": "#cert_login_btn",
    "login_id_input": "#user_id",
    "login_pw_input": "#passwd",
    "login_submit": "#login_btn",
    "cert_select_popup": ".cert_select_popup",
    "cert_pw_input": "#cert_pw",
    "cert_confirm_btn": "#cert_confirm",

    # Tax invoice search
    "search_start_date": "#schDtFrom",
    "search_end_date": "#schDtTo",
    "search_btn": "#btn_search",
    "result_table": "#resultTable",
    "result_rows": "#resultTable tbody tr",

    # Tax invoice detail
    "invoice_number": "#taxInvoiceNum",
    "issue_date": "#issueDate",
    "supplier_brn": "#invoicerCorpNum",
    "supplier_name": "#invoicerCorpName",
    "buyer_brn": "#invoiceeCorpNum",
    "buyer_name": "#invoiceeCorpName",
    "supply_amount": "#supplyCostTotal",
    "tax_amount": "#taxTotal",
    "total_amount": "#totalAmount",
    "nts_confirm": "#ntsConfirmNum",

    # Tax invoice issue form
    "form_issue_date": "#writeDate",
    "form_supplier_brn": "#invoicerCorpNum",
    "form_supplier_name": "#invoicerCorpName",
    "form_buyer_brn": "#invoiceeCorpNum",
    "form_buyer_name": "#invoiceeCorpName",
    "form_supply_amount": "#supplyCostTotal",
    "form_tax_amount": "#taxTotal",
    "form_submit": "#btn_issue",

    # Common
    "loading_indicator": ".loading",
    "alert_popup": ".alert_popup",
    "confirm_btn": ".confirm_btn",
    "close_btn": ".close_btn",
}

# Error messages
ERROR_MESSAGES = {
    "LOGIN_FAILED": "로그인에 실패했습니다",
    "CERT_EXPIRED": "인증서가 만료되었습니다",
    "CERT_INVALID": "인증서 비밀번호가 올바르지 않습니다",
    "SESSION_EXPIRED": "세션이 만료되었습니다",
    "NETWORK_ERROR": "네트워크 오류가 발생했습니다",
    "SEARCH_FAILED": "조회에 실패했습니다",
    "ISSUE_FAILED": "발급에 실패했습니다",
}

# Timeouts (in milliseconds)
TIMEOUTS = {
    "page_load": 60000,
    "navigation": 30000,
    "element_wait": 10000,
    "animation": 500,
}
