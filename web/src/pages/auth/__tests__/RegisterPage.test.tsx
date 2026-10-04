import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { RegisterPage } from '../RegisterPage';

describe('RegisterPage', () => {
  // Step 2 has required/type="email" inputs. Without noValidate the browser's
  // constraint check blocks submit, so the zod messages never render.
  it('shows the app validation messages when step 2 is submitted empty', async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <RegisterPage />
      </MemoryRouter>
    );

    await user.type(screen.getByLabelText(/회사명/), '주식회사 테스트');
    await user.type(screen.getByLabelText(/사업자등록번호/), '123-45-67890');
    await user.click(screen.getByRole('button', { name: '다음' }));

    await user.click(await screen.findByRole('button', { name: '가입하기' }));

    expect(await screen.findByText('이름은 최소 2자 이상이어야 합니다.')).toBeInTheDocument();
    expect(screen.getByText('올바른 이메일 형식이 아닙니다.')).toBeInTheDocument();
  });
});
