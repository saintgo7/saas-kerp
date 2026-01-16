import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  // ==========================================================================
  // Login Page Tests
  // ==========================================================================

  test.describe('Login Page', () => {
    test('should display login form', async ({ page }) => {
      await page.goto('/login');

      await expect(page.getByRole('heading', { name: '로그인' })).toBeVisible();
      await expect(page.getByLabel(/이메일/)).toBeVisible();
      await expect(page.getByLabel(/비밀번호/)).toBeVisible();
      await expect(page.getByRole('button', { name: '로그인' })).toBeVisible();
    });

    test('should show validation errors for empty form', async ({ page }) => {
      await page.goto('/login');

      await page.getByRole('button', { name: '로그인' }).click();

      // Wait for validation errors
      await expect(page.getByText('올바른 이메일 형식이 아닙니다.')).toBeVisible({
        timeout: 5000,
      });
    });

    test('should show error for invalid email format', async ({ page }) => {
      await page.goto('/login');

      await page.getByLabel(/이메일/).fill('invalid-email');
      await page.getByLabel(/비밀번호/).fill('password123');
      await page.getByRole('button', { name: '로그인' }).click();

      await expect(page.getByText('올바른 이메일 형식이 아닙니다.')).toBeVisible();
    });

    test('should show error for short password', async ({ page }) => {
      await page.goto('/login');

      await page.getByLabel(/이메일/).fill('test@example.com');
      await page.getByLabel(/비밀번호/).fill('short');
      await page.getByRole('button', { name: '로그인' }).click();

      await expect(
        page.getByText('비밀번호는 최소 8자 이상이어야 합니다.')
      ).toBeVisible();
    });

    test('should toggle password visibility', async ({ page }) => {
      await page.goto('/login');

      const passwordInput = page.getByLabel(/비밀번호/);
      await passwordInput.fill('mypassword123');

      // Initially password type
      await expect(passwordInput).toHaveAttribute('type', 'password');

      // Click toggle button (eye icon)
      await page.locator('button[type="button"]').first().click();

      // Should now be text type
      await expect(passwordInput).toHaveAttribute('type', 'text');
    });

    test('should navigate to register page', async ({ page }) => {
      await page.goto('/login');

      await page.getByRole('link', { name: '회원가입' }).click();

      await expect(page).toHaveURL(/.*register/);
    });

    test('should navigate to forgot password page', async ({ page }) => {
      await page.goto('/login');

      await page.getByRole('link', { name: '비밀번호 찾기' }).click();

      await expect(page).toHaveURL(/.*forgot-password/);
    });
  });

  // ==========================================================================
  // Register Page Tests
  // ==========================================================================

  test.describe('Register Page', () => {
    test('should display registration form', async ({ page }) => {
      await page.goto('/register');

      await expect(page.getByRole('heading', { name: '회원가입' })).toBeVisible();
      await expect(page.getByLabel(/이메일/)).toBeVisible();
      await expect(page.getByLabel(/비밀번호/)).toBeVisible();
      await expect(page.getByLabel(/이름/)).toBeVisible();
      await expect(page.getByLabel(/회사명/)).toBeVisible();
      await expect(page.getByLabel(/사업자등록번호/)).toBeVisible();
    });

    test('should validate required fields', async ({ page }) => {
      await page.goto('/register');

      await page.getByRole('button', { name: '회원가입' }).click();

      // At least one validation error should appear
      await expect(page.locator('.text-destructive')).toHaveCount(1, {
        timeout: 5000,
      });
    });

    test('should navigate to login page', async ({ page }) => {
      await page.goto('/register');

      await page.getByRole('link', { name: '로그인' }).click();

      await expect(page).toHaveURL(/.*login/);
    });
  });

  // ==========================================================================
  // Protected Routes Tests
  // ==========================================================================

  test.describe('Protected Routes', () => {
    test('should redirect to login when accessing dashboard without auth', async ({
      page,
    }) => {
      await page.goto('/dashboard');

      // Should redirect to login
      await expect(page).toHaveURL(/.*login/);
    });

    test('should redirect to login when accessing accounting without auth', async ({
      page,
    }) => {
      await page.goto('/accounting/voucher');

      // Should redirect to login
      await expect(page).toHaveURL(/.*login/);
    });
  });

  // ==========================================================================
  // Session Tests (with mocked auth)
  // ==========================================================================

  test.describe('Session Management', () => {
    test.skip('should persist login state after page reload', async ({
      page,
    }) => {
      // This test requires setting up auth state
      // Skipped until we have proper auth mocking in E2E

      await page.goto('/login');

      // TODO: Mock successful login
      // await page.getByLabel(/이메일/).fill('test@example.com');
      // await page.getByLabel(/비밀번호/).fill('password123');
      // await page.getByRole('button', { name: '로그인' }).click();

      // await expect(page).toHaveURL(/.*dashboard/);

      // // Reload page
      // await page.reload();

      // // Should still be on dashboard
      // await expect(page).toHaveURL(/.*dashboard/);
    });
  });
});
