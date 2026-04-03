import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useRawInputStore } from '@/stores/rawinputStore'
import { rawinputService } from '@/services/rawinputService'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/rawinputService', () => ({
  rawinputService: {
    list: vi.fn(),
    reprocess: vi.fn(),
  },
}))

const makeItem = (overrides = {}) => ({
  id: 'id-1',
  sourceType: 'voice',
  status: 'failed',
  rawContent: 'test',
  retryCount: 3,
  maxRetries: 5,
  createdAt: '2026-04-03T00:00:00Z',
  ...overrides,
})

describe('useRawInputStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchList populates items and total', async () => {
    vi.mocked(rawinputService.list).mockResolvedValue({
      items: [makeItem()],
      total: 1,
      page: 1,
      rowsPerPage: 25,
    })
    const store = useRawInputStore()
    await store.fetchList()
    expect(store.items).toHaveLength(1)
    expect(store.total).toBe(1)
  })

  it('reprocess calls service and refreshes list', async () => {
    const updated = makeItem({ status: 'pending', retryCount: 0 })
    vi.mocked(rawinputService.reprocess).mockResolvedValue(updated)
    vi.mocked(rawinputService.list).mockResolvedValue({
      items: [updated],
      total: 1,
      page: 1,
      rowsPerPage: 25,
    })
    const store = useRawInputStore()
    await store.reprocess('id-1')
    expect(rawinputService.reprocess).toHaveBeenCalledWith('id-1')
    expect(rawinputService.list).toHaveBeenCalled()
  })

  it('setStatusFilter resets page to 1 and fetches with filter', async () => {
    vi.mocked(rawinputService.list).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      rowsPerPage: 25,
    })
    const store = useRawInputStore()
    store.page = 3
    await store.setStatusFilter('failed')
    expect(store.statusFilter).toBe('failed')
    expect(store.page).toBe(1)
    expect(rawinputService.list).toHaveBeenCalledWith(
      expect.objectContaining({ status: 'failed', page: 1 }),
    )
  })
})
