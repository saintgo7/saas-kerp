-- K-ERP v0.2 Seed: System Permissions
-- These are global permissions, not tenant-specific

INSERT INTO permissions (code, name, module, description) VALUES
-- Company management
('company.view', 'View Company Info', 'company', 'View company profile and settings'),
('company.edit', 'Edit Company Info', 'company', 'Edit company profile and settings'),

-- User management
('user.view', 'View Users', 'user', 'View user list and profiles'),
('user.create', 'Create Users', 'user', 'Create new users'),
('user.edit', 'Edit Users', 'user', 'Edit user information'),
('user.delete', 'Delete Users', 'user', 'Deactivate or delete users'),
('user.role', 'Manage User Roles', 'user', 'Assign roles to users'),

-- Role management
('role.view', 'View Roles', 'role', 'View roles and permissions'),
('role.create', 'Create Roles', 'role', 'Create new roles'),
('role.edit', 'Edit Roles', 'role', 'Edit role permissions'),
('role.delete', 'Delete Roles', 'role', 'Delete roles'),

-- Accounting - Accounts
('account.view', 'View Chart of Accounts', 'accounting', 'View account list'),
('account.create', 'Create Accounts', 'accounting', 'Create new accounts'),
('account.edit', 'Edit Accounts', 'accounting', 'Edit account information'),
('account.delete', 'Deactivate Accounts', 'accounting', 'Deactivate accounts'),

-- Accounting - Vouchers
('voucher.view', 'View Vouchers', 'accounting', 'View journal entries'),
('voucher.create', 'Create Vouchers', 'accounting', 'Create journal entries'),
('voucher.edit', 'Edit Vouchers', 'accounting', 'Edit draft vouchers'),
('voucher.delete', 'Delete Vouchers', 'accounting', 'Delete draft vouchers'),
('voucher.approve', 'Approve Vouchers', 'accounting', 'Approve pending vouchers'),
('voucher.post', 'Post Vouchers', 'accounting', 'Post approved vouchers to ledger'),
('voucher.reverse', 'Reverse Vouchers', 'accounting', 'Create reversal entries'),

-- Accounting - Reports
('report.trial_balance', 'View Trial Balance', 'report', 'View trial balance report'),
('report.ledger', 'View General Ledger', 'report', 'View account ledger'),
('report.income_statement', 'View Income Statement', 'report', 'View profit and loss'),
('report.balance_sheet', 'View Balance Sheet', 'report', 'View financial position'),
('report.cash_flow', 'View Cash Flow', 'report', 'View cash flow statement'),

-- Tax Invoices
('invoice.view', 'View Invoices', 'invoice', 'View tax invoices'),
('invoice.create', 'Create Invoices', 'invoice', 'Create tax invoices'),
('invoice.edit', 'Edit Invoices', 'invoice', 'Edit draft invoices'),
('invoice.delete', 'Delete Invoices', 'invoice', 'Delete draft invoices'),
('invoice.issue', 'Issue Invoices', 'invoice', 'Issue to National Tax Service'),
('invoice.scrape', 'Scrape Invoices', 'invoice', 'Scrape from Hometax'),

-- Partners
('partner.view', 'View Partners', 'partner', 'View trading partners'),
('partner.create', 'Create Partners', 'partner', 'Create partners'),
('partner.edit', 'Edit Partners', 'partner', 'Edit partner information'),
('partner.delete', 'Delete Partners', 'partner', 'Delete partners'),

-- HR - Employees
('employee.view', 'View Employees', 'hr', 'View employee list'),
('employee.view_salary', 'View Employee Salary', 'hr', 'View salary information'),
('employee.view_personal', 'View Personal Info', 'hr', 'View sensitive personal info'),
('employee.create', 'Create Employees', 'hr', 'Create new employees'),
('employee.edit', 'Edit Employees', 'hr', 'Edit employee information'),
('employee.delete', 'Terminate Employees', 'hr', 'Process termination'),

-- HR - Organization
('department.view', 'View Departments', 'hr', 'View department structure'),
('department.manage', 'Manage Departments', 'hr', 'Create/edit/delete departments'),
('position.view', 'View Positions', 'hr', 'View position list'),
('position.manage', 'Manage Positions', 'hr', 'Create/edit/delete positions'),

-- Payroll
('payroll.view', 'View Payroll', 'payroll', 'View payroll records'),
('payroll.view_own', 'View Own Payroll', 'payroll', 'View own payslips'),
('payroll.calculate', 'Calculate Payroll', 'payroll', 'Run payroll calculation'),
('payroll.approve', 'Approve Payroll', 'payroll', 'Approve payroll for payment'),
('payroll.pay', 'Execute Payment', 'payroll', 'Execute bank transfers'),

-- Insurance
('insurance.view', 'View Insurance', 'insurance', 'View insurance records'),
('insurance.report', 'Submit Reports', 'insurance', 'Submit insurance reports'),
('insurance.manage', 'Manage Insurance', 'insurance', 'Manage credentials and settings'),

-- System
('system.audit', 'View Audit Logs', 'system', 'View system audit trail'),
('system.settings', 'System Settings', 'system', 'Manage system configuration'),
('system.backup', 'System Backup', 'system', 'Perform data backup/restore')

ON CONFLICT (code) DO NOTHING;
