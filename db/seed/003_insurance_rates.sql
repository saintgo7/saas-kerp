-- K-ERP v0.2 Seed: 2026 Social Insurance Rates
-- Korean social insurance contribution rates

-- NPS (National Pension Service / 국민연금)
INSERT INTO social_insurance_rates (effective_year, effective_month, insurance_type, employee_rate, employer_rate, min_base, max_base, notes)
VALUES
(2026, 1, 'nps', 0.045, 0.045, 370000, 5900000, 'Total 9%, split 50/50');

-- NHIS (National Health Insurance / 건강보험)
INSERT INTO social_insurance_rates (effective_year, effective_month, insurance_type, employee_rate, employer_rate, min_base, max_base, notes)
VALUES
(2026, 1, 'nhis', 0.03545, 0.03545, NULL, NULL, 'Total 7.09%, split 50/50');

-- NHIS LTC (Long-term Care Insurance / 장기요양보험)
INSERT INTO social_insurance_rates (effective_year, effective_month, insurance_type, employee_rate, employer_rate, min_base, max_base, notes)
VALUES
(2026, 1, 'nhis_ltc', 0.004591, 0.004591, NULL, NULL, '12.95% of NHIS premium');

-- EI (Employment Insurance / 고용보험)
INSERT INTO social_insurance_rates (effective_year, effective_month, insurance_type, employee_rate, employer_rate, min_base, max_base, notes)
VALUES
(2026, 1, 'ei', 0.009, 0.009, NULL, NULL, 'Employee 0.9%, Employer varies by size');

-- WCI (Workers Compensation Insurance / 산재보험)
-- Rate varies by industry, using average rate
INSERT INTO social_insurance_rates (effective_year, effective_month, insurance_type, employee_rate, employer_rate, min_base, max_base, notes)
VALUES
(2026, 1, 'wci', 0, 0.0164, NULL, NULL, 'Employer only, rate varies by industry (avg 1.64%)');

-- Income Tax Tables for 2026
INSERT INTO income_tax_tables (effective_year, min_amount, max_amount, base_tax, tax_rate)
VALUES
(2026, 0, 14000000, 0, 0.06),
(2026, 14000000, 50000000, 840000, 0.15),
(2026, 50000000, 88000000, 6240000, 0.24),
(2026, 88000000, 150000000, 15360000, 0.35),
(2026, 150000000, 300000000, 37060000, 0.38),
(2026, 300000000, 500000000, 94060000, 0.40),
(2026, 500000000, 1000000000, 174060000, 0.42),
(2026, 1000000000, NULL, 384060000, 0.45);

COMMENT ON TABLE social_insurance_rates IS 'Rates for NPS, NHIS, EI, WCI';
COMMENT ON TABLE income_tax_tables IS 'Korean income tax brackets';
