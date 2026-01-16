-- K-ERP v0.2 Seed: Standard Korean Chart of Accounts (K-IFRS)
-- This is a template - actual accounts are created per company

-- Function to create standard chart of accounts for a company
CREATE OR REPLACE FUNCTION create_standard_accounts(p_company_id UUID)
RETURNS void AS $$
BEGIN
    -- ==============================
    -- 1. ASSETS (자산)
    -- ==============================

    -- 1.1 Current Assets (유동자산)
    INSERT INTO accounts (company_id, code, name, name_en, level, account_type, account_nature, account_category, path) VALUES
    (p_company_id, '1', '자산', 'Assets', 1, 'asset', 'debit', NULL, '1'),
    (p_company_id, '11', '유동자산', 'Current Assets', 2, 'asset', 'debit', 'current', '1.11'),
    (p_company_id, '1101', '현금및현금성자산', 'Cash and Cash Equivalents', 3, 'asset', 'debit', 'current', '1.11.1101'),
    (p_company_id, '110101', '현금', 'Cash', 4, 'asset', 'debit', 'current', '1.11.1101.110101'),
    (p_company_id, '110102', '보통예금', 'Checking Account', 4, 'asset', 'debit', 'current', '1.11.1101.110102'),
    (p_company_id, '110103', '정기예금', 'Time Deposit', 4, 'asset', 'debit', 'current', '1.11.1101.110103'),
    (p_company_id, '1102', '단기금융자산', 'Short-term Financial Assets', 3, 'asset', 'debit', 'current', '1.11.1102'),
    (p_company_id, '1103', '매출채권', 'Accounts Receivable', 3, 'asset', 'debit', 'current', '1.11.1103'),
    (p_company_id, '110301', '외상매출금', 'Trade Receivables', 4, 'asset', 'debit', 'current', '1.11.1103.110301'),
    (p_company_id, '110302', '받을어음', 'Notes Receivable', 4, 'asset', 'debit', 'current', '1.11.1103.110302'),
    (p_company_id, '110303', '대손충당금', 'Allowance for Bad Debts', 4, 'asset', 'credit', 'current', '1.11.1103.110303'),
    (p_company_id, '1104', '기타채권', 'Other Receivables', 3, 'asset', 'debit', 'current', '1.11.1104'),
    (p_company_id, '110401', '미수금', 'Accrued Receivables', 4, 'asset', 'debit', 'current', '1.11.1104.110401'),
    (p_company_id, '110402', '미수수익', 'Accrued Revenue', 4, 'asset', 'debit', 'current', '1.11.1104.110402'),
    (p_company_id, '110403', '선급금', 'Advance Payments', 4, 'asset', 'debit', 'current', '1.11.1104.110403'),
    (p_company_id, '110404', '선급비용', 'Prepaid Expenses', 4, 'asset', 'debit', 'current', '1.11.1104.110404'),
    (p_company_id, '1105', '재고자산', 'Inventories', 3, 'asset', 'debit', 'current', '1.11.1105'),
    (p_company_id, '110501', '상품', 'Merchandise', 4, 'asset', 'debit', 'current', '1.11.1105.110501'),
    (p_company_id, '110502', '제품', 'Finished Goods', 4, 'asset', 'debit', 'current', '1.11.1105.110502'),
    (p_company_id, '110503', '원재료', 'Raw Materials', 4, 'asset', 'debit', 'current', '1.11.1105.110503'),
    (p_company_id, '110504', '재공품', 'Work in Process', 4, 'asset', 'debit', 'current', '1.11.1105.110504'),

    -- 1.2 Non-current Assets (비유동자산)
    (p_company_id, '12', '비유동자산', 'Non-current Assets', 2, 'asset', 'debit', 'non_current', '1.12'),
    (p_company_id, '1201', '장기금융자산', 'Long-term Financial Assets', 3, 'asset', 'debit', 'non_current', '1.12.1201'),
    (p_company_id, '1202', '유형자산', 'Property, Plant and Equipment', 3, 'asset', 'debit', 'non_current', '1.12.1202'),
    (p_company_id, '120201', '토지', 'Land', 4, 'asset', 'debit', 'non_current', '1.12.1202.120201'),
    (p_company_id, '120202', '건물', 'Buildings', 4, 'asset', 'debit', 'non_current', '1.12.1202.120202'),
    (p_company_id, '120203', '건물감가상각누계액', 'Accumulated Depreciation - Buildings', 4, 'asset', 'credit', 'non_current', '1.12.1202.120203'),
    (p_company_id, '120204', '기계장치', 'Machinery', 4, 'asset', 'debit', 'non_current', '1.12.1202.120204'),
    (p_company_id, '120205', '기계장치감가상각누계액', 'Accumulated Depreciation - Machinery', 4, 'asset', 'credit', 'non_current', '1.12.1202.120205'),
    (p_company_id, '120206', '차량운반구', 'Vehicles', 4, 'asset', 'debit', 'non_current', '1.12.1202.120206'),
    (p_company_id, '120207', '차량운반구감가상각누계액', 'Accumulated Depreciation - Vehicles', 4, 'asset', 'credit', 'non_current', '1.12.1202.120207'),
    (p_company_id, '120208', '비품', 'Furniture and Fixtures', 4, 'asset', 'debit', 'non_current', '1.12.1202.120208'),
    (p_company_id, '120209', '비품감가상각누계액', 'Accumulated Depreciation - F&F', 4, 'asset', 'credit', 'non_current', '1.12.1202.120209'),
    (p_company_id, '1203', '무형자산', 'Intangible Assets', 3, 'asset', 'debit', 'non_current', '1.12.1203'),
    (p_company_id, '120301', '영업권', 'Goodwill', 4, 'asset', 'debit', 'non_current', '1.12.1203.120301'),
    (p_company_id, '120302', '소프트웨어', 'Software', 4, 'asset', 'debit', 'non_current', '1.12.1203.120302'),
    (p_company_id, '1204', '기타비유동자산', 'Other Non-current Assets', 3, 'asset', 'debit', 'non_current', '1.12.1204'),
    (p_company_id, '120401', '보증금', 'Deposits', 4, 'asset', 'debit', 'non_current', '1.12.1204.120401'),

    -- ==============================
    -- 2. LIABILITIES (부채)
    -- ==============================

    (p_company_id, '2', '부채', 'Liabilities', 1, 'liability', 'credit', NULL, '2'),
    (p_company_id, '21', '유동부채', 'Current Liabilities', 2, 'liability', 'credit', 'current', '2.21'),
    (p_company_id, '2101', '매입채무', 'Trade Payables', 3, 'liability', 'credit', 'current', '2.21.2101'),
    (p_company_id, '210101', '외상매입금', 'Accounts Payable', 4, 'liability', 'credit', 'current', '2.21.2101.210101'),
    (p_company_id, '210102', '지급어음', 'Notes Payable', 4, 'liability', 'credit', 'current', '2.21.2101.210102'),
    (p_company_id, '2102', '단기차입금', 'Short-term Borrowings', 3, 'liability', 'credit', 'current', '2.21.2102'),
    (p_company_id, '2103', '기타유동부채', 'Other Current Liabilities', 3, 'liability', 'credit', 'current', '2.21.2103'),
    (p_company_id, '210301', '미지급금', 'Accrued Payables', 4, 'liability', 'credit', 'current', '2.21.2103.210301'),
    (p_company_id, '210302', '미지급비용', 'Accrued Expenses', 4, 'liability', 'credit', 'current', '2.21.2103.210302'),
    (p_company_id, '210303', '선수금', 'Advances Received', 4, 'liability', 'credit', 'current', '2.21.2103.210303'),
    (p_company_id, '210304', '예수금', 'Withholdings', 4, 'liability', 'credit', 'current', '2.21.2103.210304'),
    (p_company_id, '210305', '부가세예수금', 'VAT Payable', 4, 'liability', 'credit', 'current', '2.21.2103.210305'),
    (p_company_id, '210306', '미지급법인세', 'Income Tax Payable', 4, 'liability', 'credit', 'current', '2.21.2103.210306'),

    (p_company_id, '22', '비유동부채', 'Non-current Liabilities', 2, 'liability', 'credit', 'non_current', '2.22'),
    (p_company_id, '2201', '장기차입금', 'Long-term Borrowings', 3, 'liability', 'credit', 'non_current', '2.22.2201'),
    (p_company_id, '2202', '퇴직급여충당부채', 'Retirement Benefit Obligations', 3, 'liability', 'credit', 'non_current', '2.22.2202'),
    (p_company_id, '2203', '이연법인세부채', 'Deferred Tax Liabilities', 3, 'liability', 'credit', 'non_current', '2.22.2203'),

    -- ==============================
    -- 3. EQUITY (자본)
    -- ==============================

    (p_company_id, '3', '자본', 'Equity', 1, 'equity', 'credit', NULL, '3'),
    (p_company_id, '31', '자본금', 'Share Capital', 2, 'equity', 'credit', NULL, '3.31'),
    (p_company_id, '3101', '보통주자본금', 'Common Stock', 3, 'equity', 'credit', NULL, '3.31.3101'),
    (p_company_id, '32', '자본잉여금', 'Capital Surplus', 2, 'equity', 'credit', NULL, '3.32'),
    (p_company_id, '3201', '주식발행초과금', 'Share Premium', 3, 'equity', 'credit', NULL, '3.32.3201'),
    (p_company_id, '33', '이익잉여금', 'Retained Earnings', 2, 'equity', 'credit', NULL, '3.33'),
    (p_company_id, '3301', '이익준비금', 'Legal Reserve', 3, 'equity', 'credit', NULL, '3.33.3301'),
    (p_company_id, '3302', '미처분이익잉여금', 'Unappropriated Retained Earnings', 3, 'equity', 'credit', NULL, '3.33.3302'),
    (p_company_id, '3303', '당기순이익', 'Net Income', 3, 'equity', 'credit', NULL, '3.33.3303'),

    -- ==============================
    -- 4. REVENUE (수익)
    -- ==============================

    (p_company_id, '4', '수익', 'Revenue', 1, 'revenue', 'credit', NULL, '4'),
    (p_company_id, '41', '매출', 'Sales', 2, 'revenue', 'credit', 'operating', '4.41'),
    (p_company_id, '4101', '상품매출', 'Merchandise Sales', 3, 'revenue', 'credit', 'operating', '4.41.4101'),
    (p_company_id, '4102', '제품매출', 'Product Sales', 3, 'revenue', 'credit', 'operating', '4.41.4102'),
    (p_company_id, '4103', '용역매출', 'Service Revenue', 3, 'revenue', 'credit', 'operating', '4.41.4103'),
    (p_company_id, '42', '영업외수익', 'Non-operating Income', 2, 'revenue', 'credit', 'non_operating', '4.42'),
    (p_company_id, '4201', '이자수익', 'Interest Income', 3, 'revenue', 'credit', 'non_operating', '4.42.4201'),
    (p_company_id, '4202', '배당금수익', 'Dividend Income', 3, 'revenue', 'credit', 'non_operating', '4.42.4202'),
    (p_company_id, '4203', '외환차익', 'Foreign Exchange Gains', 3, 'revenue', 'credit', 'non_operating', '4.42.4203'),
    (p_company_id, '4204', '유형자산처분이익', 'Gain on Disposal of PPE', 3, 'revenue', 'credit', 'non_operating', '4.42.4204'),
    (p_company_id, '4205', '잡이익', 'Miscellaneous Income', 3, 'revenue', 'credit', 'non_operating', '4.42.4205'),

    -- ==============================
    -- 5. EXPENSES (비용)
    -- ==============================

    (p_company_id, '5', '비용', 'Expenses', 1, 'expense', 'debit', NULL, '5'),
    (p_company_id, '51', '매출원가', 'Cost of Sales', 2, 'expense', 'debit', 'cost', '5.51'),
    (p_company_id, '5101', '상품매출원가', 'Cost of Merchandise Sold', 3, 'expense', 'debit', 'cost', '5.51.5101'),
    (p_company_id, '5102', '제품매출원가', 'Cost of Goods Sold', 3, 'expense', 'debit', 'cost', '5.51.5102'),
    (p_company_id, '52', '판매비와관리비', 'Selling and Administrative Expenses', 2, 'expense', 'debit', 'operating', '5.52'),
    (p_company_id, '5201', '급여', 'Salaries', 3, 'expense', 'debit', 'operating', '5.52.5201'),
    (p_company_id, '5202', '퇴직급여', 'Retirement Benefits', 3, 'expense', 'debit', 'operating', '5.52.5202'),
    (p_company_id, '5203', '복리후생비', 'Employee Benefits', 3, 'expense', 'debit', 'operating', '5.52.5203'),
    (p_company_id, '5204', '여비교통비', 'Travel and Transportation', 3, 'expense', 'debit', 'operating', '5.52.5204'),
    (p_company_id, '5205', '접대비', 'Entertainment', 3, 'expense', 'debit', 'operating', '5.52.5205'),
    (p_company_id, '5206', '통신비', 'Communication', 3, 'expense', 'debit', 'operating', '5.52.5206'),
    (p_company_id, '5207', '수도광열비', 'Utilities', 3, 'expense', 'debit', 'operating', '5.52.5207'),
    (p_company_id, '5208', '세금과공과', 'Taxes and Dues', 3, 'expense', 'debit', 'operating', '5.52.5208'),
    (p_company_id, '5209', '감가상각비', 'Depreciation', 3, 'expense', 'debit', 'operating', '5.52.5209'),
    (p_company_id, '5210', '지급임차료', 'Rent', 3, 'expense', 'debit', 'operating', '5.52.5210'),
    (p_company_id, '5211', '보험료', 'Insurance', 3, 'expense', 'debit', 'operating', '5.52.5211'),
    (p_company_id, '5212', '차량유지비', 'Vehicle Maintenance', 3, 'expense', 'debit', 'operating', '5.52.5212'),
    (p_company_id, '5213', '운반비', 'Shipping', 3, 'expense', 'debit', 'operating', '5.52.5213'),
    (p_company_id, '5214', '소모품비', 'Supplies', 3, 'expense', 'debit', 'operating', '5.52.5214'),
    (p_company_id, '5215', '도서인쇄비', 'Books and Printing', 3, 'expense', 'debit', 'operating', '5.52.5215'),
    (p_company_id, '5216', '수선비', 'Repairs', 3, 'expense', 'debit', 'operating', '5.52.5216'),
    (p_company_id, '5217', '광고선전비', 'Advertising', 3, 'expense', 'debit', 'operating', '5.52.5217'),
    (p_company_id, '5218', '지급수수료', 'Commissions', 3, 'expense', 'debit', 'operating', '5.52.5218'),
    (p_company_id, '5219', '대손상각비', 'Bad Debt Expense', 3, 'expense', 'debit', 'operating', '5.52.5219'),
    (p_company_id, '5220', '무형자산상각비', 'Amortization', 3, 'expense', 'debit', 'operating', '5.52.5220'),
    (p_company_id, '5221', '잡비', 'Miscellaneous Expenses', 3, 'expense', 'debit', 'operating', '5.52.5221'),
    (p_company_id, '53', '영업외비용', 'Non-operating Expenses', 2, 'expense', 'debit', 'non_operating', '5.53'),
    (p_company_id, '5301', '이자비용', 'Interest Expense', 3, 'expense', 'debit', 'non_operating', '5.53.5301'),
    (p_company_id, '5302', '외환차손', 'Foreign Exchange Losses', 3, 'expense', 'debit', 'non_operating', '5.53.5302'),
    (p_company_id, '5303', '유형자산처분손실', 'Loss on Disposal of PPE', 3, 'expense', 'debit', 'non_operating', '5.53.5303'),
    (p_company_id, '5304', '기부금', 'Donations', 3, 'expense', 'debit', 'non_operating', '5.53.5304'),
    (p_company_id, '5305', '잡손실', 'Miscellaneous Losses', 3, 'expense', 'debit', 'non_operating', '5.53.5305'),
    (p_company_id, '54', '법인세비용', 'Income Tax Expense', 2, 'expense', 'debit', 'tax', '5.54'),
    (p_company_id, '5401', '법인세비용', 'Corporate Income Tax', 3, 'expense', 'debit', 'tax', '5.54.5401');

    -- Update parent_id references
    UPDATE accounts a
    SET parent_id = p.id
    FROM accounts p
    WHERE a.company_id = p_company_id
      AND p.company_id = p_company_id
      AND a.level > 1
      AND p.code = LEFT(a.code, LENGTH(a.code) - 2)
      AND a.parent_id IS NULL;

END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION create_standard_accounts IS 'Creates standard K-IFRS chart of accounts for a new company';
