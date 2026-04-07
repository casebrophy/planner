import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ActivityHistory from '@/components/shared/ActivityHistory.vue'

vi.mock('@/services/activityLogService', () => ({
  activityLogService: {
    list: vi.fn(),
  },
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mockListResult(items: any[] = [], total?: number) {
  return { items, total: total ?? items.length, page: 1, rowsPerPage: 20 }
}

function makeActivityLog(overrides: Record<string, unknown> = {}) {
  return {
    id: 'log-1',
    subjectType: 'task',
    subjectId: 'task-1',
    value: 'completed',
    loggedAt: new Date(Date.now() - 3600000).toISOString(),
    createdAt: new Date(Date.now() - 3600000).toISOString(),
    ...overrides,
  }
}

describe('ActivityHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows loading state while fetching', async () => {
    const { activityLogService } = await import('@/services/activityLogService')

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    let resolveList!: (value: any) => void
    vi.mocked(activityLogService.list).mockReturnValue(
      new Promise((resolve) => {
        resolveList = resolve
      }),
    )

    const wrapper = mount(ActivityHistory, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Loading...')

    resolveList(mockListResult())
    wrapper.unmount()
  })

  it('shows empty state when no entries', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    vi.mocked(activityLogService.list).mockResolvedValue(mockListResult())

    const wrapper = mount(ActivityHistory, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('No activity logged yet.')
    wrapper.unmount()
  })

  it('renders activity entries when data is available', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    const entries = [
      makeActivityLog({ id: 'log-1', value: 'completed task' }),
      makeActivityLog({ id: 'log-2', value: 'reviewed' }),
    ]
    vi.mocked(activityLogService.list).mockResolvedValue(mockListResult(entries, 2))

    const wrapper = mount(ActivityHistory, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('completed task')
    expect(wrapper.text()).toContain('reviewed')
    wrapper.unmount()
  })

  it('shows "logged" for entries with no value', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    const entries = [makeActivityLog({ value: null })]
    vi.mocked(activityLogService.list).mockResolvedValue(mockListResult(entries, 1))

    const wrapper = mount(ActivityHistory, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('logged')
    wrapper.unmount()
  })

  it('shows relative timestamp for each entry', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    const entries = [makeActivityLog({ loggedAt: new Date(Date.now() - 3600000).toISOString() })]
    vi.mocked(activityLogService.list).mockResolvedValue(mockListResult(entries, 1))

    const wrapper = mount(ActivityHistory, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    await flushPromises()

    expect(wrapper.text()).toMatch(/hour ago|hours ago|minutes ago/)
    wrapper.unmount()
  })

  it('shows error message when fetch fails', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    vi.mocked(activityLogService.list).mockRejectedValue(new Error('Network error'))

    const wrapper = mount(ActivityHistory, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Network error')
    wrapper.unmount()
  })

  it('shows a generic error when a non-Error is thrown', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    vi.mocked(activityLogService.list).mockRejectedValue('unexpected')

    const wrapper = mount(ActivityHistory, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Failed to load activity')
    wrapper.unmount()
  })

  it('renders the "Activity History" heading', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    vi.mocked(activityLogService.list).mockResolvedValue(mockListResult())

    const wrapper = mount(ActivityHistory, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Activity History')
    wrapper.unmount()
  })

  it('calls list with the correct subject filter', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    vi.mocked(activityLogService.list).mockResolvedValue(mockListResult())

    mount(ActivityHistory, {
      props: { subjectType: 'context', subjectId: 'ctx-42' },
    })

    await flushPromises()

    expect(activityLogService.list).toHaveBeenCalledWith(
      expect.objectContaining({
        filter: expect.objectContaining({
          subjectType: 'context',
          subjectId: 'ctx-42',
        }),
      }),
    )
  })

  it('exposes a reload method that re-fetches data', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    vi.mocked(activityLogService.list).mockResolvedValue(mockListResult())

    const wrapper = mount(ActivityHistory, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    await flushPromises()
    expect(activityLogService.list).toHaveBeenCalledTimes(1)

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await (wrapper.vm as any).reload()
    await flushPromises()

    expect(activityLogService.list).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })
})
