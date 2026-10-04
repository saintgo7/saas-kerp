import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/__tests__/mocks/server';
import { accountsApi } from '@/api';

// The API caps page_size at 100; listAll must follow total_pages.
describe('accountsApi.listAll', () => {
  it('collects every page', async () => {
    server.use(
      http.get('/api/v1/accounts', ({ request }) => {
        const page = Number(new URL(request.url).searchParams.get('page') ?? 1);
        return HttpResponse.json({
          success: true,
          data: [{ id: `acc-${page}`, code: `${page}01`, name: `p${page}` }],
          meta: { pagination: { page, per_page: 100, total: 2, total_pages: 2 } },
        });
      })
    );

    const res = await accountsApi.listAll({ isActive: true });

    expect(res.data.items.map((a) => a.code)).toEqual(['101', '201']);
  });
});
