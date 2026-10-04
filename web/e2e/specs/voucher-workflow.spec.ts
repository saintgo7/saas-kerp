import { test, expect, Page } from '@playwright/test';

// These specs drive the real API: the voucher pages load accounts and the
// session is checked server-side, so a fake token in localStorage is torn down
// on the first 401. CI seeds a demo admin (db/seed/004_demo_company.sql) and
// passes its credentials in E2E_USER_EMAIL / E2E_USER_PASSWORD.
const E2E_USER_EMAIL = process.env.E2E_USER_EMAIL;
const E2E_USER_PASSWORD = process.env.E2E_USER_PASSWORD;
const hasBackendUser = !!E2E_USER_EMAIL && !!E2E_USER_PASSWORD;

// Account pickers are native <select>s with a "계정선택" placeholder; the
// header's 전표유형 select is a combobox too, so match on the placeholder.
function accountSelects(page: Page) {
  return page
    .locator('select')
    .filter({ has: page.locator('option', { hasText: '계정선택' }) });
}

async function login(page: Page) {
  await page.goto('/login');
  await page.getByLabel(/이메일/).fill(E2E_USER_EMAIL!);
  await page.getByLabel(/비밀번호/).fill(E2E_USER_PASSWORD!);
  await page.getByRole('button', { name: '로그인' }).click();
  await expect(page).toHaveURL(/.*dashboard/, { timeout: 10000 });
}

test.describe('Voucher Workflow', () => {
  // Needs a running API with a seeded user (see login above).
  test.skip(!hasBackendUser, 'E2E_USER_EMAIL/E2E_USER_PASSWORD not set');

  // ==========================================================================
  // Voucher Form Page Tests
  // ==========================================================================

  test.describe('Voucher Form Page', () => {
    test.beforeEach(async ({ page }) => {
      await login(page);
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
      await expect(accountSelects(page)).toHaveCount(2);
    });

    test('should add new entry row', async ({ page }) => {
      await page.goto('/accounting/voucher/new');

      await page.getByRole('button', { name: /분개 추가/ }).click();

      await expect(accountSelects(page)).toHaveCount(3);
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
      await accountSelects(page).first().selectOption({ label: '110101 현금' });
      await accountSelects(page).nth(1).selectOption({ label: '4101 상품매출' });

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
      await login(page);
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
      await login(page);
    });

    test('should create, view, and manage voucher', async ({ page }) => {
      // Unique per run so re-runs against the same database stay unambiguous.
      const description = `Test voucher - E2E ${Date.now()}`;

      // 1. Navigate to voucher list
      await page.goto('/accounting/voucher');

      // 2. Click create new voucher
      await page.getByRole('link', { name: /전표 작성|신규/ }).click();

      // 3. Fill voucher form
      await page.getByLabel(/적요/).fill(description);

      // Select accounts
      await accountSelects(page).first().selectOption({ label: '110101 현금' });
      await accountSelects(page).nth(1).selectOption({ label: '4101 상품매출' });

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
      await expect(page.getByText(description)).toBeVisible();
    });
  });
});

// ==========================================================================
// Accessibility Tests
// ==========================================================================

test.describe('Accessibility', () => {
  test.skip(!hasBackendUser, 'E2E_USER_EMAIL/E2E_USER_PASSWORD not set');

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('voucher form should have proper labels', async ({ page }) => {
    await page.goto('/accounting/voucher/new');

    // Check that important form fields have labels
    await expect(page.getByLabel(/전표일자/)).toBeVisible();
    await expect(page.getByLabel(/적요/)).toBeVisible();
  });

  test('voucher form should be keyboard navigable', async ({ page }) => {
    await page.goto('/accounting/voucher/new');
    // The route is lazy-loaded; tab only once the form is on screen.
    await expect(page.getByLabel(/전표일자/)).toBeVisible();

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
