import { test, expect, Page } from '@playwright/test';

// Helper function to login (to be used when auth is working)
async function login(page: Page, email: string, password: string) {
  await page.goto('/login');
  await page.getByLabel(/이메일/).fill(email);
  await page.getByLabel(/비밀번호/).fill(password);
  await page.getByRole('button', { name: '로그인' }).click();
  await expect(page).toHaveURL(/.*dashboard/, { timeout: 10000 });
}

// Helper to set auth state directly (for testing without actual login)
async function setAuthState(page: Page) {
  await page.evaluate(() => {
    const mockUser = {
      id: 'test-user-id',
      email: 'test@example.com',
      name: 'Test User',
      role: 'admin',
      companyId: 'test-company-id',
    };

    localStorage.setItem('kerp_access_token', 'test-access-token');
    localStorage.setItem('kerp_refresh_token', 'test-refresh-token');
    localStorage.setItem(
      'kerp_user',
      JSON.stringify({ state: { user: mockUser, isAuthenticated: true } })
    );
  });
}

test.describe('Voucher Workflow', () => {
  // Skip these tests when running in CI without backend
  test.skip(({ browserName }) => {
    return process.env.CI === 'true' && !process.env.BACKEND_URL;
  });

  // ==========================================================================
  // Voucher Form Page Tests
  // ==========================================================================

  test.describe('Voucher Form Page', () => {
    test.beforeEach(async ({ page }) => {
      await setAuthState(page);
    });

    test('should display voucher form', async ({ page }) => {
      await page.goto('/accounting/voucher/new');

      await expect(page.getByRole('heading', { name: '전표 작성' })).toBeVisible();
      await expect(page.getByText('기본 정보')).toBeVisible();
      await expect(page.getByText('분개 입력')).toBeVisible();
    });

    test('should have today as default date', async ({ page }) => {
      await page.goto('/accounting/voucher/new');

      const today = new Date().toISOString().split('T')[0];
      const dateInput = page.getByLabel(/전표일자/);

      await expect(dateInput).toHaveValue(today);
    });

    test('should show initial two entry rows', async ({ page }) => {
      await page.goto('/accounting/voucher/new');

      // Should have 2 account selects
      const accountSelects = page.getByRole('combobox');
      await expect(accountSelects).toHaveCount(2);
    });

    test('should add new entry row', async ({ page }) => {
      await page.goto('/accounting/voucher/new');

      await page.getByRole('button', { name: /분개 추가/ }).click();

      const accountSelects = page.getByRole('combobox');
      await expect(accountSelects).toHaveCount(3);
    });

    test('should show unbalanced badge initially', async ({ page }) => {
      await page.goto('/accounting/voucher/new');

      await expect(page.getByText('불균형')).toBeVisible();
    });

    test('should show balanced badge when debit equals credit', async ({
      page,
    }) => {
      await page.goto('/accounting/voucher/new');

      // Enter debit amount
      const debitInputs = page.locator('input[type="number"]');
      await debitInputs.first().fill('10000');

      // Enter credit amount (second row, credit column)
      await debitInputs.nth(3).fill('10000');

      await expect(page.getByText('균형')).toBeVisible();
    });

    test('should show difference when unbalanced', async ({ page }) => {
      await page.goto('/accounting/voucher/new');

      const debitInputs = page.locator('input[type="number"]');
      await debitInputs.first().fill('10000');
      await debitInputs.nth(3).fill('5000');

      await expect(page.getByText(/차이:/)).toBeVisible();
    });

    test('should validate required fields on submit', async ({ page }) => {
      await page.goto('/accounting/voucher/new');

      // Try to submit without filling required fields
      await page.getByRole('button', { name: /저장/ }).first().click();

      // Should show validation error for description
      await expect(page.getByText('적요를 입력하세요')).toBeVisible({
        timeout: 5000,
      });
    });

    test('should validate balance on submit', async ({ page }) => {
      await page.goto('/accounting/voucher/new');

      // Fill description
      await page.getByLabel(/적요/).fill('Test voucher');

      // Select accounts
      await page.getByRole('combobox').first().click();
      await page.getByText('101 현금').click();

      await page.getByRole('combobox').nth(1).click();
      await page.getByText('401 상품매출').click();

      // Enter unbalanced amounts
      const numberInputs = page.locator('input[type="number"]');
      await numberInputs.first().fill('10000');
      await numberInputs.nth(3).fill('5000');

      // Submit
      await page.getByRole('button', { name: /저장/ }).first().click();

      // Should show balance error
      await expect(
        page.getByText('차변과 대변의 합계가 일치해야 합니다')
      ).toBeVisible({ timeout: 5000 });
    });

    test('should navigate back on cancel', async ({ page }) => {
      await page.goto('/accounting/voucher/new');

      const currentUrl = page.url();

      await page.getByRole('button', { name: /취소/ }).click();

      // Should navigate back (URL should change)
      await expect(page).not.toHaveURL(currentUrl);
    });
  });

  // ==========================================================================
  // Voucher List Page Tests
  // ==========================================================================

  test.describe('Voucher List Page', () => {
    test.beforeEach(async ({ page }) => {
      await setAuthState(page);
    });

    test('should display voucher list page', async ({ page }) => {
      await page.goto('/accounting/voucher');

      await expect(
        page.getByRole('heading', { name: /전표/ }).first()
      ).toBeVisible();
    });

    test('should have create voucher button', async ({ page }) => {
      await page.goto('/accounting/voucher');

      await expect(
        page.getByRole('link', { name: /전표 작성|신규/ })
      ).toBeVisible();
    });

    test('should navigate to voucher form on create click', async ({
      page,
    }) => {
      await page.goto('/accounting/voucher');

      await page.getByRole('link', { name: /전표 작성|신규/ }).click();

      await expect(page).toHaveURL(/.*voucher.*new/);
    });
  });

  // ==========================================================================
  // Complete Voucher Workflow Tests
  // ==========================================================================

  test.describe('Complete Workflow', () => {
    test.beforeEach(async ({ page }) => {
      await setAuthState(page);
    });

    test.skip('should create, view, and manage voucher', async ({ page }) => {
      // This test requires backend API
      // Skipped until backend is available

      // 1. Navigate to voucher list
      await page.goto('/accounting/voucher');

      // 2. Click create new voucher
      await page.getByRole('link', { name: /전표 작성|신규/ }).click();

      // 3. Fill voucher form
      await page.getByLabel(/적요/).fill('Test voucher - E2E');

      // Select accounts
      await page.getByRole('combobox').first().click();
      await page.getByText('101 현금').click();

      await page.getByRole('combobox').nth(1).click();
      await page.getByText('401 상품매출').click();

      // Enter balanced amounts
      const numberInputs = page.locator('input[type="number"]');
      await numberInputs.first().fill('50000');
      await numberInputs.nth(3).fill('50000');

      // Verify balanced
      await expect(page.getByText('균형')).toBeVisible();

      // 4. Submit voucher
      await page.getByRole('button', { name: /저장/ }).first().click();

      // 5. Should navigate to list
      await expect(page).toHaveURL(/.*accounting.*voucher/);

      // 6. New voucher should appear in list
      await expect(page.getByText('Test voucher - E2E')).toBeVisible();
    });
  });
});

// ==========================================================================
// Accessibility Tests
// ==========================================================================

test.describe('Accessibility', () => {
  test('voucher form should have proper labels', async ({ page }) => {
    // Set auth state
    await page.evaluate(() => {
      localStorage.setItem('kerp_access_token', 'test-token');
    });

    await page.goto('/accounting/voucher/new');

    // Check that important form fields have labels
    await expect(page.getByLabel(/전표일자/)).toBeVisible();
    await expect(page.getByLabel(/적요/)).toBeVisible();
  });

  test('voucher form should be keyboard navigable', async ({ page }) => {
    // Set auth state
    await page.evaluate(() => {
      localStorage.setItem('kerp_access_token', 'test-token');
    });

    await page.goto('/accounting/voucher/new');

    // Tab through form elements
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');

    // Should be able to focus inputs
    const focusedElement = await page.evaluate(() =>
      document.activeElement?.tagName.toLowerCase()
    );
    expect(['input', 'textarea', 'button', 'select', 'a']).toContain(
      focusedElement
    );
  });
});
