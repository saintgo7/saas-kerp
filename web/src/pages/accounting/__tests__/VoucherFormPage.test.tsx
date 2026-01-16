import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { VoucherFormPage } from '../VoucherFormPage';

// Mock navigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// Wrapper component with Router
function renderVoucherFormPage() {
  return render(
    <BrowserRouter>
      <VoucherFormPage />
    </BrowserRouter>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mockNavigate.mockReset();
});

describe('VoucherFormPage', () => {
  // ==========================================================================
  // Rendering Tests
  // ==========================================================================

  describe('rendering', () => {
    it('should render page title and description', () => {
      renderVoucherFormPage();

      expect(screen.getByRole('heading', { name: '전표 작성' })).toBeInTheDocument();
      expect(screen.getByText('새로운 회계 전표를 작성합니다.')).toBeInTheDocument();
    });

    it('should render basic info section', () => {
      renderVoucherFormPage();

      expect(screen.getByText('기본 정보')).toBeInTheDocument();
      expect(screen.getByLabelText(/전표일자/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/적요/i)).toBeInTheDocument();
    });

    it('should render entry section with table headers', () => {
      renderVoucherFormPage();

      expect(screen.getByText('분개 입력')).toBeInTheDocument();
      expect(screen.getByText('계정과목')).toBeInTheDocument();
      expect(screen.getByText('차변')).toBeInTheDocument();
      expect(screen.getByText('대변')).toBeInTheDocument();
    });

    it('should render initial two entry rows', () => {
      renderVoucherFormPage();

      // Should have 2 account select dropdowns (initial entries)
      const accountSelects = screen.getAllByRole('combobox');
      expect(accountSelects).toHaveLength(2);
    });

    it('should render action buttons', () => {
      renderVoucherFormPage();

      expect(screen.getByRole('button', { name: /저장/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /취소/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /분개 추가/i })).toBeInTheDocument();
    });

    it('should show unbalanced badge initially', () => {
      renderVoucherFormPage();

      expect(screen.getByText('불균형')).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Entry Management Tests
  // ==========================================================================

  describe('entry management', () => {
    it('should add new entry row', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      const addButton = screen.getByRole('button', { name: /분개 추가/i });

      expect(screen.getAllByRole('combobox')).toHaveLength(2);

      await user.click(addButton);

      expect(screen.getAllByRole('combobox')).toHaveLength(3);
    });

    it('should remove entry row', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      // Add third entry first (need > 2 to enable delete)
      const addButton = screen.getByRole('button', { name: /분개 추가/i });
      await user.click(addButton);

      expect(screen.getAllByRole('combobox')).toHaveLength(3);

      // Find and click delete button (Trash icon)
      const deleteButtons = screen.getAllByRole('button').filter(
        (btn) => btn.querySelector('[data-lucide="trash-2"]') !== null ||
                 btn.className.includes('destructive')
      );

      // Click first delete button that's enabled
      const enabledDeleteBtn = deleteButtons.find(btn => !btn.hasAttribute('disabled'));
      if (enabledDeleteBtn) {
        await user.click(enabledDeleteBtn);
        expect(screen.getAllByRole('combobox')).toHaveLength(2);
      }
    });

    it('should disable delete when only 2 entries', () => {
      renderVoucherFormPage();

      // All delete buttons should be disabled when only 2 entries
      const deleteButtons = screen.getAllByRole('button').filter(
        (btn) => btn.querySelector('svg') !== null
      );

      // Find buttons with trash icon (destructive variant)
      const trashButtons = deleteButtons.filter(
        (btn) => btn.className.includes('destructive') ||
                 btn.querySelector('[class*="destructive"]')
      );

      trashButtons.forEach((btn) => {
        expect(btn).toBeDisabled();
      });
    });
  });

  // ==========================================================================
  // Balance Calculation Tests
  // ==========================================================================

  describe('balance calculation', () => {
    it('should show balanced badge when debit equals credit', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      // Get debit and credit inputs
      const numberInputs = screen.getAllByRole('spinbutton');
      const debitInput = numberInputs[0];
      const creditInput = numberInputs[1];

      // Enter balanced amounts
      await user.clear(debitInput);
      await user.type(debitInput, '10000');
      await user.clear(creditInput);
      await user.type(creditInput, '10000');

      await waitFor(() => {
        expect(screen.getByText('균형')).toBeInTheDocument();
      });
    });

    it('should show unbalanced badge when debit not equals credit', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      const numberInputs = screen.getAllByRole('spinbutton');
      const debitInput = numberInputs[0];
      const creditInput = numberInputs[1];

      await user.clear(debitInput);
      await user.type(debitInput, '10000');
      await user.clear(creditInput);
      await user.type(creditInput, '5000');

      await waitFor(() => {
        expect(screen.getByText('불균형')).toBeInTheDocument();
      });
    });

    it('should calculate totals correctly', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      // Add third entry
      await user.click(screen.getByRole('button', { name: /분개 추가/i }));

      const numberInputs = screen.getAllByRole('spinbutton');

      // Entry 1: Debit 10000
      await user.clear(numberInputs[0]);
      await user.type(numberInputs[0], '10000');

      // Entry 2: Credit 5000
      await user.clear(numberInputs[3]);
      await user.type(numberInputs[3], '5000');

      // Entry 3: Credit 5000
      await user.clear(numberInputs[5]);
      await user.type(numberInputs[5], '5000');

      await waitFor(() => {
        expect(screen.getByText('균형')).toBeInTheDocument();
      });
    });

    it('should show difference amount when unbalanced', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      const numberInputs = screen.getAllByRole('spinbutton');

      await user.clear(numberInputs[0]);
      await user.type(numberInputs[0], '10000');
      await user.clear(numberInputs[1]);
      await user.type(numberInputs[1], '3000');

      await waitFor(() => {
        expect(screen.getByText(/차이:/)).toBeInTheDocument();
      });
    });
  });

  // ==========================================================================
  // Form Validation Tests
  // ==========================================================================

  describe('form validation', () => {
    it('should show error for empty voucher date', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      const dateInput = screen.getByLabelText(/전표일자/i);
      await user.clear(dateInput);

      const submitButton = screen.getAllByRole('button', { name: /저장/i })[0];
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('전표일자를 입력하세요')).toBeInTheDocument();
      });
    });

    it('should show error for empty description', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      const submitButton = screen.getAllByRole('button', { name: /저장/i })[0];
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('적요를 입력하세요')).toBeInTheDocument();
      });
    });

    it('should show error for unbalanced entries', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      // Fill in basic info
      const descriptionInput = screen.getByLabelText(/적요/i);
      await user.type(descriptionInput, 'Test voucher');

      // Fill entries with unbalanced amounts
      const accountSelects = screen.getAllByRole('combobox');
      await user.click(accountSelects[0]);
      await user.click(screen.getByText('101 현금'));

      await user.click(accountSelects[1]);
      await user.click(screen.getByText('401 상품매출'));

      const numberInputs = screen.getAllByRole('spinbutton');
      await user.clear(numberInputs[0]);
      await user.type(numberInputs[0], '10000');
      await user.clear(numberInputs[1]);
      await user.type(numberInputs[1], '5000');

      const submitButton = screen.getAllByRole('button', { name: /저장/i })[0];
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('차변과 대변의 합계가 일치해야 합니다')).toBeInTheDocument();
      });
    });

    it('should show error for missing account code', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      const descriptionInput = screen.getByLabelText(/적요/i);
      await user.type(descriptionInput, 'Test voucher');

      // Don't select account codes, just enter amounts
      const numberInputs = screen.getAllByRole('spinbutton');
      await user.clear(numberInputs[0]);
      await user.type(numberInputs[0], '10000');
      await user.clear(numberInputs[3]);
      await user.type(numberInputs[3], '10000');

      const submitButton = screen.getAllByRole('button', { name: /저장/i })[0];
      await user.click(submitButton);

      await waitFor(() => {
        expect(screen.getByText('계정과목을 선택하세요')).toBeInTheDocument();
      });
    });
  });

  // ==========================================================================
  // Navigation Tests
  // ==========================================================================

  describe('navigation', () => {
    it('should navigate back on cancel click', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      const cancelButton = screen.getByRole('button', { name: /취소/i });
      await user.click(cancelButton);

      expect(mockNavigate).toHaveBeenCalledWith(-1);
    });

    it('should navigate back on back button click', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      // Find the back arrow button (first ghost button with icon)
      const buttons = screen.getAllByRole('button');
      const backButton = buttons.find(
        (btn) =>
          btn.className.includes('ghost') &&
          btn.querySelector('svg') !== null &&
          btn.getAttribute('type') !== 'submit'
      );

      if (backButton) {
        await user.click(backButton);
        expect(mockNavigate).toHaveBeenCalledWith(-1);
      }
    });
  });

  // ==========================================================================
  // Submission Tests
  // ==========================================================================

  describe('form submission', () => {
    it('should submit valid form and navigate', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      // Fill description
      const descriptionInput = screen.getByLabelText(/적요/i);
      await user.type(descriptionInput, 'Test voucher description');

      // Select accounts
      const accountSelects = screen.getAllByRole('combobox');
      await user.click(accountSelects[0]);
      await user.click(screen.getByText('101 현금'));

      await user.click(accountSelects[1]);
      await user.click(screen.getByText('401 상품매출'));

      // Enter balanced amounts
      const numberInputs = screen.getAllByRole('spinbutton');
      await user.clear(numberInputs[0]);
      await user.type(numberInputs[0], '10000');
      await user.clear(numberInputs[3]);
      await user.type(numberInputs[3], '10000');

      // Submit
      const submitButton = screen.getAllByRole('button', { name: /저장/i })[0];
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/accounting/voucher');
      });
    });
  });

  // ==========================================================================
  // Account Selection Tests
  // ==========================================================================

  describe('account selection', () => {
    it('should display account options', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      const accountSelects = screen.getAllByRole('combobox');
      await user.click(accountSelects[0]);

      expect(screen.getByText('101 현금')).toBeInTheDocument();
      expect(screen.getByText('102 보통예금')).toBeInTheDocument();
      expect(screen.getByText('401 상품매출')).toBeInTheDocument();
      expect(screen.getByText('501 상품매입')).toBeInTheDocument();
    });

    it('should select account and update display', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      const accountSelects = screen.getAllByRole('combobox');
      await user.click(accountSelects[0]);
      await user.click(screen.getByText('101 현금'));

      // The select should now show the selected value
      expect(accountSelects[0]).toHaveTextContent('101 현금');
    });
  });

  // ==========================================================================
  // Date Input Tests
  // ==========================================================================

  describe('date input', () => {
    it('should have today as default date', () => {
      renderVoucherFormPage();

      const dateInput = screen.getByLabelText(/전표일자/i) as HTMLInputElement;
      const today = new Date().toISOString().split('T')[0];

      expect(dateInput.value).toBe(today);
    });

    it('should allow date change', async () => {
      const user = userEvent.setup();
      renderVoucherFormPage();

      const dateInput = screen.getByLabelText(/전표일자/i) as HTMLInputElement;
      await user.clear(dateInput);
      await user.type(dateInput, '2026-01-15');

      expect(dateInput.value).toBe('2026-01-15');
    });
  });
});
