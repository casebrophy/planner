import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import StreakDisplay from '@/components/shared/StreakDisplay.vue'

vi.mock('@/services/activityLogService', () => ({
  activityLogService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    getStreaks: vi.fn(),
  },
}))

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), dismiss: vi.fn() }),
}))

describe('StreakDisplay', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders nothing when no streak data is available', () => {
    const wrapper = mount(StreakDisplay, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    expect(wrapper.find('.flex').exists()).toBe(false)
  })

  it('displays current, longest, and total streak values when streak is loaded', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    vi.mocked(activityLogService.getStreaks).mockResolvedValue({
      current: 5,
      longest: 12,
      totalCount: 42,
    })

    const { useActivityLogStore } = await import('@/stores/activityLogStore')
    const store = useActivityLogStore()

    // Manually set streak data on the store after mount so the computed updates
    const wrapper = mount(StreakDisplay, {
      props: { subjectType: 'task', subjectId: 'task-1' },
    })

    store.streaks['task:task-1'] = { current: 5, longest: 12, totalCount: 42 }
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('5')
    expect(wrapper.text()).toContain('12')
    expect(wrapper.text()).toContain('42')
    expect(wrapper.text()).toContain('current')
    expect(wrapper.text()).toContain('best')
    expect(wrapper.text()).toContain('total')
    wrapper.unmount()
  })

  it('calls fetchStreaks with the correct subject type and id on mount', async () => {
    const { activityLogService } = await import('@/services/activityLogService')
    vi.mocked(activityLogService.getStreaks).mockResolvedValue({
      current: 1,
      longest: 3,
      totalCount: 10,
    })

    mount(StreakDisplay, {
      props: { subjectType: 'context', subjectId: 'ctx-99' },
    })

    // Allow onMounted to run
    await new Promise((r) => setTimeout(r, 0))

    expect(activityLogService.getStreaks).toHaveBeenCalledWith('context', 'ctx-99')
  })
})
